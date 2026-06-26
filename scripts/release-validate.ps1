param(
    [int]$Port = 18081,
    [string]$MqttBrokerUrl = "",
    [switch]$SkipRace
)

$ErrorActionPreference = "Stop"

$RepoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$TempRoot = Join-Path ([System.IO.Path]::GetTempPath()) ("nexus-release-validate-" + [System.Guid]::NewGuid().ToString("N"))
$ConfigPath = Join-Path $RepoRoot "config-dev.yml"
$ConfigBackup = $null
$ServerProcess = $null
$BaseUrl = "http://127.0.0.1:$Port"

function Write-Step {
    param([string]$Message)
    Write-Host ""
    Write-Host "==> $Message"
}

function Invoke-Checked {
    param(
        [string]$FilePath,
        [string[]]$Arguments,
        [hashtable]$Environment = @{}
    )

    $oldValues = @{}
    foreach ($key in $Environment.Keys) {
        $oldValues[$key] = [Environment]::GetEnvironmentVariable($key, "Process")
        [Environment]::SetEnvironmentVariable($key, [string]$Environment[$key], "Process")
    }

    try {
        & $FilePath @Arguments
        if ($LASTEXITCODE -ne 0) {
            throw "$FilePath $($Arguments -join ' ') failed with exit code $LASTEXITCODE"
        }
    } finally {
        foreach ($key in $Environment.Keys) {
            [Environment]::SetEnvironmentVariable($key, $oldValues[$key], "Process")
        }
    }
}

function Invoke-JsonPost {
    param(
        [string]$Path,
        [hashtable]$Body
    )

    $json = $Body | ConvertTo-Json -Depth 20
    $response = Invoke-RestMethod -Method Post -Uri "$BaseUrl$Path" -ContentType "application/json" -Body $json
    if ($response.code -lt 200 -or $response.code -ge 300) {
        throw "POST $Path failed: code=$($response.code) message=$($response.message)"
    }
    return $response.data
}

function Wait-Healthy {
    $deadline = (Get-Date).AddSeconds(20)
    do {
        try {
            $response = Invoke-RestMethod -Method Get -Uri "$BaseUrl/health" -TimeoutSec 2
            if ($response.code -eq 200 -and $response.data.status -eq "ok") {
                return
            }
        } catch {
            Start-Sleep -Milliseconds 500
        }
    } while ((Get-Date) -lt $deadline)

    $stdout = Join-Path $TempRoot "server.out.log"
    $stderr = Join-Path $TempRoot "server.err.log"
    if (Test-Path -LiteralPath $stdout) {
        Write-Host "server stdout:"
        Get-Content -LiteralPath $stdout -Tail 40
    }
    if (Test-Path -LiteralPath $stderr) {
        Write-Host "server stderr:"
        Get-Content -LiteralPath $stderr -Tail 40
    }
    throw "server did not become healthy at $BaseUrl"
}

function Stop-Server {
    if ($script:ServerProcess -and -not $script:ServerProcess.HasExited) {
        Stop-Process -Id $script:ServerProcess.Id -Force
        $script:ServerProcess.WaitForExit(5000) | Out-Null
    }
    $script:ServerProcess = $null
}

function Start-Server {
    param([string]$BinaryPath)

    $stdout = Join-Path $TempRoot "server.out.log"
    $stderr = Join-Path $TempRoot "server.err.log"
    $script:ServerProcess = Start-Process -FilePath $BinaryPath `
        -WorkingDirectory $RepoRoot `
        -WindowStyle Hidden `
        -RedirectStandardOutput $stdout `
        -RedirectStandardError $stderr `
        -PassThru
    Wait-Healthy
}

try {
    Set-Location $RepoRoot
    New-Item -ItemType Directory -Path $TempRoot | Out-Null
    $ConfigBackup = Join-Path $TempRoot "config-dev.yml.bak"
    Copy-Item -LiteralPath $ConfigPath -Destination $ConfigBackup -Force

    Write-Step "Go tests"
    Invoke-Checked "go" @("test", "-count=1", "./...")

    if (-not $SkipRace) {
        Write-Step "Race tests"
        Invoke-Checked "go" @("test", "-race", "-count=1", "./...") @{ CGO_ENABLED = "1" }
    }

    Write-Step "Build server and demo products"
    $serverBin = Join-Path $TempRoot "nexus-core.exe"
    Invoke-Checked "go" @("build", "-o", $serverBin, ".")
    Invoke-Checked "go" @("build", "./cmd/demo-product", "./cmd/protocol-demo-product")

    Write-Step "Start temporary SQLite server"
    $dbPath = (Join-Path $TempRoot "release-validate.db").Replace("\", "/")
    $mqttClientID = "nexus-core-release-validate-" + [System.Guid]::NewGuid().ToString("N")
    $config = @"
port: $Port
swagger_enabled: true
auto_open_browser: false
swagger_url: /swagger/index.html
swagger_doc_url: /swagger/doc.json
db_list:
  default_db_name: test
  connect_list:
    - name: test
      db_type: sqlite
      db_path: "$dbPath"
      max_open_conns: 20
      max_idle_conns: 10
      conn_max_lifetime: 30
mqtt:
  broker_url: "$MqttBrokerUrl"
  client_id: "$mqttClientID"
  username: ""
  password: ""
  publish_timeout_seconds: 5
control:
  dispatch_timeout_seconds: 5
  dispatch_max_retries: 0
  node_online_ttl_seconds: 120
"@
    Set-Content -LiteralPath $ConfigPath -Value $config -Encoding utf8
    Start-Server $serverBin

    Write-Step "Swagger UI smoke check"
    $swagger = Invoke-WebRequest -Method Get -Uri "$BaseUrl/swagger/index.html" -UseBasicParsing
    if ($swagger.StatusCode -ne 200 -or $swagger.Content -notmatch "Swagger UI") {
        throw "Swagger UI smoke check failed"
    }
    $swaggerDoc = Invoke-WebRequest -Method Get -Uri "$BaseUrl/swagger/doc.json" -UseBasicParsing
    if ($swaggerDoc.Content -notmatch "Nexus Core API" -or $swaggerDoc.Content -notmatch "/access/register") {
        throw "Swagger doc content check failed"
    }

    Write-Step "Demo product E2E"
    Invoke-Checked "go" @(
        "run", "./cmd/demo-product",
        "-server", $BaseUrl,
        "-device", "release-demo-basic",
        "-heartbeats", "2",
        "-heartbeat-interval", "100ms"
    )

    Write-Step "Protocol conversion product E2E"
    Invoke-Checked "go" @(
        "run", "./cmd/protocol-demo-product",
        "-server", $BaseUrl,
        "-device", "release-demo-protocol"
    )

    Write-Step "Scheduled release restart recovery"
    $suffix = (Get-Date).ToString("yyyyMMddHHmmss")
    $product = Invoke-JsonPost "/products" @{
        name = "restart-release-product-$suffix"
        description = "release validation restart recovery"
    }
    $releaseDate = (Get-Date).ToUniversalTime().AddSeconds(4).ToString("o")
    $versionCode = "restart-$suffix"
    Invoke-JsonPost "/products/versions" @{
        product_id = [uint32]$product.id
        version_code = $versionCode
        release_method = 1
        release_date = $releaseDate
        description = "scheduled release restart recovery"
    } | Out-Null
    $license = Invoke-JsonPost "/licenses" @{
        product_id = [uint32]$product.id
        validity_hours = 24
        max_nodes = 1
        max_concurrent = 0
        remark = "restart recovery validation"
    }

    Stop-Server
    Start-Sleep -Seconds 6
    Start-Server $serverBin

    $registration = Invoke-JsonPost "/access/register" @{
        device_code = "release-restart-node"
        license_key = $license.license_key
        product_id = [uint32]$product.id
        version_code = $versionCode
    }
    if (-not $registration.node_id) {
        throw "scheduled release was not available after restart"
    }

    Write-Step "SQLite migration restart rehearsal"
    Stop-Server
    Start-Server $serverBin
    Wait-Healthy

    if ([string]::IsNullOrWhiteSpace($MqttBrokerUrl)) {
        Write-Host ""
        Write-Host "MQTT broker validation skipped: pass -MqttBrokerUrl tcp://host:1883 to enable a real broker config smoke run."
    } else {
        Write-Host ""
        Write-Host "MQTT broker configured for server startup: $MqttBrokerUrl"
        Write-Host "Note: command publish/subscribe acceptance still requires an external MQTT subscriber."
    }

    Write-Host ""
    Write-Host "release validation passed"
} finally {
    Stop-Server
    if ($ConfigBackup -and (Test-Path -LiteralPath $ConfigBackup)) {
        Copy-Item -LiteralPath $ConfigBackup -Destination $ConfigPath -Force
    }
    if (Test-Path -LiteralPath $TempRoot) {
        Remove-Item -LiteralPath $TempRoot -Recurse -Force
    }
}

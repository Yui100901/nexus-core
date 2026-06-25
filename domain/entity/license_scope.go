package entity

type ScopeStatus int

const (
	ScopeStatusEnabled ScopeStatus = iota + 1
	ScopeStatusDisabled
)

type LicenseProductScope struct {
	ID        uint
	LicenseID uint
	ProductID uint
	Status    ScopeStatus
}

type LicenseServiceScope struct {
	ID                uint
	LicenseID         uint
	ServiceIdentifier string
	Status            ScopeStatus
}

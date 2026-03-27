package service

import "nexus-core/persistence/repository"

//
// @Author yfy2001
// @Date 2026/3/26 14 52
//

var (
	productRepo        = repository.NewProductRepository()
	productVersionRepo = repository.NewProductVersionRepository()
	nodeRepo           = repository.NewNodeRepository()
	licenseRepo        = repository.NewLicenseRepository()
)

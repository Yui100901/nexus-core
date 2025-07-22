package repository

import (
	"nexus-core/persistence/mapper"
)

//
// @Author yfy2001
// @Date 2025/7/22 10 16
//

type DeviceRepository struct {
	dm *mapper.DeviceMapper
	am *mapper.AuthMapper
}

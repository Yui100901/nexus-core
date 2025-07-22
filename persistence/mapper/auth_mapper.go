package mapper

import (
	"nexus-core/persistence/base"
	"nexus-core/persistence/model"
)

//
// @Author yfy2001
// @Date 2025/7/22 10 27
//

type AuthMapper struct {
	*base.Mapper[model.Auth]
}

func NewAuthMapper() *AuthMapper {
	return &AuthMapper{
		Mapper: base.NewMapper[model.Auth](),
	}
}

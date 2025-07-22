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
	bm *base.Mapper[model.Auth]
}

func NewAuthMapper() *AuthMapper {
	return &AuthMapper{
		bm: base.NewMapper[model.Auth](),
	}
}

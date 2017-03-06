package valid

import (
	"errors"
	a "github.com/domac/mafio/agent"
)

var nullErr = errors.New("null data")

const ModuleName = "valid"

type DefaultFilterService struct {
	Ctx *a.Context
}

func New() *DefaultFilterService {
	return &DefaultFilterService{}
}

func (self *DefaultFilterService) SetContext(ctx *a.Context) {
	self.Ctx = ctx
}

//过滤
func (self *DefaultFilterService) DoFilter(data []byte) ([]byte, error) {
	if data == nil {
		return nil, nullErr
	}
	return data, nil
}

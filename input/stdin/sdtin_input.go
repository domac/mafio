package stdin

import (
	"fmt"
	a "github.com/domac/mafio/agent"
)

const ModuleName = "stdin"

//标准输入
type StdinInputService struct {
	ctx *a.Context
}

func New() *StdinInputService {
	return &StdinInputService{}
}

func (self *StdinInputService) SetContext(ctx *a.Context) {
	self.ctx = ctx
}

func (self *StdinInputService) StartInput() {
	for i := 0; i < 1; i++ {
		select {
		case self.ctx.Agentd.Inchan <- []byte(fmt.Sprintf("%d", i)):
		case <-self.ctx.Agentd.GetExitCh():
			goto exit
		}
	}
exit:
	self.ctx.Logger().Warning("input close")
}

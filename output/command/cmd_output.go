package command

import (
	a "github.com/domac/mafio/agent"
	p "github.com/domac/mafio/packet"
)

const ModuleName = "command"

type CommandOutputService struct {
	agentd *a.Context
}

func New() *CommandOutputService {
	return &CommandOutputService{}
}

func (self *CommandOutputService) SetContext(ctx *a.Context) {
	self.agentd = ctx
}

func (self *CommandOutputService) Reflesh() {

}

func (self *CommandOutputService) DoWrite(packets []*p.Packet) {

	for _, pp := range packets {
		println(string(pp.Data))
	}
}

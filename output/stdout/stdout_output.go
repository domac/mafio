package stdout

import (
	"fmt"
	a "github.com/domac/mafio/agent"
	p "github.com/domac/mafio/packet"
)

const ModuleName = "stdout"

type StdoutOutputService struct {
	agentd *a.Context
}

func New() *StdoutOutputService {
	return &StdoutOutputService{}
}

func (self *StdoutOutputService) SetContext(ctx *a.Context) {
	self.agentd = ctx
}

func (self *StdoutOutputService) Reflesh() {

}

func (self *StdoutOutputService) DoWrite(packets []*p.Packet) {

	for _, pp := range packets {
		fmt.Println("output :", string(pp.Data))
	}
}

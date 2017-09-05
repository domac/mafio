package command

import (
	a "github.com/domac/mafio/agent"
	p "github.com/domac/mafio/packet"
	"github.com/domac/mafio/util"
	"sync"
)

const ModuleName = "command"

//执行命令输出
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

//命令调用
func (self *CommandOutputService) cmdCall(cmd string, wg *sync.WaitGroup) {
	defer func() {
		wg.Done()
	}()

	if strings.HasPrefix(cmd, "\"") && strings.HasSuffix(cmd, "\"") {
		cmd = cmd[1 : len(cmd)-1]
	}

	self.agentd.Logger().Infof("[%s] start", cmd)

	cmds := []string{"sh", cmd}
	_, err := util.ScriptRun(cmds, 0)

	if err != nil {
		self.agentd.Logger().Error(err)
	}
	self.agentd.Logger().Infof("[%s] end", cmd)
}

func (self *CommandOutputService) DoWrite(packets []*p.Packet) {
	wg := sync.WaitGroup{}
	for _, pp := range packets {
		wg.Add(1)
		self.cmdCall(string(pp.Data), &wg)
	}
	wg.Wait()
}

package logrotator

import (
	a "github.com/domac/mafio/agent"
	p "github.com/domac/mafio/packet"
)

const ModuleName = "logr"

type LogROutputService struct {
	agentd *a.Context
	writer *RotatingWriter
}

func New() *LogROutputService {

	opts := &Options{
		RotateDaily: false,
		Compress:    true,
		MaximumSize: 1024 * 1024 * 1024, //1G
	}

	writer, _ := NewWriter("/tmp/log.txt", opts)

	return &LogROutputService{writer: writer}
}

func (self *LogROutputService) SetContext(ctx *a.Context) {
	self.agentd = ctx
}

func (self *LogROutputService) Reflesh() {

}

func (self *LogROutputService) DoWrite(packets []*p.Packet) {

	for _, pp := range packets {
		//println(string(pp.Data))
		self.writer.Write(pp.Data)
	}
}

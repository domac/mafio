package logrotator

import (
	a "github.com/domac/mafio/agent"
	p "github.com/domac/mafio/packet"
	"github.com/domac/mafio/util"
	"strings"
)

const ModuleName = "logr"

type LogROutputService struct {
	ctx    *a.Context
	writer *RotatingWriter
}

func New() *LogROutputService {
	return &LogROutputService{}
}

func (self *LogROutputService) SetContext(ctx *a.Context) {

	self.ctx = ctx

	//默认输出路径
	outputPath := "/tmp/dump.log"

	opts := &Options{
		RotateDaily: false,
		Compress:    true,
		MaximumSize: 1024 * 1024 * 1024, //1G
	}

	formatStr := self.ctx.Agentd.GetOptions().FormatStr
	formatStr = strings.TrimSpace(formatStr)

	//参数化配置
	if formatStr != "" {
		functionMap, err := util.JsonStringToMap(formatStr)
		if err != nil {
			self.ctx.Logger().Errorln(err)
		} else {
			//参数配置路径
			logr_path_config, snExist := functionMap["logr_path"]
			if snExist {
				outputPath = logr_path_config.(string)
			}
		}
	}

	ctx.Logger().Infof("logr output path: %s", outputPath)

	writer, _ := NewWriter(outputPath, opts)
	self.writer = writer
}

func (self *LogROutputService) Reflesh() {

}

func (self *LogROutputService) DoWrite(packets []*p.Packet) {

	for _, pp := range packets {
		//println(string(pp.Data))
		self.writer.Write(pp.Data)
	}
}

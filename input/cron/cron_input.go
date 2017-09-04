package cron

import (
	a "github.com/domac/mafio/agent"
	"github.com/domac/mafio/util"
	"github.com/robfig/cron"
	"os"
)

const ModuleName = "cron"

//文件输入服务
type CronInputService struct {
	ctx *a.Context
}

func New() *CronInputService {
	return &CronInputService{}
}

func (self *CronInputService) SetContext(ctx *a.Context) {
	self.ctx = ctx
}

func (self *CronInputService) Reflesh() {

}

//开启文件监听
func (self *CronInputService) StartInput() {
	self.ctx.Logger().Infof("start cron input service")
	cronTab := cron.New()
	cronTab.Start()

	configMap, ok := self.ctx.Agentd.GetOptions().PluginsConfigs[ModuleName]
	if !ok {
		self.ctx.Logger().Error("cron input config not found")
		os.Exit(2)
	}

	cron_map, ok := configMap["cron_map"]
	if !ok {
		self.ctx.Logger().Error("cron input config-cron_map not found")
		os.Exit(2)
	}

	//cron 作业信息
	cron_map_dict := cron_map.(map[string]interface{})
	for express, v := range cron_map_dict {
		jobs, _ := util.Interface2Stringslice(v)
		self.ctx.Logger().Infof("load job : %s", express)
		func(jobList []string) {
			cronTab.AddFunc(express, func() {
				for _, j := range jobList {
					self.ctx.Agentd.Inchan <- []byte(j)
				}
			})
		}(jobs)
	}

	select {
	case <-self.ctx.Agentd.GetExitCh():
		goto EXIT
	}
EXIT:
	self.ctx.Logger().Infoln("cron input exit")
}

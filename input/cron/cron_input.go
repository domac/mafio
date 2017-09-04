package cron

import (
	a "github.com/domac/mafio/agent"
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
	println("cron service")
}

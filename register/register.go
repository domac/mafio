package register

import (
	a "github.com/domac/mafio/agent"
	valid "github.com/domac/mafio/filter/default"
	fi "github.com/domac/mafio/input/file"
	"github.com/domac/mafio/input/stdin"
	"github.com/domac/mafio/input/tcpdump"
	"github.com/domac/mafio/output/logrotator"
	"github.com/domac/mafio/output/rabbitmq"
	"github.com/domac/mafio/output/stdout"
)

//插件注册初始化
func Init() {

	//---------- 注册输入插件
	a.RegistInput(fi.ModuleName, fi.New())
	a.RegistInput(stdin.ModuleName, stdin.New())
	a.RegistInput(tcpdump.ModuleName, tcpdump.New())

	//---------- 注册过滤器插件
	a.RegistFilter(valid.ModuleName, valid.New())

	//---------- 注册s输出插件
	a.RegistOutput(stdout.ModuleName, stdout.New())
	a.RegistOutput(rabbitmq.ModuleName, rabbitmq.New())
	a.RegistOutput(logrotator.ModuleName, logrotator.New())
}

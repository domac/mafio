package agent

import (
	p "github.com/domac/mafio/packet"
)

var (
	FilterServiceMap = map[string]FilterService{}
	InputServiceMap  = map[string]InputService{}
	OutputServiceMap = map[string]OutputService{}
)

func RegistFilter(name string, f FilterService) {
	FilterServiceMap[name] = f
}

func RegistInput(name string, i InputService) {
	InputServiceMap[name] = i
}

func RegistOutput(name string, o OutputService) {
	OutputServiceMap[name] = o
}

//输入服务接口
type InputService interface {
	SetContext(*Context)
	StartInput()
	Reflesh()
}

//输出服务接口
type OutputService interface {
	SetContext(*Context)
	DoWrite([]*p.Packet)
	Reflesh()
}

//过滤服务接口
type FilterService interface {
	SetContext(*Context)
	DoFilter([]byte) ([]byte, error)
}

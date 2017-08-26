package agent

import (
	"github.com/domac/mafio/version"
	"net"
	"os"
	"reflect"
	"sync"
)

//*****************************************
//
// agent后台服务进程 (Daemon Processor)
//
//*****************************************

type Agentd struct {
	sync.RWMutex                           //同步锁
	opts                      *Options     //配置参数选项
	httpListener              net.Listener //http监听器
	waitGroup                 WaitGroupWrapper
	messageCollectStartedChan chan int

	Inchan   chan []byte //数据输入通道
	Outchan  chan []byte //数据输出通道
	exitChan chan int

	isExit bool //退出标识
	paused bool //暂停标识
}

func convertArg(arg interface{}, kind reflect.Kind) (val reflect.Value, ok bool) {
	val = reflect.ValueOf(arg)
	if val.Kind() == kind {
		ok = true
	}
	return
}

//获取配置
func convertConfig(arg interface{}) (out []string, ok bool) {
	//类型转换
	slice, success := convertArg(arg, reflect.Slice)
	if !success {
		ok = false
		return
	}

	c := slice.Len()
	out = make([]string, c)
	for i := 0; i < c; i++ {
		tmp := slice.Index(i).Interface()
		if tmp != nil {
			//强制转换为字符串
			out[i] = tmp.(string)
		}
	}

	return out, true
}

//创建后台进程对象
func New(opts *Options, pluginsConf interface{}) *Agentd {

	//加载插件的配置信息
	configs, ok := convertConfig(pluginsConf)
	if ok {
		opts.LoadPluginsConf(configs)
	}

	a := &Agentd{
		opts:                      opts,
		exitChan:                  make(chan int),
		Inchan:                    make(chan []byte, opts.MaxReadChannelSize),
		Outchan:                   make(chan []byte, opts.MaxWriteChannelSize),
		messageCollectStartedChan: make(chan int),
		paused: false,
	}
	a.opts.Logger.Infof(version.Verbose("mafio"))
	return a
}

func (self *Agentd) GetOptions() *Options {
	return self.opts

}

// 清空agent的数据
// 输出通道和输入通道的消息会被立刻清理
func (self *Agentd) Empty() error {
	self.Lock()
	defer self.Unlock()
	for {
		select {
		case <-self.Outchan:
		case <-self.Inchan:
		default:
			goto finish
		}
	}
finish:
	return nil
}

func (self *Agentd) GetExitCh() chan int {
	return self.exitChan
}

//后台程序退出
func (self *Agentd) Exit() {
	self.opts.Logger.Warnf("agentd program is exiting ...")
	if self.httpListener != nil {
		self.httpListener.Close()
	}
	close(self.exitChan)
	close(self.Inchan)
	close(self.Outchan)
	self.isExit = true
	self.waitGroup.Wait()
}

//主程序入口
//Agent主要逻辑入口
func (self *Agentd) Main() {
	ctx := &Context{self}
	httpListener, err := net.Listen("tcp", self.opts.HTTPAddress)
	if err != nil {
		self.opts.Logger.Errorf("listen (%s) failed - %s", self.opts.HTTPAddress, err)
		os.Exit(1)
	}
	self.Lock()
	self.httpListener = httpListener
	self.Unlock()
	//开启自身的 api 服务端
	apiServer := newAPIServer(ctx)
	//开启对外提供的api服务
	self.waitGroup.Wrap(func() {
		Serve(self.httpListener, apiServer, "HTTP", self.opts.Logger)
	})

	//性能监控(使用influxDB)
	if self.opts.InfluxdbAddr != "" {
		self.waitGroup.Wrap(func() { ctx.monitor() })
	}

	//异步output处理
	self.waitGroup.Wrap(func() { ctx.messagesPush() })

	// messageCollectStartedCha用于同步输出与输入的流程
	// 这样可以保证输出器的初始化工作完成后,才进行数据采集的工作
	// 可以避免因为输出器因为某些原因无法工作,导致数据不断采集而无消费
	// 这样容易导致内存消息堆积,引起无法控制的情况
	<-self.messageCollectStartedChan

	//异步filer处理
	self.waitGroup.Wrap(func() { ctx.messagesFilted() })

	//异步intput处理
	self.waitGroup.Wrap(func() { ctx.messagePull() })
}

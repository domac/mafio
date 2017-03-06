package agent

import (
	pk "github.com/domac/mafio/packet"
	"math"
	"os"
	"time"
)

//ioloop主要是定义三大类型插件的执行方式

//消息拉取(input)
func (self *Context) messagePull() {

	//获取输入插件
	inputName := self.Agentd.opts.Input

	self.Logger().Infof("[INPUT]current input: <%s>", inputName)

	inputInstance, ok := InputServiceMap[inputName]
	if !ok {
		self.Logger().Errorln("no input found")
		os.Exit(1)
	}
	inputInstance.SetContext(self)
	inputInstance.StartInput()
}

//消息过滤(filter)
//从iput读入数据,并处理,最后把过滤后的数据丢到输出通道
func (self *Context) messagesFilted() {

	filterName := self.Agentd.opts.Filter

	self.Logger().Infof("[FILTER]current filter: <%s>", filterName)

	filterInstance, ok := FilterServiceMap[filterName]
	if !ok {
		self.Logger().Errorln("no filter found")
		os.Exit(1)
	}
	filterInstance.SetContext(self)
	for {
		select {
		case data, ok := <-self.Agentd.Inchan:
			if ok {
				d, err := filterInstance.DoFilter(data)
				if err == nil {
					self.Agentd.Outchan <- d
				}
			}
		case <-self.Agentd.exitChan:
			goto exit
		}
	}
exit:
	self.Logger().Warnln("filter is closing now")

}

//消息发送(output)
//消息输出的基础设施环境初始化优先
//这样可以最大限度降低消息积压
//因为如果负责消费输出的环境没初始化好,那些生产者输入器就会
//短时间制造很多数据,容易积压
func (self *Context) messagesPush() {

	maxWirteBulkSize := self.Agentd.opts.MaxWriteBulkSize
	//批量bulk
	packets := make([]*pk.Packet, 0, maxWirteBulkSize)

	outputName := self.Agentd.opts.Output

	self.Logger().Infof("[OUTPUT]current output: <%s>", outputName)

	outputInstance, ok := OutputServiceMap[outputName]
	if !ok {
		self.Logger().Errorln("no output found")
		os.Exit(1)
	}
	outputInstance.SetContext(self)
	//关闭messageCollectStartedChan, 宣告输出器的初始化工作已经完成
	//其它工作组件可以往下走
	close(self.Agentd.messageCollectStartedChan)
	for {
		select {
		case data, ok := <-self.Agentd.Outchan:
			if ok {
				pkg := pk.NewPacket(data)
				//output.DoWrite(pkg)
				packets = append(packets, pkg)

				//计算当前输出通道的实际需求大小
				chanlen := int(math.Min(float64(len(self.Agentd.Outchan)), float64(maxWirteBulkSize)))

				//如果channel的长度还有数据, 批量最多读取maxWirteBulkSize条数据,再合并写出
				//减少系统调用
				//减少网络传输, 提高资源利用率
				for i := 0; i < chanlen; i++ {
					p := <-self.Agentd.Outchan
					if nil != p {
						rpkg := pk.NewPacket(p)
						packets = append(packets, rpkg)
					}
				}

				//输出数据存在的情况下
				if len(packets) > 0 {
					outputInstance.DoWrite(packets)
					//回收包裹空间, 清理内存
					packets = packets[:0]
				}
			}
		case <-self.Agentd.exitChan:
			goto exit
		default:
			//让子弹飞一个会儿
			time.Sleep(400 * time.Millisecond)
		}
	}
exit:
	self.Logger().Warnln("output is closing now")
}

//性能监控
func (ctx *Context) monitor() {
	db_name := ctx.Agentd.opts.AgentGroup
	influxDB_addr := ctx.Agentd.opts.InfluxdbAddr
	ctx.Logger().Infof("[MONITOR]Ready to monitor with influxDB addr: <%s>, db: <%s>", influxDB_addr, db_name)
	DoMetrics(influxDB_addr, db_name, "", "", time.Second*5)
}

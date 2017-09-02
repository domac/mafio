package tcpdump

import (
	"errors"
	"fmt"
	a "github.com/domac/mafio/agent"
	"github.com/domac/mafio/util"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"github.com/google/gopacket/tcpassembly"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
)

//流量嗅探
const ModuleName = "tcpdump"

var ERR_EOF = errors.New("EOF")

//文件输入服务
type TcpDumpService struct {
	ctx              *a.Context
	PacketDataSource gopacket.PacketDataSource
	Decoder          gopacket.Decoder

	requestAssembler *tcpassembly.Assembler
	httpPorts        []string
	tcpPorts         []string
	targetProcPorts  []string
	portMap          map[string]string
	snaplen          int
	ttlPerMinutes    int

	quit chan bool
}

func New() *TcpDumpService {
	return &TcpDumpService{
		quit: make(chan bool),
	}
}

func (self *TcpDumpService) SetPacketDataSource(handle gopacket.PacketDataSource, decoder gopacket.Decoder) {
	self.PacketDataSource = handle
	self.Decoder = decoder
}

func (self *TcpDumpService) SetContext(ctx *a.Context) {
	self.ctx = ctx
}

func (self *TcpDumpService) Reflesh() {

}

//获取input的配置信息
func (self *TcpDumpService) GetInputConfigMap() (map[string]interface{}, bool) {
	configMap, ok := self.ctx.Agentd.GetOptions().PluginsConfigs[ModuleName]
	return configMap, ok
}

//开始执行输入
func (self *TcpDumpService) StartInput() {

	//初始化需要监控的端口
	err := self.initListenPort()

	if err != nil {
		self.ctx.Logger().Errorln(err)
		os.Exit(2)
	}

	//生成BPF表达式
	bpf := self.GenerateBpf()

	self.ctx.Logger().Infof("config http ports : %s", self.httpPorts)
	self.ctx.Logger().Infof("config tcp ports : %s", self.tcpPorts)
	self.ctx.Logger().Infof("config snaplen : %d", self.snaplen)
	self.ctx.Logger().Infof("config ttl per minutes : %d", self.ttlPerMinutes)
	self.ctx.Logger().Infof("config bpf : %s", bpf)

	//后台进程一直工作
	self.ctx.Logger().Infoln("daemon job start")
	err = self.startTcpDump(bpf)

	if err != nil {
		return
	}

	//存在TTL的情况
	if self.ttlPerMinutes > 0 {
		time.AfterFunc(time.Duration(self.ttlPerMinutes)*time.Second, func() {
			//退出
			close(self.quit)
		})
		<-self.quit
		self.ctx.Logger().Infoln("input exit now")
		os.Exit(2)
	}
}

//开始嗅探
func (self *TcpDumpService) startTcpDump(bpf string) error {

	deviceList := findDevices()

	// Set up assemblies
	requestStreamFactory := &httpStreamFactory{ctx: self.ctx, portMap: self.portMap, quitChan: self.quit}
	requestStreamPool := tcpassembly.NewStreamPool(requestStreamFactory)
	self.requestAssembler = tcpassembly.NewAssembler(requestStreamPool)

	for _, device := range deviceList {
		self.ctx.Logger().Infof("Net Device : %s", device)
		go self.startListen(device, bpf)
	}
	return nil
}

//开始监听
func (self *TcpDumpService) startListen(faceName string, filter string) {

	handle, err := pcap.OpenLive(faceName, int32(self.snaplen), true, 500)
	if err != nil {
		self.ctx.Logger().Fatal(err)
		return
	}

	if handle != nil {
		defer handle.Close()
	}

	if err := handle.SetBPFFilter(filter); err != nil {
		self.ctx.Logger().Fatal(err)
	}

	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())

	//设置包数据源相关参数
	self.SetPacketDataSource(handle, handle.LinkType())

	//获取数据包通道
	packetsChan := self.getPacketsChan(packetSource)

	//消费dump-packets通道的数据包
	ticker := time.Tick(time.Minute)
	for {
		select {
		case packet := <-packetsChan:
			err = self.processPacket(packet)
			if err != nil {
				return
			}
		case <-ticker:
			//每一分钟,自动刷新之前2分钟都处于不活跃的连接信息
			self.requestAssembler.FlushOlderThan(time.Now().Add(time.Minute * -2))
		}
	}

}

//设置监听端口
func (self *TcpDumpService) initListenPort() error {
	self.httpPorts = []string{"80"}
	self.tcpPorts = []string{}
	self.targetProcPorts = []string{}
	self.portMap = make(map[string]string)
	self.snaplen = 1600
	self.ttlPerMinutes = 10

	configMap, ok := self.GetInputConfigMap()

	//-f='{"logr_path":"/tmp/rr.log","snaplen":"65535","ttl_per_minutes":"15","http_ports":["80","443","8080","10029"],"tcp_ports":[],"target_processes":["kafka","rabbitmq","vms"]}'
	formatStr := self.ctx.Agentd.GetOptions().FormatStr
	formatStr = strings.TrimSpace(formatStr)

	//参数化配置
	if formatStr != "" {
		self.ctx.Logger().Infof("function string : %s", formatStr)
		functionMap, err := util.JsonStringToMap(formatStr)
		if err != nil {
			self.ctx.Logger().Errorln(err)
		} else {
			configMap = functionMap
		}
		ok = true
	}

	if ok {
		http_ports, isExist := configMap["http_ports"]
		if isExist {
			configHttpPorts, _ := convertConfig(http_ports)
			self.httpPorts = configHttpPorts
		}

		tcp_ports, isExist := configMap["tcp_ports"]
		if isExist {
			configTcpPorts, _ := convertConfig(tcp_ports)
			self.tcpPorts = configTcpPorts
		}

		//获取监控组件的信息
		target_processes, hasProcess := configMap["target_processes"]
		if hasProcess {
			target_processes_list, _ := convertConfig(target_processes)
			for _, processName := range target_processes_list {
				if processName == "" {
					continue
				}
				processPorts, err := getPortsByProcessName(processName)
				if err != nil {
					self.ctx.Logger().Infof(">>>>> target process [%s] found nothing about port", processName)
					continue
				}
				self.ctx.Logger().Infof(">>>>> target process [%s] found port: %s", processName, processPorts)
				for _, pp := range processPorts {
					self.portMap[pp] = processName
					self.targetProcPorts = append(self.targetProcPorts, pp)
				}

			}
		}

		tmp_snaplen, snExist := configMap["snaplen"]
		if snExist {
			tmpSnaplen := tmp_snaplen.(string)
			snaplen, _ := strconv.Atoi(tmpSnaplen)
			self.snaplen = snaplen
		}

		tmp_ttl, snExist := configMap["ttl_per_minutes"]
		if snExist {
			ttlStr := tmp_ttl.(string)
			//self.ttlPerMinutes = ttl
			ttlStr = strings.TrimSpace(ttlStr)
			if ttlStr == "" {
				self.ttlPerMinutes = 0
			} else {
				ttl, err := strconv.Atoi(ttlStr)
				if err != nil {
					ttl = 0
				}

				if ttl >= 50 {
					ttl = 50
				}
				self.ttlPerMinutes = ttl
			}
		}

	} else {
		return errors.New("no tcpdump input config found")
	}
	return nil
}

//获取dump包的数据通道
func (self *TcpDumpService) getPacketsChan(packetSource *gopacket.PacketSource) chan gopacket.Packet {
	sourcePacketsChannel := make(chan gopacket.Packet, 5000)
	go func() {
		for {
			//读当前需要处理的包
			packet, err := packetSource.NextPacket()
			if err == io.EOF {
				return
			} else if err == nil {
				sourcePacketsChannel <- packet
			}
		}
	}()
	return sourcePacketsChannel
}

//数据包处理
func (self *TcpDumpService) processPacket(packet gopacket.Packet) error {
	if packet == nil {
		return ERR_EOF
	}

	if !(packet.NetworkLayer() == nil || packet.TransportLayer() == nil || packet.TransportLayer().LayerType() != layers.LayerTypeTCP) {
		tcp := packet.TransportLayer().(*layers.TCP)
		dstPort := fmt.Sprintf("%d", tcp.DstPort)

		isHttp := self.checkHttpPort(dstPort)

		if isHttp {
			self.requestAssembler.AssembleWithTimestamp(packet.NetworkLayer().NetworkFlow(), tcp, packet.Metadata().Timestamp)
		} else {
			//处理TCP包
			srcIp := packet.NetworkLayer().NetworkFlow().Src().String()
			dstIp := packet.NetworkLayer().NetworkFlow().Dst().String()
			srcPort := fmt.Sprintf("%d", tcp.SrcPort)

			pkgType := "tcp" //默认包类型
			if pt, ok := self.portMap[dstPort]; ok {
				pkgType = pt
			}

			result := fmt.Sprintf(
				"SrcIp:%s\nSrcPort:%s\nDstIp:%s\nDstPort:%s\nMethod:%s\nUrl:%s\nPkgType:%s\n",
				srcIp,
				srcPort,
				dstIp,
				dstPort,
				"", "",
				pkgType,
			)

			//结果处理
			//结果处理
			select {
			case self.ctx.Agentd.Inchan <- []byte(result):
			default: //读channel撑不住的情况,就放弃当前数据
				println("drop tcp pack")
			}
		}

	}
	return nil
}

//检查是否http端口
func (self *TcpDumpService) checkHttpPort(port string) bool {
	for _, p := range self.httpPorts {
		if p == port {
			return true
		}
	}
	return false
}

//生成BPF表达式
func (self *TcpDumpService) GenerateBpf() string {
	portCondition := []string{}

	//http
	for _, hp := range self.httpPorts {
		portCondition = append(portCondition, fmt.Sprintf("dst port %s", hp))
	}

	//tcp
	for _, tp := range self.tcpPorts {
		portCondition = append(portCondition, fmt.Sprintf("dst port %s", tp))
	}

	//监控组件进程
	for _, targetProcPort := range self.targetProcPorts {
		portCondition = append(portCondition, fmt.Sprintf("dst port %s", targetProcPort))
	}

	cond := strings.Join(portCondition, " or ")
	bpf := fmt.Sprintf("tcp and (%s)", cond)
	return bpf
}

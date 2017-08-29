package tcpdump

import (
	"bufio"
	"errors"
	"fmt"
	a "github.com/domac/mafio/agent"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"github.com/google/gopacket/tcpassembly"
	"github.com/google/gopacket/tcpassembly/tcpreader"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"
)

//流量嗅探
const ModuleName = "tcpdump"

var ERR_EOF = errors.New("EOF")

//文件输入服务
type HttpDumpService struct {
	ctx              *a.Context
	PacketDataSource gopacket.PacketDataSource
	Decoder          gopacket.Decoder

	requestAssembler *tcpassembly.Assembler
	httpPorts        []string
	tcpPorts         []string
}

func New() *HttpDumpService {
	return &HttpDumpService{}
}

func (self *HttpDumpService) SetPacketDataSource(handle gopacket.PacketDataSource, decoder gopacket.Decoder) {
	self.PacketDataSource = handle
	self.Decoder = decoder
}

func (self *HttpDumpService) SetContext(ctx *a.Context) {
	self.ctx = ctx
}

func (self *HttpDumpService) Reflesh() {

}

//开始执行输入
func (self *HttpDumpService) StartInput() {
	self.httpPorts = []string{"80"}
	self.tcpPorts = []string{""}
	snaplen := 1600
	configMap, ok := self.ctx.Agentd.GetOptions().PluginsConfigs[ModuleName]
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

		tmpSnaplen := configMap["snaplen"].(string)
		snaplen, _ = strconv.Atoi(tmpSnaplen)
	}

	bpf := self.GenerateBpf()

	self.ctx.Logger().Infof("config http ports : %s", self.httpPorts)
	self.ctx.Logger().Infof("config tcp ports : %s", self.tcpPorts)
	self.ctx.Logger().Infof("config snaplen : %d", snaplen)
	self.ctx.Logger().Infof("config bpf : %s", bpf)
	//bpf := "tcp and (dst port 80 or dst port 8080 or dst port 443 or dst port 10029)"

	err := self.startTcpDump(snaplen, bpf)
	if err != nil {
		return
	}
}

//继承 tcpassembly.StreamFactory
type httpStreamFactory struct {
	ctx *a.Context
}

// httpStream 负责处理 http 请求.
type httpStream struct {
	net, transport gopacket.Flow
	r              tcpreader.ReaderStream
	ctx            *a.Context
}

func (h *httpStreamFactory) New(net, transport gopacket.Flow) tcpassembly.Stream {
	hstream := &httpStream{
		net:       net,
		transport: transport,
		r:         tcpreader.NewReaderStream(),
		ctx:       h.ctx,
	}
	go hstream.run()

	return &hstream.r
}

func (h *httpStream) run() {
	buf := bufio.NewReader(&h.r)
	for {
		req, err := http.ReadRequest(buf)
		if err == io.EOF {
			// EOF 返回
			return
		} else if err != nil {

		} else {
			result := fmt.Sprintf(
				"SrcIp:%s\nSrcPort:%s\nDstIp:%s\nDstPort:%s\nMethod:%s\nUrl:%s\n",
				h.net.Src().String(),
				h.transport.Src().String(),
				h.net.Dst().String(),
				h.transport.Dst().String(),
				req.Method, req.URL.String())

			//结果处理
			req.Body.Close()

			select {
			case h.ctx.Agentd.Inchan <- []byte(result):
			default: //读channel撑不住的情况,就放弃当前数据
				println("drop http pack")
				continue
			}

		}
	}
}

func isLoopback(device pcap.Interface) bool {
	if len(device.Addresses) == 0 {
		return false
	}

	switch device.Addresses[0].IP.String() {
	case "127.0.0.1", "::1":
		return true
	}

	return false
}

//获取要监听的网卡
func findDevices() []string {
	devices, err := pcap.FindAllDevs()

	if err != nil {
		return []string{}
	}

	interfaces := []string{}
	for _, device := range devices {

		//不处理没绑定地址的网卡
		if len(device.Addresses) == 0 || isLoopback(device) {
			continue
		}

		if strings.HasPrefix(device.Name, "lo") {
			continue
		}

		//如果是绑定的网卡,立刻返回
		if strings.HasPrefix(device.Name, "bond") {
			return []string{device.Name}
		}
		interfaces = append(interfaces, device.Name)
	}
	return interfaces

}

//开始嗅探
func (self *HttpDumpService) startTcpDump(snaplen int, bpf string) error {

	deviceList := findDevices()

	// Set up assemblies
	requestStreamFactory := &httpStreamFactory{ctx: self.ctx}
	requestStreamPool := tcpassembly.NewStreamPool(requestStreamFactory)
	self.requestAssembler = tcpassembly.NewAssembler(requestStreamPool)

	for _, device := range deviceList {
		self.ctx.Logger().Infof("Net Device : %s", device)
		go self.startListen(device, snaplen, bpf)
	}

	return nil
}

//捕获http包
func (self *HttpDumpService) startListen(faceName string, snaplen int, filter string) {
	handle, err := pcap.OpenLive(faceName, int32(snaplen), true, 500)
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

//获取dump包的数据通道
func (self *HttpDumpService) getPacketsChan(packetSource *gopacket.PacketSource) chan gopacket.Packet {
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

func (self *HttpDumpService) processPacket(packet gopacket.Packet) error {
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
			result := fmt.Sprintf(
				"SrcIp:%s\nSrcPort:%s\nDstIp:%s\nDstPort:%s\nMethod:%s\nUrl:%s\n",
				srcIp,
				srcPort,
				dstIp,
				dstPort,
				"", "")
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
func (self *HttpDumpService) checkHttpPort(port string) bool {
	for _, p := range self.httpPorts {
		if p == port {
			return true
		}
	}
	return false
}

func (self *HttpDumpService) GenerateBpf() string {
	portCondition := []string{}

	for _, hp := range self.httpPorts {
		portCondition = append(portCondition, fmt.Sprintf("dst port %s", hp))
	}

	for _, tp := range self.tcpPorts {
		portCondition = append(portCondition, fmt.Sprintf("dst port %s", tp))
	}
	cond := strings.Join(portCondition, " or ")
	bpf := fmt.Sprintf("tcp and (%s)", cond)
	return bpf
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

func convertArg(arg interface{}, kind reflect.Kind) (val reflect.Value, ok bool) {
	val = reflect.ValueOf(arg)
	if val.Kind() == kind {
		ok = true
	}
	return
}

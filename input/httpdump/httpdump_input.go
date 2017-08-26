package httpdump

import (
	"fmt"
	a "github.com/domac/mafio/agent"

	"bufio"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"github.com/google/gopacket/tcpassembly"
	"github.com/google/gopacket/tcpassembly/tcpreader"
	"io"
	"net/http"
	"strings"
	"time"
)

//流量嗅探
const ModuleName = "httpdump"

//文件输入服务
type HttpDumpService struct {
	ctx *a.Context
}

func New() *HttpDumpService {
	return &HttpDumpService{}
}

func (self *HttpDumpService) SetContext(ctx *a.Context) {
	self.ctx = ctx
}

func (self *HttpDumpService) Reflesh() {

}

func (self *HttpDumpService) StartInput() {

	snaplen := 1600
	bpf := "tcp and (dst port 80 or dst port 8080 or dst port 443 or dst port 10029)"
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
			//log.Println("Error reading stream", h.net, h.transport, ":", err)
		} else {
			result := fmt.Sprintf(
				"SrcIp:%s\nSrcPort:%s\nDstIp:%s\nDstPort:%s\nMethod:%s\nUrl:%s\n",
				h.net.Src().String(),
				h.transport.Src().String(),
				h.net.Dst().String(),
				h.transport.Dst().String(),
				req.Method, req.URL.String())

			//结果处理

			select {
			case h.ctx.Agentd.Inchan <- []byte(result):
			default: //读channel撑不住的情况,就放弃当前数据
				println("drop input pack")
				continue
			}

		}
	}
}

//获取要监听的网卡
func getNetDevices() []string {
	devices, _ := pcap.FindAllDevs()
	interfaces := []string{}
	for _, device := range devices {
		if len(device.Addresses) == 0 {
			continue
		}

		if strings.HasPrefix(device.Name, "lo") {
			continue
		}

		if strings.HasPrefix(device.Name, "bond") {
			return []string{device.Name}
		}
		interfaces = append(interfaces, device.Name)
	}
	return interfaces

}

func (self *HttpDumpService) getAssembler() *tcpassembly.Assembler {
	// 设置 assembly
	streamFactory := &httpStreamFactory{self.ctx}
	streamPool := tcpassembly.NewStreamPool(streamFactory)
	assembler := tcpassembly.NewAssembler(streamPool)
	return assembler
}

//开始嗅探
func (self *HttpDumpService) startTcpDump(snaplen int, bpf string) error {

	deviceList := getNetDevices()
	assembler := self.getAssembler()

	for _, device := range deviceList {
		self.ctx.Logger().Infof("Net Device : %s", device)
		go self.capturePackets(device, assembler, snaplen, bpf)
	}

	return nil
}

func (self *HttpDumpService) capturePackets(faceName string, assembler *tcpassembly.Assembler, snaplen int, filter string) {
	handle, err := pcap.OpenLive(faceName, int32(snaplen), true, 500)
	if err != nil {
		self.ctx.Logger().Fatal(err)
	}

	if err := handle.SetBPFFilter(filter); err != nil {
		self.ctx.Logger().Fatal(err)
	}

	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	packets := packetSource.Packets()
	ticker := time.Tick(time.Minute)

	for {
		select {
		case packet := <-packets:
			//数据包为空, 代表 pcap文件到结尾了
			if packet == nil {
				return
			}

			if packet.NetworkLayer() == nil || packet.TransportLayer() == nil || packet.TransportLayer().LayerType() != layers.LayerTypeTCP {
				self.ctx.Logger().Println("Unusable packet")
				continue
			}

			if packet.TransportLayer() == nil {
				continue
			}

			tcp, ok := packet.TransportLayer().(*layers.TCP)

			if !ok {
				continue
			}

			assembler.AssembleWithTimestamp(packet.NetworkLayer().NetworkFlow(), tcp, packet.Metadata().Timestamp)

		case <-ticker:
			//每一分钟,自动刷新之前2分钟都处于不活跃的连接信息
			assembler.FlushOlderThan(time.Now().Add(time.Minute * -2))
		}
	}
}

package tcpdump

import (
	"bufio"
	"fmt"
	a "github.com/domac/mafio/agent"
	"github.com/google/gopacket"
	"github.com/google/gopacket/tcpassembly"
	"github.com/google/gopacket/tcpassembly/tcpreader"
	"io"
	"net/http"
)

//dump结果信息
type DumpResult struct {
	SrcIp   string `json:"srcip"`
	SrcPort string `json:"srcport"`
	DstIp   string `json:"dstip"`
	DstPort string `json:"dstport"`
	Method  string `json:"method"`
	Url     string `json:"url"`
	PkgType string `json:"pkgtype"`
}

//继承 tcpassembly.StreamFactory
type httpStreamFactory struct {
	ctx     *a.Context
	portMap map[string]string
}

// httpStream 负责处理 http 请求.
type httpStream struct {
	net, transport gopacket.Flow
	r              tcpreader.ReaderStream
	ctx            *a.Context
	portMap        map[string]string
}

func (h *httpStreamFactory) New(net, transport gopacket.Flow) tcpassembly.Stream {
	hstream := &httpStream{
		net:       net,
		transport: transport,
		r:         tcpreader.NewReaderStream(),
		ctx:       h.ctx,
		portMap:   h.portMap,
	}
	go hstream.run()

	return &hstream.r
}

func (h *httpStream) run() {

	defer func() {
		if e := recover(); e != nil {
			h.ctx.Logger().Error(e)
		}
	}()

	buf := bufio.NewReader(&h.r)
	for {
		req, err := http.ReadRequest(buf)
		if err == io.EOF {
			// EOF 返回
			return
		} else if err != nil {

		} else {

			pkgType := "http"

			if pt, ok := h.portMap[h.transport.Dst().String()]; ok {
				pkgType = pt
			}

			result := fmt.Sprintf(
				"SrcIp:%s\nSrcPort:%s\nDstIp:%s\nDstPort:%s\nMethod:%s\nUrl:%s\nPkgType:%s\n",
				h.net.Src().String(),
				h.transport.Src().String(),
				h.net.Dst().String(),
				h.transport.Dst().String(),
				req.Method, req.URL.String(),
				pkgType,
			)

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

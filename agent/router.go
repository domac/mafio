package agent

import (
	"bytes"
	"github.com/domac/mafio/util"
	"github.com/domac/mafio/version"
	"github.com/julienschmidt/httprouter"
	"net/http"
	"net/http/pprof"
)

//负责提供agent的对外API服务
type ApiServer struct {
	ctx    *Context     //上下文
	router http.Handler //路由
}

//HTTP 服务
func newAPIServer(ctx *Context) *ApiServer {

	log := Log(ctx.Agentd.opts.Logger)

	router := httprouter.New()
	router.HandleMethodNotAllowed = true
	router.PanicHandler = LogPanicHandler(ctx.Agentd.opts.Logger)
	router.NotFound = LogNotFoundHandler(ctx.Agentd.opts.Logger)
	router.MethodNotAllowed = LogMethodNotAllowedHandler(ctx.Agentd.opts.Logger)

	s := &ApiServer{
		ctx:    ctx,
		router: router,
	}

	//内置监控
	router.GET("/debug/pprof/*pprof", innerPprofHandler)

	//在这里注册路由服务
	router.Handle("GET", "/version", Decorate(s.versionHandler, log, Default)) //json格式输出
	router.Handle("GET", "/debug", Decorate(s.pprofHandler, log, PlainText))   //文本形式输出
	router.Handle("GET", "/ping", Decorate(s.pingHandler, log, PlainText))     //文本形式输出
	router.Handle("GET", "/empty", Decorate(s.emptyHandler, log, PlainText))   //文本形式输出
	return s
}

func (s *ApiServer) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	s.router.ServeHTTP(w, req)
}

func (s *ApiServer) versionHandler(w http.ResponseWriter, req *http.Request, ps httprouter.Params) (interface{}, error) {
	version := version.Verbose("mafio")
	res := NewResult(RESULT_CODE_FAIL, true, "", version)
	return res, nil
}

//调用内置的pprof
func innerPprofHandler(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	switch p.ByName("pprof") {
	case "/cmdline":
		pprof.Cmdline(w, r)
	case "/profile":
		pprof.Profile(w, r)
	case "/symbol":
		pprof.Symbol(w, r)
	default:
		pprof.Index(w, r)
	}
}

//Ping
func (s *ApiServer) pingHandler(w http.ResponseWriter, req *http.Request, ps httprouter.Params) (interface{}, error) {
	return "OK", nil
}

//输出性能信息
func (s *ApiServer) pprofHandler(w http.ResponseWriter, req *http.Request, ps httprouter.Params) (interface{}, error) {
	paramReq, err := NewReqParams(req)
	if err != nil {
		return nil, err
	}
	cmd, _ := paramReq.Get("cmd")
	buf := bytes.Buffer{}
	util.ProcessInput(cmd, &buf)
	return buf.String(), nil
}

//通道数据清空
func (s *ApiServer) emptyHandler(w http.ResponseWriter, req *http.Request, ps httprouter.Params) (interface{}, error) {
	s.ctx.Agentd.Empty()
	return "empty is finish", nil
}

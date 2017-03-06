package agent

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
)

type logWriter struct {
	Logger
}

func (l logWriter) Write(p []byte) (int, error) {
	l.Logger.Infof(string(p))
	return len(p), nil

}

//简单清爽的Http服务
func Serve(listener net.Listener, handler http.Handler, proto string, l Logger) {
	l.Infof(fmt.Sprintf("[%s]istening on %s", proto, listener.Addr()))

	server := &http.Server{
		Handler:  handler,
		ErrorLog: log.New(logWriter{}, "", 0)}

	err := server.Serve(listener)
	if err != nil && !strings.Contains(err.Error(), "use of closed network connection") {
		l.Errorf(fmt.Sprintf("ERROR: http.Serve() - %s", err))
	}
	l.Infof(fmt.Sprintf("%s: closing %s", proto, listener.Addr()))
}

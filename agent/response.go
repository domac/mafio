package agent

import (
	"encoding/json"
	"fmt"
	"github.com/julienschmidt/httprouter"
	"io"
	"net/http"
	"time"
)

type Decorator func(APIHandler) APIHandler

//定义API的类型
type APIHandler func(http.ResponseWriter, *http.Request, httprouter.Params) (interface{}, error)

const (
	RESULT_CODE_SUCCESS int = iota
	RESULT_CODE_FAIL
)

type Result struct {
	Code    int
	Success bool
	Message string
	Object  interface{}
}

func NewResult(code int, success bool, msg string, obj interface{}) *Result {
	return &Result{
		Code:    code,
		Success: success,
		Message: msg,
		Object:  obj,
	}
}

func (r Result) Error() string {
	return r.Message
}

func acceptVersion(req *http.Request) int {
	if req.Header.Get("accept") == "application/vnd.mafioagent; version=1.0" {
		return 1
	}

	return 0
}

func PlainText(f APIHandler) APIHandler {
	return func(w http.ResponseWriter, req *http.Request, ps httprouter.Params) (interface{}, error) {
		code := 200
		data, err := f(w, req, ps)
		if err != nil {
			code = err.(Result).Code
			data = err.Error()
		}
		switch d := data.(type) {
		case string:
			w.WriteHeader(code)
			io.WriteString(w, d)
		case []byte:
			w.WriteHeader(code)
			w.Write(d)
		default:
			panic(fmt.Sprintf("unknown response type %T", data))
		}
		return nil, nil
	}
}

func Default(f APIHandler) APIHandler {
	return func(w http.ResponseWriter, req *http.Request, ps httprouter.Params) (interface{}, error) {
		data, err := f(w, req, ps)
		if err != nil {
			RespondDefault(w, err.(Result).Code, err)
			return nil, nil
		}
		RespondDefault(w, 200, data)
		return nil, nil
	}
}

func RespondDefault(w http.ResponseWriter, code int, data interface{}) {
	var response []byte
	var err error
	var isJSON bool

	if code == 200 {
		switch data.(type) {
		case string:
			response = []byte(data.(string))
		case []byte:
			response = data.([]byte)
		case nil:
			response = []byte{}
		default:
			isJSON = true
			response, err = json.Marshal(data)
			if err != nil {
				code = 500
				data = err
			}
		}
	}

	if code != 200 {
		isJSON = true
		response = []byte(fmt.Sprintf(`{"message":"%s"}`, data))
	}

	if isJSON {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
	}
	w.Header().Set("X-mafio-Content-Type", "mafio; version=1.0")
	w.WriteHeader(code)
	w.Write(response)
}

func Decorate(f APIHandler, ds ...Decorator) httprouter.Handle {
	decorated := f
	for _, decorate := range ds {
		decorated = decorate(decorated)
	}
	return func(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
		decorated(w, req, ps)
	}
}

func Log(l Logger) Decorator {
	return func(f APIHandler) APIHandler {
		return func(w http.ResponseWriter, req *http.Request, ps httprouter.Params) (interface{}, error) {
			start := time.Now()
			response, err := f(w, req, ps)
			elapsed := time.Since(start)
			status := 200
			if e, ok := err.(Result); ok {
				status = e.Code
			}
			l.Infof(fmt.Sprintf("%d %s %s (%s) %s",
				status, req.Method, req.URL.RequestURI(), req.RemoteAddr, elapsed))
			return response, err
		}
	}
}

func LogPanicHandler(l Logger) func(w http.ResponseWriter, req *http.Request, p interface{}) {
	return func(w http.ResponseWriter, req *http.Request, p interface{}) {
		l.Errorf(fmt.Sprintf("ERROR: panic in HTTP handler - %s", p))
		Decorate(func(w http.ResponseWriter, req *http.Request, ps httprouter.Params) (interface{}, error) {
			return nil, Result{500, false, "INTERNAL_ERROR", nil}
		}, Log(l), Default)(w, req, nil)
	}
}

func LogNotFoundHandler(l Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		Decorate(func(w http.ResponseWriter, req *http.Request, ps httprouter.Params) (interface{}, error) {
			return nil, Result{404, false, "NOT_FOUND", nil}
		}, Log(l), Default)(w, req, nil)
	})
}

func LogMethodNotAllowedHandler(l Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		Decorate(func(w http.ResponseWriter, req *http.Request, ps httprouter.Params) (interface{}, error) {
			return nil, Result{405, false, "METHOD_NOT_ALLOWED", nil}
		}, Log(l), Default)(w, req, nil)
	})
}

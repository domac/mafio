package util

import (
	"github.com/valyala/fasthttp"
	"time"
)

func NewHostClient(proxy string, readTimeout time.Duration) *fasthttp.HostClient {
	c := &fasthttp.HostClient{
		Addr:        proxy,
		ReadTimeout: readTimeout,
	}
	return c
}

func NewHttpClient(proxy string, readTimeout time.Duration) *fasthttp.Client {
	c := &fasthttp.Client{
		ReadTimeout: readTimeout,
	}
	return c
}

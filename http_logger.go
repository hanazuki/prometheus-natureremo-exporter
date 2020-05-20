package main

import (
	"fmt"
	"net/http"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

type HttpLogger struct {
	Logger    log.Logger
	Transport http.RoundTripper
}

func (h HttpLogger) RoundTrip(req *http.Request) (res *http.Response, err error) {
	res, err = h.Transport.RoundTrip(req)

	level.Info(h.Logger).Log("msg", fmt.Sprintf("HTTP %s %s -> %s", req.Method, req.URL, res.Status))

	return
}

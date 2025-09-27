package middleware

import (
	"io"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"

	"github.com/openai/openai-go/v2/option"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/log"
)

type stringReadCloser struct {
	io.Reader
}

func (stringReadCloser) Close() error {
	return nil
}

func NewStringReadCloser(s string) io.ReadCloser {
	return stringReadCloser{strings.NewReader(s)}
}

func Middleware(model *api.Model, vars *api.Vars) option.Middleware {
	return func(req *http.Request, next option.MiddlewareNext) (*http.Response, error) {
		start := time.Now()

		ctx := req.Context()
		if log.GetLogger(ctx).IsTrace() {
			reqData, _ := httputil.DumpRequest(req, true)
			log.GetLogger(ctx).Debugf(">REQUEST: %s\n", string(reqData))
		}

		var resp *http.Response
		var err error

		if vars.Config.DryRun {
			resp, err = fake(req, model, vars)
		} else {
			// Call the next middleware in the chain.
			resp, err = next(req)
		}

		if log.GetLogger(ctx).IsTrace() {
			resData, _ := httputil.DumpResponse(resp, true)
			log.GetLogger(ctx).Debugf(">RESPONSE: %s\n", string(resData))
		}

		took := time.Since(start).Milliseconds()
		var status int
		if resp != nil {
			status = resp.StatusCode
		}
		log.GetLogger(ctx).Debugf("Status: %d, %s request for %s took %dms\n", status, req.Method, req.URL, took)

		return resp, err
	}
}

package log

import (
	"net/http"
	"net/http/httputil"
	"time"

	"github.com/openai/openai-go/option"
)

func Middleware() option.Middleware {
	return func(req *http.Request, next option.MiddlewareNext) (*http.Response, error) {
		start := time.Now()

		reqData, _ := httputil.DumpRequest(req, true)
		Debugln(">>>REQUEST:\n", string(reqData))

		// Call the next middleware in the chain.
		resp, err := next(req)

		resData, _ := httputil.DumpResponse(resp, true)
		Debugln("<<<RESPONSE:\n", string(resData))

		took := time.Since(start).Milliseconds()
		Debugf("Status: %d, %s request for %s took %dms\n", resp.StatusCode, req.Method, req.URL, took)

		return resp, err
	}
}

package log

import (
	"io"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"

	"github.com/openai/openai-go/option"
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

func Middleware(dryRun bool, dryrunContent string) option.Middleware {
	return func(req *http.Request, next option.MiddlewareNext) (*http.Response, error) {
		start := time.Now()

		if IsTrace() {
			reqData, _ := httputil.DumpRequest(req, true)
			Debugln(">>>REQUEST:\n", string(reqData))
		}

		var resp *http.Response
		var err error

		if dryRun {
			resp = &http.Response{
				StatusCode: 200,
				Body:       NewStringReadCloser(dryrunContent),
			}
		} else {
			// Call the next middleware in the chain.
			resp, err = next(req)
		}

		if IsTrace() {
			resData, _ := httputil.DumpResponse(resp, true)
			Debugln("<<<RESPONSE:\n", string(resData))
		}

		took := time.Since(start).Milliseconds()
		var status int
		if resp != nil {
			status = resp.StatusCode
		}
		Debugf("Status: %d, %s request for %s took %dms\n", status, req.Method, req.URL, took)

		return resp, err
	}
}

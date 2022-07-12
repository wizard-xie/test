package filter

import (
	"bytes"
	"net/http"
	"net/url"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

// filterFunc adapter kratos http.HandleFunc.
type filterFunc = func(http.Handler) http.Handler

var _ http.ResponseWriter = (*responseWriter)(nil)

// responseWriter implement the http.ResponseWriter interface.
// in order to record the value written by the client calling the Write() method.
type responseWriter struct {
	w    http.ResponseWriter
	buf  bytes.Buffer
	code int
}

func (c *responseWriter) Header() http.Header {
	return c.w.Header()
}

func (c *responseWriter) Write(b []byte) (int, error) {
	c.buf.Write(b)

	return c.w.Write(b)
}

func (c *responseWriter) WriteHeader(statusCode int) {
	c.code = statusCode
	c.w.WriteHeader(statusCode)
}

type httpInfo struct {
	r *http.Request
	w *responseWriter
}

func (hi *httpInfo) form() zap.Field {
	_ = hi.r.ParseForm()

	return zap.Reflect("form", hi.r.Form)
}

func (hi *httpInfo) query() zap.Field {
	var q url.Values
	if hi.r.URL == nil {
		q = url.Values{}
	} else {
		q = hi.r.URL.Query()
	}

	return zap.Reflect("query", q)
}

func (hi *httpInfo) vars() zap.Field {
	raws := mux.Vars(hi.r)
	vars := make(url.Values, len(raws))
	for k, v := range raws {
		vars[k] = []string{v}
	}

	return zap.Reflect("vars", vars)
}

func (hi *httpInfo) method() zap.Field {
	return zap.String("method", hi.r.Method)
}

func (hi *httpInfo) host() zap.Field {
	return zap.String("host", hi.r.Host)
}

func (hi *httpInfo) path() zap.Field {
	return zap.String("path", hi.r.RequestURI)
}

func (hi *httpInfo) requestHeader() zap.Field {
	return zap.Reflect("requestHeader", hi.r.Header)
}

func (hi *httpInfo) requestBody() zap.Field {
	buf := new(bytes.Buffer)
	_, _ = buf.ReadFrom(hi.r.Body)

	return zap.String("requestBody", buf.String())
}

func (hi *httpInfo) responseHeader() zap.Field {
	return zap.Reflect("responseHeader", hi.w.Header())
}

func (hi *httpInfo) code() zap.Field {
	return zap.Int("responseCode", hi.w.code)
}

func (hi *httpInfo) reponseBody() zap.Field {
	return zap.ByteString("responseBody", hi.w.buf.Bytes())
}

func (hi *httpInfo) log() {
	logger.Info(
		"http request info",
		zap.Namespace("httpInfo"),
		hi.host(),
		hi.method(),
		hi.path(),
		hi.form(),
		hi.query(),
		hi.vars(),
		hi.requestBody(),
		hi.requestHeader(),
		hi.code(),
		hi.responseHeader(),
		hi.reponseBody(),
	)
}

// LogHTTPInfo the http filter to record http request and respose info.
func LogHTTPInfo() filterFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rw := &responseWriter{w: w}
			next.ServeHTTP(rw, r)
			info := &httpInfo{w: rw, r: r}
			info.log()
		})
	}
}

// TODO: do log splitting can see https://blog.csdn.net/lfhlzh/article/details/106151419
var logger zap.Logger

func init() {
	logger = *zap.NewExample()
}

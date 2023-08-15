package tracing

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"mime"
	"net/http"
	"strings"

	"github.com/DoOR-Team/goutils/common"
	dlog "github.com/DoOR-Team/goutils/log"
	"github.com/DoOR-Team/goutils/trace"
	"github.com/DoOR-Team/goutils/tracing/serious"
	jsoniter "github.com/json-iterator/go"
	"github.com/jtolds/gls"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go/log"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

type HTTPResponseMeta struct {
	Header        http.Header `json:",omitempty"`
	StatusCode    int
	ContentLength int64
}

type responseMultiWriter struct {
	w      http.ResponseWriter
	status int
	buf    *bytes.Buffer
}

func (h *responseMultiWriter) Header() http.Header {
	return h.w.Header()
}

func (h *responseMultiWriter) Write(b []byte) (int, error) {
	h.buf.Write(b)
	return h.w.Write(b)
}

func (h *responseMultiWriter) WriteHeader(c int) {
	h.status = c
	h.w.WriteHeader(c)
}

// NewHTTPHandler 建议底部中间件自行处理panic的recover问题，供记录错误
func NewHTTPHandler(handler http.Handler,
	options ...TracingOption) http.Handler {
	opts := &tracingOptions{}
	for _, opt := range options {
		opt(opts)
	}

	fn := func(w http.ResponseWriter, r *http.Request) {
		var checkReqErr error
		if opts.httpCheckRequestFunc != nil {
			err := opts.httpCheckRequestFunc(r, true)
			if err == ErrUnneededTracing {
				handler.ServeHTTP(w, r)
				return
			} else if err != nil {
				checkReqErr = err
			}
		}

		tr := opentracing.GlobalTracer()
		spanCtx, _ := tr.Extract(opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(r.Header))

		var op string
		if opts.opNameFunc != nil {
			op = opts.opNameFunc()
		}
		if op == "" {
			op = "HTTP " + r.Method + " " + r.URL.Path
		}
		sp := tr.StartSpan(op, ext.RPCServerOption(spanCtx))
		defer sp.Finish()
		defer func() {
			r := recover() // 简单recover记录下，再丢出去
			if r != nil {
				ext.Error.Set(sp, true)
				perr, ok := r.(error)
				if !ok {
					perr = fmt.Errorf(fmt.Sprintln(r))
				}
				sp.LogFields(ErrorField(errors.Wrap(perr, "panic")))
				serious.SignSerious(sp, true)
				w.WriteHeader(500)
				fmt.Fprintf(w, common.ToJsonString(map[string]interface{}{
					"errText": "系统错误",
					"errCode": 1,
					"data":    "",
				}))
				// 消费这个错，并报出来
				dlog.Error(string(trace.PanicTrace(5)))
			}
		}()

		if checkReqErr != nil {
			ext.Error.Set(sp, true) // 标记错误
			sp.LogFields(ErrorField(checkReqErr))
		}

		// 设置tag
		ext.Component.Set(sp, "http")
		ext.HTTPMethod.Set(sp, r.Method)
		ext.HTTPUrl.Set(sp, r.URL.String())

		// 记录请求信息
		traceRequestWithSpan(sp, r, !opts.disableTracingHTTPRequstBody, opts.maxBodyLogSize)

		// 请求返回结果双写
		multiWriter := &responseMultiWriter{
			w:   w,
			buf: bytes.NewBuffer([]byte{}),
		}

		// 处理请求，使用gls包裹
		r = r.WithContext(opentracing.ContextWithSpan(r.Context(), sp)) // 传递下span到ctx
		setGlsTracingSpan(sp, func() {
			handler.ServeHTTP(multiWriter, r)
		})

		// uid
		uid := r.Context().Value("uid")
		if uid != nil {
			sp.SetBaggageItem(BaggageItemKeyUserID, fmt.Sprint(uid))
			sp.SetTag(TagKeyUserID, fmt.Sprint(uid))
		}

		// 记录返回码，设置tag
		ext.HTTPStatusCode.Set(sp, uint16(multiWriter.status))

		// 记录返回结果
		traceResponseWithSpan(sp, multiWriter.Header(), multiWriter.status, int64(multiWriter.buf.Len()), multiWriter.buf.Bytes(), true, !opts.disableTracingHTTPResponseBody, opts.maxBodyLogSize, opts.httpCheckResponseFunc)
	}

	return http.HandlerFunc(fn)
}

// DoHTTPRequest ...
func DoHTTPRequest(client *http.Client,
	req *http.Request,
	options ...TracingOption) (*http.Response, error) {

	opts := &tracingOptions{}
	for _, opt := range options {
		opt(opts)
	}

	var checkReqErr error
	if opts.httpCheckRequestFunc != nil {
		err := opts.httpCheckRequestFunc(req, false)
		if err == ErrUnneededTracing {
			return client.Do(req)
		} else if err != nil {
			checkReqErr = err
		}
	}

	ctx := req.Context()
	if opentracing.SpanFromContext(ctx) == nil {
		// 如果ctx里没传，就从gls获取
		glsSpan := getGlsTracingSpan()
		if glsSpan != nil {
			ctx = opentracing.ContextWithSpan(ctx, glsSpan)
			req = req.WithContext(ctx)
		}
	}
	var parentCtx opentracing.SpanContext
	if parent := opentracing.SpanFromContext(ctx); parent != nil {
		parentCtx = parent.Context()
	}

	var op string
	if opts.opNameFunc != nil {
		op = opts.opNameFunc()
	}
	if op == "" {
		op = fmt.Sprintf("HTTP_CLI %s %s", req.Method, req.URL.Path)
	}
	sp := opentracing.GlobalTracer().StartSpan(op, opentracing.ChildOf(parentCtx))
	defer sp.Finish()
	defer func() {
		r := recover() // 简单recover记录下，再丢出去
		if r != nil {
			ext.Error.Set(sp, true)
			perr, ok := r.(error)
			if !ok {
				perr = fmt.Errorf(fmt.Sprintln(r))
			}
			sp.LogFields(ErrorField(errors.Wrap(perr, "panic")))
			serious.SignSerious(sp, true)
			panic(r)
		}
	}()

	if checkReqErr != nil {
		ext.Error.Set(sp, true) // 标记错误
		sp.LogFields(ErrorField(checkReqErr))
	}

	// 设置tag和carrier
	ext.SpanKindRPCClient.Set(sp)
	ext.Component.Set(sp, "http")
	ext.HTTPMethod.Set(sp, req.Method)
	ext.HTTPUrl.Set(sp, req.URL.String())

	carrier := opentracing.HTTPHeadersCarrier(req.Header)
	sp.Tracer().Inject(sp.Context(), opentracing.HTTPHeaders, carrier)

	// 记录请求信息
	traceRequestWithSpan(sp, req, !opts.disableTracingHTTPRequstBody, opts.maxBodyLogSize)

	// 执行请求
	resp, doErr := client.Do(req)

	// uid
	uid := sp.BaggageItem(BaggageItemKeyUserID)
	if uid != "" {
		sp.SetTag(TagKeyUserID, uid)
	}

	if resp != nil {
		// 记录返回码，设置tag
		ext.HTTPStatusCode.Set(sp, uint16(resp.StatusCode))

		// 记录返回结果
		var body []byte
		if resp.ContentLength != 0 && !opts.disableTracingHTTPResponseBody {
			var err error
			body, err = ioutil.ReadAll(resp.Body)
			if err != nil {
				ext.Error.Set(sp, true)
				sp.LogFields(ErrorField(errors.Wrap(err, "Reading response.body failed")))
			}
			resp.Body.Close()
			resp.Body = ioutil.NopCloser(bytes.NewBuffer(body))
		}

		checkRespFunc := opts.httpCheckResponseFunc
		if doErr != nil { // 如果本身do已经产生错误了，就无需检查结果集了
			checkRespFunc = nil
		}
		traceResponseWithSpan(sp, resp.Header, resp.StatusCode, resp.ContentLength, body, false, !opts.disableTracingHTTPResponseBody, opts.maxBodyLogSize, checkRespFunc)
	}

	if doErr != nil {
		// 记录doErr
		ext.Error.Set(sp, true)
		sp.LogFields(ErrorField(errors.Wrap(doErr, "Do request failed")))
	}

	return resp, doErr
}

func traceRequestWithSpan(sp opentracing.Span, req *http.Request, traceBody bool, maxBodyLogSize int) {
	requestMeta := &struct {
		Header        http.Header `json:",omitempty"`
		ContentLength int64
	}{
		Header:        req.Header,
		ContentLength: req.ContentLength,
	}
	jsn, err := jsoniter.Marshal(requestMeta)
	if err != nil {
		ext.Error.Set(sp, true)
		sp.LogFields(ErrorField(errors.Wrap(err, "Marshal request.meta failed")))
	} else {
		sp.LogFields(log.String("request.meta", string(jsn)))
	}

	if !traceBody {
		return
	}

	if req.ContentLength > 0 {
		isUpload := false
		v := req.Header.Get("Content-Type")
		if v != "" {
			d, _, err := mime.ParseMediaType(v)
			if err == nil && d == "multipart/form-data" {
				isUpload = true
			}
		}

		if !isUpload {
			body, err := ioutil.ReadAll(req.Body)
			defer func() {
				// TODO: ioutil.NopCloser 的话还是不太好，可能会引起没Close
				// 这里似乎被调用了两次，需要确认咋回事
				req.Body.Close()
				req.Body = ioutil.NopCloser(bytes.NewBuffer(body))
			}()

			if err != nil {
				ext.Error.Set(sp, true)
				sp.LogFields(ErrorField(errors.Wrap(err, "Reading request.body failed")))
			} else {
				sp.LogFields(log.String("request.body", pruneBodyLog(string(body), maxBodyLogSize)))
			}
		}
	}
}

func traceResponseWithSpan(sp opentracing.Span,
	header http.Header,
	statusCode int,
	contentLength int64,
	body []byte,
	server bool,
	traceBody bool, maxBodyLogSize int,
	checkResponseFunc func(meta *HTTPResponseMeta, body []byte, forServer bool) error) {

	responseMeta := &HTTPResponseMeta{
		Header:        header,
		StatusCode:    statusCode,
		ContentLength: contentLength,
	}
	jsn, err := jsoniter.Marshal(responseMeta)
	if err != nil {
		ext.Error.Set(sp, true)
		sp.LogFields(ErrorField(errors.Wrap(err, "Marshal response.meta failed")))
	} else {
		sp.LogFields(log.String("response.meta", string(jsn)))
	}

	if responseMeta.ContentLength != 0 {
		if traceBody && len(body) > 0 {
			sp.LogFields(log.String("response.body", pruneBodyLog(string(body), maxBodyLogSize)))
		}

		if checkResponseFunc == nil {
			return
		}

		err = checkResponseFunc(responseMeta, body, server)
		if err != nil {
			ext.Error.Set(sp, true) // 标记错误
			sp.LogFields(ErrorField(err))
		}
	}
}

var hetuCheckReqFunc = func(r *http.Request, forServer bool) error {
	if r.Method == "HEAD" ||
		r.Method == "OPTION" {
		return nil
	}

	if strings.HasPrefix(r.URL.Path, "/debug/pprof") ||
		strings.HasSuffix(r.URL.Path, "/status") ||
		strings.HasSuffix(r.URL.Path, "/healthz") ||
		strings.HasPrefix(r.URL.Path, "/metrics") {
		return ErrUnneededTracing
	}

	return nil
}

var hetuCheckRespFunc = func(meta *HTTPResponseMeta, body []byte, isServer bool) error {
	if strings.Contains(meta.Header.Get("Content-Type"), "application/json") {
		e := &struct {
			DebugInfo string `json:"debugInfo"`
			ErrText   string `json:"errText"`
			ErrCode   int64  `json:"errCode"`
		}{}

		if err := jsoniter.Unmarshal(body, e); err != nil {
			if string(body) == "ok" {
				return nil
			}
			return errors.Wrap(err, "Unmarshal response.body failed")
		}

		if (e.ErrCode != 0 && e.ErrCode != 1000 && e.ErrCode == 100) || len(body) <= 0 { // body没内容肯定也是要认为是业务端错误
			msg := "无错误信息"
			if e.DebugInfo != "" {
				msg = e.DebugInfo
			} else if e.ErrText != "" {
				msg = e.ErrText
			} else if len(body) <= 0 {
				msg = "返回body为空"
			}

			if isServer { // server的业务错误不需要记录在这里的堆栈信息
				return fmt.Errorf(msg)
			}

			return errors.Errorf(msg)
		}
	}

	return nil
}

func NewTracingHTTPHandler(handler http.Handler, options ...TracingOption) http.Handler {
	options = append(options, TracingHTTPCheckRequestFunc(hetuCheckReqFunc))
	options = append(options, TracingHTTPCheckResponseFunc(hetuCheckRespFunc))
	return NewHTTPHandler(handler, options...)
}

func DoHetuHTTPRequest(client *http.Client,
	req *http.Request,
	options ...TracingOption) (*http.Response, error) {
	options = append(options, TracingHTTPCheckRequestFunc(hetuCheckReqFunc))
	options = append(options, TracingHTTPCheckResponseFunc(hetuCheckRespFunc))
	return DoHTTPRequest(client, req, options...)
}

type HttpWriteTracer struct {
	w        http.ResponseWriter
	bytesbuf *bytes.Buffer
}

func (h HttpWriteTracer) Header() http.Header {
	return h.w.Header()
}
func (h HttpWriteTracer) Write(b []byte) (int, error) {
	h.bytesbuf.Write(b)
	return h.w.Write(b)
}
func (h HttpWriteTracer) WriteHeader(c int) {
	h.w.WriteHeader(c)
}

func AddTraceIDAndUserIDAndPrev(traceID, userID, prevMethod, prevApp string, contextCall func()) {
	mgr.SetValues(
		gls.Values{traceIdKey: traceID,
			userIdKey:     userID,
			prevMethodKey: prevMethod,
			prevAppKey:    prevApp,
		},
		contextCall,
	)
}

func HttpTraceMiddleware(h http.Handler) http.Handler {
	var handler http.Handler
	handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var traceID, nextTraceID, userID string
		cookiestr := make([]string, 0)
		for index := range r.Cookies() {
			cookiestr = append(cookiestr, r.Cookies()[index].String())
		}
		ctx := r.Context()
		traceID = MakeNewTraceID()
		nextTraceID = MakeNewFrom(traceID)
		var ok = false
		userID, ok = ctx.Value("uid").(string)
		if !ok {
			userIDByte, _ := json.Marshal(ctx.Value("uid"))
			userID = string(userIDByte)
		}
		httpTraceBuf := make([]byte, 0)
		httpWriteTracer := HttpWriteTracer{w: w, bytesbuf: bytes.NewBuffer(httpTraceBuf)}
		AddTraceIDAndUserIDAndPrev(nextTraceID, userID, r.RequestURI, viper.GetString("k8s_appname"), func() {
			h.ServeHTTP(httpWriteTracer, r)
		})
	})

	if Enable {
		oldHandler := handler

		// handler = NewHTTPHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler = NewTracingHTTPHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			oldHandler.ServeHTTP(w, r)
		}), TracingMaxBodyLogSize(5*1024*1024))
	}

	// 去重判断再加一层
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if GetTraceID() != "" {
			h.ServeHTTP(w, r)
			return
		}
		handler.ServeHTTP(w, r)
	})
}

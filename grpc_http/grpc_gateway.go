package grpc_http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"context"

	"github.com/gorilla/mux"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	spb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/DoOR-Team/goutils/derror"
)

type Srv struct {
	CloseFunc func() error
	ServeFunc func() error
}

func (h *Srv) Close() error {
	return h.CloseFunc()
}

func (h *Srv) Serve() error {
	return h.ServeFunc()
}

type Cli struct {
	CloseFunc func() error
}

func (c *Cli) Close() error {
	return c.CloseFunc()
}

/*
grpc-gateway 相关的

例子:
gateWayMux := runtime.NewServeMux(
		runtime.WithProtoErrorHandler(grpc_http.HTTPProtoErrorHandler),
		runtime.WithMarshalerOption("application/json", &grpc_http.JSONMarshaler{}),
	)
*/

type ErrorResponse struct {
	Message string `json:"message"`
	Code    uint32 `json:"code"`
}

const defaultMessage = "系统错误"

func wrapErrorResponse(err error) *ErrorResponse {
	if err == nil {
		return &ErrorResponse{
			Message: "异常系统错误",
			Code:    uint32(codes.Unknown),
		}
	}
	msg := defaultMessage

	// body里的code，基本上这个code都符合grpc code规范

	return &ErrorResponse{
		Message: msg,
		Code:    1,
	}
}

var HTTPProtoErrorHandler runtime.ProtoErrorHandlerFunc = func(ctx context.Context, mux *runtime.ServeMux, marshaler runtime.Marshaler, w http.ResponseWriter, req *http.Request, err error) {
	if err != nil {
		// 需要保证最终传递进DefaultHTTPProtoErrorHandler的是*statusError
		// 因为例如使用了NativeClient的话，这里就是原rpc方法直接返回的error信息了
		// err = derror.GrpcStatus(err).Err()
		err = derror.GrpcCheckStatus(err).Err()
	}
	runtime.DefaultHTTPProtoErrorHandler(ctx, mux, marshaler, w, req, err)
}

// An Encoder writes JSON values to an output stream.
type Encoder struct {
	w          io.Writer
	err        error
	escapeHTML bool

	indentBuf    *bytes.Buffer
	indentPrefix string
	indentValue  string
}

type JSONMarshaler struct{}

func (*JSONMarshaler) ContentType() string {
	return "application/json"
}

func (m *JSONMarshaler) Marshal(v interface{}) ([]byte, error) {
	if s, ok := v.(*spb.Status); ok {
		err := status.FromProto(s).Err()
		resp := wrapErrorResponse(err)
		return json.Marshal(resp)
	}

	return json.Marshal(v)
}

func (m *JSONMarshaler) Unmarshal(data []byte, v interface{}) error {
	if _, ok := v.(*spb.Status); ok {
		errResp := &ErrorResponse{}
		err := json.Unmarshal(data, errResp)
		if err == nil {
			panic("无法从ErrorResponse转换成*spb.Status")
			// er := wrapError(errResp)
			// status := agierror.GrpcStatus(er).Proto()
			// proto.Merge(s, status)
			// return nil
		}
	}

	return json.Unmarshal(data, v)
}

func (m *JSONMarshaler) NewDecoder(r io.Reader) runtime.Decoder {
	return json.NewDecoder(r)
}

func (m *JSONMarshaler) NewEncoder(w io.Writer) runtime.Encoder {
	return json.NewEncoder(w)
}

//func AgiProfiling(r *mux.Router) {
//	r.HandleFunc("/debug/pprof/", pprof.Index)
//	r.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
//	r.HandleFunc("/debug/pprof/profile", pprof.Profile)
//	r.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
//	r.HandleFunc("/debug/pprof/trace", pprof.Trace)
//}

func NewHttpRouter() *mux.Router {
	r := mux.NewRouter()
	r.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "ok")
	})
	//AgiProfiling(r)
	return r
}

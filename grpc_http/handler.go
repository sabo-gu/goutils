package grpc_http

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/go-kit/kit/endpoint"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	gcontext "golang.org/x/net/context"

	"github.com/DoOR-Team/goutils/derror"
	"github.com/DoOR-Team/goutils/log"
)

const AccessControlAllowHeaders = "Origin, Content-Type, Token, Authorization, Mode, x-requested-with"

// const AccessControlAllowHeaders = "Origin, Content-Type, Token, Authorization, Mode"

// HandleHTTPGateWay 绑定HTPPHandler 到 grpc-gateway的服务
func HandleHTTPGateWay(method, path string, h http.Handler, mux *runtime.ServeMux) {
	tips := strings.Split(path, "/")
	pattern := runtime.MustPattern(runtime.NewPattern(1, []int{2, 0, 2, 1, 2, 2}, tips, ""))
	mux.Handle(method, pattern, GateWayHandlerGen(h))
}

// GateWayHandlerGen  把普通的HTTP Handler 转换成 GrpcGateway 支持的Handler
func GateWayHandlerGen(h http.Handler) runtime.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
		h.ServeHTTP(w, r)
	}
}

func MiddleWare(component, method string, e endpoint.Endpoint) endpoint.Endpoint {
	return RecoverMiddleware(method)(e)
}

func AddCookieWithDomain(key, value, domain string, w http.ResponseWriter) {
	maxAge := int(time.Hour * 24 * 30 / time.Second)
	cookie := &http.Cookie{
		Name:     key,
		Value:    value,
		Path:     "/",
		HttpOnly: false,
		Domain:   domain,
		MaxAge:   maxAge,
	}
	http.SetCookie(w, cookie)
}

func AddCookie(key, value string, w http.ResponseWriter) {
	cookie := &Cookie{
		Name:     key,
		Value:    value,
		Path:     "/",
		HttpOnly: false,
		Expires:  time.Now().Add(time.Second * 60 * 60 * 24 * 30),
	}
	// http.SetCookie(w, cookie)
	w.Header().Add("set-cookie", cookie.String())
}

func MakeServiceEndpoint(service interface{}, method string) endpoint.Endpoint {
	v := reflect.ValueOf(service)
	methodFunc := v.MethodByName(method)
	if methodFunc.Kind() == reflect.Invalid || methodFunc.IsNil() {
		log.Panicf("绑定的方法名:%s不存在", method)
	}
	return func(ctx context.Context, req interface{}) (interface{}, error) {
		gctx := gcontext.Context(ctx)
		params := make([]reflect.Value, 2)
		params[0] = reflect.ValueOf(gctx)
		params[1] = reflect.ValueOf(req)
		results := methodFunc.Call(params)
		if results[1].Interface() == nil {
			return results[0].Interface(), nil
		} else {
			return results[0].Interface(), results[1].Interface().(error)
		}
	}
}

type DecodeRequestFunc func(context.Context, *http.Request) (request interface{}, err error)

func MakePostRequestDecoder(req interface{}) kithttp.DecodeRequestFunc {
	return func(ctx context.Context, r *http.Request) (out interface{}, err error) {
		reqType := reflect.TypeOf(req).Elem()
		req1 := reflect.New(reqType).Interface()
		if err := json.NewDecoder(r.Body).Decode(req1); err != nil {
			return nil, err
		}
		return req1, nil
	}
}
func MakeFormFilePostRequstDecoder(req interface{}) kithttp.DecodeRequestFunc {
	return func(ctx context.Context, r *http.Request) (out interface{}, err error) {
		reqType := reflect.TypeOf(req).Elem()
		req1 := reflect.New(reqType).Interface()
		// reqValue := reflect.ValueOf(req)
		file, _, err := r.FormFile("file")
		if err != nil {
			return nil, err
		}
		var formValues = make(url.Values)
		content, err := ioutil.ReadAll(file)
		if err != nil {
			return nil, err
		}
		formValues.Set("file", string(content))
		err = GetForm2Struct(req1, formValues)
		if err != nil {
			return nil, err
		}
		return req1, nil
	}
}

func MakeGetRequestDecoder(req interface{}) kithttp.DecodeRequestFunc {
	return func(ctx context.Context, r *http.Request) (out interface{}, err error) {
		r.ParseForm()
		reqType := reflect.TypeOf(req).Elem()
		req1 := reflect.New(reqType).Interface()
		err = GetForm2Struct(req1, r.Form)
		if err != nil {
			return nil, err
		}
		return req1, nil
	}
}

func strFirstToUpper(str string) string {
	result := ""
	vv := []rune(str)
	if vv[0] >= 'a' && vv[1] <= 'z' {
		vv[0] -= 32
	}
	result += string(vv)
	return result
}

func GetForm2Struct(req interface{}, formMap url.Values) error {
	ref := reflect.ValueOf(req).Elem()
	defer func() {
		err := recover()
		if err != nil {
			log.Printf("解析GET参数异常:req=%+v,formMap=%+v,err:%+v", req, formMap, err)
			return
		}
	}()
	for k, v := range formMap {
		if len(v) == 0 {
			continue
		}
		fName := strFirstToUpper(k)
		t := ref.FieldByName(Snack2Camel(fName))
		if t.Kind() == reflect.Invalid {
			t = ref.FieldByName(fName)
		}
		if len(v) == 1 {
			vStr := string(v[0])
			switch t.Kind() {
			case reflect.Bool:
				b, err := strconv.ParseBool(vStr)
				if err != nil {
					continue
				}
				t.SetBool(b)
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				b, err := strconv.ParseInt(vStr, 10, 64)
				if err != nil {
					continue
				}
				t.SetInt(b)
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uintptr, reflect.Uint64:
				b, err := strconv.ParseUint(vStr, 10, 64)
				if err != nil {
					continue
				}
				t.SetUint(b)
			case reflect.Float32, reflect.Float64:
				b, err := strconv.ParseFloat(vStr, 64)
				if err != nil {
					continue
				}
				t.SetFloat(b)
			case reflect.String:
				t.SetString(vStr)
			case reflect.Interface:
				log.Println("不支持interface类型")
			case reflect.Struct:
				log.Println("不支持struct类型")
			case reflect.Map:
				log.Println("不支持map类型")
			case reflect.Slice:
				log.Println("不支持Slice类型")
			case reflect.Array:
				log.Println("不支持Array类型")
			case reflect.Ptr:
				log.Println("不支持Ptr类型")
			default:
			}
		} else {
			log.Println("不支持Slice类型")
		}
	}
	return nil
}

func Snack2Camel(Snack string) string {
	var buf []byte = make([]byte, 0)
	if Snack[0] >= 'a' && Snack[0] <= 'z' {
		buf = append(buf, Snack[0]-32)
	} else {
		buf = append(buf, Snack[0])
	}
	for i := 1; i < len(Snack); i++ {
		if Snack[i-1] == '_' && Snack[i] >= 'a' && Snack[i] <= 'z' {
			buf = append(buf, Snack[i]-32)
		} else if Snack[i] != '_' {
			buf = append(buf, Snack[i])
		}
	}
	return string(buf)
}

// MakeHandler returns a handler for the biz service.
/*
agikit.MakePostHttpHandler 参数解释：
	 1. 把post的body转化成的目标request
	 2. 需要代理的服务接口
	 3. 组件名称
	 4. 代理的服务方法名
*/
func MakePostHttpHandler(request, bs interface{}, component, method string) http.Handler {
	opts := []kithttp.ServerOption{
		//		kithttp.ServerErrorLogger(logger),
		kithttp.ServerErrorEncoder(EncodeError),
	}

	Handler := kithttp.NewServer(
		MiddleWare(component, method, MakeServiceEndpoint(bs, method)),
		MakePostRequestDecoder(request),
		EncodeResponse,
		opts...,
	)
	return AccessControl(Handler)
	//	return MakePostHttpHandlerWithEncoder(request, bs, component, method, EncodeResponse)
}

func MakePostHttpHandlerWithDecoder(request, bs interface{}, component, method string, decoder kithttp.DecodeRequestFunc) http.Handler {
	opts := []kithttp.ServerOption{
		//		kithttp.ServerErrorLogger(logger),
		kithttp.ServerErrorEncoder(EncodeError),
	}

	Handler := kithttp.NewServer(
		MiddleWare(component, method, MakeServiceEndpoint(bs, method)),
		decoder,
		EncodeResponse,
		opts...,
	)
	return AccessControl(Handler)
	//	return MakePostHttpHandlerWithEncoder(request, bs, component, method, EncodeResponse)
}

/*func MakeFilePostHttpHandler(request, bs interface{}, component, method string,encoder kithttp.EncodeResponseFunc ,targetEndpoint kithttp.Endpoint) http.Handler {
	var logger log.Logger
	logger = log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
	logger = log.With(logger, "ts", log.DefaultTimestampUTC)
	logger = log.With(logger, "component", component)

	opts := []kithttp.ServerOption{
		kithttp.ServerErrorLogger(logger),
		kithttp.ServerErrorEncoder(EncodeError),
	}

	Handler := kithttp.NewServer(
		MiddleWare(component, method, MakeServiceEndpoint(bs, method)),
//		MiddleWare(component,method,targetEndpoint)
		MakePostRequestDecoder(request),
		encoder,
		opts...,
	)
	return AccessControl(Handler)

	//	return MakePostHttpHandlerWithEncoder(request, bs, component, method, FileEncodeResponse)
}
*/

func MakePostHttpHandlerWithEncoder(request, bs interface{}, component, method string, encoder kithttp.EncodeResponseFunc) http.Handler {

	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(EncodeError),
	}

	Handler := kithttp.NewServer(
		MiddleWare(component, method, MakeServiceEndpoint(bs, method)),
		MakePostRequestDecoder(request),
		encoder,
		opts...,
	)
	return AccessControl(Handler)
}

// MakeHandler returns a handler for the biz service.
func MakeGetHttpHandler(request, bs interface{}, component, method string) http.Handler {

	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(EncodeError),
	}

	Handler := kithttp.NewServer(
		MiddleWare(component, method, MakeServiceEndpoint(bs, method)),
		MakeGetRequestDecoder(request),
		EncodeResponse,
		opts...,
	)
	return AccessControl(Handler)
	// return MakeGetHttpHandlerWithDecoder(request, bs, component, method, EncodeResponse)
}

// MakeHandler returns a handler for the biz service.
func MakeGetHttpHandlerWithDecoder(request, bs interface{}, component, method string, decoder kithttp.DecodeRequestFunc) http.Handler {

	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(EncodeError),
	}

	Handler := kithttp.NewServer(
		MiddleWare(component, method, MakeServiceEndpoint(bs, method)),
		decoder,
		EncodeResponse,
		opts...,
	)
	return AccessControl(Handler)
}
func MakeGetHttpHandlerWithEncoder(request, bs interface{}, component, method string, encoder kithttp.EncodeResponseFunc) http.Handler {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(EncodeError),
	}
	Handler := kithttp.NewServer(
		MiddleWare(component, method, MakeServiceEndpoint(bs, method)),
		MakeGetRequestDecoder(request),
		encoder,
		opts...,
	)
	return AccessControl(Handler)

}
func MakeFileUpLoadHttpHandler(request, bs interface{}, component, method string) http.Handler {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(EncodeError),
	}
	Handler := kithttp.NewServer(
		MiddleWare(component, method, MakeServiceEndpoint(bs, method)),
		MakeFormFilePostRequstDecoder(request),
		EncodeResponse,
		opts...,
	)
	return AccessControl(Handler)

}

func MakeFileUploadHandler(req interface{}) kithttp.DecodeRequestFunc {
	return func(ctx context.Context, r *http.Request) (out interface{}, err error) {
		reqType := reflect.TypeOf(req).Elem()
		req1 := reflect.New(reqType).Interface()

		// 若使用MultipartReader方法的话，就是能拿到一个Reader，以流的形式玩，而不是直接拿全数据
		file, header, err := r.FormFile("file")
		defer func() {
			if file != nil {
				file.Close()
			}
		}()
		if err != nil {
			return nil, derror.NewNoTipsError(err.Error())
		}
		content, err := ioutil.ReadAll(file)
		if err != nil {
			return nil, err
		}

		formValues := make(url.Values)
		formValues.Set("FileContent", string(content))
		formValues.Set("FileName", header.Filename)
		formValues.Set("FileContentType", header.Header.Get("Content-Type"))

		err = GetForm2Struct(req1, formValues)
		if err != nil {
			return nil, err
		}
		err = GetForm2Struct(req1, r.Form)
		if err != nil {
			return nil, err
		}

		return req1, nil
	}
}

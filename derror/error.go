package derror

import (
	"encoding/json"
	"fmt"
	"runtime"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ErrInterface interface {
	GetInfo() string
	Error() string
	Code() int
	NeedTips() bool
	Message() string
}

const (
	NoTipErrorCode = -1
	TipErrorCode   = 1
)

// 用于定义一些预设的错误值,从1000开始
var NotLogin = &Error{ErrCode: 97, Msg: "用户未登录"}

type GatewayError Error

func (e GatewayError) Error() string {
	str, _ := json.Marshal(e)
	return string(str)
}

type Error struct {
	ErrCode int
	Msg     string
	Info    string
}

func (e Error) string() string {
	return "info:" + e.Info + " Code:" + strconv.Itoa(e.ErrCode) + " Msg:" + e.Msg
}

func GetInfo() string {
	_, file, line, ok := runtime.Caller(2)
	if !ok {
		return ""
	}
	i := strings.LastIndex(file, "/")
	fileName := file[i+1:]
	return fileName + ":" + strconv.Itoa(line)
}

func New(code int, msg string) *Error {
	e := &Error{ErrCode: code, Msg: msg}
	e.Info = GetInfo()
	return e
}

func (e Error) Error() string {
	// jsonobj, _ := json.Marshal(e)
	// return string(jsonobj)
	return e.Msg // + e.info
}

func (e Error) Message() string {
	return e.Msg
}

func (e Error) Code() int {
	return e.ErrCode
}

func (e Error) NeedTips() bool {
	if e.ErrCode == NoTipErrorCode {
		return false
	}
	return true
}

// 返回需要提示的异常，比如用户名不允许为空等
func NewTipsError(msg string) *Error {
	e := &Error{ErrCode: TipErrorCode, Msg: msg}
	e.Info = GetInfo()
	return e
}

// 返回不提示的错误，比如系统数据库异常等，统一处理为系统异常
// 都提示系统错误，errcode=-1
func NewNoTipsError(msg string) *Error {
	e := &Error{ErrCode: NoTipErrorCode, Msg: msg}
	e.Info = GetInfo()
	return e
}

// 支持直接 format
func NewNoTipsErrorf(format string, a ...interface{}) *Error {
	return NewNoTipsError(fmt.Sprintf(format, a...))
}

// 返回GrpcStatus，但若是*WrapError，则塞入其中，透传出去
func GrpcCheckStatus(err error) *status.Status {
	er, ok := errors.Cause(err).(*Error)
	if ok && er != nil {
		s := status.New(codes.Code(er.ErrCode), er.Message())
		return s
	}
	return status.Convert(err)
}

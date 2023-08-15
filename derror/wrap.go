package derror

import (
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	// 使用时请使用 errors.WithStack(derror.UnimplementedErr) ，这样才能重新记录调用堆栈
	UnimplementedErr = Wrap(fmt.Errorf("Unimplemented"), WithCode(uint32(codes.Unimplemented)))
)

const UnknownCode = uint32(codes.Unknown)

// 实现error
func (e *WrapError) Error() string {
	return e.GetMessage()
}

type WrapOption func(*WrapError)

func WithCode(code uint32) WrapOption {
	return func(options *WrapError) {
		options.Code = code
	}
}

func WithUnack(unack ...bool) WrapOption {
	return func(options *WrapError) {
		if len(unack) > 0 {
			options.Unack = unack[0]
		} else {
			options.Unack = true
		}
	}
}

func WithTips(tips ...bool) WrapOption {
	return func(options *WrapError) {
		if len(tips) > 0 {
			options.Tips = tips[0]
		} else {
			options.Tips = true
		}
	}
}

// 友好信息比tips优先级高，必定会透出到前端
func WithFriendlyMessage(friendlyMessage string) WrapOption {
	return func(options *WrapError) {
		options.FriendlyMessage = friendlyMessage
	}
}

func WithLevel(level ErrorLevel) WrapOption {
	return func(options *WrapError) {
		options.Level = level
	}
}

func Code(err error) uint32 {
	er, ok := errors.Cause(err).(*WrapError)
	if ok && er != nil {
		return er.GetCode()
	}
	return UnknownCode
}

func IsUnack(err error) bool {
	er, ok := errors.Cause(err).(*WrapError)
	if ok && er != nil {
		return er.GetUnack()
	}
	return false
}

func ShouldTips(err error) bool {
	er, ok := errors.Cause(err).(*WrapError)
	if ok && er != nil {
		return er.GetTips()
	}
	return false
}

func FriendlyMessage(err error) string {
	er, ok := errors.Cause(err).(*WrapError)
	if ok && er != nil {
		return er.GetFriendlyMessage()
	}
	return ""
}

func Level(err error) ErrorLevel {
	er, ok := errors.Cause(err).(*WrapError)
	if ok && er != nil {
		return er.GetLevel()
	}
	return ErrorLevel_Common
}

func Wrap(err error, options ...WrapOption) error {
	if err == nil {
		return nil
	}

	// 拿取 *WrapError 内容
	er, ok := errors.Cause(err).(*WrapError)
	if !ok {
		er = &WrapError{
			Message: err.Error(),
			Code:    uint32(UnknownCode),
		}
		err = er
	}

	// 极端情况下还是有可能为nil的
	if er == nil {
		return nil
	}
	// 设置新的属性
	for _, opt := range options {
		opt(er)
	}

	// 如果没包过，就包一下，否则也不包了，保留起源的堆栈信息
	type causer interface {
		Cause() error
	}
	if _, ok = err.(causer); !ok {
		err = errors.WithStack(er)
	}

	return err
}

// 还是提供几个相对更常用的一个方法吧

// 快速WithTips的Wrap方法
func WrapWithTips(err error, options ...WrapOption) error {
	options = append(options, WithTips())
	return Wrap(err, options...)
}

// 一般需要单独构造的error信息的话，大部分都是需要tips的，所以这种场景下，此方法常用
func WrapfWithTips(format string, a ...interface{}) error {
	return Wrap(fmt.Errorf(format, a...), WithTips())
}

// 这个一般不会用到，原因看WrapfWithTips注释
func Wrapf(format string, a ...interface{}) error {
	return Wrap(fmt.Errorf(format, a...))
}

// // 快速设置友好信息
// func WrapWithFMf(err error, format string, a ...interface{}) error {
// 	return Wrap(err, WithFriendlyMessage(fmt.Sprintf(format, a...)))
// }

// 快速设置友好信息
func WrapWithFriendlyMessagef(err error, format string, a ...interface{}) error {
	return Wrap(err, WithFriendlyMessage(fmt.Sprintf(format, a...)))
}

// 返回GrpcStatus，但若是*WrapError，则塞入其中，透传出去
func GrpcStatus(err error) *status.Status {
	er, ok := errors.Cause(err).(*WrapError)
	if ok && er != nil {
		s, _ := status.New(codes.Code(er.Code), er.Message).WithDetails(er)
		return s
	}

	return status.Convert(err)
}

// 从Status里转出*WrapError
func WrapWithGrpcStatus(s *status.Status) error {
	ds := s.Details()
	if len(ds) > 0 {
		if m, ok := ds[0].(proto.Message); ok && m != nil {
			if er, ok := m.(*WrapError); ok && er != nil {
				return Wrap(er)
			}
		}
	}

	if s.Err() == nil {
		return nil
	}

	return Wrap(fmt.Errorf(s.Message()), WithCode(uint32(s.Code())))
}

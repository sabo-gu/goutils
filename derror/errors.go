package derror

import (
	"fmt"

	"github.com/pkg/errors"
)

func NewError(e interface{}, forceTips ...bool) error {
	str := ""
	forceTip := false
	if err, ok := e.(error); ok {
		str = err.Error()
		forceTip = false
	} else if s, ok := e.(string); ok {
		str = s
		forceTip = true
	}

	if len(forceTips) > 0 {
		forceTip = forceTips[0]
	}

	if forceTip {
		//AgiError的类型判断利用 errors.Cause
		return errors.WithStack(NewTipsError(str))
		// return errors.WithStack(fmt.Errorf(str))
	}

	return errors.WithStack(NewNoTipsError(str))
	// return errors.WithStack(fmt.Errorf(str))
}

func Errorf(format string, a ...interface{}) error {
	msg := fmt.Sprintf(format, a...)
	return NewError(msg, true)
}

func NoTipsErrorf(format string, a ...interface{}) error {
	msg := fmt.Sprintf(format, a...)
	return NewError(msg, false)
}

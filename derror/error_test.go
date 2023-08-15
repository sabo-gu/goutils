package derror

import (
	"fmt"
	"testing"
)

func test_test() error {
	return NewTipsError("测试")
}

func TestGetInfo(t *testing.T) {
	e := test_test()
	fmt.Printf("%#v", e.(*Error))
}

func TestGatewayError(t *testing.T) {
	e := test_test().(*Error)
	ge := (*GatewayError)(e)
	fmt.Println(ge.Error())
	fmt.Println(e.Error())
}

type zzz struct {
	c int
}

func TestNestedStruct(t *testing.T) {
	x := struct {
		*zzz
		b int
	}{
		b: 5,
	}
	fmt.Println(x)
}

func returnWrapError2() error {
	var err error
	err = &WrapError{}
	err = Wrap(err, WithTips())
	err = Wrap(err, WithUnack())
	err = Wrap(err, WithTips(false))
	return err
}

func returnWrapError1() error {
	err := returnWrapError2()
	err = Wrap(err, WithTips(false))
	return err
}

func TestWrapError(t *testing.T) {
	err := returnWrapError1()
	err = Wrap(err, WithCode(1024))
	err = Wrap(err, WithFriendlyMessage("你好，baby"))

	fmt.Printf("%+v\n", err)

	if IsUnack(err) {
		fmt.Println("is unack")
	} else {
		fmt.Println("is not unack")
	}

	if ShouldTips(err) {
		fmt.Println("should tips")
	} else {
		fmt.Println("should not tips")
	}

	fmt.Println(Code(err))
	fmt.Println(FriendlyMessage(err))
}

func TestGrpcStatus(t *testing.T) {
	err := fmt.Errorf("orig")
	err = Wrap(err, WithTips())
	err = Wrap(err, WithUnack())
	err = Wrap(err, WithTips(false))
	err = Wrap(err, WithCode(2), WithFriendlyMessage("你好，baby"))

	fmt.Printf("err:\n%+v\n\n", err)

	s := GrpcStatus(err)
	fmt.Printf("status:\n%+v\n\n", s.Proto())

	// 重置了err，从status里获取了
	err = WrapWithGrpcStatus(s)
	fmt.Printf("er:\n%+v\n\n", err)

	if IsUnack(err) {
		fmt.Println("is unack")
	} else {
		fmt.Println("is not unack")
	}

	if ShouldTips(err) {
		fmt.Println("should tips")
	} else {
		fmt.Println("should not tips")
	}

	fmt.Println(Code(err))
	fmt.Println(FriendlyMessage(err))

	fmt.Println(Wrap(nil))
}

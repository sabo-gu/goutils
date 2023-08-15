package grpc_http

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io"
	"log"

	"github.com/bitly/go-simplejson"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"

	"github.com/DoOR-Team/goutils/derror"
	"github.com/DoOR-Team/goutils/djson"
)

// JSONBuiltin is a Marshaler which marshals/unmarshals into/from JSON
// with the standard "encoding/json" package of Golang.
// Although it is generally faster for simple proto messages than JSONPb,
// it does not support advanced features of protobuf, e.g. map, oneof, ....
/*
// Marshal marshals "v" into byte sequence.
	Marshal(v interface{}) ([]byte, error)
	// Unmarshal unmarshals "data" into "v".
	// "v" must be a pointer value.
	Unmarshal(data []byte, v interface{}) error
	// NewDecoder returns a Decoder which reads byte sequence from "r".
	NewDecoder(r io.Reader) Decoder
	// NewEncoder returns an Encoder which writes bytes sequence into "w".
	NewEncoder(w io.Writer) Encoder
	// ContentType returns the Content-Type which this marshaler is responsible for.
	ContentType() string
*/
type JSONBuiltin struct{}

type emptyResult struct {}

// ContentType always Returns "application/json".
func (*JSONBuiltin) ContentType() string {
	return "application/json"
}

type errorBody struct {
	Error string `protobuf:"bytes,1,name=error" json:"error"`
	Code  int32  `protobuf:"varint,2,name=code" json:"code"`
}

func GobDeepCopy(dst, src interface{}) error {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(src); err != nil {
		log.Printf("进行深度拷贝发生异常，src=%+v,dst=%+v", src, dst)
		return err
	}
	return gob.NewDecoder(bytes.NewBuffer(buf.Bytes())).Decode(dst)
}
func IsErrorMsg(v interface{}) (*errorBody, bool) {
	m, err := simplejson.NewJson([]byte(djson.ToJsonString(v)))
	if err != nil {
		return nil, false
	}
	mmap, err := m.Map()
	if err != nil {
		fmt.Println(err)
		return nil, false
	}
	if len(mmap) == 2 {
		errStr, err := m.Get("Msg").String()
		if err != nil {
			//fmt.Println(err.Error())
			return nil, false
		}
		codeNum, err := m.Get("ErrCode").Int()
		if err != nil {
			return nil, false
		}
		eb := &errorBody{
			Error: errStr,
			Code:  int32(codeNum),
		}
		return eb, true
	} else if len(mmap) == 3 {
		errStr, err := m.Get("Msg").String()
		if err != nil {
			fmt.Println(err.Error())
			return nil, false
		}
		codeNum, err := m.Get("ErrCode").Int()
		if err != nil {
			return nil, false
		}
		_, err = m.Get("Info").String()
		if err != nil {
			return nil, false
		}
		eb := &errorBody{
			Error: errStr,
			Code:  int32(codeNum),
		}
		return eb, true
	}
	return nil, false
}

func IsErrorBody(v interface{}) (*errorBody, bool) {
	m, err := simplejson.NewJson([]byte(djson.ToJsonString(v)))
	if err != nil {
		return nil, false
	}
	mmap, err := m.Map()
	if err != nil {
		fmt.Println(err)
		return nil, false
	}
	if len(mmap) == 2 {
		errStr, err := m.Get("message").String()
		if err != nil {
			//		fmt.Println(err.Error())
			return nil, false
		}
		codeNum, err := m.Get("code").Int()
		if err != nil {
			return nil, false
		}
		eb := &errorBody{
			Error: errStr,
			Code:  int32(codeNum),
		}
		return eb, true
	}
	return nil, false
}

// Marshal marshals "v" into JSON
func (j *JSONBuiltin) Marshal(v interface{}) ([]byte, error) {
	var eb *errorBody
	eb1, ok := IsErrorMsg(v)
	if ok {
		eb = eb1
	} else {
		eb2, ok2 := IsErrorBody(v)
		if ok2 {
			ok = ok2
			eb = eb2
		}
	}
	if ok {
		// err := &derror.Error{}
		//整个error体可能被存储在eb.Error中
		// marshalErr := json.Unmarshal([]byte(eb.Error), err)
		// if marshalErr != nil {
		// 	err = &derror.Error{
		// 		ErrCode: int(eb.Code),
		// 		Msg:     eb.Error,
		// 	}
		// }
		if eb.Code == derror.NoTipErrorCode {
			return json.Marshal(map[string]interface{}{
				"errText":   "系统错误",
				"errCode":   eb.Code,
				"data":      emptyResult{},
				"debugInfo": eb.Error,
			})
		} else {
			return json.Marshal(map[string]interface{}{
				"errText": eb.Error,
				"errCode": eb.Code,
				"data":    emptyResult{},
			})
		}
	}
	// log.Println("Is not ErrorBody")
	return json.Marshal(map[string]interface{}{
		"errText": "",
		"errCode": 0,
		"data":    v,
	})
	// }
	// return Marshal(v)
}

// Unmarshal unmarshals JSON data into "v".
func (j *JSONBuiltin) Unmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

// NewDecoder returns a Decoder which reads JSON stream from "r".
func (j *JSONBuiltin) NewDecoder(r io.Reader) runtime.Decoder {
	return json.NewDecoder(r)
}

// NewEncoder returns an Encoder which writes JSON stream into "w".
func (j *JSONBuiltin) NewEncoder(w io.Writer) runtime.Encoder {
	return json.NewEncoder(w)
}

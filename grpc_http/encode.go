package grpc_http

import (
	"encoding/json"
	"net/http"

	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"

	"github.com/DoOR-Team/goutils/derror"
)

// encode errors from business-logic
func EncodeError(_ context.Context, err error, w http.ResponseWriter) {
	if w.Header().Get("Access-Control-Allow-Origin") == "" {
		w.Header().Set("Content-Type", "application/json; charset=utf-8;")
		w.Header().Set("Access-Control-Allow-Origin", viper.GetString("access_control_allow_origin"))
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", AccessControlAllowHeaders)
		w.Header().Set("Access-Control-Allow-Credentials", "true")
	}
	oriErr := err
	if err, ok := errors.Cause(err).(*derror.Error); ok {
		errText := err.Msg
		debugInfo := ""
		if !err.NeedTips() {
			debugInfo = err.Error()
			errText = "系统错误"
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"errText":   errText,
			"errCode":   err.Code(),
			"data":      "",
			"debugInfo": debugInfo,
		})
	} else {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"errText": oriErr.Error(),
			"errCode": 1,
			"data":    "",
		})
	}
}

func EncodeResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	if w.Header().Get("Access-Control-Allow-Origin") == "" {
		w.Header().Set("Content-Type", "application/json; charset=utf-8;")
		w.Header().Set("Access-Control-Allow-Origin", viper.GetString("access_control_allow_origin"))
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", AccessControlAllowHeaders)
		w.Header().Set("Access-Control-Allow-Credentials", "true")
	}

	err := json.NewEncoder(w).Encode(map[string]interface{}{
		"errText": "成功",
		"errCode": 0,
		"data":    response,
	})
	return err
}

// encode errors from business-logic
func EncodeGrpcError(_ context.Context, err error, w http.ResponseWriter) {
	if w.Header().Get("Access-Control-Allow-Origin") == "" {
		w.Header().Set("Content-Type", "application/json; charset=utf-8;")
		w.Header().Set("Access-Control-Allow-Origin", viper.GetString("access_control_allow_origin"))
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", AccessControlAllowHeaders)
		w.Header().Set("Access-Control-Allow-Credentials", "true")
	}
	if err, ok := errors.Cause(err).(*derror.Error); ok {
		if err.NeedTips() {
			json.NewEncoder(w).Encode(&errorBody{
				Error: "alert:" + err.Message(),
				Code:  int32(codes.OK),
			})
		} else {
			json.NewEncoder(w).Encode(&errorBody{
				Error: err.Error(),
				Code:  int32(codes.OK),
			})
		}

	} else {
		json.NewEncoder(w).Encode(&errorBody{
			Error: err.Error(),
			Code:  int32(codes.OK),
		})
	}
}

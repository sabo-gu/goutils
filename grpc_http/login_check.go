package grpc_http

import (
	"net/http"
	"strings"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/spf13/viper"
	"golang.org/x/net/context"

	"github.com/DoOR-Team/goutils/derror"
)

const CtxKeyRemoteIp = "remoteIP"
const SessionUid = "uid"
const DEBUG = "debug"
const USERINFO = "user_info"

var debugUid = UserClaims{
	Uid:     "debug_id",
	RealUid: "debug_id",
	Name:    "debug",
}

type UserClaims struct {
	*jwt.StandardClaims
	Uid     string
	RealUid string
	Name    string
}

func ParseToken(tokenSrt string, SecretKey []byte) (*UserClaims, error) {
	var token *jwt.Token
	token, err := jwt.ParseWithClaims(tokenSrt, &UserClaims{}, func(*jwt.Token) (interface{}, error) {
		return SecretKey, nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*UserClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, derror.NewNoTipsError("token解析失败或token无效")
}

func LoginHttpMidWare(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ips := r.Header.Get("X-Forwarded-For")
		if ips != "" {
			ctx := r.Context()
			ctx = context.WithValue(ctx, CtxKeyRemoteIp, ips)
			r = r.WithContext(ctx)
		}
		debug := false
		if r.Header.Get("Mode") == "debug" {
			debug = true
		}

		// 更换token获取方式，如果为空，还使用原有方式
		auth := r.Header.Get("Authorization")
		prefix := "Bearer "

		// Bearer Auth 模式
		if auth != "" && strings.HasPrefix(auth, prefix) {
			auth = auth[len(prefix):]
		} else {
			if auth == "" {
				// Header Token模式
				auth = r.Header.Get("Token")
			}
			if auth == "" {
				// 参数access_token模式
				auth = r.FormValue("access_token")
			}
		}
		if auth == "" && !debug {
			EncodeError(r.Context(), derror.NotLogin, w)
			return
		}
		// log.Info("auth:", auth)

		authInfo, err := ParseToken(auth, []byte(viper.GetString("token_secret")))

		if err != nil && !debug {
			EncodeError(r.Context(), derror.New(derror.NotLogin.ErrCode, "登录已超时"), w)
			return
		}
		// fmt.Println(authInfo.UserID)
		if authInfo != nil && authInfo.ExpiresAt < time.Now().Unix() && !debug {
			EncodeError(r.Context(), derror.NotLogin, w)
			return
		}
		ctx := r.Context()
		if authInfo != nil {
			ctx = context.WithValue(ctx, USERINFO, authInfo)
			ctx = context.WithValue(ctx, SessionUid, authInfo.Id)
		} else if debug {
			ctx = context.WithValue(ctx, USERINFO, &debugUid)
			ctx = context.WithValue(ctx, SessionUid, debugUid.Uid)
		} else {
			EncodeError(r.Context(), derror.NotLogin, w)
			return
		}

		if debug {
			ctx = context.WithValue(ctx, DEBUG, debug)
		}
		r = r.WithContext(ctx)

		h.ServeHTTP(w, r)
	})
}

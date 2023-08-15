package grpc_http

import (
	"context"
	"net"
	"net/http"
	"strings"

	"github.com/spf13/viper"

	"github.com/DoOR-Team/goutils/log"
	"github.com/DoOR-Team/goutils/tracing"
	"github.com/DoOR-Team/goutils/waitgroup"
)

func WrapListenAndServe(r http.Handler) error {
	lis, err := net.Listen("tcp", viper.GetString("http_addr"))
	if err != nil {
		log.Panicf("listen grpc_http(%s) failed - %+v", lis.Addr().String(), err)
	}

	s := &http.Server{
		Handler: r,
		// ErrorLog: log.New(os.Stdout, "", 0),
	}
	err = waitgroup.AddModAndWrapServer("HTTP_Server", &Srv{
		ServeFunc: func() error {
			log.Infof("listen grpc_http(%s)", lis.Addr().String())
			return s.Serve(lis)
		},
		CloseFunc: func() error {
			return s.Close()
		},
	})
	if err != nil {
		panic(err)
	}
	return err
}

func WrapListenAndServeWithTls(r http.Handler, crtfile, keyfile string) error {
	lis, err := net.Listen("tcp", viper.GetString("http_addr"))
	if err != nil {
		log.Panicf("listen grpc_http(%s) failed - %+v", lis.Addr().String(), err)
	}

	s := &http.Server{
		Handler: r,
		// ErrorLog: log.New(os.Stdout, "", 0),
	}
	err = waitgroup.AddModAndWrapServer("HTTP_Server", &Srv{
		ServeFunc: func() error {
			log.Infof("listen grpc_http(%s)", lis.Addr().String())
			return s.ServeTLS(lis, crtfile, keyfile)
		},
		CloseFunc: func() error {
			return s.Close()
		},
	})
	if err != nil {
		panic(err)
	}
	return err
}

const HETU_HTTP_METHOD string = "HETU_HTTP_METHOD"
const HETU_HTTP_COOKIE string = "HETU_HTTP_COOKIE"
const HETU_HTTP_REQUESTURI string = "HETU_HTTP_REQUESTURI"

func OptionControl(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" {
			origin = r.Referer()
			if origin != "" && strings.HasSuffix(origin, "/") {
				origin = origin[:len(origin)-1]
			}
		}
		if viper.GetString("access_control_allow_origin") == "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		} else {
			originList := strings.Split(viper.GetString("access_control_allow_origin"), ",")
			for _, allowOrigin := range originList {
				if origin == allowOrigin {
					w.Header().Set("Access-Control-Allow-Origin", origin)
				}
			}
			//w.Header().Set("Access-Control-Allow-Origin", viper.GetString("access_control_allow_origin"))
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8;")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", AccessControlAllowHeaders)
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		//w.Header().Set("Connection", "close")
		if r.Method == "OPTIONS" {
			return
		}

		h.ServeHTTP(w, r)
	})
}

func TraceControl(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// if r.URL.Path == "/healthz" {
		// 	h.ServeHTTP(w, r)
		// 	return
		// }

		ctx := r.Context()
		if ctx.Value(HETU_HTTP_METHOD) == nil {
			ctx = context.WithValue(ctx, HETU_HTTP_METHOD, r.Method)
			cookiestr := make([]string, 0)
			for index := range r.Cookies() {
				cookiestr = append(cookiestr, r.Cookies()[index].String())
			}
			ctx = context.WithValue(ctx, HETU_HTTP_COOKIE, cookiestr)
			ctx = context.WithValue(ctx, HETU_HTTP_REQUESTURI, r.RequestURI)
			r = r.WithContext(ctx)
		}
		tracing.HttpTraceMiddleware(h).ServeHTTP(w, r)
	})

}

func AccessControl(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" {
			origin = r.Referer()
			if origin != "" && strings.HasSuffix(origin, "/") {
				origin = origin[:len(origin)-1]
			}
		}
		if viper.GetString("access_control_allow_origin") == "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		} else {
			originList := strings.Split(viper.GetString("access_control_allow_origin"), ",")
			for _, allowOrigin := range originList {
				if origin == allowOrigin {
					w.Header().Set("Access-Control-Allow-Origin", origin)
				}
			}
			//w.Header().Set("Access-Control-Allow-Origin", viper.GetString("access_control_allow_origin"))1
		}
		if strings.HasSuffix(r.URL.Path, ".png") || strings.HasSuffix(r.URL.Path, ".jpg") {
			w.Header().Set("Content-Type", "image/gif")
		} else if strings.HasSuffix(r.URL.Path, ".wav") {
			w.Header().Set("Content-Type", "audio/x-wav")
		} else {
			w.Header().Set("Content-Type", "application/json; charset=utf-8;")
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", AccessControlAllowHeaders)
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		//w.Header().Set("Connection", "close")
		if r.Method == "OPTIONS" {
			return
		}
		ctx := r.Context()
		if ctx.Value(HETU_HTTP_METHOD) == nil {
			ctx = context.WithValue(ctx, HETU_HTTP_METHOD, r.Method)
			cookiestr := make([]string, 0)
			for index := range r.Cookies() {
				cookiestr = append(cookiestr, r.Cookies()[index].String())
			}
			ctx = context.WithValue(ctx, HETU_HTTP_COOKIE, cookiestr)
			ctx = context.WithValue(ctx, HETU_HTTP_REQUESTURI, r.RequestURI)
			r = r.WithContext(ctx)
		}
		h.ServeHTTP(w, r)
		// h.ServeHTTP(w, r)
	})
}

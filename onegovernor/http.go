/*
 *  ┏┓      ┏┓
 *┏━┛┻━━━━━━┛┻┓
 *┃　　　━　　  ┃
 *┃   ┳┛ ┗┳   ┃
 *┃           ┃
 *┃     ┻     ┃
 *┗━━━┓     ┏━┛
 *　　 ┃　　　┃神兽保佑
 *　　 ┃　　　┃代码无BUG！
 *　　 ┃　　　┗━━━┓
 *　　 ┃         ┣┓
 *　　 ┃         ┏┛
 *　　 ┗━┓┓┏━━┳┓┏┛
 *　　   ┃┫┫  ┃┫┫
 *      ┗┻┛　 ┗┻┛
 @Time    : 2024/11/12 -- 11:43
 @Author  : 亓官竹 ❤️ MONEY
 @Copyright 2024 亓官竹
 @Description: http.go
*/

package onegovernor

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"runtime/debug"
	"strings"
	"time"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/xneogo/eins/onelog"
	"github.com/xneogo/eins/oneprom"
)

type Ignore func(w http.ResponseWriter, r *http.Request) bool

func Recover(h http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rvr := recover(); rvr != nil {
				if rvr == http.ErrAbortHandler {
					// we don't recover http.ErrAbortHandler so the response
					// to the client is aborted, this should not be logged
					panic(rvr)
				}

				ctx := r.Context()
				httpRequest, _ := httputil.DumpRequest(r, false)
				if err, ok := rvr.(error); ok {
					onelog.Ctx(ctx).Error().Err(err).Str("trace", string(debug.Stack())).Bytes("request", httpRequest).Msg("panic")
				} else {
					onelog.Ctx(ctx).Error().Any("error", rvr).Str("trace", string(debug.Stack())).Bytes("request", httpRequest).Msg("panic")
				}

				w.WriteHeader(http.StatusInternalServerError)
			}
		}()

		h.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}

func Prom(ignores ...Ignore) func(h http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			for _, ignore := range ignores {
				if ignore(w, r) {
					h.ServeHTTP(w, r)
					return
				}
			}

			handlerName := strings.Trim(r.URL.Path, "/")
			handlerName = strings.ReplaceAll(handlerName, "/", "_")
			handlerName = fmt.Sprintf("%s-%s", handlerName, r.Method)
			handlerName = strings.ToUpper(handlerName)

			oneprom.ServerStartedCounter(handlerName)

			ww := chimiddleware.NewWrapResponseWriter(w, r.ProtoMajor)
			h.ServeHTTP(ww, r)

			oneprom.ServerHandledCounter(handlerName, ww.Status())
			oneprom.ServerHandledHistogram(handlerName, time.Since(start))
		}

		return http.HandlerFunc(fn)
	}
}

func Logger(ignores ...Ignore) func(h http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			ctx := r.Context()
			requestID := r.Header.Get("X-Request-ID")
			if requestID == "" {
				requestID = uuid.NewString()
			}
			w.Header().Set("X-Request-ID", requestID)
			ctx = context.WithValue(ctx, requestIdCtxKey, requestID)
			logger := onelog.Ctx(ctx).With().Str("requestId", requestID).Logger()
			ctx = onelog.WithContext(ctx, logger)
			r = r.WithContext(ctx)

			for _, ignore := range ignores {
				if ignore(w, r) {
					h.ServeHTTP(w, r)
					return
				}
			}

			ww := chimiddleware.NewWrapResponseWriter(w, r.ProtoMajor)
			h.ServeHTTP(ww, r)

			log.Ctx(ctx).Info().
				Int("status", ww.Status()).
				Int("size", ww.BytesWritten()).
				Str("method", r.Method).
				Str("host", r.Host).
				Str("path", r.URL.Path).
				Str("query", r.URL.RawQuery).
				Str("ip", r.RemoteAddr).
				Str("ua", r.UserAgent()).
				Str("refer", r.Referer()).
				Str("cost", time.Since(start).String()).
				Msg("request")
		}

		return http.HandlerFunc(fn)
	}
}

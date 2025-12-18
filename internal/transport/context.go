package transport

import (
	"context"
	"net/http"
)

type ctxKey string

const (
	requestKey        ctxKey = "httpRequest"
	responseWriterKey ctxKey = "httpResponseWriter"
)

func WithHTTP(ctx context.Context, r *http.Request, w http.ResponseWriter) context.Context {
	ctx = context.WithValue(ctx, requestKey, r)
	ctx = context.WithValue(ctx, responseWriterKey, w)
	return ctx
}

func GetRequest(ctx context.Context) *http.Request {
	r, _ := ctx.Value(requestKey).(*http.Request)
	return r
}

func GetResponseWriter(ctx context.Context) http.ResponseWriter {
	w, _ := ctx.Value(responseWriterKey).(http.ResponseWriter)
	return w
}

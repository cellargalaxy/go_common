package util

import "context"

const ignoreErrKey = "ignoreErr"

func GetCtxValue(ctx context.Context, key string) interface{} {
	return ctx.Value(key)
}

func SetCtxValue(ctx context.Context, key string, value interface{}) context.Context {
	return context.WithValue(ctx, key, value)
}

func IsIgnoreErr(ctx context.Context) bool {
	isIgnoreP := GetCtxValue(ctx, ignoreErrKey)
	isIgnore, ok := isIgnoreP.(bool)
	return isIgnore && ok
}

func SetIgnoreErr(ctx context.Context, isIgnoreErr bool) context.Context {
	if IsIgnoreErr(ctx) == isIgnoreErr {
		return ctx
	}
	return SetCtxValue(ctx, ignoreErrKey, isIgnoreErr)
}

func GenCtx() context.Context {
	ctx := context.Background()
	ctx = SetLogId(ctx)
	return ctx
}

func CtxDone(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}

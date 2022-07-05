package util

import "context"

const ignoreErrKey = "ignoreErr"

func IsIgnoreErr(ctx context.Context) bool {
	isIgnoreP := GetCtxValue(ctx, ignoreErrKey)
	isIgnore, ok := isIgnoreP.(bool)
	return isIgnore && ok
}

func SetIgnoreErr(ctx context.Context, isIgnoreErr bool) context.Context {
	return SetCtxValue(ctx, ignoreErrKey, isIgnoreErr)
}

func GetCtxValue(ctx context.Context, key string) interface{} {
	return ctx.Value(key)
}

func SetCtxValue(ctx context.Context, key string, value interface{}) context.Context {
	return context.WithValue(ctx, key, value)
}

func GenCtx() context.Context {
	ctx := context.Background()
	ctx = SetLogId(ctx)
	return ctx
}

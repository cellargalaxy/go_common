package util

import "context"

const ignoreErrKey = "ignoreErr"

func GetCtxValue[T any](ctx context.Context, key string) T {
	value := ctx.Value(key)
	object, _ := value.(T)
	return object
}
func SetCtxValue(ctx context.Context, key string, value interface{}) context.Context {
	return context.WithValue(ctx, key, value)
}

func IsIgnoreErr(ctx context.Context) bool {
	return GetCtxValue[bool](ctx, ignoreErrKey)
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
func CopyCtx(old context.Context) context.Context {
	ctx := GenCtx()
	ctx = SetIgnoreErr(ctx, IsIgnoreErr(old))
	ctx = SetCtxValue(ctx, LogIdKey, GetLogId(old))
	ctx = SetCtxValue(ctx, ReqIdKey, GetReqId(old))
	return ctx
}

func CancelCtx(cancels ...func()) {
	for i := range cancels {
		if cancels[i] == nil {
			continue
		}
		cancels[i]()
	}
}
func CtxDone(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}

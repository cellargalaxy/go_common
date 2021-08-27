package util

import "context"

const ignoreErrKey = "ignoreErr"

func IsIgnoreErr(ctx context.Context) bool {
	isIgnoreP := ctx.Value(ignoreErrKey)
	isIgnore, ok := isIgnoreP.(bool)
	return isIgnore && ok
}

func SetIgnoreErr(ctx context.Context, isIgnoreErr bool) context.Context {
	return context.WithValue(ctx, ignoreErrKey, isIgnoreErr)
}

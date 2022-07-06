package util

func init() {
	ctx := GenCtx()
	initCache()
	initRegexp()
	initHttp(ctx)
}

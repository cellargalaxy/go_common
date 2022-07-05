package util

func init() {
	ctx := GenCtx()
	initHttp(ctx)
	initCache()
	initRegexp()
}

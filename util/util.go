package util

func init() {
	ctx := GenCtx()
	initCache(ctx)
	initRegexp()
	initHttp(ctx)
}

func Init(serverName string) {
	InitOs(serverName)
	InitDefaultLog()
}

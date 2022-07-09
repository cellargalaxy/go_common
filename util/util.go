package util

func init() {
	ctx := GenCtx()
	initCache()
	initRegexp()
	initHttp(ctx)
}

func Init(serverName string) {
	InitOs(serverName)
	InitDefaultLog()
}

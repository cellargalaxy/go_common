package util

func init() {
	ctx := GenCtx()
	initRegexp()
	initHttp(ctx)
}

func Init(serverName string) {
	InitOs(serverName)
	InitDefaultLog()
}

package util

func init() {
	ctx := GenCtx()
	InitDefaultLog()
	initRegexp()
	initHttp(ctx)
}

func Init(serverName string) {
	InitOs(serverName)
}

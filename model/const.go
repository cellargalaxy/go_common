package model

const (
	SuccessCode    = 1
	FailCode       = 2
	ReRequestCode  = 3
	IllegalUriCode = 4
	NotLoginCode   = 5
)

const (
	PingPath   = "/api/ping"
	StaticPath = "/static"
	DebugPath  = "/debug"
	PprofPath  = "/pprof"
)

const (
	DbBatchLen = 1000
)

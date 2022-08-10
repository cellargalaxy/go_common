package model

const (
	HttpSuccessCode    = 1
	HttpFailCode       = 2
	HttpReRequestCode  = 3
	HttpIllegalUriCode = 4
)

const (
	PingPath   = "/ping"
	StaticPath = "/static"
	DebugPath  = "/debug"
	PprofPath  = "/pprof"
)

const DbMaxBatchAddLength = 1000

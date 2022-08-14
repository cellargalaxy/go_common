package model

const (
	HttpSuccessCode    = 1
	HttpFailCode       = 2
	HttpReRequestCode  = 3
	HttpIllegalUriCode = 4
)

const (
	BoolYes = 1
	BoolNo  = 2
)

func Bool2Const(value bool) int {
	if value {
		return BoolYes
	}
	return BoolNo
}
func Const2Bool(value int) bool {
	return value == BoolYes
}

const (
	PingPath   = "/ping"
	StaticPath = "/static"
	DebugPath  = "/debug"
	PprofPath  = "/pprof"
)

const (
	DbMaxBatchAddLength = 1000
)

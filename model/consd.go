package model

const (
	HttpSuccessCode    = 1
	HttpFailCode       = 2
	HttpReRequestCode  = 3
	HttpIllegalUriCode = 4
)

type Bool int

const (
	BoolYes Bool = 1
	BoolNo  Bool = 2
)

func Bool2Const(value bool) Bool {
	if value {
		return BoolYes
	}
	return BoolNo
}
func Const2Bool(value Bool) bool {
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

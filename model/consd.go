package model

const (
	HttpSuccessCode    = 1
	HttpFailCode       = 2
	HttpReRequestCode  = 3
	HttpIllegalUriCode = 4
)

type Bool int

func (this Bool) Bool() bool {
	return this == BoolYes
}

const (
	BoolYes Bool = 1
	BoolNo  Bool = 2
)

func NewBool(value bool) Bool {
	if value {
		return BoolYes
	}
	return BoolNo
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

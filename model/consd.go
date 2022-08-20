package model

const (
	HttpSuccessCode    = 1
	HttpFailCode       = 2
	HttpReRequestCode  = 3
	HttpIllegalUriCode = 4
)
const (
	BoolYes Bool = 1
	BoolNo  Bool = 2
)

type Bool int

func (this Bool) Value() bool {
	return this == BoolYes
}

func NewBool(value bool) Bool {
	if value {
		return BoolYes
	}
	return BoolNo
}

type Int int

func (this Int) Value() int {
	return int(this)
}

func NewInt(value int) Int {
	return Int(value)
}

type Int64 int64

func (this Int64) Value() int64 {
	return int64(this)
}

func NewInt64(value int64) Int64 {
	return Int64(value)
}

type Float64 float64

func (this Float64) Value() float64 {
	return float64(this)
}

func NewFloat64(value float64) Float64 {
	return Float64(value)
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

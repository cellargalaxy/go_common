package util

import (
	"testing"
)

func TestTIme(t *testing.T) {
	ctx := GenCtx()

	ttt, err := ParseStr2Time(ctx, DateLayout_2006_01_02, "2023-01-01", E8Loc)
	if err != nil {
		t.Errorf("%+v", err)
		return
	}
	if Time2MsTs(ttt) != int64(1672502400000) {
		t.Errorf(`if Time2MsTs(ttt) != int64(1672502400000) {`)
		return
	}
}

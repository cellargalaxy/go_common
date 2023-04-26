package util

import (
	"testing"
	"time"
)

func TestGenId(t *testing.T) {
	ctx := GenCtx()

	now := time.Now()

	id := GenIdByTime(now)
	tt, err := ParseId(ctx, id)
	if err != nil {
		t.Errorf("%+v", err)
		return
	}
	if tt.Unix() != now.Unix() {
		t.Errorf(`if tt.Unix() != now.Unix() {`)
		return
	}
}

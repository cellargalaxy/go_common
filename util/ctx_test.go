package util

import (
	"context"
	"testing"
)

func TestCtx(t *testing.T) {
	ctx := GenCtx()

	ctx = SetIgnoreErr(ctx, true)
	if !IsIgnoreErr(ctx) {
		t.Errorf(`if !IsIgnoreErr(ctx) {`)
		return
	}
	ctx, fun := context.WithCancel(ctx)
	fun()
	if !CtxDone(ctx) {
		t.Errorf(`if !CtxDone(ctx) {`)
		return
	}
}

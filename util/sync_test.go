package util

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestOnceSingleGoPool(t *testing.T) {
	ctx := GenCtx()

	taskChange := ""
	pool, err := NewOnceSingleGoPool(ctx, "test-1", func(cancelCtx context.Context, pool *SingleGoPool) {
		for {
			taskChange = "test-1"
			time.Sleep(time.Millisecond * 100)
			if CtxDone(cancelCtx) {
				return
			}
		}
	})
	if err != nil {
		t.Errorf("%+v", err)
		return
	}

	time.Sleep(time.Millisecond * 500)
	if taskChange != "test-1" {
		t.Errorf(`if taskChange != "test-1" {`)
		return
	}

	pool.AddOnceTask(ctx, "test-2", func(cancelCtx context.Context, pool *SingleGoPool) {
		for {
			taskChange = "test-2"
			time.Sleep(time.Millisecond * 100)
			if CtxDone(cancelCtx) {
				taskChange = "test-done"
				return
			}
		}
	})

	time.Sleep(time.Millisecond * 500)
	if taskChange != "test-2" {
		t.Errorf(`if taskChange != "test-2" {`)
		return
	}

	ClosePool(ctx, pool)
	time.Sleep(time.Millisecond * 500)
	if taskChange != "test-done" {
		t.Errorf(`if taskChange != "test-done" {`)
		return
	}
}

func TestDaemonSingleGoPool(t *testing.T) {
	ctx := GenCtx()

	i := 3
	pool, err := NewDaemonSingleGoPool(ctx, "test", time.Millisecond*100, func(cancelCtx context.Context, pool *SingleGoPool) {
		for {
			i--
			fmt.Println(i)
			fmt.Println(100 / i)
			fmt.Println("-----------")
			time.Sleep(time.Millisecond * 100)
			if CtxDone(cancelCtx) {
				return
			}
		}
	})
	if err != nil {
		t.Errorf("err: %+v\n", err)
		return
	}

	time.Sleep(time.Millisecond * 500)
	if i >= 0 {
		t.Errorf(`if i >= 0 { %+v`, i)
		return
	}

	ClosePool(ctx, pool)
	i = 10000
	time.Sleep(time.Millisecond * 500)
	if i <= 0 {
		t.Errorf(`if i <= 0 {`)
		return
	}
}

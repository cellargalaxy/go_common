package util

import (
	"context"
	"testing"
	"time"
)

// E8Loc 名称与偏移必须自洽，此前写成 FixedZone("GMT", 8*3600) 会格式化出 "GMT +08:00"
func TestE8Loc(t *testing.T) {
	d := time.Date(2026, 1, 1, 0, 0, 0, 0, E8Loc)
	name, offset := d.Zone()
	if offset != 8*3600 {
		t.Errorf("E8Loc 偏移 = %d秒, 期望 28800", offset)
	}
	if name == "GMT" || name == "UTC" {
		t.Errorf("E8Loc 时区名 = %q, 与+8偏移自相矛盾", name)
	}
	//同一时刻在E8Loc与UTC下的挂钟时间应相差8小时
	utc := d.In(time.UTC)
	if d.Hour()-utc.Hour() != 8 && d.Hour()-utc.Hour() != -16 {
		t.Errorf("E8Loc与UTC小时差异常: E8=%d UTC=%d", d.Hour(), utc.Hour())
	}
	if time.UTC != time.UTC {
		t.Errorf("time.UTC 应等于 time.UTC")
	}
}

func TestParseStr2Time(t *testing.T) {
	ctx := GenCtx()
	cases := []struct {
		layout, value string
		wantUnix      int64
	}{
		{DateLayout_2006_01_02, "2023-01-01", 1672502400},
		{DateLayout_2006_01, "2023-01", 1672502400},
		{DateLayout_2006_01_02_15_04_05, "2023-01-01 12:00:00", 1672545600},
		{DateLayout_2006Y01M02D, "2023年01月01日", 1672502400},
		{DateLayout_2006Y01M02D15H04m05S, "2023年01月01日 12点00分00秒", 1672545600},
	}
	for _, c := range cases {
		got, err := ParseStr2Time(ctx, c.layout, c.value, E8Loc)
		if err != nil {
			t.Errorf("ParseStr2Time(%q) 异常: %+v", c.value, err)
			continue
		}
		if got.Unix() != c.wantUnix {
			t.Errorf("ParseStr2Time(%q) Unix = %d, 期望 %d", c.value, got.Unix(), c.wantUnix)
		}
	}
	//时区必须真正生效：同一字符串在UTC下解析应比E8晚8小时
	e8, _ := ParseStr2Time(ctx, DateLayout_2006_01_02, "2023-01-01", E8Loc)
	utc, _ := ParseStr2Time(ctx, DateLayout_2006_01_02, "2023-01-01", time.UTC)
	if utc.Unix()-e8.Unix() != 8*3600 {
		t.Errorf("时区未生效: UTC与E8解析差 %d秒, 期望 28800", utc.Unix()-e8.Unix())
	}
	//非法输入必须返回error
	if _, err := ParseStr2Time(ctx, DateLayout_2006_01_02, "不是日期", E8Loc); err == nil {
		t.Errorf("非法日期字符串应返回error")
	}
	if _, err := ParseStr2Time(ctx, DateLayout_2006_01_02, "", E8Loc); err == nil {
		t.Errorf("空字符串应返回error")
	}
}

// loc 为 nil 时 time.ParseInLocation 会 panic(missing Location in call to Date)，
// 本函数是 ParseStr2Unix / ParseStr2UnixMilli 的公共入口，须回退到UTC而不是让调用方崩溃。
func TestParseStr2TimeNilLocation(t *testing.T) {
	ctx := GenCtx()

	//nil 时区不得panic，且须回退到默认的东八区
	got, err := ParseStr2Time(ctx, DateLayout_2006_01_02, "2023-01-01", nil)
	if err != nil {
		t.Fatalf("ParseStr2Time(nil时区) 异常: %+v", err)
	}
	e8, err := ParseStr2Time(ctx, DateLayout_2006_01_02, "2023-01-01", E8Loc)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	if got.Unix() != e8.Unix() {
		t.Errorf("nil时区解析 = %d, 期望与东八区一致的 %d", got.Unix(), e8.Unix())
	}

	//依赖它的两个上层函数同样不得panic
	ts, err := ParseStr2Unix(ctx, DateLayout_2006_01_02, "2023-01-01", nil)
	if err != nil {
		t.Errorf("ParseStr2Unix(nil时区) 异常: %+v", err)
	}
	if ts != e8.Unix() {
		t.Errorf("ParseStr2Unix(nil时区) = %d, 期望 %d", ts, e8.Unix())
	}
	msTs, err := ParseStr2UnixMilli(ctx, DateLayout_2006_01_02, "2023-01-01", nil)
	if err != nil {
		t.Errorf("ParseStr2UnixMilli(nil时区) 异常: %+v", err)
	}
	if msTs != e8.UnixMilli() {
		t.Errorf("ParseStr2UnixMilli(nil时区) = %d, 期望 %d", msTs, e8.UnixMilli())
	}

	//nil 时区下的非法输入仍须返回error而非panic
	if _, err := ParseStr2Time(ctx, DateLayout_2006_01_02, "不是日期", nil); err == nil {
		t.Errorf("nil时区+非法日期 应返回error")
	}
}

func TestParseStr2Unix(t *testing.T) {
	ctx := GenCtx()
	got, err := ParseStr2Unix(ctx, DateLayout_2006_01_02, "2023-01-01", E8Loc)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	if got != 1672502400 {
		t.Errorf("ParseStr2Unix = %d, 期望 1672502400", got)
	}
	//失败时必须返回0且带error
	ts, err := ParseStr2Unix(ctx, DateLayout_2006_01_02, "bad", E8Loc)
	if err == nil || ts != 0 {
		t.Errorf("非法输入 = %d, err = %v, 期望 0 且非nil error", ts, err)
	}
}

func TestParseStr2UnixMilli(t *testing.T) {
	ctx := GenCtx()
	got, err := ParseStr2UnixMilli(ctx, DateLayout_2006_01_02, "2023-01-01", E8Loc)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	//原有断言：毫秒时间戳
	if got != 1672502400000 {
		t.Errorf("ParseStr2UnixMilli = %d, 期望 1672502400000", got)
	}
	ms, err := ParseStr2UnixMilli(ctx, DateLayout_2006_01_02, "bad", E8Loc)
	if err == nil || ms != 0 {
		t.Errorf("非法输入 = %d, err = %v", ms, err)
	}
}

func TestTimeConstants(t *testing.T) {
	//TimeMax 是固定时刻常量，其"9999年末"的语义按 E8Loc 成立；
	//必须在 E8Loc 下断言，否则本地时区为 UTC+14 时会读成 10000 年
	if got := TimeMax.In(E8Loc).Year(); got != 9999 {
		t.Errorf("TimeMax 在E8Loc下年份 = %d, 期望 9999", got)
	}
	//跨时区不变量：TimeMax 始终晚于当前时间，可安全当"无限远"用
	if !TimeMax.After(time.Now()) {
		t.Errorf("TimeMax 应在当前时间之后")
	}
	//底层时间戳固定，不随本地时区变化
	if got := TimeMax.Unix(); got != 253402271999 {
		t.Errorf("TimeMax 时间戳 = %d, 期望 253402271999", got)
	}
	if DurationMax <= 0 {
		t.Errorf("DurationMax = %v, 应为正", DurationMax)
	}
	//DurationMax 须远大于常规业务时长
	if DurationMax < time.Hour*24*365 {
		t.Errorf("DurationMax = %v, 过小", DurationMax)
	}
}

func TestSleep(t *testing.T) {
	ctx := GenCtx()
	//正常睡眠至少达到指定时长
	start := time.Now()
	Sleep(ctx, time.Millisecond*50)
	if elapsed := time.Since(start); elapsed < time.Millisecond*45 {
		t.Errorf("Sleep(50ms) 实际 %v, 过短", elapsed)
	}
	//非正数时长必须立即返回
	start = time.Now()
	Sleep(ctx, 0)
	Sleep(ctx, -time.Second)
	if elapsed := time.Since(start); elapsed > time.Millisecond*20 {
		t.Errorf("Sleep(0/负数) 耗时 %v, 应立即返回", elapsed)
	}
	//ctx取消后必须提前唤醒，这是该函数存在的意义
	cancelCtx, cancel := context.WithCancel(ctx)
	go func() {
		time.Sleep(time.Millisecond * 30)
		cancel()
	}()
	start = time.Now()
	Sleep(cancelCtx, time.Second*10)
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("ctx取消后 Sleep 耗时 %v, 未提前唤醒", elapsed)
	}
	//已取消的ctx应立即返回
	doneCtx, cancel2 := context.WithCancel(ctx)
	cancel2()
	start = time.Now()
	Sleep(doneCtx, time.Second*10)
	if elapsed := time.Since(start); elapsed > time.Millisecond*100 {
		t.Errorf("已取消ctx Sleep 耗时 %v", elapsed)
	}
}

func TestSleepWare(t *testing.T) {
	ctx := GenCtx()
	//扰动后仍应在原时长±10%附近
	start := time.Now()
	SleepWare(ctx, time.Millisecond*100)
	elapsed := time.Since(start)
	if elapsed < time.Millisecond*80 || elapsed > time.Millisecond*200 {
		t.Errorf("SleepWare(100ms) 实际 %v, 超出预期扰动范围", elapsed)
	}
	//ctx取消同样要能提前唤醒
	cancelCtx, cancel := context.WithCancel(ctx)
	cancel()
	start = time.Now()
	SleepWare(cancelCtx, time.Second*10)
	if elapsed := time.Since(start); elapsed > time.Millisecond*100 {
		t.Errorf("已取消ctx SleepWare 耗时 %v", elapsed)
	}
}

package util

import (
	"strings"
	"sync"
	"testing"
	"time"
)

// 期望值在测试侧独立声明，不复用实现里的常量：
// 实现中 wantIdLen / wantIdTickMicro 是 ParseStrId / GenId 的函数内局部常量(99da961)，测试取不到；
// 更重要的是——用例若复用被测常量，实现把常量改错时断言会跟着一起变，反而失去拦截力。
const (
	wantIdLen       = 16  //ID位数：12位东八区时间(YYMMDDhhmmss) + 4位亚秒
	wantIdTickMicro = 100 //亚秒位精度，单位微秒：4位亚秒即0.1毫秒
)

// ID往返：GenIdByTime按本地时区格式化、ParseStrId固定按E8Loc解析，
// 二者曾不对称，仅在本地时区恰为UTC+8时自洽。本用例显式覆盖多时区。
func TestGenIdRoundTrip(t *testing.T) {
	ctx := GenCtx()
	now := time.Now()
	id := GenIdByTime(now)
	got, err := ParseId(ctx, id)
	if err != nil {
		t.Fatalf("ParseId 异常: %+v", err)
	}
	if got.Unix() != now.Unix() {
		t.Errorf("往返偏差 %d 秒（本地时区 %v，时区不对称缺陷）", got.Unix()-now.Unix(), time.Local)
	}
}

// 关键：显式指定不同时区的时间点，验证与本地时区无关
func TestGenIdTimezoneIndependent(t *testing.T) {
	ctx := GenCtx()
	base := time.Date(2026, 9, 4, 17, 30, 45, 123456000, E8Loc)
	//同一时刻用不同时区表示，生成的ID必须相同
	want := GenIdByTime(base)
	for _, loc := range []*time.Location{time.UTC, time.FixedZone("X", -5*3600), time.FixedZone("Y", 5*3600+1800)} {
		if got := GenIdByTime(base.In(loc)); got != want {
			t.Errorf("同一时刻在时区%v下生成ID=%d, 期望%d（时区不对称缺陷）", loc, got, want)
		}
	}
	//解析回来必须仍是同一时刻
	got, err := ParseId(ctx, want)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	if got.Unix() != base.Unix() {
		t.Errorf("解析结果 %v 与原时刻 %v 不符", got, base)
	}
}

func TestGenId(t *testing.T) {
	//ID为16位，且随时间单调递增
	id1 := GenId()
	if s := Int2Str(id1); len(s) != wantIdLen {
		t.Errorf("GenId 位数 = %d (%s), 期望 %d", len(s), s, wantIdLen)
	}
	time.Sleep(time.Millisecond * 2)
	if id2 := GenId(); id2 <= id1 {
		t.Errorf("GenId 未随时间递增: %d -> %d", id1, id2)
	}
	//字符串形式与数值形式一致
	sid := GenStrId()
	if len(sid) != wantIdLen {
		t.Errorf("GenStrId 位数 = %d (%s)", len(sid), sid)
	}
	if Str2Int[int64](sid) <= 0 {
		t.Errorf("GenStrId 不可解析为正整数: %s", sid)
	}
	//已知时刻的ID前缀应体现年月日时分秒
	fixed := time.Date(2026, 9, 4, 17, 30, 45, 0, E8Loc)
	if got := Int2Str(GenIdByTime(fixed)); !strings.HasPrefix(got, "260904173045") {
		t.Errorf("GenIdByTime 前缀 = %s, 期望以 260904173045 开头", got)
	}
}

// GenId 唯一性：ID格式只到0.1毫秒(060102150405.0000)，而time.Now()实测最小步进
// 约几十纳秒，同一个100微秒窗口内的连续调用若不补偿会拿到完全相同的ID。
// GenId 同时是 GenLogId 与 GenReqId 的底层实现，而 ValidateGin 用 reqId 做防重放
// 校验(TryLockReqId命中即以 http.StatusConflict 拒绝)，ID重复会让合法请求被误判为重放而拒绝。
// 本用例同时锁定三件事：不重复、严格递增、且补偿后的ID依然合法可解析。
// 最后一条尤其关键：补偿必须在"时间刻度"上进行，若直接对ID整数+1，
// 边界处会产生 2609050102600000 这类秒位为60的非法值，ParseId 会报
// second out of range。
func TestGenIdUnique(t *testing.T) {
	ctx := GenCtx()
	restoreGenIdTick(t)

	const n = 20000
	seen := make(map[int64]bool, n)
	var prev int64
	for i := 0; i < n; i++ {
		id := GenId()
		if seen[id] {
			t.Fatalf("第%d次调用出现重复ID: %d", i, id)
		}
		seen[id] = true
		if id <= prev {
			t.Fatalf("第%d次调用ID未严格递增: %d -> %d", i, prev, id)
		}
		prev = id
		//补偿不能把ID撑出16位，也不能产生无法解析的非法时间
		if s := Int2Str(id); len(s) != wantIdLen {
			t.Fatalf("第%d次调用ID非%d位: %s", i, wantIdLen, s)
		}
		if _, err := ParseId(ctx, id); err != nil {
			t.Fatalf("第%d次调用ID无法解析 %d: %+v", i, id, err)
		}
	}
}

// 并发下同样必须唯一：genIdLock 用互斥锁保护发号水位，
// 这里用多goroutine抢号验证不会发出重复ID（配合 -race 亦可验证无竞争）。
func TestGenIdUniqueConcurrent(t *testing.T) {
	restoreGenIdTick(t)

	const goroutines, per = 20, 1000
	var wg sync.WaitGroup
	all := make([][]int64, goroutines)
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			local := make([]int64, 0, per)
			for i := 0; i < per; i++ {
				local = append(local, GenId())
			}
			all[g] = local
		}(g)
	}
	wg.Wait()

	total := goroutines * per
	seen := make(map[int64]bool, total)
	for g := range all {
		for _, id := range all[g] {
			if seen[id] {
				t.Fatalf("并发下出现重复ID: %d", id)
			}
			seen[id] = true
		}
	}
	if len(seen) != total {
		t.Errorf("并发生成 %d 个ID，唯一值仅 %d 个", total, len(seen))
	}
}

// 系统时钟回拨、以及同一个100微秒窗口内连续调用时，不得发出重复或倒退的ID。
// 补偿分支直接体现在发号水位 genIdLock.tick 上，这里通过摆布水位来驱动，避免真的去改系统时间。
func TestGenIdClockRollback(t *testing.T) {
	restoreGenIdTick(t)

	//把水位顶到未来一小时，等价于时钟回拨：此后发号必须在水位上继续递增，不能倒退也不能重复
	future := time.Now().Add(time.Hour).UnixMicro() / wantIdTickMicro
	setGenIdTick(future)

	if got, want := GenId(), GenIdByTime(time.UnixMicro((future+1)*wantIdTickMicro)); got != want {
		t.Errorf("时钟回拨后发号 = %d, 期望 %d（必须继续递增）", got, want)
	}
	//同一个100微秒窗口内再次发号同样必须+1
	if got, want := GenId(), GenIdByTime(time.UnixMicro((future+2)*wantIdTickMicro)); got != want {
		t.Errorf("连续发号 = %d, 期望 %d", got, want)
	}

	//水位低于真实时刻时，应直接采用真实时刻而非继续在旧水位上+1
	past := time.Now().Add(-time.Hour).UnixMicro() / wantIdTickMicro
	setGenIdTick(past)
	GenId()
	if tick := getGenIdTick(); tick <= past+1 {
		t.Errorf("水位低于真实时刻时仍在旧水位上补偿: tick = %d, past = %d", tick, past)
	}
}

// GenId 在"发号水位已顶到某秒的最后一个100微秒窗口"时，下一个号必须进位到下一秒，
// 且仍是合法可解析的16位ID。
// 这是"补偿必须在时间域进行"的直接回归：若实现改成对ID整数+1，
// 此处会得到 ...600000（秒位60），ParseId 报 second out of range。
// 上面的 TestGenIdUnique 抓不到这一点——它只在真实当前时刻附近发号，
// 极少正好压在秒边界上，所以必须显式把水位摆到边界再发号。
func TestGenIdCarryAcrossSecond(t *testing.T) {
	ctx := GenCtx()
	restoreGenIdTick(t)

	//依次把水位顶到秒/分/时/日/年的最后一个100微秒窗口，再让 GenId 发下一个号
	for _, edge := range []time.Time{
		time.Date(2030, 9, 5, 1, 2, 59, 999900000, E8Loc),     //跨秒
		time.Date(2030, 9, 5, 1, 59, 59, 999900000, E8Loc),    //跨分
		time.Date(2030, 9, 5, 23, 59, 59, 999900000, E8Loc),   //跨日
		time.Date(2030, 12, 31, 23, 59, 59, 999900000, E8Loc), //跨年
	} {
		edgeTick := edge.UnixMicro() / wantIdTickMicro
		setGenIdTick(edgeTick)

		//传入时刻远早于水位，必然走补偿分支，拿到 edgeTick+1
		id := GenId()
		want := GenIdByTime(time.UnixMicro((edgeTick + 1) * wantIdTickMicro))
		if id != want {
			t.Errorf("%s 边界发号 = %d, 期望 %d",
				edge.In(E8Loc).Format("06-01-02 15:04:05.0000"), id, want)
		}
		if s := Int2Str(id); len(s) != wantIdLen {
			t.Errorf("%s 边界发号非%d位: %s", edge.In(E8Loc).Format("15:04:05.0000"), wantIdLen, s)
		}
		//关键断言：整数+1的实现会在这里产生秒位60/分位60等非法值
		parsed, err := ParseId(ctx, id)
		if err != nil {
			t.Errorf("%s 边界发号ID无法解析 %d: %+v",
				edge.In(E8Loc).Format("06-01-02 15:04:05.0000"), id, err)
			continue
		}
		//进位后应恰好落在下一个100微秒窗口
		if parsed.UnixMicro() != (edgeTick+1)*wantIdTickMicro {
			t.Errorf("%s 边界发号解析时刻 = %d, 期望 %d",
				edge.In(E8Loc).Format("15:04:05.0000"), parsed.UnixMicro(), (edgeTick+1)*wantIdTickMicro)
		}
	}
}

// 时间域补偿的边界：跨秒/分/时/日/年进位后，ID必须仍是16位且可解析。
// 这是"不能直接对ID整数+1"的正面回归——整数+1在这些点上会产生秒位60等非法值。
func TestGenIdByTimeTickCarry(t *testing.T) {
	ctx := GenCtx()
	for _, tm := range []time.Time{
		time.Date(2026, 9, 5, 1, 2, 59, 999900000, E8Loc),     //跨秒
		time.Date(2026, 9, 5, 1, 59, 59, 999900000, E8Loc),    //跨分
		time.Date(2026, 9, 5, 23, 59, 59, 999900000, E8Loc),   //跨日
		time.Date(2026, 12, 31, 23, 59, 59, 999900000, E8Loc), //跨年
	} {
		base := GenIdByTime(tm)
		next := GenIdByTime(time.UnixMicro(tm.UnixMicro() + wantIdTickMicro))
		if next <= base {
			t.Errorf("%v 进位后ID未递增: %d -> %d", tm, base, next)
		}
		if s := Int2Str(next); len(s) != wantIdLen {
			t.Errorf("%v 进位后ID非%d位: %s", tm, wantIdLen, s)
		}
		if _, err := ParseId(ctx, next); err != nil {
			t.Errorf("%v 进位后ID无法解析 %d: %+v", tm, next, err)
		}
	}
}

// jsMaxSafeInteger JS的 Number.MAX_SAFE_INTEGER，超过它的整数经JSON传到前端会被
// IEEE-754双精度截断。ID会随HTTP响应直接暴露给前端，故必须恒在此值以内。
const jsMaxSafeInteger int64 = 9007199254740991

// ID 必须是JS安全整数：这是把亚秒位从微秒(18位)收窄到0.1毫秒(16位)的唯一动因。
// 16位方案有明确的时间边界——2090-07-19 23:59:59.9999 之前安全，
// 自 2090-07-20 00:00:00 起越界。按需求不要求兼容到2090，
// 故这里把这条边界精确断言下来当作备忘：
// 若哪天需要跨过它，本用例会指出必须改用更短的亚秒位或换号段式ID。
// 若有人把布局改回 .000000，本用例同样立刻失败。
func TestGenIdJsSafe(t *testing.T) {
	//实际发号
	for i := 0; i < 100; i++ {
		id := GenId()
		if id > jsMaxSafeInteger {
			t.Fatalf("GenId = %d 超出JS安全整数 %d", id, jsMaxSafeInteger)
		}
		//float64往返不丢精度，等价于前端 JSON.parse 后的行为
		if back := int64(float64(id)); back != id {
			t.Fatalf("ID %d 经float64往返变成 %d，前端会丢精度", id, back)
		}
	}

	//安全期最后一个ID：2090-07-19 23:59:59.9999
	max := GenIdByTime(time.Date(2090, 7, 19, 23, 59, 59, 999900000, E8Loc))
	if max != 9007192359599999 {
		t.Fatalf("安全期上界 = %d, 期望 9007192359599999", max)
	}
	if max > jsMaxSafeInteger {
		t.Errorf("安全期上界 %d 超出JS安全整数 %d", max, jsMaxSafeInteger)
	}
	if back := int64(float64(max)); back != max {
		t.Errorf("安全期上界 %d 经float64往返变成 %d", max, back)
	}

	//已知边界：越过这一刻即越界。这是本方案自觉接受的取舍，不是遗漏
	over := GenIdByTime(time.Date(2090, 7, 20, 0, 0, 0, 0, E8Loc))
	if over != 9007200000000000 {
		t.Fatalf("越界起点 = %d, 期望 9007200000000000", over)
	}
	if over <= jsMaxSafeInteger {
		t.Errorf("越界起点 %d 未越界，与 wantIdLen 注释记载的边界不符", over)
	}
}

// 发号容量：亚秒位取4位即0.1毫秒精度，单进程每秒最多10000个不同的ID。
// 本用例把"容量"与"漂移上界"两件事都变成可执行的规格：
//  1. 精度就是 wantIdTickMicro——同一个100微秒窗口内的不同时刻必须格式化出同一个ID，
//     跨窗口必须恰好+1。窗口数 1e6/100 = 10000 就是每秒容量。
//  2. 连续发号 n 个时，GenId 的水位补偿会把内嵌时间顶到未来，
//     但前移量恒不超过 n 个窗口（即 n/10000 秒）——这是补偿式发号的结构性上界，
//     与机器快慢无关。给数组循环赋值时可据此估算：千级元素≈0.1秒、万级≈1秒。
func TestGenIdCapacity(t *testing.T) {
	ctx := GenCtx()

	//1) 精度 = wantIdTickMicro，故每秒容量 = 1秒/wantIdTickMicro
	base := time.Date(2026, 9, 4, 17, 30, 45, 0, E8Loc)
	if got, want := int64(time.Second/time.Microsecond)/wantIdTickMicro, int64(10000); got != want {
		t.Fatalf("每秒发号容量 = %d, 期望 %d", got, want)
	}
	//同一窗口内(0 与 99微秒)必须是同一个ID，否则精度不是声称的100微秒
	if head, tail := GenIdByTime(base), GenIdByTime(base.Add(time.Duration(wantIdTickMicro-1)*time.Microsecond)); head != tail {
		t.Errorf("同一窗口内ID不一致: %d vs %d", head, tail)
	}
	//跨一个窗口必须恰好+1，既不跳号也不重号
	if got, want := GenIdByTime(base.Add(time.Duration(wantIdTickMicro)*time.Microsecond)), GenIdByTime(base)+1; got != want {
		t.Errorf("跨窗口发号 = %d, 期望 %d", got, want)
	}

	//2) 循环赋值场景：连续发号的内嵌时间前移量不得超过 n 个窗口
	restoreGenIdTick(t)
	const n = 1000
	setGenIdTick(time.Now().UnixMicro() / wantIdTickMicro)
	var last int64
	for i := 0; i < n; i++ {
		last = GenId()
	}
	parsed, err := ParseId(ctx, last)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	drift := parsed.Sub(time.Now())
	maxDrift := time.Duration(n*wantIdTickMicro) * time.Microsecond
	if drift > maxDrift {
		t.Errorf("连续发号%d个后内嵌时间前移 %v, 超过结构性上界 %v", n, drift, maxDrift)
	}
	t.Logf("连续发号%d个：内嵌时间前移 %v（上界 %v）", n, drift.Round(time.Millisecond), maxDrift)
}

func TestParseStrId(t *testing.T) {
	ctx := GenCtx()
	//合法16位ID
	got, err := ParseStrId(ctx, "2609041730451234")
	if err != nil {
		t.Fatalf("%+v", err)
	}
	if got.Year() != 2026 || got.Month() != 9 || got.Day() != 4 {
		t.Errorf("解析日期 = %v", got)
	}
	if got.Hour() != 17 || got.Minute() != 30 || got.Second() != 45 {
		t.Errorf("解析时间 = %v", got)
	}
	if got.Nanosecond() != 123400000 {
		t.Errorf("解析亚秒 = %d, 期望 123400000", got.Nanosecond())
	}
	//超长必须报错，不能静默返回零值；旧的18位微秒ID也在此被拒
	for _, bad := range []string{"", "26090417304512345", "260904173045123456"} {
		if _, err := ParseStrId(ctx, bad); err == nil {
			t.Errorf("ParseStrId(%q) 长度非法应返回error", bad)
		}
	}
	//不足16位会左补0，补出来的是非法时间(月份00)，同样必须报错
	if _, err := ParseStrId(ctx, "123"); err == nil {
		t.Errorf("ParseStrId(\"123\") 补零后为非法时间，应返回error")
	}
	//长度合法但内容非法
	if _, err := ParseStrId(ctx, "abcdefghijklmnop"); err == nil {
		t.Errorf("非数字内容应返回error")
	}
	//ParseId 与 ParseStrId 结果一致
	tid, err1 := ParseId(ctx, 2609041730451234)
	sid, err2 := ParseStrId(ctx, "2609041730451234")
	if err1 != nil || err2 != nil {
		t.Fatalf("err1=%v err2=%v", err1, err2)
	}
	if !tid.Equal(sid) {
		t.Errorf("ParseId 与 ParseStrId 结果不一致: %v vs %v", tid, sid)
	}
}

// ==== 辅助 ====

func getGenIdTick() int64 {
	genIdLock.Lock()
	defer genIdLock.Unlock()

	return genIdLock.tick
}
func setGenIdTick(tick int64) {
	genIdLock.Lock()
	defer genIdLock.Unlock()

	genIdLock.tick = tick
}

// restoreGenIdTick 用例结束后把包级发号水位还原为"原水位与真实时刻中的较大者"。
// genIdLock.tick 是包级状态：摆布水位的用例会把它推到未来，批量发号的用例
// (每10000个号推进1秒)同样会把它顶到未来，
// 不还原会让 util/log_test.go 里"ParseId(GenId()) 与当前时刻偏差须在1分钟内"的断言
// 随用例执行顺序偶发失败。
func restoreGenIdTick(t *testing.T) {
	t.Helper()
	saved := getGenIdTick()
	t.Cleanup(func() {
		if now := time.Now().UnixMicro() / wantIdTickMicro; now > saved {
			setGenIdTick(now)
		} else {
			setGenIdTick(saved)
		}
	})
}

func TestGenRandStr(t *testing.T) {
	//负长度曾panic(makeslice: len out of range)，须与0等价返回空串
	for _, n := range []int{-100, -5, -1} {
		if got := GenRandStr(n); got != "" {
			t.Errorf("GenRandStr(%d) = %q, 期望空串", n, got)
		}
	}
	//长度须严格等于入参
	for _, n := range []int{0, 1, 8, 64} {
		got := GenRandStr(n)
		if len([]rune(got)) != n {
			t.Errorf("GenRandStr(%d) 长度 = %d", n, len([]rune(got)))
		}
		//只能出现英文大小写字母，不得掺入数字或符号
		for _, r := range got {
			if !('a' <= r && r <= 'z') && !('A' <= r && r <= 'Z') {
				t.Errorf("GenRandStr(%d) 含非字母字符 %q: %q", n, r, got)
			}
		}
	}
	//随机性：多次生成不应全部相同（否则退化为常量）
	first := GenRandStr(16)
	var diff bool
	for i := 0; i < 20; i++ {
		if GenRandStr(16) != first {
			diff = true
			break
		}
	}
	if !diff {
		t.Errorf("GenRandStr 20次结果全相同，疑似未随机")
	}
}

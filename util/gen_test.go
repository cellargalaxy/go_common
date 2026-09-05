package util

import (
	"strings"
	"sync"
	"testing"
	"time"
)

// ID往返：GenIdByTime按本地时区格式化、ParseStringId固定按E8Loc解析，
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
	for _, loc := range []*time.Location{UTCLoc, time.FixedZone("X", -5*3600), time.FixedZone("Y", 5*3600+1800)} {
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
	//ID为18位，且随时间单调递增
	id1 := GenId()
	if s := Int2String(id1); len(s) != 18 {
		t.Errorf("GenId 位数 = %d (%s), 期望 18", len(s), s)
	}
	time.Sleep(time.Millisecond * 2)
	if id2 := GenId(); id2 <= id1 {
		t.Errorf("GenId 未随时间递增: %d -> %d", id1, id2)
	}
	//字符串形式与数值形式一致
	sid := GenStringId()
	if len(sid) != 18 {
		t.Errorf("GenStringId 位数 = %d (%s)", len(sid), sid)
	}
	if String2Int[int64](sid) <= 0 {
		t.Errorf("GenStringId 不可解析为正整数: %s", sid)
	}
	//已知时刻的ID前缀应体现年月日时分秒
	fixed := time.Date(2026, 9, 4, 17, 30, 45, 0, E8Loc)
	if got := Int2String(GenIdByTime(fixed)); !strings.HasPrefix(got, "260904173045") {
		t.Errorf("GenIdByTime 前缀 = %s, 期望以 260904173045 开头", got)
	}
}

// GenId 唯一性：ID格式只到微秒(060102150405.000000)，而time.Now()实测最小步进
// 约几十纳秒，同一微秒内的连续调用曾拿到完全相同的ID——实测顺序2万次仅约2700个
// 不同值(碰撞率86%)，20并发下碰撞率约97%。
// GenId 同时是 GenLogId 与 GenReqId 的底层实现，而 ValidateGin 用 reqId 做防重放
// 校验(existReqId命中即以ReRequestCode拒绝)，ID重复会让合法请求被误判为重放而拒绝。
// 本用例同时锁定三件事：不重复、严格递增、且补偿后的ID依然合法可解析。
// 最后一条尤其关键：补偿必须在"微秒时刻"上进行，若直接对ID整数+1，
// 边界处会产生 260905010260000000 这类秒位为60的非法值，ParseId 会报
// second out of range。
func TestGenIdUnique(t *testing.T) {
	ctx := GenCtx()
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
		//补偿不能把ID撑出18位，也不能产生无法解析的非法时间
		if s := Int2String(id); len(s) != 18 {
			t.Fatalf("第%d次调用ID非18位: %s", i, s)
		}
		if _, err := ParseId(ctx, id); err != nil {
			t.Fatalf("第%d次调用ID无法解析 %d: %+v", i, id, err)
		}
	}
}

// 并发下同样必须唯一：nextIdMicro 用互斥锁保护发号水位，
// 这里用多goroutine抢号验证不会发出重复ID（配合 -race 亦可验证无竞争）。
func TestGenIdUniqueConcurrent(t *testing.T) {
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

// 系统时钟回拨时不得发出重复或倒退的ID：nextIdMicro 的 micro <= last 分支
// 同时覆盖"同一微秒"与"时钟回拨"两种情况。
// 直接驱动 nextIdMicro 来构造回拨，避免真的去改系统时间。
// 注意 lastIdMicro 是包级发号水位，会被同包其它用例推高到真实当前时刻，
// 故这里必须先接管并在结束后还原，否则断言结果取决于用例执行顺序。
func TestNextIdMicroClockRollback(t *testing.T) {
	lastIdMicro.Lock()
	saved := lastIdMicro.micro
	lastIdMicro.Unlock()
	t.Cleanup(func() {
		lastIdMicro.Lock()
		//还原为原水位与真实时刻中的较大者，避免影响后续用例的递增断言
		if now := time.Now().UnixMicro(); now > saved {
			lastIdMicro.micro = now
		} else {
			lastIdMicro.micro = saved
		}
		lastIdMicro.Unlock()
	})

	base := time.Date(2026, 9, 5, 1, 2, 3, 0, E8Loc)
	//把水位压到 base 之前，让 base 成为"正常前进"的时刻
	lastIdMicro.Lock()
	lastIdMicro.micro = base.UnixMicro() - 1
	lastIdMicro.Unlock()

	//水位低于真实时刻时，应直接采用传入时刻
	if got := nextIdMicro(base); got != base.UnixMicro() {
		t.Fatalf("正常前进时发号 = %d, 期望采用真实时刻 %d", got, base.UnixMicro())
	}
	first := base.UnixMicro()
	//同一时刻再次发号必须+1
	if second := nextIdMicro(base); second != first+1 {
		t.Errorf("同一时刻二次发号 = %d, 期望 %d", second, first+1)
	}
	//时钟回拨1小时，仍必须严格递增，不能倒退也不能重复
	if back := nextIdMicro(base.Add(-time.Hour)); back != first+2 {
		t.Errorf("时钟回拨后发号 = %d, 期望 %d（必须继续递增）", back, first+2)
	}
	//时钟正常前进到远大于水位处，应直接采用真实时刻而非继续+1
	ahead := base.Add(time.Hour)
	if got := nextIdMicro(ahead); got != ahead.UnixMicro() {
		t.Errorf("时钟前进后发号 = %d, 期望采用真实时刻 %d", got, ahead.UnixMicro())
	}
}

// GenId 在"发号水位已顶到某秒的最后一微秒"时，下一个号必须进位到下一秒，
// 且仍是合法可解析的18位ID。
// 这是"补偿必须在时间域进行"的直接回归：若实现改成对ID整数+1，
// 此处会得到 ...60000000（秒位60），ParseId 报 second out of range。
// 上面的 TestGenIdUnique 抓不到这一点——它只在真实当前时刻附近发号，
// 极少正好压在秒边界上，所以必须显式把水位摆到边界再发号。
func TestGenIdCarryAcrossSecond(t *testing.T) {
	ctx := GenCtx()
	lastIdMicro.Lock()
	saved := lastIdMicro.micro
	lastIdMicro.Unlock()
	t.Cleanup(func() {
		lastIdMicro.Lock()
		if now := time.Now().UnixMicro(); now > saved {
			lastIdMicro.micro = now
		} else {
			lastIdMicro.micro = saved
		}
		lastIdMicro.Unlock()
	})

	//依次把水位顶到秒/分/时/日/年的最后一微秒，再让 GenId 发下一个号
	for _, edge := range []time.Time{
		time.Date(2030, 9, 5, 1, 2, 59, 999999000, E8Loc),     //跨秒
		time.Date(2030, 9, 5, 1, 59, 59, 999999000, E8Loc),    //跨分
		time.Date(2030, 9, 5, 23, 59, 59, 999999000, E8Loc),   //跨日
		time.Date(2030, 12, 31, 23, 59, 59, 999999000, E8Loc), //跨年
	} {
		edgeMicro := edge.UnixMicro()
		lastIdMicro.Lock()
		lastIdMicro.micro = edgeMicro
		lastIdMicro.Unlock()

		//传入时刻远早于水位，必然走补偿分支，拿到 edgeMicro+1
		id := GenId()
		want := GenIdByTime(time.UnixMicro(edgeMicro + 1))
		if id != want {
			t.Errorf("%s 边界发号 = %d, 期望 %d",
				edge.In(E8Loc).Format("06-01-02 15:04:05.000000"), id, want)
		}
		if s := Int2String(id); len(s) != 18 {
			t.Errorf("%s 边界发号非18位: %s", edge.In(E8Loc).Format("15:04:05.000000"), s)
		}
		//关键断言：整数+1的实现会在这里产生秒位60/分位60等非法值
		parsed, err := ParseId(ctx, id)
		if err != nil {
			t.Errorf("%s 边界发号ID无法解析 %d: %+v",
				edge.In(E8Loc).Format("06-01-02 15:04:05.000000"), id, err)
			continue
		}
		//进位后应恰好落在下一微秒
		if parsed.UnixMicro() != edgeMicro+1 {
			t.Errorf("%s 边界发号解析时刻 = %d, 期望 %d",
				edge.In(E8Loc).Format("15:04:05.000000"), parsed.UnixMicro(), edgeMicro+1)
		}
	}
}

// 微秒域补偿的边界：跨秒/分/时/日/年进位后，ID必须仍是18位且可解析。
// 这是"不能直接对ID整数+1"的正面回归——整数+1在这些点上会产生秒位60等非法值。
func TestGenIdByTimeMicroCarry(t *testing.T) {
	ctx := GenCtx()
	for _, tm := range []time.Time{
		time.Date(2026, 9, 5, 1, 2, 59, 999999000, E8Loc),     //跨秒
		time.Date(2026, 9, 5, 1, 59, 59, 999999000, E8Loc),    //跨分
		time.Date(2026, 9, 5, 23, 59, 59, 999999000, E8Loc),   //跨日
		time.Date(2026, 12, 31, 23, 59, 59, 999999000, E8Loc), //跨年
	} {
		base := GenIdByTime(tm)
		next := GenIdByTime(time.UnixMicro(tm.UnixMicro() + 1))
		if next <= base {
			t.Errorf("%v 进位后ID未递增: %d -> %d", tm, base, next)
		}
		if s := Int2String(next); len(s) != 18 {
			t.Errorf("%v 进位后ID非18位: %s", tm, s)
		}
		if _, err := ParseId(ctx, next); err != nil {
			t.Errorf("%v 进位后ID无法解析 %d: %+v", tm, next, err)
		}
	}
}

func TestParseStringId(t *testing.T) {
	ctx := GenCtx()
	//合法18位ID
	got, err := ParseStringId(ctx, "260904173045123456")
	if err != nil {
		t.Fatalf("%+v", err)
	}
	if got.Year() != 2026 || got.Month() != 9 || got.Day() != 4 {
		t.Errorf("解析日期 = %v", got)
	}
	if got.Hour() != 17 || got.Minute() != 30 || got.Second() != 45 {
		t.Errorf("解析时间 = %v", got)
	}
	//长度非18必须报错，不能静默返回零值
	for _, bad := range []string{"", "123", "26090417304512345", "2609041730451234567"} {
		if _, err := ParseStringId(ctx, bad); err == nil {
			t.Errorf("ParseStringId(%q) 长度非法应返回error", bad)
		}
	}
	//长度合法但内容非法
	if _, err := ParseStringId(ctx, "abcdefghijklmnopqr"); err == nil {
		t.Errorf("非数字内容应返回error")
	}
	//ParseId 与 ParseStringId 结果一致
	tid, err1 := ParseId(ctx, 260904173045123456)
	sid, err2 := ParseStringId(ctx, "260904173045123456")
	if err1 != nil || err2 != nil {
		t.Fatalf("err1=%v err2=%v", err1, err2)
	}
	if !tid.Equal(sid) {
		t.Errorf("ParseId 与 ParseStringId 结果不一致: %v vs %v", tid, sid)
	}
}

func TestGenGoLabel(t *testing.T) {
	ctx := GenCtx()
	code := "type Demo struct {\n\tUserName string //用户名\n\tAge int //年龄\n}"
	got := GenGoLabel(ctx, code)
	//驼峰字段名须转为下划线json标签
	if !strings.Contains(got, `json:"user_name"`) {
		t.Errorf("缺少 json:\"user_name\" 标签:\n%s", got)
	}
	if !strings.Contains(got, `json:"age"`) {
		t.Errorf("缺少 json:\"age\" 标签:\n%s", got)
	}
	//注释须保留
	if !strings.Contains(got, "用户名") || !strings.Contains(got, "年龄") {
		t.Errorf("注释丢失:\n%s", got)
	}
	//花括号行原样保留
	if !strings.Contains(got, "type Demo struct {") {
		t.Errorf("结构体声明行被破坏:\n%s", got)
	}

	//额外标签
	got = GenGoLabel(ctx, code, "gorm", "csv")
	if !strings.Contains(got, `gorm:"user_name"`) || !strings.Contains(got, `csv:"user_name"`) {
		t.Errorf("额外标签未生成:\n%s", got)
	}

	//空代码原样返回，不能panic
	if got := GenGoLabel(ctx, ""); got != "" {
		t.Errorf("空代码 = %q", got)
	}
	//字段少于两段的行应被跳过而非panic
	if got := GenGoLabel(ctx, "type A struct {\n\tOnlyOne\n}"); !strings.Contains(got, "OnlyOne") {
		t.Errorf("单字段行处理异常:\n%s", got)
	}
}

func TestGenModel2Sql(t *testing.T) {
	ctx := GenCtx()
	code := "type UserInfo struct {\n\tUserName string //用户名\n\tAge int //年龄\n\tScore float64 //分数\n\tBigNum int64 //大数\n\tCreateTime time.Time //创建时间\n}"
	got := GenModel2Sql(ctx, code)

	//表名由驼峰转下划线
	if !strings.Contains(got, "CREATE TABLE `user_info`") {
		t.Errorf("表名错误:\n%s", got)
	}
	//Go类型到MySQL类型的映射
	checks := map[string]string{
		"`user_name` varchar(255)": "string->varchar",
		"`age` int(11)":            "int->int(11)",
		"`score` double":           "float64->double",
		"`big_num` bigint(20)":     "int64->bigint",
		"`create_time` datetime":   "time.Time->datetime",
	}
	for want, desc := range checks {
		if !strings.Contains(got, want) {
			t.Errorf("类型映射缺失(%s): 期望包含 %q\n%s", desc, want, got)
		}
	}
	//默认值
	if !strings.Contains(got, "DEFAULT ''") {
		t.Errorf("字符串默认值缺失:\n%s", got)
	}
	//注释须落入COMMENT
	if !strings.Contains(got, "COMMENT '用户名'") {
		t.Errorf("COMMENT 缺失:\n%s", got)
	}
	//固定脚手架字段
	for _, want := range []string{"`id`", "AUTO_INCREMENT", "`created_at`", "`updated_at`", "PRIMARY KEY (`id`)", "ENGINE = InnoDB", "utf8mb4"} {
		if !strings.Contains(got, want) {
			t.Errorf("缺少固定片段 %q:\n%s", want, got)
		}
	}
	//空代码原样返回
	if got := GenModel2Sql(ctx, ""); got != "" {
		t.Errorf("空代码 = %q", got)
	}
	//未知类型：列类型原样透出以便人工核对，但绝不能把Go类型名当成默认值再输出一次，
	//否则会生成 "`flag` bool NOT NULL bool" 这种语法必然非法的SQL。
	got = GenModel2Sql(ctx, "type A struct {\n\tFlag bool //标记\n}")
	if !strings.Contains(got, "`flag` bool") {
		t.Errorf("未知类型的列类型未原样透出:\n%s", got)
	}
	if strings.Contains(got, "NOT NULL bool") {
		t.Errorf("未知类型把Go类型名输出到了DEFAULT位置，生成非法SQL:\n%s", got)
	}
	//未知类型不应带DEFAULT子句
	for _, line := range strings.Split(got, "\n") {
		if strings.Contains(line, "`flag`") && strings.Contains(line, "DEFAULT") {
			t.Errorf("未知类型不应生成DEFAULT子句: %q", line)
		}
	}
	//已知类型的DEFAULT必须仍然正常输出，确认上面的修正没有把默认值整体去掉
	known := GenModel2Sql(ctx, "type A struct {\n\tNum int //数字\n\tName string //名字\n}")
	if !strings.Contains(known, "DEFAULT 0") || !strings.Contains(known, "DEFAULT ''") {
		t.Errorf("已知类型的DEFAULT子句缺失:\n%s", known)
	}
}

// getBdType / getBdDefaultValue 的逐类型映射，
// 重点：未知类型不得把Go类型名当作SQL默认值（会产生非法SQL）
func TestGetBdTypeAndDefaultValue(t *testing.T) {
	typeCases := map[string]string{
		"int": "int(11)", "int64": "bigint(20)", "string": "varchar(255)",
		"time.Time": "datetime", "float32": "float", "float64": "double",
	}
	for in, want := range typeCases {
		if got := getBdType(in); got != want {
			t.Errorf("getBdType(%q) = %q, 期望 %q", in, got, want)
		}
	}
	//未知类型原样透出，便于人工发现并补充映射
	for _, in := range []string{"bool", "[]byte", "MyStruct", ""} {
		if got := getBdType(in); got != in {
			t.Errorf("getBdType(%q) = %q, 期望原样返回", in, got)
		}
	}

	defaultCases := map[string]string{
		"int": "DEFAULT 0", "int64": "DEFAULT 0", "string": "DEFAULT ''",
		"float32": "DEFAULT 0", "float64": "DEFAULT 0",
		//time.Time 不设默认值
		"time.Time": "",
	}
	for in, want := range defaultCases {
		if got := getBdDefaultValue(in); got != want {
			t.Errorf("getBdDefaultValue(%q) = %q, 期望 %q", in, got, want)
		}
	}
	//关键回归：未知类型必须返回空串，而不是把Go类型名当默认值
	for _, in := range []string{"bool", "[]byte", "MyStruct", ""} {
		if got := getBdDefaultValue(in); got != "" {
			t.Errorf("getBdDefaultValue(%q) = %q, 期望空串（返回类型名会生成非法SQL）", in, got)
		}
	}
}

func TestCreateRandString(t *testing.T) {
	//负长度曾panic(makeslice: len out of range)，须与0等价返回空串
	for _, n := range []int{-100, -5, -1} {
		if got := CreateRandString(n); got != "" {
			t.Errorf("CreateRandString(%d) = %q, 期望空串", n, got)
		}
	}
	//长度须严格等于入参
	for _, n := range []int{0, 1, 8, 64} {
		got := CreateRandString(n)
		if len([]rune(got)) != n {
			t.Errorf("CreateRandString(%d) 长度 = %d", n, len([]rune(got)))
		}
		//只能出现英文大小写字母，不得掺入数字或符号
		for _, r := range got {
			if !('a' <= r && r <= 'z') && !('A' <= r && r <= 'Z') {
				t.Errorf("CreateRandString(%d) 含非字母字符 %q: %q", n, r, got)
			}
		}
	}
	//随机性：多次生成不应全部相同（否则退化为常量）
	first := CreateRandString(16)
	var diff bool
	for i := 0; i < 20; i++ {
		if CreateRandString(16) != first {
			diff = true
			break
		}
	}
	if !diff {
		t.Errorf("CreateRandString 20次结果全相同，疑似未随机")
	}
}

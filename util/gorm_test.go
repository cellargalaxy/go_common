package util

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/utils/tests"
)

// captureLog 临时把logrus输出重定向到buffer，用于断言日志过滤行为；
// 用完自动恢复，避免影响其他用例
func captureLog(t *testing.T, fn func()) string {
	t.Helper()
	var buf bytes.Buffer
	old := logrus.StandardLogger().Out
	oldLevel := logrus.GetLevel()
	logrus.SetOutput(&buf)
	logrus.SetLevel(logrus.DebugLevel)
	t.Cleanup(func() {
		logrus.SetOutput(old)
		logrus.SetLevel(oldLevel)
	})
	fn()
	logrus.SetOutput(old)
	logrus.SetLevel(oldLevel)
	return buf.String()
}

func TestDefaultSqlLen(t *testing.T) {
	if DefaultSqlLen != 512 {
		t.Errorf("DefaultSqlLen = %d, 期望 512", DefaultSqlLen)
	}
}

func TestNewGormLog(t *testing.T) {
	ignore := []error{gorm.ErrRecordNotFound}
	got := NewGormLog(ignore, 100, true, false, true, false, true)
	if got.SqlLen != 100 {
		t.Errorf("SqlLen = %d", got.SqlLen)
	}
	if !got.InsertShow || got.DeleteShow || !got.SelectShow || got.UpdateShow || !got.OtherShow {
		t.Errorf("开关字段未按序赋值: %+v", got)
	}
	if len(got.IgnoreErrs) != 1 {
		t.Errorf("IgnoreErrs = %v", got.IgnoreErrs)
	}
}

// 默认配置的语义：忽略RecordNotFound，且只展示删/查/其他
func TestNewDefaultGormLog(t *testing.T) {
	got := NewDefaultGormLog()
	if got.SqlLen != DefaultSqlLen {
		t.Errorf("SqlLen = %d, 期望 %d", got.SqlLen, DefaultSqlLen)
	}
	if got.InsertShow {
		t.Errorf("默认不应展示INSERT")
	}
	if !got.DeleteShow {
		t.Errorf("默认应展示DELETE")
	}
	if !got.SelectShow {
		t.Errorf("默认应展示SELECT")
	}
	if got.UpdateShow {
		t.Errorf("默认不应展示UPDATE")
	}
	if !got.OtherShow {
		t.Errorf("默认应展示其他语句")
	}
	//默认须忽略 ErrRecordNotFound，否则查询无结果会刷错误日志
	var found bool
	for _, err := range got.IgnoreErrs {
		if errors.Is(gorm.ErrRecordNotFound, err) {
			found = true
		}
	}
	if !found {
		t.Errorf("默认未忽略 ErrRecordNotFound: %v", got.IgnoreErrs)
	}
}

// LogMode 须返回自身，保证gorm设置级别后配置不丢失
func TestGormLogLogMode(t *testing.T) {
	gormLog := NewGormLog(nil, 77, true, true, true, true, true)
	got := gormLog.LogMode(logger.Info)
	typed, ok := got.(GormLog)
	if !ok {
		t.Fatalf("LogMode 返回类型 = %T, 期望 GormLog", got)
	}
	if typed.SqlLen != 77 {
		t.Errorf("LogMode 后配置丢失: SqlLen = %d", typed.SqlLen)
	}
}

func TestGormLogInfoWarnError(t *testing.T) {
	ctx := GenCtx()
	gormLog := NewDefaultGormLog()

	//三个级别都须把格式化参数正确带出
	out := captureLog(t, func() { gormLog.Info(ctx, "信息%s-%d", "A", 1) })
	if !strings.Contains(out, "信息A-1") {
		t.Errorf("Info 输出 = %q", out)
	}
	out = captureLog(t, func() { gormLog.Warn(ctx, "告警%s", "B") })
	if !strings.Contains(out, "告警B") {
		t.Errorf("Warn 输出 = %q", out)
	}
	out = captureLog(t, func() { gormLog.Error(ctx, "错误%s", "C") })
	if !strings.Contains(out, "错误C") {
		t.Errorf("Error 输出 = %q", out)
	}
}

// Trace 按语句类型过滤：关掉的类型不得输出
func TestGormLogTraceFilterByType(t *testing.T) {
	ctx := GenCtx()
	//只开 SELECT
	gormLog := NewGormLog(nil, DefaultSqlLen, false, false, true, false, false)

	out := captureLog(t, func() {
		gormLog.Trace(ctx, time.Now(), func() (string, int64) { return "SELECT * FROM t", 1 }, nil)
	})
	if !strings.Contains(out, "SELECT * FROM t") {
		t.Errorf("SELECT 已开启但未输出: %q", out)
	}
	//其余类型均应被过滤
	for _, sql := range []string{"INSERT INTO t VALUES(1)", "DELETE FROM t", "UPDATE t SET a=1", "SHOW TABLES"} {
		out = captureLog(t, func() {
			gormLog.Trace(ctx, time.Now(), func() (string, int64) { return sql, 1 }, nil)
		})
		if strings.Contains(out, sql) {
			t.Errorf("%q 未开启却被输出: %q", sql, out)
		}
	}

	//全开时各类型都应输出
	allOn := NewGormLog(nil, DefaultSqlLen, true, true, true, true, true)
	for _, sql := range []string{"INSERT INTO t VALUES(1)", "DELETE FROM t", "UPDATE t SET a=1", "SELECT 1", "SHOW TABLES"} {
		out = captureLog(t, func() {
			allOn.Trace(ctx, time.Now(), func() (string, int64) { return sql, 1 }, nil)
		})
		if !strings.Contains(out, sql) {
			t.Errorf("全开时 %q 未输出: %q", sql, out)
		}
	}
}

// 忽略名单中的错误不得打错误日志；不在名单中的必须打
func TestGormLogTraceIgnoreErrs(t *testing.T) {
	ctx := GenCtx()
	//关闭所有语句展示，以便区分"错误日志"与"语句日志"
	gormLog := NewGormLog([]error{gorm.ErrRecordNotFound}, DefaultSqlLen, false, false, false, false, false)

	//被忽略的错误：不应出现错误日志
	out := captureLog(t, func() {
		gormLog.Trace(ctx, time.Now(), func() (string, int64) { return "SELECT 1", 0 }, gorm.ErrRecordNotFound)
	})
	if strings.Contains(out, "level=error") || strings.Contains(out, "[ERRO]") {
		t.Errorf("被忽略的错误仍打了错误日志: %q", out)
	}
	//包装后的忽略错误同样须被识别（实现用 errors.Is）
	out = captureLog(t, func() {
		wrapped := errors.WithMessage(gorm.ErrRecordNotFound, "上层包装")
		gormLog.Trace(ctx, time.Now(), func() (string, int64) { return "SELECT 1", 0 }, wrapped)
	})
	if strings.Contains(out, "level=error") || strings.Contains(out, "[ERRO]") {
		t.Errorf("包装后的忽略错误未被识别: %q", out)
	}

	//未被忽略的错误：必须打错误日志，且带上SQL
	out = captureLog(t, func() {
		gormLog.Trace(ctx, time.Now(), func() (string, int64) { return "SELECT 2", 0 }, errors.Errorf("真实数据库错误"))
	})
	if !strings.Contains(out, "真实数据库错误") {
		t.Errorf("未忽略的错误未输出: %q", out)
	}
	if !strings.Contains(out, "SELECT 2") {
		t.Errorf("错误日志未带SQL: %q", out)
	}

	//空忽略名单：任何错误都要打
	noIgnore := NewGormLog(nil, DefaultSqlLen, false, false, false, false, false)
	out = captureLog(t, func() {
		noIgnore.Trace(ctx, time.Now(), func() (string, int64) { return "SELECT 3", 0 }, gorm.ErrRecordNotFound)
	})
	if !strings.Contains(out, "SELECT 3") {
		t.Errorf("空忽略名单时错误未输出: %q", out)
	}
}

// SqlLen 截断：超长SQL须被裁剪，避免日志爆炸
func TestGormLogTraceSqlTruncate(t *testing.T) {
	ctx := GenCtx()
	longSQL := "SELECT " + strings.Repeat("x", 1000)

	//限长10：输出不应包含完整SQL
	gormLog := NewGormLog(nil, 10, true, true, true, true, true)
	out := captureLog(t, func() {
		gormLog.Trace(ctx, time.Now(), func() (string, int64) { return longSQL, 1 }, nil)
	})
	if strings.Contains(out, longSQL) {
		t.Errorf("超长SQL未被截断")
	}
	if !strings.Contains(out, "SELECT xxx") {
		t.Errorf("截断后应保留前缀: %q", out)
	}

	//SqlLen<=0 表示不截断
	noLimit := NewGormLog(nil, 0, true, true, true, true, true)
	out = captureLog(t, func() {
		noLimit.Trace(ctx, time.Now(), func() (string, int64) { return longSQL, 1 }, nil)
	})
	if !strings.Contains(out, longSQL) {
		t.Errorf("SqlLen=0 时不应截断")
	}

	//SQL短于限长时不应被改动
	shortLimit := NewGormLog(nil, 1000, true, true, true, true, true)
	out = captureLog(t, func() {
		shortLimit.Trace(ctx, time.Now(), func() (string, int64) { return "SELECT 1", 1 }, nil)
	})
	if !strings.Contains(out, "SELECT 1") {
		t.Errorf("短SQL被误改: %q", out)
	}
}

// Trace 须记录耗时，且空SQL不能panic
func TestGormLogTraceEdge(t *testing.T) {
	ctx := GenCtx()
	gormLog := NewGormLog(nil, DefaultSqlLen, true, true, true, true, true)

	//耗时字段须出现
	out := captureLog(t, func() {
		gormLog.Trace(ctx, time.Now().Add(-50*time.Millisecond), func() (string, int64) { return "SELECT 1", 1 }, nil)
	})
	if !strings.Contains(out, "elapsed") {
		t.Errorf("未记录耗时: %q", out)
	}
	//空SQL：走 OtherShow 分支，不能panic
	out = captureLog(t, func() {
		gormLog.Trace(ctx, time.Now(), func() (string, int64) { return "", 0 }, nil)
	})
	_ = out
	//小写语句：实现按大写前缀匹配，会落到 other 分支
	out = captureLog(t, func() {
		gormLog.Trace(ctx, time.Now(), func() (string, int64) { return "select 1", 1 }, nil)
	})
	if !strings.Contains(out, "select 1") {
		t.Errorf("小写语句应走other分支输出: %q", out)
	}
}

// ==== GormService 的无DB守卫路径 ====
// 注：增删改的真实落库需要数据库驱动；不触达DB的早返回与参数校验分支在此覆盖，
// 而 Select/SelectOne 的SQL构造逻辑改用gorm自带的 DummyDialector + DryRun 覆盖（见下方）。

type fakeGormObject struct {
	Id   int64
	Name string
}

func (this fakeGormObject) TableName() string { return "fake_object" }

type fakeGormInquiry struct {
	fakeGormObject
	PageNum  int
	PageSize int
}

func (this fakeGormInquiry) GetPageNum() int  { return this.PageNum }
func (this fakeGormInquiry) GetPageSize() int { return this.PageSize }

// fakeGormHandler 的 Where 返回nil，用于驱动"条件为空"的守卫分支
type fakeGormHandler struct {
	whereNil bool
}

func (this *fakeGormHandler) GetName(ctx context.Context) string { return "假对象" }
func (this *fakeGormHandler) GetDb(ctx context.Context, where *gorm.DB) *gorm.DB {
	return where
}
func (this *fakeGormHandler) Where(ctx context.Context, where *gorm.DB, inquiry fakeGormInquiry) *gorm.DB {
	if this.whereNil {
		return nil
	}
	return where
}

func TestGormServiceInsertEmpty(t *testing.T) {
	ctx := GenCtx()
	service := &GormService[fakeGormObject, fakeGormInquiry]{GormHandler: &fakeGormHandler{}}

	//空入参须直接返回，不触达数据库（否则这里会因DB为nil而panic）
	got, err := service.Insert(ctx)
	if err != nil {
		t.Errorf("空插入应无错误: %+v", err)
	}
	if len(got) != 0 {
		t.Errorf("空插入返回 = %v", got)
	}
}

func TestGormServiceUpdateNil(t *testing.T) {
	ctx := GenCtx()
	service := &GormService[fakeGormObject, fakeGormInquiry]{GormHandler: &fakeGormHandler{}}

	//nil 对象须直接返回，不触达数据库
	got, count, err := service.Update(ctx, nil)
	if err != nil {
		t.Errorf("nil 更新应无错误: %+v", err)
	}
	if got != nil {
		t.Errorf("nil 更新返回 = %v", got)
	}
	if count != 0 {
		t.Errorf("nil 更新影响行数 = %d, 期望 0", count)
	}
}

// Where 返回nil时，Delete 必须拒绝执行——否则会退化成全表删除
func TestGormServiceDeleteNilWhere(t *testing.T) {
	ctx := GenCtx()
	service := &GormService[fakeGormObject, fakeGormInquiry]{GormHandler: &fakeGormHandler{whereNil: true}}

	err := service.Delete(ctx, fakeGormInquiry{})
	if err == nil {
		t.Errorf("删除条件为空时必须报错，避免误删全表")
	}
	if !strings.Contains(err.Error(), "条件为空") {
		t.Errorf("错误信息 = %v, 应说明条件为空", err)
	}
}

// GormObject/GormInquiry 接口实现须自洽
func TestGormInterfaces(t *testing.T) {
	var object GormObject = fakeGormObject{}
	if got := object.TableName(); got != "fake_object" {
		t.Errorf("TableName = %q", got)
	}
	var inquiry GormInquiry = fakeGormInquiry{PageNum: 2, PageSize: 20}
	if got := inquiry.GetPageNum(); got != 2 {
		t.Errorf("GetPageNum = %d", got)
	}
	if got := inquiry.GetPageSize(); got != 20 {
		t.Errorf("GetPageSize = %d", got)
	}
	//Inquiry 须同时满足 GormObject
	if got := inquiry.TableName(); got != "fake_object" {
		t.Errorf("Inquiry.TableName = %q", got)
	}
}

// ==== Select / SelectOne ====
// 用 gorm 自带的 DummyDialector + DryRun 覆盖SQL构造逻辑：
// DryRun 只生成SQL不真正执行，因此无需数据库、也无需引入额外测试driver。

// dryRunHandler 提供一个 DryRun 的 *gorm.DB，并记录最终用于查询的 where，
// 以便断言分页子句是否被正确拼装
type dryRunHandler struct {
	db       *gorm.DB
	captured *gorm.DB
	forceErr error
	whereNil bool
}

func (this *dryRunHandler) GetName(ctx context.Context) string { return "假对象" }
func (this *dryRunHandler) GetDb(ctx context.Context, where *gorm.DB) *gorm.DB {
	if where != nil {
		return where
	}
	//每次都用全新会话，避免用例之间条件串味
	return this.db.Session(&gorm.Session{DryRun: true, NewDB: true})
}
func (this *dryRunHandler) Where(ctx context.Context, where *gorm.DB, inquiry fakeGormInquiry) *gorm.DB {
	if this.whereNil {
		return nil
	}
	if where == nil {
		return nil
	}
	if this.forceErr != nil {
		where.AddError(this.forceErr)
	}
	if inquiry.Id != 0 {
		where = where.Where("id = ?", inquiry.Id)
	}
	this.captured = where
	return where
}

func newDryRunService(t *testing.T, handler *dryRunHandler) *GormService[fakeGormObject, fakeGormInquiry] {
	t.Helper()
	db, err := gorm.Open(tests.DummyDialector{}, &gorm.Config{DryRun: true})
	if err != nil {
		t.Fatalf("打开DryRun数据库异常: %+v", err)
	}
	handler.db = db
	return &GormService[fakeGormObject, fakeGormInquiry]{GormHandler: handler}
}

// 分页语义：pageSize 决定 LIMIT，pageNum 决定 OFFSET=(pageNum-1)*pageSize
func TestGormServiceSelectPaging(t *testing.T) {
	ctx := GenCtx()
	cases := []struct {
		pageNum, pageSize int
		wantLimit         bool
		wantLimitValue    int
		wantOffset        int
	}{
		//第3页、每页10条 -> LIMIT 10 OFFSET 20
		{3, 10, true, 10, 20},
		//第1页 -> OFFSET 0
		{1, 10, true, 10, 0},
		//pageSize<=0 表示不分页，不应拼 LIMIT
		{0, 0, false, 0, 0},
		//只有pageSize：仅限制条数，不偏移
		{0, 5, true, 5, 0},
	}
	for _, c := range cases {
		handler := &dryRunHandler{}
		service := newDryRunService(t, handler)
		_, _, err := service.Select(ctx, fakeGormInquiry{
			fakeGormObject: fakeGormObject{Id: 7},
			PageNum:        c.pageNum,
			PageSize:       c.pageSize,
		})
		if err != nil {
			t.Fatalf("pageNum=%d pageSize=%d 查询异常: %+v", c.pageNum, c.pageSize, err)
		}
		if handler.captured == nil {
			t.Fatalf("pageNum=%d pageSize=%d 未捕获到查询条件", c.pageNum, c.pageSize)
		}
		limitClause, ok := handler.captured.Statement.Clauses["LIMIT"]
		if !c.wantLimit {
			if ok {
				t.Errorf("pageNum=%d pageSize=%d 不应拼装LIMIT子句", c.pageNum, c.pageSize)
			}
			continue
		}
		if !ok {
			t.Fatalf("pageNum=%d pageSize=%d 缺少LIMIT子句", c.pageNum, c.pageSize)
		}
		limit, ok := limitClause.Expression.(clause.Limit)
		if !ok {
			t.Fatalf("LIMIT子句类型 = %T", limitClause.Expression)
		}
		if limit.Limit == nil {
			t.Fatalf("pageNum=%d pageSize=%d LIMIT值为空", c.pageNum, c.pageSize)
		}
		if *limit.Limit != c.wantLimitValue {
			t.Errorf("pageNum=%d pageSize=%d LIMIT = %d, 期望 %d", c.pageNum, c.pageSize, *limit.Limit, c.wantLimitValue)
		}
		if limit.Offset != c.wantOffset {
			t.Errorf("pageNum=%d pageSize=%d OFFSET = %d, 期望 %d", c.pageNum, c.pageSize, limit.Offset, c.wantOffset)
		}
	}
}

// 查询条件须真正下推到SQL，否则会退化成全表查询
func TestGormServiceSelectWhereApplied(t *testing.T) {
	ctx := GenCtx()
	handler := &dryRunHandler{}
	service := newDryRunService(t, handler)

	list, count, err := service.Select(ctx, fakeGormInquiry{fakeGormObject: fakeGormObject{Id: 7}})
	if err != nil {
		t.Fatalf("查询异常: %+v", err)
	}
	//DryRun 不真正执行，结果集为空、count为0，但不得报错
	if len(list) != 0 {
		t.Errorf("DryRun 结果集 = %v, 期望空", list)
	}
	if count != 0 {
		t.Errorf("DryRun count = %d, 期望 0", count)
	}
	//WHERE 必须带上条件
	whereClause, ok := handler.captured.Statement.Clauses["WHERE"]
	if !ok {
		t.Fatalf("缺少WHERE子句，查询条件未下推")
	}
	where, ok := whereClause.Expression.(clause.Where)
	if !ok {
		t.Fatalf("WHERE子句类型 = %T", whereClause.Expression)
	}
	if len(where.Exprs) == 0 {
		t.Errorf("WHERE 条件为空，等同全表查询")
	}
	//表名须取自 TableName()
	if got := handler.captured.Statement.Table; got != "fake_object" {
		t.Errorf("查询表名 = %q, 期望 fake_object", got)
	}
}

// 底层错误必须被包装返回，不能被吞掉
func TestGormServiceSelectError(t *testing.T) {
	ctx := GenCtx()
	handler := &dryRunHandler{forceErr: gorm.ErrInvalidField}
	service := newDryRunService(t, handler)

	list, _, err := service.Select(ctx, fakeGormInquiry{})
	if err == nil {
		t.Fatalf("底层错误未返回")
	}
	if !strings.Contains(err.Error(), "异常") {
		t.Errorf("错误信息 = %v, 应说明查询异常", err)
	}
	if list != nil {
		t.Errorf("出错时结果集 = %v, 期望 nil", list)
	}
}

// ErrRecordNotFound 不算错误：应返回空结果而非error，否则调用方每次查空都要处理异常
func TestGormServiceSelectRecordNotFound(t *testing.T) {
	ctx := GenCtx()
	handler := &dryRunHandler{forceErr: gorm.ErrRecordNotFound}
	service := newDryRunService(t, handler)

	list, count, err := service.Select(ctx, fakeGormInquiry{})
	if err != nil {
		t.Errorf("RecordNotFound 应被视为空结果，而非错误: %+v", err)
	}
	if len(list) != 0 {
		t.Errorf("结果集 = %v, 期望空", list)
	}
	if count != 0 {
		t.Errorf("count = %d, 期望 0", count)
	}
}

// SelectOne 是 Select 的包装：空结果返回 nil,nil；出错则透传error
func TestGormServiceSelectOne(t *testing.T) {
	ctx := GenCtx()

	//空结果：须返回 nil 且不报错
	handler := &dryRunHandler{}
	service := newDryRunService(t, handler)
	got, err := service.SelectOne(ctx, fakeGormInquiry{fakeGormObject: fakeGormObject{Id: 7}})
	if err != nil {
		t.Errorf("空结果不应报错: %+v", err)
	}
	if got != nil {
		t.Errorf("空结果 = %v, 期望 nil", got)
	}

	//出错：须透传error且不返回对象
	errHandler := &dryRunHandler{forceErr: gorm.ErrInvalidField}
	errService := newDryRunService(t, errHandler)
	got, err = errService.SelectOne(ctx, fakeGormInquiry{})
	if err == nil {
		t.Errorf("底层错误未透传")
	}
	if got != nil {
		t.Errorf("出错时返回 = %v, 期望 nil", got)
	}
}

// Where 返回nil时 Select 的行为：与 Delete 的显式守卫形成对比。
// 原用例只在未panic时 t.Logf 提示，无论实现怎么变都不会失败，等于没有校验；
// 这里改为断言"要么panic、要么返回error"，即绝不允许静默当成无条件查询成功返回数据。
func TestGormServiceSelectNilWhereNoGuard(t *testing.T) {
	ctx := GenCtx()
	handler := &dryRunHandler{whereNil: true}
	service := newDryRunService(t, handler)

	var panicked bool
	var err error
	var list []*fakeGormObject
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
			}
		}()
		list, _, err = service.Select(ctx, fakeGormInquiry{})
	}()

	//Delete 侧已有显式守卫（见 TestGormServiceDeleteNilWhere）。
	//Select 侧至少不能"无守卫且无错误地"返回结果集，否则等同于放行无条件查询。
	if !panicked && err == nil {
		t.Errorf("Where返回nil时 Select 既未panic也未报错（结果集=%v），"+
			"相当于放行无条件查询，应与Delete一致补充显式守卫", list)
	}
}

// ==== Insert / Update / Delete 的SQL构造 ====
// 同样用 DryRun 覆盖：只校验生成的SQL语义，不需要真实数据库
//
// 这里另起一套 RecordObject/RecordInquiry 夹具：内嵌类型名必须是导出的，
// 否则 gorm 反射读取"通过未导出内嵌字段提升上来的主键"会panic
// （reflect.Value.Interface: cannot return value obtained from unexported field）

type RecordObject struct {
	Id   int64
	Name string
}

func (this RecordObject) TableName() string { return "record_object" }

type RecordInquiry struct {
	RecordObject
	PageNum  int
	PageSize int
}

func (this RecordInquiry) GetPageNum() int  { return this.PageNum }
func (this RecordInquiry) GetPageSize() int { return this.PageSize }

// recordHandler 记录最终执行的SQL，用于断言语句类型与目标表
type recordHandler struct {
	db   *gorm.DB
	sqls []string
}

func (this *recordHandler) GetName(ctx context.Context) string { return "假对象" }
func (this *recordHandler) GetDb(ctx context.Context, where *gorm.DB) *gorm.DB {
	if where != nil {
		return where
	}
	return this.db.Session(&gorm.Session{DryRun: true, NewDB: true})
}
func (this *recordHandler) Where(ctx context.Context, where *gorm.DB, inquiry RecordInquiry) *gorm.DB {
	if where == nil {
		//Delete 传入的是nil，需要自行起一个会话，否则无法构造SQL
		where = this.db.Session(&gorm.Session{DryRun: true, NewDB: true}).Model(&RecordObject{})
	}
	if inquiry.Id != 0 {
		where = where.Where("id = ?", inquiry.Id)
	}
	return where
}
func (this *recordHandler) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	sql, _ := fc()
	this.sqls = append(this.sqls, sql)
}
func (this *recordHandler) LogMode(logger.LogLevel) logger.Interface      { return this }
func (this *recordHandler) Info(context.Context, string, ...interface{})  {}
func (this *recordHandler) Warn(context.Context, string, ...interface{})  {}
func (this *recordHandler) Error(context.Context, string, ...interface{}) {}

func newRecordService(t *testing.T) (*GormService[RecordObject, RecordInquiry], *recordHandler) {
	t.Helper()
	handler := &recordHandler{}
	db, err := gorm.Open(tests.DummyDialector{}, &gorm.Config{DryRun: true, Logger: handler})
	if err != nil {
		t.Fatalf("打开DryRun数据库异常: %+v", err)
	}
	handler.db = db
	return &GormService[RecordObject, RecordInquiry]{GormHandler: handler}, handler
}

// 非空插入须生成 INSERT，且落到正确的表
func TestGormServiceInsertSql(t *testing.T) {
	ctx := GenCtx()
	service, handler := newRecordService(t)

	got, err := service.Insert(ctx, &RecordObject{Id: 1, Name: "甲"}, &RecordObject{Id: 2, Name: "乙"})
	if err != nil {
		t.Fatalf("插入异常: %+v", err)
	}
	//入参须原样返回，供调用方拿回带自增ID的对象
	if len(got) != 2 {
		t.Errorf("返回对象数 = %d, 期望 2", len(got))
	}
	if len(handler.sqls) == 0 {
		t.Fatalf("未生成任何SQL")
	}
	sql := handler.sqls[len(handler.sqls)-1]
	if !strings.HasPrefix(sql, "INSERT") {
		t.Errorf("生成的SQL = %q, 期望以 INSERT 开头", sql)
	}
	if !strings.Contains(sql, "record_object") {
		t.Errorf("SQL未落到 record_object 表: %q", sql)
	}
	//两条记录须在同一条语句里批量插入
	if !strings.Contains(sql, "甲") || !strings.Contains(sql, "乙") {
		t.Errorf("批量插入未包含全部记录: %q", sql)
	}
}

// 更新须生成 UPDATE；Select("*") 意味着零值字段也要被写入
func TestGormServiceUpdateSql(t *testing.T) {
	ctx := GenCtx()
	service, handler := newRecordService(t)

	object := &RecordObject{Id: 5, Name: ""}
	got, _, err := service.Update(ctx, object)
	if err != nil {
		t.Fatalf("更新异常: %+v", err)
	}
	if got != object {
		t.Errorf("须原样返回入参对象")
	}
	if len(handler.sqls) == 0 {
		t.Fatalf("未生成任何SQL")
	}
	sql := handler.sqls[len(handler.sqls)-1]
	if !strings.HasPrefix(sql, "UPDATE") {
		t.Errorf("生成的SQL = %q, 期望以 UPDATE 开头", sql)
	}
	//实现里用了 Select("*")，因此空字符串的 name 也必须出现在SET中，
	//否则"清空某字段"的更新会静默丢失
	if !strings.Contains(sql, "name") {
		t.Errorf("Select(\"*\") 应使零值字段也被更新, SQL = %q", sql)
	}
}

// 删除须生成 DELETE 且带上WHERE，避免全表删除
func TestGormServiceDeleteSql(t *testing.T) {
	ctx := GenCtx()
	service, handler := newRecordService(t)

	err := service.Delete(ctx, RecordInquiry{RecordObject: RecordObject{Id: 9}})
	if err != nil {
		t.Fatalf("删除异常: %+v", err)
	}
	if len(handler.sqls) == 0 {
		t.Fatalf("未生成任何SQL")
	}
	sql := handler.sqls[len(handler.sqls)-1]
	if !strings.HasPrefix(sql, "DELETE") {
		t.Errorf("生成的SQL = %q, 期望以 DELETE 开头", sql)
	}
	//必须带WHERE，否则是全表删除
	if !strings.Contains(strings.ToUpper(sql), "WHERE") {
		t.Errorf("DELETE 未带WHERE，存在全表删除风险: %q", sql)
	}
	if !strings.Contains(sql, "9") {
		t.Errorf("DELETE 未带上查询条件的值: %q", sql)
	}
}

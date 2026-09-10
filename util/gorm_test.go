package util

import (
	"bytes"
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
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

func TestLogSqlLen(t *testing.T) {
	if LogSqlLen != 512 {
		t.Errorf("LogSqlLen = %d, 期望 512", LogSqlLen)
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
	if got.SqlLen != LogSqlLen {
		t.Errorf("SqlLen = %d, 期望 %d", got.SqlLen, LogSqlLen)
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
	gormLog := NewGormLog(nil, LogSqlLen, false, false, true, false, false)

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
	allOn := NewGormLog(nil, LogSqlLen, true, true, true, true, true)
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
	gormLog := NewGormLog([]error{gorm.ErrRecordNotFound}, LogSqlLen, false, false, false, false, false)

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
	noIgnore := NewGormLog(nil, LogSqlLen, false, false, false, false, false)
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
	gormLog := NewGormLog(nil, LogSqlLen, true, true, true, true, true)

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

// ==== Transaction 与各 Handler ====
// 增删改查的真实落库需要数据库驱动，这里统一用 gorm 自带的 DummyDialector + DryRun
// 覆盖SQL构造与影响行数，不触达DB的早返回与参数校验分支一并覆盖。

type fakeGormObject struct {
	Id   int64
	Name string
}

func (this fakeGormObject) TableName() string { return "fake_object" }

type fakeGormInquiry struct {
	Id       int64
	PageNum  int
	PageSize int
}

// fakeInquiryHandler 记录每个钩子被调用的次数与最终SQL，用于断言 Where/Order/Limit 的编排顺序
type fakeInquiryHandler struct {
	whereNil  bool
	whereCall int
	orderCall int
	limitCall int
}

func (this *fakeInquiryHandler) Where(ctx context.Context, tx *gorm.DB, inquiry fakeGormInquiry) *gorm.DB {
	this.whereCall++
	if this.whereNil {
		return nil
	}
	if inquiry.Id != 0 {
		tx = tx.Where("id = ?", inquiry.Id)
	}
	return tx
}
func (this *fakeInquiryHandler) Order(ctx context.Context, tx *gorm.DB, inquiry fakeGormInquiry) *gorm.DB {
	this.orderCall++
	return tx.Order("id DESC")
}
func (this *fakeInquiryHandler) Limit(ctx context.Context, tx *gorm.DB, inquiry fakeGormInquiry) *gorm.DB {
	this.limitCall++
	if inquiry.PageSize <= 0 {
		return tx
	}
	return tx.Limit(inquiry.PageSize).Offset((inquiry.PageNum - 1) * inquiry.PageSize)
}

// sqlRecorder 记录 DryRun 下构造出的SQL
type sqlRecorder struct {
	sqls []string
}

func (this *sqlRecorder) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	sql, _ := fc()
	this.sqls = append(this.sqls, sql)
}
func (this *sqlRecorder) LogMode(logger.LogLevel) logger.Interface      { return this }
func (this *sqlRecorder) Info(context.Context, string, ...interface{})  {}
func (this *sqlRecorder) Warn(context.Context, string, ...interface{})  {}
func (this *sqlRecorder) Error(context.Context, string, ...interface{}) {}

// dryRunConnPool 只提供事务的开启与提交语义，DryRun下不会真的执行SQL，
// 用于覆盖 Transaction；gorm自带的 DummyDialector 不带连接池，Begin 会直接报 invalid transaction
type dryRunConnPool struct{}

func (this *dryRunConnPool) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	return nil, nil
}
func (this *dryRunConnPool) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return nil, nil
}
func (this *dryRunConnPool) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return nil, nil
}
func (this *dryRunConnPool) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return nil
}
func (this *dryRunConnPool) BeginTx(ctx context.Context, opts *sql.TxOptions) (gorm.ConnPool, error) {
	return &dryRunTx{}, nil
}

// dryRunTx 与 dryRunConnPool 必须是不同类型：只有事务对象才实现 Commit/Rollback，
// 否则gorm会把外层连接池也当成"已在事务中"，转而走SavePoint，DummyDialector不支持而报 unsupported driver
type dryRunTx struct {
	dryRunConnPool
}

func (this *dryRunTx) Commit() error   { return nil }
func (this *dryRunTx) Rollback() error { return nil }

func newDryRunDb(t *testing.T) (*gorm.DB, *sqlRecorder) {
	t.Helper()
	recorder := &sqlRecorder{}
	db, err := gorm.Open(tests.DummyDialector{}, &gorm.Config{DryRun: true, Logger: recorder, ConnPool: &dryRunConnPool{}})
	if err != nil {
		t.Fatalf("打开DryRun数据库异常: %+v", err)
	}
	return db, recorder
}

func (this *sqlRecorder) contains(keyword string) bool {
	for i := range this.sqls {
		if strings.Contains(this.sqls[i], keyword) {
			return true
		}
	}
	return false
}

// 空入参须直接返回，不触达数据库（否则DB为nil时会panic）
func TestInsertHandlerEmpty(t *testing.T) {
	ctx := GenCtx()
	handler := NewInsertHandler[fakeGormObject]("假对象")

	if err := handler.Transaction(ctx, nil); err != nil {
		t.Errorf("空插入应无错误: %+v", err)
	}
	if handler.Count != 0 {
		t.Errorf("空插入影响行数 = %d, 期望 0", handler.Count)
	}
}

func TestInsertHandlerSql(t *testing.T) {
	ctx := GenCtx()
	db, recorder := newDryRunDb(t)
	handler := NewInsertHandler("假对象", &fakeGormObject{Id: 1, Name: "a"}, &fakeGormObject{Id: 2, Name: "b"})

	if err := handler.Transaction(ctx, db); err != nil {
		t.Fatalf("插入异常: %+v", err)
	}
	if !recorder.contains("INSERT") || !recorder.contains("fake_object") {
		t.Errorf("未生成落到 fake_object 的INSERT: %v", recorder.sqls)
	}
}

// nil 对象须直接返回，不触达数据库
func TestUpdateHandlerNil(t *testing.T) {
	ctx := GenCtx()
	handler := NewUpdateHandler[fakeGormObject]("假对象", nil)

	if err := handler.Transaction(ctx, nil); err != nil {
		t.Errorf("nil更新应无错误: %+v", err)
	}
	if handler.Count != 0 {
		t.Errorf("nil更新影响行数 = %d, 期望 0", handler.Count)
	}
}

func TestUpdateHandlerSql(t *testing.T) {
	ctx := GenCtx()
	db, recorder := newDryRunDb(t)
	handler := NewUpdateHandler("假对象", &fakeGormObject{Id: 1, Name: "a"})

	if err := handler.Transaction(ctx, db); err != nil {
		t.Fatalf("更新异常: %+v", err)
	}
	if !recorder.contains("UPDATE") || !recorder.contains("fake_object") {
		t.Errorf("未生成落到 fake_object 的UPDATE: %v", recorder.sqls)
	}
}

func TestDeleteHandlerSql(t *testing.T) {
	ctx := GenCtx()
	db, recorder := newDryRunDb(t)
	inquiryHandler := &fakeInquiryHandler{}
	handler := NewDeleteHandler[fakeGormObject]("假对象", fakeGormInquiry{Id: 7}, inquiryHandler)

	if err := handler.Transaction(ctx, db.Model(&fakeGormObject{})); err != nil {
		t.Fatalf("删除异常: %+v", err)
	}
	if !recorder.contains("DELETE") || !recorder.contains("fake_object") {
		t.Errorf("未生成落到 fake_object 的DELETE: %v", recorder.sqls)
	}
	//Where/Order/Limit 三个钩子都必须被调用，缺一会让条件或分页静默失效
	if inquiryHandler.whereCall != 1 || inquiryHandler.orderCall != 1 || inquiryHandler.limitCall != 1 {
		t.Errorf("钩子调用次数 where=%d order=%d limit=%d, 期望各1次",
			inquiryHandler.whereCall, inquiryHandler.orderCall, inquiryHandler.limitCall)
	}
}

// 分页语义：pageSize 决定 LIMIT，pageNum 决定 OFFSET=(pageNum-1)*pageSize。
// 本用例当前必然失败，用于标记已知缺陷：Count 与 Find 复用同一个tx，
// 而 gorm 只在 Statement.SQL 为空时才重建SQL(callbacks/query.go)，
// 于是 Find 直接复用了 Count 那条语句，Order/Limit 失效、查回的是count结果。
// 修法是让 Count 走独立会话：tx.Session(&gorm.Session{}).Count(&this.Count)
func TestSelectHandlerPaging(t *testing.T) {
	ctx := GenCtx()
	db, recorder := newDryRunDb(t)
	inquiryHandler := &fakeInquiryHandler{}
	handler := NewSelectHandler[fakeGormObject]("假对象", fakeGormInquiry{Id: 7, PageNum: 2, PageSize: 20}, inquiryHandler)

	if err := handler.Transaction(ctx, db); err != nil {
		t.Fatalf("查询异常: %+v", err)
	}
	if !recorder.contains("SELECT") || !recorder.contains("fake_object") {
		t.Errorf("未生成落到 fake_object 的SELECT: %v", recorder.sqls)
	}
	if !recorder.contains("LIMIT 20") {
		t.Errorf("未按 pageSize 生成LIMIT: %v", recorder.sqls)
	}
	if !recorder.contains("OFFSET 20") {
		t.Errorf("未按 pageNum 生成OFFSET: %v", recorder.sqls)
	}
	//Count 先于 Order/Limit 执行，故 Where 会被调用一次，Order/Limit 各一次
	if inquiryHandler.whereCall != 1 || inquiryHandler.orderCall != 1 || inquiryHandler.limitCall != 1 {
		t.Errorf("钩子调用次数 where=%d order=%d limit=%d, 期望各1次",
			inquiryHandler.whereCall, inquiryHandler.orderCall, inquiryHandler.limitCall)
	}
}

// 无结果时 GetOne 必须返回nil而不是越界panic
func TestSelectHandlerGetOne(t *testing.T) {
	ctx := GenCtx()
	db, _ := newDryRunDb(t)
	handler := NewSelectHandler[fakeGormObject]("假对象", fakeGormInquiry{}, &fakeInquiryHandler{})

	if got := handler.GetOne(); got != nil {
		t.Errorf("空结果 GetOne = %v, 期望 nil", got)
	}
	if err := handler.Transaction(ctx, db); err != nil {
		t.Fatalf("查询异常: %+v", err)
	}
	if got := handler.GetOne(); got != nil {
		t.Errorf("DryRun无数据时 GetOne = %v, 期望 nil", got)
	}
	//Object 初始化为空切片而非nil，调用方可直接range
	if handler.Object == nil {
		t.Errorf("Object 未初始化为空切片")
	}
	//有数据时须返回首个元素
	handler.Object = []*fakeGormObject{{Id: 1}, {Id: 2}}
	if got := handler.GetOne(); got == nil || got.Id != 1 {
		t.Errorf("GetOne = %v, 期望首个元素", got)
	}
}

// Transaction 须按顺序执行全部handler，任一报错则中断并向上抛出
type errHandler struct {
	called int
	err    error
}

func (this *errHandler) Transaction(ctx context.Context, tx *gorm.DB) error {
	this.called++
	return this.err
}

func TestTransaction(t *testing.T) {
	ctx := GenCtx()
	db, _ := newDryRunDb(t)

	first := &errHandler{}
	second := &errHandler{}
	if err := Transaction(ctx, db, first, second); err != nil {
		t.Fatalf("事务异常: %+v", err)
	}
	if first.called != 1 || second.called != 1 {
		t.Errorf("handler 调用次数 = %d/%d, 期望各1次", first.called, second.called)
	}

	//前一个handler报错时，后续handler不得再执行
	failed := &errHandler{err: errors.Errorf("处理失败")}
	skipped := &errHandler{}
	err := Transaction(ctx, db, failed, skipped)
	if err == nil {
		t.Fatalf("handler报错时事务必须返回错误")
	}
	if !strings.Contains(err.Error(), "处理失败") {
		t.Errorf("错误未透传: %v", err)
	}
	if skipped.called != 0 {
		t.Errorf("前置handler报错后仍执行了后续handler")
	}

	//无handler时须正常返回
	if err = Transaction(ctx, db); err != nil {
		t.Errorf("空handler事务异常: %+v", err)
	}
}

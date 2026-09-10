package util

import (
	"context"
	"os"
	"path"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

// testProcStart 记录测试进程启动时刻，logDirRoot 记录 util/log 的绝对路径。
//
// Go 先执行被测包（util.go）的 init()，再执行同包测试文件的 init()。util.go 的
// InitDefaultLog() 在 serverName 为空时用 GenStrId()（时间戳）当目录名，
// 故每跑一次测试都会在 util/log/<时间戳>/ 新增一个日志目录（复核时已累积约280个、23MB）。
// 这里在切换CWD前记下路径，供 TestMain 前后清理这些目录。
var logDirRoot string

// testLogServerName 测试进程内重新初始化日志时使用的固定 serverName，
// 避免 TestMain 再次 InitDefaultLog() 时又按时间戳新建一个目录。
//
// 注意不能改用设置 server_name 环境变量的办法来根治：GetServerName() 会优先读该变量，
// 而 TestInitOsAndGetServerName 正是在验证「环境变量 > InitOs 默认值 > 空值回落」这套
// 优先级，进程级预设环境变量会直接让该用例失真。
const testLogServerName = "go_common_test"

func init() {
	//必须在切换CWD之前记录，此时相对路径 log 仍指向 util 包目录
	if wd, err := os.Getwd(); err == nil {
		logDirRoot = path.Join(wd, "log")
	}
	Init("go_common")
	//测试模式下关闭gin的调试日志，避免污染用例输出
	gin.SetMode(gin.TestMode)
}

// isGenIdDirName 判断目录名是否为 GenStrId() 生成的纯数字时间戳ID。
// 只有这种名字才可能是"serverName为空时自动兜底生成"的日志目录，
// 用户显式指定的 serverName（如 go_common、svc）不会被误删。
func isGenIdDirName(name string) bool {
	if name == "" {
		return false
	}
	for _, r := range name {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// cleanTestLogDirs 清理 util/log 下由测试进程产生的日志目录。
//
// 需要清理两类：
//  1. 纯数字时间戳目录：serverName 为空时 InitDefaultLog() 用 GenStrId() 兜底命名，
//     每次运行新增一个（复核时已累积约280个/23MB）；
//  2. 固定 serverName 目录（go_common / go_common_test 等）：由 init() 与用例中的日志
//     写入不断追加，lumberjack 还会滚动出多个 1MB 备份（复核时约11MB）。
//
// 判据只用目录名形态，不附加"修改时间不早于本进程启动"：用例中若发生 panic，进程会在
// TestMain 收尾逻辑前退出，那一次的目录就会残留，下次运行时它已成"历史目录"从而被时间
// 判据永久跳过，于是不断累积。这些目录全部由测试进程自身产生（生产代码跑在各自部署目录，
// 不会写到本仓库的 util/log），可以安全清理。
func cleanTestLogDirs() {
	if logDirRoot == "" {
		return
	}
	entries, err := os.ReadDir(logDirRoot)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		//时间戳兜底目录，或测试进程使用的固定 serverName 目录
		if !isGenIdDirName(name) && name != "go_common" && name != testLogServerName {
			continue
		}
		_ = os.RemoveAll(path.Join(logDirRoot, name))
	}
	//目录空了就一并移除，保持仓库干净
	if remain, e := os.ReadDir(logDirRoot); e == nil && len(remain) == 0 {
		_ = os.Remove(logDirRoot)
	}
}

// TestMain 把整个测试进程的工作目录切到临时目录再跑用例，并清理测试产生的日志目录。
//
// 该日志行为属于生产代码既有逻辑，本次单测复核不改动生产代码；但测试自身不应污染源码树：
//  1. chdir 到临时目录并重新初始化日志，使用例期间的日志落到临时目录（原先约144KB/次）；
//  2. 进入用例前先清一次历史残留（含此前 panic 退出留下的），跑完再清一次本轮新增的。
func TestMain(m *testing.M) {
	origin, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	if logDirRoot == "" {
		logDirRoot = path.Join(origin, "log")
	}
	//先清理历史残留：util.go 的 init() 早于本文件执行，已在源码树建了一个时间戳目录
	cleanTestLogDirs()

	tmp, err := os.MkdirTemp("", "gocommon_testmain_")
	if err != nil {
		panic(err)
	}
	if err = os.Chdir(tmp); err != nil {
		panic(err)
	}
	//关键：切目录后重新初始化日志，否则 lumberjack 仍按原 CWD 落盘到源码树。
	//这里传固定 serverName，避免再产生一个时间戳目录（此时CWD已在临时目录，仅为稳妥）。
	Init(testLogServerName)
	//用例默认期望 serverName 为 go_common（codec/gin 等用例会与 GetServerName() 对比），
	//日志初始化完成后即刻复原，不影响后续断言
	Init("go_common")

	code := m.Run()

	//先切回原目录，否则临时目录删除后 CWD 失效
	_ = os.Chdir(origin)
	_ = os.RemoveAll(tmp)
	//再清理本轮新增的兜底日志目录，保持仓库干净
	cleanTestLogDirs()
	os.Exit(code)
}

// ==== 测试公用辅助 ====

// timeAfterMs 返回指定毫秒后触发的channel，用于超时断言
func timeAfterMs(ms int) <-chan time.Time {
	return time.After(time.Duration(ms) * time.Millisecond)
}

// newTestDir 创建一个测试专用临时目录，并注册自动清理，避免用例间互相干扰
func newTestDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "gocommon_test_")
	if err != nil {
		t.Fatalf("创建临时目录失败: %+v", err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

// newTestFile 在临时目录下生成含指定内容的文件，返回其路径
func newTestFile(t *testing.T, content string) string {
	t.Helper()
	dir := newTestDir(t)
	filePath := path.Join(dir, "test.txt")
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("写入测试文件失败: %+v", err)
	}
	return filePath
}

// chdir 切换工作目录并在用例结束后自动切回，用于隔离产生相对路径文件的用例
func chdir(t *testing.T, dir string) {
	t.Helper()
	old, err := os.Getwd()
	if err != nil {
		t.Fatalf("获取工作目录失败: %+v", err)
	}
	if err = os.Chdir(dir); err != nil {
		t.Fatalf("切换工作目录失败: %+v", err)
	}
	t.Cleanup(func() { os.Chdir(old) })
}

// ctxWithTimeoutMs 生成指定毫秒后自动取消的ctx
func ctxWithTimeoutMs(ctx context.Context, ms int) (context.Context, func()) {
	return context.WithTimeout(ctx, time.Duration(ms)*time.Millisecond)
}

// ErrTestSentinel 测试用哨兵错误，用于验证错误是否被原样透传
var ErrTestSentinel = errors.Errorf("测试哨兵错误")

// Init 须可重复调用而不panic（多个包的init可能重复触发）
func TestInit(t *testing.T) {
	Init("go_common_retest")
	if GetServerName() != "go_common_retest" {
		t.Errorf("Init 未设置 serverName, got %q", GetServerName())
	}
	//恢复其他用例依赖的原值
	Init("go_common")
	if GetServerName() != "go_common" {
		t.Errorf("恢复 serverName 失败, got %q", GetServerName())
	}
}

// 包级init须完成各子系统初始化，否则后续函数会空指针
func TestPackageInitialized(t *testing.T) {
	//initRegexp
	if numRegexp == nil {
		t.Errorf("numRegexp 未初始化")
	}
	if !ContainNum("1") {
		t.Errorf("正则功能异常")
	}
	//initHttp：http客户端应已就绪
	ctx := GenCtx()
	if NewHttpClientReq(ctx) == nil {
		t.Errorf("NewHttpClientReq 返回 nil，initHttp 未生效")
	}
}

package util

import (
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestInitOsAndGetServerName(t *testing.T) {
	old := defaultServerName
	t.Cleanup(func() { defaultServerName = old })

	InitOs("mysvc")
	if got := GetServerName(); got != "mysvc" {
		t.Errorf("GetServerName = %q, 期望 mysvc", got)
	}
	//环境变量优先级高于 InitOs 设置的默认值
	t.Setenv(serverNameKey, "env-svc")
	if got := GetServerName(); got != "env-svc" {
		t.Errorf("环境变量应优先, got %q", got)
	}
	//环境变量为空时回落到默认值
	t.Setenv(serverNameKey, "")
	if got := GetServerName(); got != "mysvc" {
		t.Errorf("空环境变量应回落默认值, got %q", got)
	}
}

func TestGetEnv(t *testing.T) {
	//未设置时为空
	if got := GetEnv("GO_COMMON_NOT_EXIST_KEY"); got != "" {
		t.Errorf("不存在的环境变量 = %q", got)
	}
	t.Setenv("GO_COMMON_TEST_KEY", "val")
	if got := GetEnv("GO_COMMON_TEST_KEY"); got != "val" {
		t.Errorf("GetEnv = %q", got)
	}
}

func TestGetEnvString(t *testing.T) {
	//不存在时用默认值
	if got := GetEnvString("GO_COMMON_NOT_EXIST_KEY", "def"); got != "def" {
		t.Errorf("GetEnvString = %q, 期望 def", got)
	}
	t.Setenv("GO_COMMON_STR", "actual")
	if got := GetEnvString("GO_COMMON_STR", "def"); got != "actual" {
		t.Errorf("GetEnvString = %q, 期望 actual", got)
	}
	//空字符串视为未设置，回落默认值
	t.Setenv("GO_COMMON_STR", "")
	if got := GetEnvString("GO_COMMON_STR", "def"); got != "def" {
		t.Errorf("空值应回落默认值, got %q", got)
	}
}

func TestGetEnvInt(t *testing.T) {
	//不存在时用默认值
	if got := GetEnvInt("GO_COMMON_NOT_EXIST_KEY", 42); got != 42 {
		t.Errorf("GetEnvInt = %d, 期望 42", got)
	}
	t.Setenv("GO_COMMON_INT", "100")
	if got := GetEnvInt("GO_COMMON_INT", 42); got != 100 {
		t.Errorf("GetEnvInt = %d, 期望 100", got)
	}
	//负数
	t.Setenv("GO_COMMON_INT", "-7")
	if got := GetEnvInt("GO_COMMON_INT", 42); got != -7 {
		t.Errorf("GetEnvInt(负数) = %d", got)
	}
	//非法值必须回落默认值而非0
	for _, bad := range []string{"", "abc", "1.5", "1e3", " 1 "} {
		t.Setenv("GO_COMMON_INT", bad)
		if got := GetEnvInt("GO_COMMON_INT", 42); got != 42 {
			t.Errorf("非法值 %q 应回落默认值, got %d", bad, got)
		}
	}
	//不同整型宽度
	t.Setenv("GO_COMMON_INT", "300")
	if got := GetEnvInt[int64]("GO_COMMON_INT", 0); got != 300 {
		t.Errorf("int64 = %d", got)
	}
	//已知行为：超出目标类型范围会被静默截断（int8 装 300 得 44），
	//调用方需自行保证取值范围
	if got := GetEnvInt[int8]("GO_COMMON_INT", 0); got != 44 {
		t.Errorf("int8 截断行为变化: got %d, 期望 44(300截断)", got)
	}

	//64位边界值必须精确解析，不得被截断
	t.Setenv("GO_COMMON_INT", "9223372036854775807")
	if got := GetEnvInt[int64]("GO_COMMON_INT", 0); got != math.MaxInt64 {
		t.Errorf("MaxInt64 = %d, 期望 %d", got, int64(math.MaxInt64))
	}
	t.Setenv("GO_COMMON_INT", "-9223372036854775808")
	if got := GetEnvInt[int64]("GO_COMMON_INT", 0); got != math.MinInt64 {
		t.Errorf("MinInt64 = %d", got)
	}
	//本库18位ID量级必须精确
	t.Setenv("GO_COMMON_INT", "260904172648391503")
	if got := GetEnvInt[int64]("GO_COMMON_INT", 0); got != 260904172648391503 {
		t.Errorf("18位ID = %d", got)
	}

	//关键差异：超出int64范围时 GetEnvInt 检查了error，故回落默认值；
	//而 String2Int 丢弃error，沿用ParseInt的钳制值(MaxInt64)。
	//两者语义不同是有意的，这里同时锁定，避免日后被"统一"成同一种而破坏调用方预期。
	t.Setenv("GO_COMMON_INT", "99999999999999999999")
	if got := GetEnvInt[int64]("GO_COMMON_INT", 42); got != 42 {
		t.Errorf("溢出值 = %d, 期望回落默认值 42（不应钳制为MaxInt64）", got)
	}
	if got := String2Int[int64]("99999999999999999999"); got != math.MaxInt64 {
		t.Errorf("String2Int 溢出应钳制为 MaxInt64, got %d", got)
	}

	//前导零与正号
	t.Setenv("GO_COMMON_INT", "007")
	if got := GetEnvInt("GO_COMMON_INT", 42); got != 7 {
		t.Errorf(`GetEnvInt("007") = %d, 期望 7`, got)
	}
	t.Setenv("GO_COMMON_INT", "+7")
	if got := GetEnvInt("GO_COMMON_INT", 42); got != 7 {
		t.Errorf(`GetEnvInt("+7") = %d, 期望 7`, got)
	}
}

func TestGetEnvFloat(t *testing.T) {
	if got := GetEnvFloat("GO_COMMON_NOT_EXIST_KEY", 1.5); got != 1.5 {
		t.Errorf("GetEnvFloat = %v, 期望 1.5", got)
	}
	t.Setenv("GO_COMMON_FLOAT", "2.75")
	if got := GetEnvFloat("GO_COMMON_FLOAT", 1.5); got != 2.75 {
		t.Errorf("GetEnvFloat = %v, 期望 2.75", got)
	}
	//整数形式也应可解析为浮点
	t.Setenv("GO_COMMON_FLOAT", "3")
	if got := GetEnvFloat("GO_COMMON_FLOAT", 1.5); got != 3 {
		t.Errorf("GetEnvFloat(整数) = %v", got)
	}
	//科学计数法
	t.Setenv("GO_COMMON_FLOAT", "1e2")
	if got := GetEnvFloat("GO_COMMON_FLOAT", 1.5); got != 100 {
		t.Errorf("GetEnvFloat(科学计数) = %v", got)
	}
	//非法值回落默认值
	for _, bad := range []string{"", "abc", "1.2.3"} {
		t.Setenv("GO_COMMON_FLOAT", bad)
		if got := GetEnvFloat("GO_COMMON_FLOAT", 1.5); got != 1.5 {
			t.Errorf("非法值 %q 应回落默认值, got %v", bad, got)
		}
	}
	//float32
	t.Setenv("GO_COMMON_FLOAT", "0.5")
	if got := GetEnvFloat[float32]("GO_COMMON_FLOAT", 0); got != 0.5 {
		t.Errorf("float32 = %v", got)
	}
}

func TestGetEnvBool(t *testing.T) {
	//不存在时用默认值，两个方向都要验证
	if !GetEnvBool("GO_COMMON_NOT_EXIST_KEY", true) {
		t.Errorf("默认值true未生效")
	}
	if GetEnvBool("GO_COMMON_NOT_EXIST_KEY", false) {
		t.Errorf("默认值false未生效")
	}
	//大小写与空格都应被容忍
	for _, s := range []string{"true", "TRUE", "True", " true ", "\ttrue\n"} {
		t.Setenv("GO_COMMON_BOOL", s)
		if !GetEnvBool("GO_COMMON_BOOL", false) {
			t.Errorf("%q 应解析为 true", s)
		}
	}
	for _, s := range []string{"false", "FALSE", "False", " false "} {
		t.Setenv("GO_COMMON_BOOL", s)
		if GetEnvBool("GO_COMMON_BOOL", true) {
			t.Errorf("%q 应解析为 false", s)
		}
	}
	//已知边界：不支持 1/0/yes/no，会回落默认值
	for _, s := range []string{"1", "0", "yes", "no", "on", "off", "abc", ""} {
		t.Setenv("GO_COMMON_BOOL", s)
		if !GetEnvBool("GO_COMMON_BOOL", true) {
			t.Errorf("%q 非 true/false，应回落默认值true", s)
		}
		if GetEnvBool("GO_COMMON_BOOL", false) {
			t.Errorf("%q 非 true/false，应回落默认值false", s)
		}
	}
}

func TestGetHome(t *testing.T) {
	home := GetHome()
	if home == "" {
		t.Fatalf("GetHome 返回空")
	}
	//应为存在的绝对路径目录
	if !filepath.IsAbs(home) {
		t.Errorf("GetHome 非绝对路径: %q", home)
	}
	info, err := os.Stat(home)
	if err != nil {
		t.Errorf("HOME 目录不存在: %+v", err)
	} else if !info.IsDir() {
		t.Errorf("HOME 不是目录: %q", home)
	}
}

func TestGetExecFileAndFolder(t *testing.T) {
	file := GetExecFile()
	if file == "" {
		t.Fatalf("GetExecFile 返回空")
	}
	if !filepath.IsAbs(file) {
		t.Errorf("GetExecFile 非绝对路径: %q", file)
	}
	folder := GetExecFolder()
	if folder == "" {
		t.Fatalf("GetExecFolder 返回空")
	}
	//二者须自洽：folder 应为 file 的目录
	if want := filepath.Dir(file); folder != want {
		t.Errorf("GetExecFolder = %q, 期望 %q", folder, want)
	}
	if info, err := os.Stat(folder); err != nil || !info.IsDir() {
		t.Errorf("执行目录异常: %v %v", info, err)
	}
}

// Defer 是全局的panic兜底工具，必须真正捕获panic并给出堆栈
func TestDefer(t *testing.T) {
	//有panic时：err非nil且带堆栈
	var gotErr interface{}
	var gotStack string
	func() {
		defer Defer(func(err interface{}, stack string) {
			gotErr = err
			gotStack = stack
		})
		panic("测试panic")
	}()
	if gotErr == nil {
		t.Errorf("Defer 未捕获 panic")
	}
	if gotErr != "测试panic" {
		t.Errorf("捕获的panic值 = %v", gotErr)
	}
	if gotStack == "" {
		t.Errorf("panic 时应提供堆栈")
	}
	if !strings.Contains(gotStack, "TestDefer") {
		t.Errorf("堆栈未包含调用点: %s", gotStack)
	}

	//无panic时：err为nil且堆栈为空，回调仍必须被调用
	called := false
	gotStack = "未重置"
	func() {
		defer Defer(func(err interface{}, stack string) {
			called = true
			gotErr = err
			gotStack = stack
		})
	}()
	if !called {
		t.Errorf("无panic时回调未被调用")
	}
	if gotErr != nil {
		t.Errorf("无panic时 err = %v, 期望 nil", gotErr)
	}
	if gotStack != "" {
		t.Errorf("无panic时堆栈应为空, got %q", gotStack)
	}

	//panic 传入 error 类型时也应被捕获
	func() {
		defer Defer(func(err interface{}, stack string) { gotErr = err })
		panic(os.ErrNotExist)
	}()
	if gotErr != os.ErrNotExist {
		t.Errorf("error类型panic捕获 = %v", gotErr)
	}
}

func TestExecCommand(t *testing.T) {
	ctx := GenCtx()
	//标准输出
	stdout, stderr, err := ExecCommand(ctx, "echo hello")
	if err != nil {
		t.Fatalf("ExecCommand 异常: %+v", err)
	}
	if len(stdout) != 1 || stdout[0] != "hello" {
		t.Errorf("stdout = %v, 期望 [hello]", stdout)
	}
	if len(stderr) != 0 {
		t.Errorf("stderr = %v, 期望空", stderr)
	}

	//多行输出须按行切分
	stdout, _, err = ExecCommand(ctx, "printf 'a\\nb\\nc\\n'")
	if err != nil {
		t.Fatalf("%+v", err)
	}
	if len(stdout) != 3 || stdout[0] != "a" || stdout[2] != "c" {
		t.Errorf("多行输出 = %v, 期望 [a b c]", stdout)
	}

	//末行无换行也不能丢（曾因只在遇到\n时才收集而丢弃末行）
	stdout, _, err = ExecCommand(ctx, "printf 'no-newline-end'")
	if err != nil {
		t.Fatalf("%+v", err)
	}
	if len(stdout) != 1 || stdout[0] != "no-newline-end" {
		t.Errorf("末行无换行的输出 = %v, 期望 [no-newline-end]", stdout)
	}

	//错误输出须进stderr而非stdout
	stdout, stderr, err = ExecCommand(ctx, "echo oops >&2")
	if err != nil {
		t.Fatalf("%+v", err)
	}
	if len(stderr) != 1 || stderr[0] != "oops" {
		t.Errorf("stderr = %v, 期望 [oops]", stderr)
	}
	if len(stdout) != 0 {
		t.Errorf("stdout = %v, 期望空", stdout)
	}

	//非零退出码必须返回error，且已产生的输出仍要带回
	stdout, _, err = ExecCommand(ctx, "echo before; exit 3")
	if err == nil {
		t.Errorf("非零退出码应返回error")
	}
	if len(stdout) != 1 || stdout[0] != "before" {
		t.Errorf("失败命令的已有输出应保留, got %v", stdout)
	}

	//空命令：约定返回全nil且不报错
	stdout, stderr, err = ExecCommand(ctx, "")
	if err != nil || stdout != nil || stderr != nil {
		t.Errorf("空命令 = %v %v %v, 期望全nil", stdout, stderr, err)
	}
	if stdout, stderr, err = ExecCommand(ctx, "   "); err != nil || stdout != nil || stderr != nil {
		t.Errorf("空白命令 = %v %v %v, 期望全nil", stdout, stderr, err)
	}

	//不存在的命令须报错
	if _, _, err = ExecCommand(ctx, "this_command_does_not_exist_12345"); err == nil {
		t.Errorf("不存在的命令应返回error")
	}
}

// ctx 取消须能终止命令，避免goroutine泄漏
func TestExecCommandCtxCancel(t *testing.T) {
	ctx := GenCtx()
	ctx, cancel := ctxWithTimeoutMs(ctx, 300)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		_, _, err := ExecCommand(ctx, "sleep 10")
		done <- err
	}()
	select {
	case err := <-done:
		if err == nil {
			t.Errorf("被取消的命令应返回error")
		}
	case <-timeAfterMs(5000):
		t.Errorf("ctx 取消后命令未被终止，可能存在goroutine泄漏")
	}
}

// ExitSignal 注册信号回调后会调用 os.Exit(0)，无法在本进程内直接断言；
// 故用子进程模式：子进程注册回调后自己发SIGTERM，父进程校验
// ①回调确实被触发（通过子进程输出）②进程以0退出（而非被信号杀死）
func TestExitSignal(t *testing.T) {
	if os.Getenv("GO_COMMON_TEST_EXIT_SIGNAL") == "1" {
		//子进程分支
		got := make(chan os.Signal, 1)
		ExitSignal(func(signal os.Signal) {
			//回调须拿到真实信号对象
			fmt.Printf("SIGNAL_RECEIVED:%v\n", signal)
			os.Stdout.Sync()
			got <- signal
		})
		//给 signal.Notify 一点注册时间，再给自己发信号
		time.Sleep(200 * time.Millisecond)
		proc, err := os.FindProcess(os.Getpid())
		if err != nil {
			fmt.Printf("FIND_PROCESS_ERR:%v\n", err)
			os.Exit(9)
		}
		if err = proc.Signal(syscall.SIGTERM); err != nil {
			fmt.Printf("SIGNAL_ERR:%v\n", err)
			os.Exit(9)
		}
		//等 ExitSignal 内部的 os.Exit(0) 生效；若超时说明回调未被触发
		select {
		case <-got:
			time.Sleep(2 * time.Second)
			//正常情况下不会走到这里：os.Exit(0) 应已退出进程
			fmt.Println("EXIT_NOT_CALLED")
			os.Exit(8)
		case <-time.After(5 * time.Second):
			fmt.Println("CALLBACK_TIMEOUT")
			os.Exit(7)
		}
	}

	cmd := exec.Command(os.Args[0], "-test.run=^TestExitSignal$", "-test.v")
	cmd.Env = append(os.Environ(), "GO_COMMON_TEST_EXIT_SIGNAL=1")
	out, err := cmd.CombinedOutput()
	output := string(out)

	//SIGTERM 被 ExitSignal 接管后须走 os.Exit(0)，因此父进程看到的是正常退出
	if err != nil {
		t.Errorf("子进程未以0退出: %v\n输出:\n%s", err, output)
	}
	//回调必须真正被调用，且拿到的是 SIGTERM
	if !strings.Contains(output, "SIGNAL_RECEIVED:terminated") {
		t.Errorf("回调未收到SIGTERM，输出:\n%s", output)
	}
	//不应命中兜底分支
	for _, bad := range []string{"CALLBACK_TIMEOUT", "EXIT_NOT_CALLED", "SIGNAL_ERR", "FIND_PROCESS_ERR"} {
		if strings.Contains(output, bad) {
			t.Errorf("子进程命中异常分支 %s，输出:\n%s", bad, output)
		}
	}
}

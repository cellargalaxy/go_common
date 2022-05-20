package util

import (
	"bufio"
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"syscall"
)

const serverNameEnvKey = "server_name"

func Defer(ctx context.Context, callback func(ctx context.Context, err interface{}, stack string)) {
	err := recover()
	var buf [1024 * 4]byte
	n := runtime.Stack(buf[:], false)
	stack := string(buf[:n])
	callback(ctx, err, stack)
}

func GetServerName(defaultServerName string) string {
	return GetEnvString(serverNameEnvKey, defaultServerName)
}

func GetEnv(key string) string {
	return os.Getenv(key)
}

func GetEnvString(key, defaultValue string) string {
	value := GetEnv(key)
	if value == "" {
		value = defaultValue
	}
	return value
}

func GetEnvInt(key string, defaultValue int) int {
	value := GetEnv(key)
	data, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return data
}

func GetEnvFloat64(key string, defaultValue float64) float64 {
	value := GetEnv(key)
	data, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return defaultValue
	}
	return data
}

func GetEnvBool(key string, defaultValue bool) bool {
	value := GetEnv(key)
	value = strings.ToLower(value)
	switch value {
	case "true":
		return true
	case "false":
		return false
	default:
		return defaultValue
	}
}

/**
https://mojotv.cn/2019/01/17/golang-signal-restart-deamom
https://bytedance.feishu.cn/wiki/wikcnaJLXgEn5xeJWF7VSUY0qNg#xbNNDs
*/
func ExitSignal(fun func(ctx context.Context, signal os.Signal)) {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGKILL)
	go func() {
		s := <-signalChan
		ctx := CreateLogCtx()
		fun(ctx, s)
		os.Exit(0)
	}()
}

func ExecCommand(ctx context.Context, commands []string) ([]string, []string, error) {
	command := strings.Join(commands, " && ")

	cmd := exec.CommandContext(ctx, "bash", "-c", command)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		logrus.WithFields(logrus.Fields{"err": err}).Error("执行命令，异常")
		return nil, nil, fmt.Errorf("执行命令，异常: %+v", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		logrus.WithFields(logrus.Fields{"err": err}).Error("执行命令，异常")
		return nil, nil, fmt.Errorf("执行命令，异常: %+v", err)
	}
	err = cmd.Start()
	if err != nil {
		logrus.WithFields(logrus.Fields{"err": err}).Error("执行命令，异常")
		return nil, nil, fmt.Errorf("执行命令，异常: %+v", err)
	}

	stdoutReader := bufio.NewReader(stdout)
	stderrReader := bufio.NewReader(stderr)
	var stdoutLines, stderrLines []string
	go func() {
		defer Defer(ctx, func(ctx context.Context, err interface{}, stack string) {
			if err != nil {
				logrus.WithFields(logrus.Fields{"err": err}).Error("执行命令，异常")
				return
			}
		})

		stdoutLines, _ = Read2LogByReader(ctx, true, stdoutReader)
	}()
	go func() {
		defer Defer(ctx, func(ctx context.Context, err interface{}, stack string) {
			if err != nil {
				logrus.WithFields(logrus.Fields{"err": err}).Error("执行命令，异常")
				return
			}
		})

		stderrLines, _ = Read2LogByReader(ctx, true, stderrReader)
	}()

	err = cmd.Wait()
	if err != nil {
		logrus.WithFields(logrus.Fields{"err": err}).Error("执行命令，异常")
		return stdoutLines, stderrLines, fmt.Errorf("执行命令，异常: %+v", err)
	}

	return stdoutLines, stderrLines, nil
}

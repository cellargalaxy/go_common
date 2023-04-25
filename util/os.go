package util

import (
	"bufio"
	"context"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/exp/constraints"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"syscall"
)

const serverNameKey = "server_name"

var defaultServerName string

func InitOs(serverName string) {
	defaultServerName = serverName
}

func GetServerName() string {
	return GetEnvString(serverNameKey, defaultServerName)
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
func GetEnvInt[T constraints.Integer](key string, defaultValue T) T {
	value := GetEnv(key)
	data, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return T(data)
}
func GetEnvFloat[T constraints.Float](key string, defaultValue T) T {
	value := GetEnv(key)
	data, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return defaultValue
	}
	return T(data)
}
func GetEnvBool(key string, defaultValue bool) bool {
	value := GetEnv(key)
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "true":
		return true
	case "false":
		return false
	default:
		return defaultValue
	}
}

func Defer(callback func(err interface{}, stack string)) {
	err := recover()
	var stack string
	if err != nil {
		var buf [1024]byte
		n := runtime.Stack(buf[:], false)
		stack = string(buf[:n])
	}
	callback(err, stack)
}

/*
*
https://mojotv.cn/2019/01/17/golang-signal-restart-deamom
https://bytedance.feishu.cn/wiki/wikcnaJLXgEn5xeJWF7VSUY0qNg#xbNNDs
*/
func ExitSignal(fun func(signal os.Signal)) {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGKILL)
	go func() {
		fun(<-signalChan)
		os.Exit(0)
	}()
}

func ExecCommand(ctx context.Context, commands []string) ([]string, []string, error) {
	command := strings.Join(commands, " && ")

	cmd := exec.CommandContext(ctx, "bash", "-c", command)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Error("执行命令，异常")
		return nil, nil, errors.Errorf("执行命令，异常: %+v", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Error("执行命令，异常")
		return nil, nil, errors.Errorf("执行命令，异常: %+v", err)
	}
	err = cmd.Start()
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Error("执行命令，异常")
		return nil, nil, errors.Errorf("执行命令，异常: %+v", err)
	}

	stdoutReader := bufio.NewReader(stdout)
	stderrReader := bufio.NewReader(stderr)
	var stdoutLines, stderrLines []string
	go func() {
		defer Defer(func(err interface{}, stack string) {
			if err != nil {
				logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Error("执行命令，异常")
			}
		})

		stdoutLines, _ = Read2LogByReader(ctx, stdoutReader, true)
	}()
	go func() {
		defer Defer(func(err interface{}, stack string) {
			if err != nil {
				logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Error("执行命令，异常")
			}
		})

		stderrLines, _ = Read2LogByReader(ctx, stderrReader, true)
	}()

	err = cmd.Wait()
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Error("执行命令，异常")
		return stdoutLines, stderrLines, errors.Errorf("执行命令，异常: %+v", err)
	}

	return stdoutLines, stderrLines, nil
}

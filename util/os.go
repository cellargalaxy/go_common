package util

import (
	"bufio"
	"context"
	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/exp/constraints"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
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

func GetHome() string {
	home, err := homedir.Dir()
	if err != nil {
		logrus.WithFields(logrus.Fields{"err": errors.WithStack(err)}).Error("获取HOME，异常")
	}
	return home
}
func GetExecFile() string {
	execPath, err := os.Executable()
	if err != nil {
		logrus.WithFields(logrus.Fields{"err": errors.WithStack(err)}).Error("获取执行文件路径，异常")
	}
	return execPath
}
func GetExecFolder() string {
	execPath, err := os.Executable()
	if err != nil {
		logrus.WithFields(logrus.Fields{"err": errors.WithStack(err)}).Error("获取执行文件路径，异常")
	}
	return filepath.Dir(execPath)
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

func ExecCommand(ctx context.Context, command string) ([]string, []string, error) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		commands := strings.Split(command, " ")
		list := make([]string, 0, len(commands))
		for i := range commands {
			commands[i] = strings.TrimSpace(commands[i])
			if commands[i] == "" {
				continue
			}
			list = append(list, commands[i])
		}
		var name string
		var arg []string
		if len(list) > 0 {
			name = list[0]
		}
		if len(list) > 1 {
			arg = list[1:]
		}
		if name == "" {
			logrus.WithContext(ctx).WithFields(logrus.Fields{}).Warn("执行命令，命令为空")
			return nil, nil, nil
		}
		cmd = exec.CommandContext(ctx, name, arg...)
	default:
		command = strings.TrimSpace(command)
		if command == "" {
			logrus.WithContext(ctx).WithFields(logrus.Fields{}).Warn("执行命令，命令为空")
			return nil, nil, nil
		}
		cmd = exec.CommandContext(ctx, "bash", "-c", command)
	}
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
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer Defer(func(err interface{}, stack string) {
			wg.Done()
			if err != nil {
				logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Error("执行命令，异常")
			}
		})

		stdoutLines, _ = Read2LogByReader(ctx, stdoutReader, true)
	}()
	wg.Add(1)
	go func() {
		defer Defer(func(err interface{}, stack string) {
			wg.Done()
			if err != nil {
				logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Error("执行命令，异常")
			}
		})

		stderrLines, _ = Read2LogByReader(ctx, stderrReader, true)
	}()

	wg.Wait()
	err = cmd.Wait()
	if err != nil {
		logrus.WithContext(ctx).WithFields(logrus.Fields{"err": err}).Error("执行命令，异常")
		return stdoutLines, stderrLines, errors.Errorf("执行命令，异常: %+v", err)
	}

	return stdoutLines, stderrLines, nil
}

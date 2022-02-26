package util

import (
	"context"
	"os"
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

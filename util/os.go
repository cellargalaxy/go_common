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

func Defer(callback func(ctx context.Context, err interface{}, stack string)) {
	ctx := CreateLogCtx()
	err := recover()
	var buf [1024 * 4]byte
	n := runtime.Stack(buf[:], false)
	stack := string(buf[:n])
	callback(ctx, err, stack)
}

func GetServerNameWithPanic() string {
	value := GetServerName()
	if value == "" {
		panic("server_name为空")
	}
	return value
}

func GetServerName() string {
	return GetEnvString(serverNameEnvKey, "")
}

func GetEnvString(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		value = defaultValue
	}
	return value
}

func GetEnvInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	data, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return data
}

func GetEnvFloat64(key string, defaultValue float64) float64 {
	value := os.Getenv(key)
	data, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return defaultValue
	}
	return data
}

func GetEnvBool(key string, defaultValue bool) bool {
	value := os.Getenv(key)
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

package util

import "github.com/sirupsen/logrus"

func Init(serverName string) {
	initOs(serverName)
	initLog(GetServerName(), "", 1, 100, 30, logrus.InfoLevel)
	initRegexp()
	initHttp()
}

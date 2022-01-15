package util

import (
	"regexp"
)

var numRegexp *regexp.Regexp

func initRegexp() {
	var err error
	numRegexp, err = regexp.Compile("\\d+([.]\\d+)?")
	if err != nil {
		panic(err)
	}
}

func ContainNum(s string) bool {
	return numRegexp.MatchString(s)
}

func FindNum(s string) string {
	return numRegexp.FindString(s)
}

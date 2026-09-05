package util

import (
	"testing"
)

// 正则 \d+([.]\d+)? ：不含负号，也不匹配非ASCII数字
func TestContainNum(t *testing.T) {
	yes := []string{"1", "123", "12.5", "abc7", "-12.5", ".5", "12.", "a1b"}
	for _, s := range yes {
		if !ContainNum(s) {
			t.Errorf("ContainNum(%q) = false, 期望 true", s)
		}
	}
	no := []string{"", "abc", "中文", "!@#", "١٢٣"}
	for _, s := range no {
		if ContainNum(s) {
			t.Errorf("ContainNum(%q) = true, 期望 false", s)
		}
	}
}

func TestFindNum(t *testing.T) {
	cases := map[string]string{
		"123":     "123",
		"12.5":    "12.5",
		"abc123":  "123",
		"a1b2":    "1",
		"12.":     "12",
		".5":      "5",
		"":        "",
		"abc":     "",
		"中文":      "",
		"1.2.3":   "1.2",
		"价格99.9元": "99.9",
		//已知边界：正则不含负号，负号会被丢弃
		"-12.5": "12.5",
	}
	for in, want := range cases {
		if got := FindNum(in); got != want {
			t.Errorf("FindNum(%q) = %q, 期望 %q", in, got, want)
		}
	}
	//只返回首个匹配
	if got := FindNum("1 and 2"); got != "1" {
		t.Errorf("FindNum 应只返回首个匹配, got %q", got)
	}
}

// initRegexp 由包init调用，须保证正则已就绪，否则ContainNum会空指针
func TestRegexpInitialized(t *testing.T) {
	if numRegexp == nil {
		t.Fatalf("numRegexp 未初始化，initRegexp 未被调用")
	}
	//重复调用应幂等，不panic
	initRegexp()
	if !ContainNum("1") {
		t.Errorf("重复 initRegexp 后功能异常")
	}
}

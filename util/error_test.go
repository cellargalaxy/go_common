package util

import (
	"testing"

	"github.com/pkg/errors"
)

// HttpBan 是包级哨兵错误，被 http 逻辑用于判定封禁，需保证可用 errors.Is 比较
func TestHttpBanError(t *testing.T) {
	if HttpBan == nil {
		t.Fatalf("HttpBan 为 nil")
	}
	if HttpBan.Error() == "" {
		t.Errorf("HttpBan 错误信息为空")
	}
	//同一实例必须可被识别（哨兵错误的核心用途）
	if !errors.Is(HttpBan, HttpBan) {
		t.Errorf("HttpBan 无法与自身比较")
	}
	//包装后仍可被 errors.Is 识别，否则调用方判断封禁会失效
	wrapped := errors.WithMessage(HttpBan, "上层附加信息")
	if !errors.Is(wrapped, HttpBan) {
		t.Errorf("包装后的 HttpBan 无法被 errors.Is 识别")
	}
	//与其他错误必须可区分
	if errors.Is(errors.Errorf("其他错误"), HttpBan) {
		t.Errorf("无关错误被误判为 HttpBan")
	}
	//同文案的另一个错误实例不能被误判为 HttpBan：
	//哨兵错误靠实例身份而非文案匹配，这里同时锁死"文案相同也不等价"的语义。
	//（原用例写的是 HttpBan != HttpBan，恒为false，不构成任何校验）
	sameText := errors.Errorf("HTTP请求封禁")
	if errors.Is(sameText, HttpBan) {
		t.Errorf("同文案的其他错误被误判为 HttpBan")
	}
	if sameText == HttpBan {
		t.Errorf("同文案的其他错误与 HttpBan 实例相同")
	}
	//包装链上取根因仍应回到同一实例
	if errors.Cause(errors.WithMessage(HttpBan, "外层")) != HttpBan {
		t.Errorf("errors.Cause 未还原为 HttpBan 实例")
	}
}

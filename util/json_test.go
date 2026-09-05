package util

import (
	"math"
	"strings"
	"testing"
)

type jsonDemo struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
	Skip string `json:"-"`
	Omit string `json:"omit,omitempty"`
}

func TestJsonRoundTrip(t *testing.T) {
	list := []jsonDemo{{Id: 1, Name: "a"}, {Id: 2, Name: "b"}}
	text := JsonStruct2String(list)
	var got []jsonDemo
	if err := JsonString2Struct(text, &got); err != nil {
		t.Fatalf("反序列化异常: %+v", err)
	}
	if len(got) != 2 || got[0].Id != 1 || got[0].Name != "a" || got[1].Id != 2 || got[1].Name != "b" {
		t.Errorf("往返结果 = %+v", got)
	}
}

// 校验实际序列化文本，而非仅"能转回来"
func TestJsonStruct2String(t *testing.T) {
	got := JsonStruct2String(jsonDemo{Id: 1, Name: "n", Skip: "x"})
	//json标签必须生效
	if !strings.Contains(got, `"id":1`) || !strings.Contains(got, `"name":"n"`) {
		t.Errorf("json标签未生效: %s", got)
	}
	//`json:"-"` 字段必须被忽略
	if strings.Contains(got, "Skip") || strings.Contains(got, `"x"`) {
		t.Errorf(`json:"-" 字段未被忽略: %s`, got)
	}
	//omitempty 空值时不应出现
	if strings.Contains(got, "omit") {
		t.Errorf("omitempty 空值仍被输出: %s", got)
	}
	//基础类型
	cases := map[interface{}]string{
		nil: "null", 1: "1", "s": `"s"`, true: "true", 1.5: "1.5",
	}
	for in, want := range cases {
		if got := JsonStruct2String(in); got != want {
			t.Errorf("JsonStruct2String(%v) = %q, 期望 %q", in, got, want)
		}
	}
	//切片与map
	if got := JsonStruct2String([]int{1, 2}); got != "[1,2]" {
		t.Errorf("切片 = %q", got)
	}
	if got := JsonStruct2String(map[string]int{"a": 1}); got != `{"a":1}` {
		t.Errorf("map = %q", got)
	}
	//中文不应被转义成\u
	if got := JsonStruct2String(map[string]string{"k": "中文"}); !strings.Contains(got, "中文") {
		t.Errorf("中文被转义: %s", got)
	}
	//Data 版本与 String 版本必须一致
	if string(JsonStruct2Data(list1())) != JsonStruct2String(list1()) {
		t.Errorf("Data 与 String 版本不一致")
	}
}

func list1() []jsonDemo { return []jsonDemo{{Id: 1, Name: "a"}} }

// 缩进版本必须真的带缩进和换行
func TestJsonIndent(t *testing.T) {
	got := JsonStruct2StringIndent(jsonDemo{Id: 1, Name: "a"})
	if !strings.Contains(got, "\n") {
		t.Errorf("缩进版本无换行: %q", got)
	}
	if !strings.Contains(got, "  ") {
		t.Errorf("缩进版本无两空格缩进: %q", got)
	}
	//缩进版与紧凑版语义应等价
	var a, b jsonDemo
	if err := JsonString2Struct(got, &a); err != nil {
		t.Fatalf("%+v", err)
	}
	if err := JsonString2Struct(JsonStruct2String(jsonDemo{Id: 1, Name: "a"}), &b); err != nil {
		t.Fatalf("%+v", err)
	}
	if a != b {
		t.Errorf("缩进版与紧凑版语义不一致: %+v vs %+v", a, b)
	}
	//Data 与 String 版本一致
	if string(JsonStruct2DataIndent(jsonDemo{Id: 1})) != JsonStruct2StringIndent(jsonDemo{Id: 1}) {
		t.Errorf("Indent 的 Data 与 String 版本不一致")
	}
}

func TestJsonError(t *testing.T) {
	//非法json必须返回error
	var got jsonDemo
	for _, bad := range []string{"", "{", "not json", `{"id":"字符串给了int字段"}`} {
		if err := JsonString2Struct(bad, &got); err == nil {
			t.Errorf("JsonString2Struct(%q) 应返回error", bad)
		}
	}
	//Data 版本同样报错
	if err := JsonData2Struct([]byte("{{"), &got); err == nil {
		t.Errorf("JsonData2Struct 非法数据应返回error")
	}
	//不可序列化的类型：返回空并记录日志，不panic
	if got := JsonStruct2String(func() {}); got != "" {
		t.Errorf("不可序列化类型 = %q, 期望空串", got)
	}
	if got := JsonStruct2String(math.NaN()); got != "" {
		t.Errorf("NaN = %q, 期望空串", got)
	}
	//传入非指针时应报错而不是静默成功
	if err := JsonString2Struct(`{"id":1}`, got); err == nil {
		t.Errorf("传入非指针应返回error")
	}
}

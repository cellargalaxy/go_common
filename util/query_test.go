package util

import (
	"strings"
	"testing"
)

type queryDemo struct {
	Id   int    `query:"id"`
	Name string `query:"name"`
}

func TestQueryRoundTrip(t *testing.T) {
	ctx := GenCtx()
	list := []queryDemo{{Id: 1, Name: "a"}, {Id: 2, Name: "b"}}
	text := QueryStruct2String(ctx, list)
	var got []queryDemo
	if err := QueryString2Struct(ctx, text, &got); err != nil {
		t.Fatalf("反序列化异常: %+v", err)
	}
	if len(got) != 2 || got[0].Id != 1 || got[0].Name != "a" || got[1].Id != 2 || got[1].Name != "b" {
		t.Errorf("往返结果 = %+v", got)
	}
}

// 校验生成的query串格式，而非仅"能转回来"
func TestQueryStruct2String(t *testing.T) {
	ctx := GenCtx()
	got := QueryStruct2String(ctx, queryDemo{Id: 7, Name: "abc"})
	//query标签必须生效，且为 k=v& 结构
	if !strings.Contains(got, "id=7") {
		t.Errorf("id 未按标签序列化: %q", got)
	}
	if !strings.Contains(got, "name=abc") {
		t.Errorf("name 未按标签序列化: %q", got)
	}
	if !strings.Contains(got, "&") {
		t.Errorf("多字段间缺少 & 分隔: %q", got)
	}
	//Data 与 String 版本一致
	if string(QueryStruct2Data(ctx, queryDemo{Id: 1})) != QueryStruct2String(ctx, queryDemo{Id: 1}) {
		t.Errorf("Data 与 String 版本不一致")
	}
	//已知行为：零值字段会被整体省略（urlquery 默认忽略空值），
	//意味着服务端无法区分"未传"与"传了零值"，此处锁定现状
	if got := QueryStruct2String(ctx, queryDemo{}); got != "" {
		t.Errorf("零值结构体 = %q, 当前实现应省略全部零值字段", got)
	}
	if got := QueryStruct2String(ctx, queryDemo{Id: 0, Name: "x"}); got != "name=x" {
		t.Errorf("含零值字段 = %q, 期望仅输出非零字段 name=x", got)
	}
}

// 需要转义的字符必须被正确编码并能还原
func TestQueryEscape(t *testing.T) {
	ctx := GenCtx()
	special := queryDemo{Id: 1, Name: "a b&c=d?e#f"}
	text := QueryStruct2String(ctx, special)
	//原始的 & 和 = 必须被转义，否则会破坏query结构
	if strings.Contains(text, "a b") {
		t.Errorf("空格未被转义: %q", text)
	}
	var got queryDemo
	if err := QueryString2Struct(ctx, text, &got); err != nil {
		t.Fatalf("含特殊字符的往返异常: %+v", err)
	}
	if got.Name != special.Name {
		t.Errorf("特殊字符还原失败: %q, 期望 %q", got.Name, special.Name)
	}
	//中文往返
	cn := queryDemo{Id: 2, Name: "中文参数"}
	if err := QueryString2Struct(ctx, QueryStruct2String(ctx, cn), &got); err != nil {
		t.Fatalf("%+v", err)
	}
	if got.Name != "中文参数" {
		t.Errorf("中文还原失败: %q", got.Name)
	}
}

// 直接解析手写query串，验证兼容真实URL参数
func TestQueryParseHandWritten(t *testing.T) {
	ctx := GenCtx()
	var got queryDemo
	if err := QueryString2Struct(ctx, "id=5&name=hello", &got); err != nil {
		t.Fatalf("解析手写query异常: %+v", err)
	}
	if got.Id != 5 || got.Name != "hello" {
		t.Errorf("解析结果 = %+v, 期望 id=5 name=hello", got)
	}
	//缺字段时保持零值
	var partial queryDemo
	if err := QueryString2Struct(ctx, "id=9", &partial); err != nil {
		t.Fatalf("%+v", err)
	}
	if partial.Id != 9 || partial.Name != "" {
		t.Errorf("缺字段解析 = %+v", partial)
	}
	//空串应成功且为零值
	var empty queryDemo
	if err := QueryString2Struct(ctx, "", &empty); err != nil {
		t.Errorf("空串应可解析: %+v", err)
	}
	if empty.Id != 0 || empty.Name != "" {
		t.Errorf("空串应为零值: %+v", empty)
	}
	//未知字段应被忽略而非报错
	var extra queryDemo
	if err := QueryString2Struct(ctx, "id=1&unknown=x", &extra); err != nil {
		t.Errorf("未知字段应被忽略: %+v", err)
	}
	if extra.Id != 1 {
		t.Errorf("未知字段影响了解析: %+v", extra)
	}
}

func TestQueryError(t *testing.T) {
	ctx := GenCtx()
	var got queryDemo
	//类型不匹配必须报错
	if err := QueryString2Struct(ctx, "id=不是数字", &got); err == nil {
		t.Errorf("类型不匹配应返回error")
	}
	//非指针目标必须报错
	if err := QueryString2Struct(ctx, "id=1", got); err == nil {
		t.Errorf("传入非指针应返回error")
	}
	//nil 目标不能panic
	if err := QueryData2Struct(ctx, []byte("id=1"), nil); err == nil {
		t.Errorf("nil 目标应返回error")
	}
}

// 序列化侧的非法入参必须与 JsonStruct2Data、YamlStruct2Data 同一套约定：
// 记日志并返回空，不得panic（urlquery 对nil会panic）。
// 该路径常用于日志字段等旁路逻辑，panic会波及主流程。
func TestQueryStruct2DataIllegalInput(t *testing.T) {
	ctx := GenCtx()

	//nil 入参：不得panic，返回空
	if got := QueryStruct2Data(ctx, nil); len(got) != 0 {
		t.Errorf("QueryStruct2Data(nil) = %q, 期望空", got)
	}
	if got := QueryStruct2String(ctx, nil); got != "" {
		t.Errorf("QueryStruct2String(nil) = %q, 期望空串", got)
	}

	//不可序列化类型：同样只能返回空，不得panic
	for _, bad := range []interface{}{func() {}, make(chan int)} {
		if got := QueryStruct2Data(ctx, bad); len(got) != 0 {
			t.Errorf("QueryStruct2Data(%T) = %q, 期望空", bad, got)
		}
		if got := QueryStruct2String(ctx, bad); got != "" {
			t.Errorf("QueryStruct2String(%T) = %q, 期望空串", bad, got)
		}
	}

	//与 Json/Yaml 的同类入口行为保持一致：都不panic
	if got := JsonStruct2Data(nil); len(got) == 0 {
		t.Logf("JsonStruct2Data(nil) = %q", got)
	}
	if got := YamlStruct2Data(ctx, nil); len(got) == 0 {
		t.Logf("YamlStruct2Data(nil) = %q", got)
	}

	//合法入参不能被兜底误伤
	if got := QueryStruct2String(ctx, queryDemo{Id: 3, Name: "ok"}); got == "" {
		t.Errorf("合法入参被误伤为空")
	}
}

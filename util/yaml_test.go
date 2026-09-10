package util

import (
	"strings"
	"testing"
)

type yamlDemo struct {
	Id   int    `yaml:"id"`
	Name string `yaml:"name"`
	Skip string `yaml:"-"`
}

func TestYamlRoundTrip(t *testing.T) {
	ctx := GenCtx()
	list := []yamlDemo{{Id: 1, Name: "a"}, {Id: 2, Name: "b"}}
	text := YamlStruct2Str(ctx, list)
	var got []yamlDemo
	if err := YamlStr2Struct(ctx, text, &got); err != nil {
		t.Fatalf("反序列化异常: %+v", err)
	}
	if len(got) != 2 || got[0].Id != 1 || got[0].Name != "a" || got[1].Id != 2 || got[1].Name != "b" {
		t.Errorf("往返结果 = %+v", got)
	}
}

// 校验实际生成的yaml文本
func TestYamlStruct2Str(t *testing.T) {
	ctx := GenCtx()
	got := YamlStruct2Str(ctx, yamlDemo{Id: 1, Name: "n", Skip: "x"})
	//yaml标签必须生效（小写键名）
	if !strings.Contains(got, "id: 1") {
		t.Errorf("id 字段格式异常: %q", got)
	}
	//注意：yaml.v2 会给可能被误解析为其他类型的短标量加引号（如 n/y/yes/no 会被当布尔），
	//故此处只断言键名与值出现，不断言是否带引号
	if !strings.Contains(got, "name:") || !strings.Contains(got, "n") {
		t.Errorf("name 字段格式异常: %q", got)
	}
	//`yaml:"-"` 必须被忽略
	if strings.Contains(got, "skip") {
		t.Errorf(`yaml:"-" 字段未被忽略: %q`, got)
	}
	//yaml以换行分隔而非json式花括号
	if strings.HasPrefix(strings.TrimSpace(got), "{") {
		t.Errorf("输出疑似json而非yaml: %q", got)
	}
	//列表应使用 - 前缀
	listText := YamlStruct2Str(ctx, []int{1, 2})
	if !strings.Contains(listText, "- 1") {
		t.Errorf("列表格式异常: %q", listText)
	}
	//Data 与 String 版本一致
	if string(YamlStruct2Data(ctx, yamlDemo{Id: 1})) != YamlStruct2Str(ctx, yamlDemo{Id: 1}) {
		t.Errorf("Data 与 String 版本不一致")
	}
	//map与嵌套结构
	nested := map[string]yamlDemo{"k": {Id: 9, Name: "deep"}}
	text := YamlStruct2Str(ctx, nested)
	var back map[string]yamlDemo
	if err := YamlStr2Struct(ctx, text, &back); err != nil {
		t.Fatalf("%+v", err)
	}
	if back["k"].Id != 9 || back["k"].Name != "deep" {
		t.Errorf("嵌套结构往返失败: %+v", back)
	}
}

// 直接解析手写yaml，验证兼容真实配置文件写法
func TestYamlParseHandWritten(t *testing.T) {
	ctx := GenCtx()
	var got yamlDemo
	//含注释、空行与缩进
	text := "# 注释行\n\nid: 42\nname: 中文名\n"
	if err := YamlStr2Struct(ctx, text, &got); err != nil {
		t.Fatalf("解析手写yaml异常: %+v", err)
	}
	if got.Id != 42 {
		t.Errorf("id = %d, 期望 42", got.Id)
	}
	if got.Name != "中文名" {
		t.Errorf("name = %q, 期望 中文名", got.Name)
	}
	//空yaml应成功且保持零值
	var empty yamlDemo
	if err := YamlStr2Struct(ctx, "", &empty); err != nil {
		t.Errorf("空yaml应可解析: %+v", err)
	}
	if empty.Id != 0 || empty.Name != "" {
		t.Errorf("空yaml应为零值: %+v", empty)
	}
}

func TestYamlError(t *testing.T) {
	ctx := GenCtx()
	var got yamlDemo
	//语法非法必须报错
	for _, bad := range []string{"id: [unclosed", "\tid: 1", "a:\n- 1\n b: 2"} {
		if err := YamlStr2Struct(ctx, bad, &got); err == nil {
			t.Errorf("YamlStr2Struct(%q) 应返回error", bad)
		}
	}
	//类型不匹配必须报错
	if err := YamlStr2Struct(ctx, "id: 不是数字", &got); err == nil {
		t.Errorf("类型不匹配应返回error")
	}
	//Data 版本同样报错
	if err := YamlData2Struct(ctx, []byte("id: ["), &got); err == nil {
		t.Errorf("YamlData2Struct 非法数据应返回error")
	}
	//不可序列化类型返回空且不panic
	if got := YamlStruct2Str(ctx, func() {}); got != "" {
		t.Errorf("不可序列化类型 = %q, 期望空串", got)
	}
	//非指针目标应报错
	if err := YamlStr2Struct(ctx, "id: 1", got); err == nil {
		t.Errorf("传入非指针应返回error")
	}
}

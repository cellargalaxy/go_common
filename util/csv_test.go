package util

import (
	"bytes"
	"path"
	"strings"
	"testing"
)

type csvDemo struct {
	Id   int    `csv:"id"`
	Name string `csv:"name"`
}

func csvDemoList() []csvDemo {
	return []csvDemo{{Id: 1, Name: "a"}, {Id: 2, Name: "b"}}
}

// 校验实际生成的CSV文本（含表头），而非仅"能转回来"
func TestCsvStruct2String(t *testing.T) {
	ctx := GenCtx()
	got, err := CsvStruct2String(ctx, csvDemoList())
	if err != nil {
		t.Fatalf("%+v", err)
	}
	//首行须为csv标签构成的表头
	lines := strings.Split(strings.TrimSpace(strings.ReplaceAll(got, "\r\n", "\n")), "\n")
	if len(lines) != 3 {
		t.Fatalf("行数 = %d, 期望 3（表头+2行数据）: %q", len(lines), got)
	}
	if lines[0] != "id,name" {
		t.Errorf("表头 = %q, 期望 id,name", lines[0])
	}
	if lines[1] != "1,a" {
		t.Errorf("第1行 = %q, 期望 1,a", lines[1])
	}
	if lines[2] != "2,b" {
		t.Errorf("第2行 = %q, 期望 2,b", lines[2])
	}
	//Data 与 String 版本一致
	data, err := CsvStruct2Data(ctx, csvDemoList())
	if err != nil {
		t.Fatalf("%+v", err)
	}
	if string(data) != got {
		t.Errorf("Data 与 String 版本不一致")
	}
}

func TestCsvStruct2Strings(t *testing.T) {
	ctx := GenCtx()
	got, err := CsvStruct2Strings(ctx, csvDemoList())
	if err != nil {
		t.Fatalf("%+v", err)
	}
	if len(got) != 3 {
		t.Fatalf("行数 = %d, 期望 3", len(got))
	}
	if got[0][0] != "id" || got[0][1] != "name" {
		t.Errorf("表头 = %v", got[0])
	}
	if got[1][0] != "1" || got[1][1] != "a" {
		t.Errorf("第1行 = %v", got[1])
	}
	if got[2][0] != "2" || got[2][1] != "b" {
		t.Errorf("第2行 = %v", got[2])
	}
}

// 各条转换路径（Data/String/Strings/Reader/File）结果必须彼此一致
func TestCsvRoundTripAllPaths(t *testing.T) {
	ctx := GenCtx()
	want := csvDemoList()

	//Data 路径
	data, err := CsvStruct2Data(ctx, want)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	var viaData []csvDemo
	if err = CsvData2Struct(ctx, data, &viaData); err != nil {
		t.Fatalf("%+v", err)
	}
	assertCsvList(t, "Data", viaData, want)

	//String 路径
	text, err := CsvStruct2String(ctx, want)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	var viaString []csvDemo
	if err = CsvString2Struct(ctx, text, &viaString); err != nil {
		t.Fatalf("%+v", err)
	}
	assertCsvList(t, "String", viaString, want)

	//Strings 路径
	strs, err := CsvStruct2Strings(ctx, want)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	var viaStrings []csvDemo
	if err = CsvStrings2Struct(ctx, strs, &viaStrings); err != nil {
		t.Fatalf("%+v", err)
	}
	assertCsvList(t, "Strings", viaStrings, want)

	//Reader 路径
	var viaReader []csvDemo
	if err = CsvReader2Struct(ctx, strings.NewReader(text), &viaReader); err != nil {
		t.Fatalf("%+v", err)
	}
	assertCsvList(t, "Reader", viaReader, want)

	//File 路径
	dir := newTestDir(t)
	filePath := path.Join(dir, "out.csv")
	if err = CsvStruct2File(ctx, want, filePath); err != nil {
		t.Fatalf("%+v", err)
	}
	var viaFile []csvDemo
	if err = CsvFile2Struct(ctx, filePath, &viaFile); err != nil {
		t.Fatalf("%+v", err)
	}
	assertCsvList(t, "File", viaFile, want)

	//Writer 路径
	var buf bytes.Buffer
	if err = CsvStruct2Writer(ctx, want, &buf); err != nil {
		t.Fatalf("%+v", err)
	}
	if buf.String() != text {
		t.Errorf("Writer 输出与 String 不一致:\n%q\n%q", buf.String(), text)
	}
}

func assertCsvList(t *testing.T, path string, got, want []csvDemo) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("[%s] 长度 = %d, 期望 %d", path, len(got), len(want))
		return
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("[%s] 第%d条 = %+v, 期望 %+v", path, i, got[i], want[i])
		}
	}
}

func TestCsvStrings2AllPaths(t *testing.T) {
	ctx := GenCtx()
	lines := [][]string{{"h1", "h2"}, {"v1", "v2"}}

	//Strings -> String -> Strings 往返
	text, err := CsvStrings2String(ctx, lines)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	got, err := CsvString2Strings(ctx, text)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	assertStrings(t, got, lines)

	//Strings -> Data -> Strings
	data, err := CsvStrings2Data(ctx, lines)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	if got, err = CsvData2Strings(ctx, data); err != nil {
		t.Fatalf("%+v", err)
	}
	assertStrings(t, got, lines)

	//Strings -> File -> Strings
	dir := newTestDir(t)
	filePath := path.Join(dir, "x.csv")
	if err = CsvStrings2File(ctx, lines, filePath); err != nil {
		t.Fatalf("%+v", err)
	}
	if got, err = CsvFile2Strings(ctx, filePath); err != nil {
		t.Fatalf("%+v", err)
	}
	assertStrings(t, got, lines)

	//Strings -> Writer
	var buf bytes.Buffer
	if err = CsvStrings2Writer(ctx, lines, &buf); err != nil {
		t.Fatalf("%+v", err)
	}
	if buf.String() != text {
		t.Errorf("Writer 与 String 不一致: %q vs %q", buf.String(), text)
	}

	//Reader -> Strings
	if got, err = CsvReader2Strings(ctx, strings.NewReader(text)); err != nil {
		t.Fatalf("%+v", err)
	}
	assertStrings(t, got, lines)
}

func assertStrings(t *testing.T, got, want [][]string) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("行数 = %d, 期望 %d: %v", len(got), len(want), got)
		return
	}
	for i := range want {
		if len(got[i]) != len(want[i]) {
			t.Errorf("第%d行列数 = %d, 期望 %d: %v", i, len(got[i]), len(want[i]), got[i])
			continue
		}
		for j := range want[i] {
			if got[i][j] != want[i][j] {
				t.Errorf("[%d][%d] = %q, 期望 %q", i, j, got[i][j], want[i][j])
			}
		}
	}
}

// CSV 特殊字符（逗号、引号、换行、中文）必须被正确转义并还原
func TestCsvEscape(t *testing.T) {
	ctx := GenCtx()
	lines := [][]string{
		{"含,逗号", `含"引号`},
		{"含\n换行", "中文内容"},
		{"", " 前后空格 "},
	}
	text, err := CsvStrings2String(ctx, lines)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	got, err := CsvString2Strings(ctx, text)
	if err != nil {
		t.Fatalf("解析含特殊字符的CSV异常: %+v", err)
	}
	assertStrings(t, got, lines)
}

func TestCsvEmpty(t *testing.T) {
	ctx := GenCtx()
	//空行集
	text, err := CsvStrings2String(ctx, [][]string{})
	if err != nil {
		t.Fatalf("%+v", err)
	}
	got, err := CsvString2Strings(ctx, text)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	if len(got) != 0 {
		t.Errorf("空行集往返 = %v", got)
	}
	//空字符串解析
	if got, err = CsvString2Strings(ctx, ""); err != nil {
		t.Errorf("空串解析异常: %+v", err)
	}
	if len(got) != 0 {
		t.Errorf("空串 = %v", got)
	}
	//空结构体列表：仍应输出表头
	text, err = CsvStruct2String(ctx, []csvDemo{})
	if err != nil {
		t.Fatalf("%+v", err)
	}
	if !strings.Contains(text, "id") {
		t.Errorf("空列表应仍输出表头, got %q", text)
	}
}

func TestCsvError(t *testing.T) {
	ctx := GenCtx()
	//列数不一致不再报错：本库写入端(CsvStrings2Data)对参差不齐的行照写不误，
	//读取端若按首行列数强校验，会导致本库自己写出的CSV自己读不回来，
	//故读取端放宽为不校验列数（详见 CsvReader2Strings 注释与 TestCsvRaggedRoundTrip）
	if got, err := CsvString2Strings(ctx, "a,b\nc,d,e\n"); err != nil {
		t.Errorf("列数不一致不应再返回error: %+v", err)
	} else if len(got) != 2 || len(got[0]) != 2 || len(got[1]) != 3 {
		t.Errorf("列数不一致的行未按原样读回: %v", got)
	}
	//但真正的语法错误（引号未闭合等）仍必须报错，放宽列数校验不等于放弃校验
	for _, bad := range []string{"a,\"b\nc,d\n", "a,\"bc\n", "a,\"b\"x,c\n"} {
		if _, err := CsvString2Strings(ctx, bad); err == nil {
			t.Errorf("非法CSV %q 应返回error", bad)
		}
	}
	//类型不匹配
	var list []csvDemo
	if err := CsvString2Struct(ctx, "id,name\n不是数字,a\n", &list); err == nil {
		t.Errorf("类型不匹配应返回error")
	}
	//非指针目标：必须报错而非静默写入
	var fresh []csvDemo
	if err := CsvString2Struct(ctx, "id,name\n1,a\n", fresh); err == nil {
		t.Errorf("非指针目标应返回error")
	}
	//nil 目标不得panic，须转为error
	if err := CsvString2Struct(ctx, "id,name\n1,a\n", nil); err == nil {
		t.Errorf("nil 目标应返回error")
	}
	if err := CsvData2Struct(ctx, []byte("id,name\n1,a\n"), nil); err == nil {
		t.Errorf("CsvData2Struct(nil) 应返回error")
	}
	if err := CsvReader2Struct(ctx, strings.NewReader("id,name\n1,a\n"), nil); err == nil {
		t.Errorf("CsvReader2Struct(nil) 应返回error")
	}
	//不存在的文件：ReadFile2Data 返回nil数据，转换应报错或返回空而非panic
	dir := newTestDir(t)
	if _, err := CsvFile2Strings(ctx, path.Join(dir, "nope.csv")); err != nil {
		t.Logf("不存在的文件返回error: %v", err)
	}
	//非切片目标
	var single csvDemo
	if err := CsvString2Struct(ctx, "id,name\n1,a\n", &single); err == nil {
		t.Errorf("非切片目标应返回error")
	}
}

// 序列化侧的非法入参必须与反序列化侧同一套约定：一律返回error，不得panic。
// gocsv 对 nil 会 panic(reflect.Value.Type on zero Value)，须由本包兜底。
func TestCsvStruct2XxxIllegalInput(t *testing.T) {
	ctx := GenCtx()

	//nil 入参：四个出口都不得panic，必须返回error
	if _, err := CsvStruct2Data(ctx, nil); err == nil {
		t.Errorf("CsvStruct2Data(nil) 应返回error")
	}
	if _, err := CsvStruct2String(ctx, nil); err == nil {
		t.Errorf("CsvStruct2String(nil) 应返回error")
	}
	if err := CsvStruct2Writer(ctx, nil, &bytes.Buffer{}); err == nil {
		t.Errorf("CsvStruct2Writer(nil) 应返回error")
	}
	if _, err := CsvStruct2Strings(ctx, nil); err == nil {
		t.Errorf("CsvStruct2Strings(nil) 应返回error")
	}
	//nil 时不得产生半截文件
	dir := newTestDir(t)
	nilPath := path.Join(dir, "nil.csv")
	if err := CsvStruct2File(ctx, nil, nilPath); err == nil {
		t.Errorf("CsvStruct2File(nil) 应返回error")
	}
	if GetPathInfo(ctx, nilPath) != nil {
		t.Errorf("CsvStruct2File(nil) 失败后仍创建了文件")
	}

	//非切片/非数组入参：gocsv 自身返回error，同样不得panic
	for _, bad := range []interface{}{123, "abc", 1.5, true, struct{ A int }{1}} {
		if _, err := CsvStruct2Data(ctx, bad); err == nil {
			t.Errorf("CsvStruct2Data(%#v) 应返回error", bad)
		}
		if _, err := CsvStruct2String(ctx, bad); err == nil {
			t.Errorf("CsvStruct2String(%#v) 应返回error", bad)
		}
		if err := CsvStruct2Writer(ctx, bad, &bytes.Buffer{}); err == nil {
			t.Errorf("CsvStruct2Writer(%#v) 应返回error", bad)
		}
	}

	//合法空切片仍须成功，且只输出表头，不能被上面的兜底误伤
	data, err := CsvStruct2Data(ctx, []csvDemo{})
	if err != nil {
		t.Errorf("CsvStruct2Data(空切片) 不应报错: %+v", err)
	}
	if len(data) == 0 {
		t.Errorf("CsvStruct2Data(空切片) 应输出表头")
	}
	if text, err := CsvStruct2String(ctx, []csvDemo{}); err != nil || text == "" {
		t.Errorf("CsvStruct2String(空切片) = %q, err=%v", text, err)
	}
	var buf bytes.Buffer
	if err := CsvStruct2Writer(ctx, []csvDemo{}, &buf); err != nil || buf.Len() == 0 {
		t.Errorf("CsvStruct2Writer(空切片) = %q, err=%v", buf.String(), err)
	}
}

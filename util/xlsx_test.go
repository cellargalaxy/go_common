package util

import (
	"path"
	"strings"
	"testing"
)

func TestXlsxRoundTrip(t *testing.T) {
	ctx := GenCtx()
	lines := [][]string{{"id", "name"}, {"1", "a"}, {"2", "b"}}

	data, err := XlsxStrs2Data(ctx, lines)
	if err != nil {
		t.Fatalf("XlsxStrs2Data 异常: %+v", err)
	}
	//须产出真实的xlsx（zip格式，以PK开头）
	if len(data) == 0 {
		t.Fatalf("生成的xlsx为空")
	}
	if !strings.HasPrefix(string(data[:2]), "PK") {
		t.Errorf("生成的数据不是xlsx(zip)格式: %v", data[:4])
	}

	got, err := XlsxData2Strs(ctx, data)
	if err != nil {
		t.Fatalf("XlsxData2Strs 异常: %+v", err)
	}
	if len(got) != 3 {
		t.Fatalf("行数 = %d, 期望 3", len(got))
	}
	for i := range lines {
		if len(got[i]) != len(lines[i]) {
			t.Errorf("第%d行列数 = %d, 期望 %d", i, len(got[i]), len(lines[i]))
			continue
		}
		for j := range lines[i] {
			if got[i][j] != lines[i][j] {
				t.Errorf("[%d][%d] = %q, 期望 %q", i, j, got[i][j], lines[i][j])
			}
		}
	}
}

func TestXlsxFileRoundTrip(t *testing.T) {
	ctx := GenCtx()
	dir := newTestDir(t)
	filePath := path.Join(dir, "out.xlsx")
	lines := [][]string{{"h1", "h2"}, {"v1", "v2"}}

	if err := XlsxStrs2File(ctx, lines, filePath); err != nil {
		t.Fatalf("XlsxStrs2File 异常: %+v", err)
	}
	if GetFileInfo(ctx, filePath) == nil {
		t.Fatalf("xlsx文件未生成")
	}
	got, err := XlsxFile2Strs(ctx, filePath)
	if err != nil {
		t.Fatalf("XlsxFile2Strs 异常: %+v", err)
	}
	if len(got) != 2 || got[0][0] != "h1" || got[1][1] != "v2" {
		t.Errorf("文件往返结果 = %v", got)
	}
}

// 中文、特殊字符、长文本、数字样式字符串都必须原样保留
func TestXlsxContentFidelity(t *testing.T) {
	ctx := GenCtx()
	lines := [][]string{
		{"中文列", "special,\"chars"},
		{"0123", "1e5"},
		{strings.Repeat("长", 200), ""},
		{"  前后空格  ", "=SUM(A1)"},
	}
	data, err := XlsxStrs2Data(ctx, lines)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	got, err := XlsxData2Strs(ctx, data)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	if len(got) < 4 {
		t.Fatalf("行数 = %d, 期望至少 4", len(got))
	}
	//以字符串写入，"0123" 不应被转成数字 123，公式不应被求值
	if got[1][0] != "0123" {
		t.Errorf("数字样式字符串被改写: %q, 期望 0123", got[1][0])
	}
	if got[1][1] != "1e5" {
		t.Errorf("科学计数样式字符串被改写: %q, 期望 1e5", got[1][1])
	}
	if got[0][0] != "中文列" {
		t.Errorf("中文丢失: %q", got[0][0])
	}
	if got[0][1] != `special,"chars` {
		t.Errorf("特殊字符被改写: %q", got[0][1])
	}
	if got[3][1] != "=SUM(A1)" {
		t.Errorf("公式文本被求值或改写: %q", got[3][1])
	}
	if len(got[2][0]) != len(strings.Repeat("长", 200)) {
		t.Errorf("长文本被截断: %d 字节", len(got[2][0]))
	}
}

// 关键回归：非法数据不得panic（曾因先defer关闭再判err而空指针panic）
func TestXlsxData2StrsInvalid(t *testing.T) {
	ctx := GenCtx()
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("XlsxData2Strs 非法数据触发panic: %v", r)
		}
	}()
	for _, bad := range [][]byte{
		[]byte("not an xlsx file"),
		{},
		nil,
		{0x50, 0x4b, 0x03, 0x04}, //仅zip头，内容不完整
	} {
		got, err := XlsxData2Strs(ctx, bad)
		if err == nil && len(got) > 0 {
			t.Errorf("非法数据未报错且返回了内容: %v", got)
		}
	}
}

func TestXlsxEmpty(t *testing.T) {
	ctx := GenCtx()
	//空行集也应能生成合法xlsx并读回空
	data, err := XlsxStrs2Data(ctx, [][]string{})
	if err != nil {
		t.Fatalf("空行集生成异常: %+v", err)
	}
	got, err := XlsxData2Strs(ctx, data)
	if err != nil {
		t.Fatalf("空xlsx读取异常: %+v", err)
	}
	if len(got) != 0 {
		t.Errorf("空xlsx = %v", got)
	}
	//含空串格
	data, err = XlsxStrs2Data(ctx, [][]string{{"", ""}, {"x", ""}})
	if err != nil {
		t.Fatalf("%+v", err)
	}
	if got, err = XlsxData2Strs(ctx, data); err != nil {
		t.Fatalf("%+v", err)
	}
	//至少要能找到 x
	var found bool
	for i := range got {
		for j := range got[i] {
			if got[i][j] == "x" {
				found = true
			}
		}
	}
	if !found {
		t.Errorf("含空格的表丢失了非空内容: %v", got)
	}
}

// 与CSV路径联动：同一份结构体数据经csv转strings后写xlsx，内容须一致
func TestXlsxWithCsvStrings(t *testing.T) {
	ctx := GenCtx()
	strs, err := CsvStruct2Strs(ctx, csvDemoList())
	if err != nil {
		t.Fatalf("%+v", err)
	}
	data, err := XlsxStrs2Data(ctx, strs)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	got, err := XlsxData2Strs(ctx, data)
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

// 已知限制：读取硬编码只认 Sheet1，首表非此名的外部文件读不到
func TestXlsxSheetNameConstant(t *testing.T) {
	if XlsxSheetNameDefault != "Sheet1" {
		t.Errorf("XlsxSheetNameDefault = %q, 期望 Sheet1", XlsxSheetNameDefault)
	}
	ctx := GenCtx()
	//本包生成的文件必定含Sheet1，故自产自销可读
	data, err := XlsxStrs2Data(ctx, [][]string{{"a"}})
	if err != nil {
		t.Fatalf("%+v", err)
	}
	if _, err = XlsxData2Strs(ctx, data); err != nil {
		t.Errorf("本包生成的xlsx应可读回: %+v", err)
	}
}

// 不存在的文件不得panic
func TestXlsxFile2StrsMissing(t *testing.T) {
	ctx := GenCtx()
	dir := newTestDir(t)
	missing := path.Join(dir, "nope.xlsx")
	got, err := XlsxFile2Strs(ctx, missing)
	if err == nil && len(got) > 0 {
		t.Errorf("不存在的文件返回了内容: %v", got)
	}
	//只读操作不得创建文件
	if GetPathInfo(ctx, missing) != nil {
		t.Errorf("XlsxFile2Strs 创建了文件")
	}
}

package util

import (
	"bufio"
	"bytes"
	"os"
	"path"
	"strings"
	"testing"
)

func TestClearPath(t *testing.T) {
	cases := map[string]string{
		//反斜杠统一为正斜杠，便于跨平台
		`a\b\c`: "a/b/c",
		`a\b/c`: "a/b/c",
		"a/b/c": "a/b/c",
		//冗余分隔符与.被清理
		"a//b":     "a/b",
		"a/./b":    "a/b",
		"a/b/..":   "a",
		"a/b/../c": "a/c",
		"./a":      "a",
		"/a/b":     "/a/b",
		//已知边界：空串归一为"."
		"":       ".",
		".":      ".",
		"/":      "/",
		`C:\x\y`: "C:/x/y",
	}
	for in, want := range cases {
		if got := ClearPath(in); got != want {
			t.Errorf("ClearPath(%q) = %q, 期望 %q", in, got, want)
		}
	}
}

func TestCloseIo(t *testing.T) {
	ctx := GenCtx()
	//nil 元素必须被跳过而非panic
	CloseIo(ctx, nil)
	CloseIo(ctx)
	CloseIo(ctx, nil, nil)

	//多个closer都应被关闭
	c1, c2 := &countCloser{}, &countCloser{}
	CloseIo(ctx, c1, nil, c2)
	if c1.n != 1 || c2.n != 1 {
		t.Errorf("关闭次数 = %d/%d, 期望各1次", c1.n, c2.n)
	}
	//某个closer报错不应阻断其余关闭（避免资源泄漏）
	bad := &countCloser{err: os.ErrClosed}
	good := &countCloser{}
	CloseIo(ctx, bad, good)
	if good.n != 1 {
		t.Errorf("前一个closer报错后，后续closer未被关闭")
	}
}

type countCloser struct {
	n   int
	err error
}

func (this *countCloser) Close() error {
	this.n++
	return this.err
}

func TestGetPathInfo(t *testing.T) {
	ctx := GenCtx()
	dir := newTestDir(t)
	filePath := path.Join(dir, "f.txt")
	if err := os.WriteFile(filePath, []byte("data"), 0644); err != nil {
		t.Fatalf("%+v", err)
	}

	//文件
	info := GetPathInfo(ctx, filePath)
	if info == nil {
		t.Fatalf("GetPathInfo 对已存在文件返回 nil")
	}
	if info.IsDir() {
		t.Errorf("文件被判定为目录")
	}
	if info.Size() != 4 {
		t.Errorf("文件大小 = %d, 期望 4", info.Size())
	}
	//目录
	if info = GetPathInfo(ctx, dir); info == nil || !info.IsDir() {
		t.Errorf("GetPathInfo 对目录判定异常: %v", info)
	}
	//不存在
	if info = GetPathInfo(ctx, path.Join(dir, "nope")); info != nil {
		t.Errorf("不存在的路径应返回 nil, got %v", info)
	}
	if info = GetPathInfo(ctx, ""); info != nil {
		t.Errorf("空路径应返回 nil")
	}
}

// GetFileInfo 与 GetFolderInfo 必须互斥：各自过滤掉另一种类型
func TestGetFileAndFolderInfo(t *testing.T) {
	ctx := GenCtx()
	dir := newTestDir(t)
	filePath := path.Join(dir, "f.txt")
	if err := os.WriteFile(filePath, []byte("x"), 0644); err != nil {
		t.Fatalf("%+v", err)
	}

	//文件路径：GetFileInfo 有值，GetFolderInfo 为nil
	if GetFileInfo(ctx, filePath) == nil {
		t.Errorf("GetFileInfo 对文件返回 nil")
	}
	if got := GetFolderInfo(ctx, filePath); got != nil {
		t.Errorf("GetFolderInfo 对文件应返回 nil, got %v", got)
	}
	//目录路径：反之
	if GetFolderInfo(ctx, dir) == nil {
		t.Errorf("GetFolderInfo 对目录返回 nil")
	}
	if got := GetFileInfo(ctx, dir); got != nil {
		t.Errorf("GetFileInfo 对目录应返回 nil, got %v", got)
	}
	//都不存在
	nope := path.Join(dir, "nope")
	if GetFileInfo(ctx, nope) != nil || GetFolderInfo(ctx, nope) != nil {
		t.Errorf("不存在的路径都应返回 nil")
	}
}

func TestCreateFolderPath(t *testing.T) {
	ctx := GenCtx()
	dir := newTestDir(t)

	//多层目录一次创建
	target := path.Join(dir, "a", "b", "c")
	if err := CreateFolderPath(ctx, target); err != nil {
		t.Fatalf("CreateFolderPath 异常: %+v", err)
	}
	if info := GetFolderInfo(ctx, target); info == nil {
		t.Errorf("目录未创建: %s", target)
	}
	//重复创建应幂等，不报错
	if err := CreateFolderPath(ctx, target); err != nil {
		t.Errorf("重复创建应幂等: %+v", err)
	}
	//目标已是文件时必须报错
	filePath := path.Join(dir, "afile")
	if err := os.WriteFile(filePath, []byte("x"), 0644); err != nil {
		t.Fatalf("%+v", err)
	}
	if err := CreateFolderPath(ctx, filePath); err == nil {
		t.Errorf("在已有文件路径上创建目录应报错")
	}
}

func TestListFile(t *testing.T) {
	ctx := GenCtx()
	dir := newTestDir(t)

	//空目录返回空
	files, err := ListFile(ctx, dir)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	if len(files) != 0 {
		t.Errorf("空目录 = %d 个文件", len(files))
	}

	//两个文件+一个子目录
	for _, name := range []string{"a.txt", "b.txt"} {
		if err = os.WriteFile(path.Join(dir, name), []byte("x"), 0644); err != nil {
			t.Fatalf("%+v", err)
		}
	}
	if err = os.Mkdir(path.Join(dir, "sub"), 0750); err != nil {
		t.Fatalf("%+v", err)
	}
	files, err = ListFile(ctx, dir)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	if len(files) != 3 {
		t.Errorf("文件数 = %d, 期望 3", len(files))
	}
	//子目录须被标记为目录
	var dirCount int
	for i := range files {
		if files[i].IsDir() {
			dirCount++
		}
	}
	if dirCount != 1 {
		t.Errorf("目录数 = %d, 期望 1", dirCount)
	}

	//不存在的目录：约定返回 nil,nil 而非报错
	files, err = ListFile(ctx, path.Join(dir, "nope"))
	if err != nil || files != nil {
		t.Errorf("不存在的目录 = %v, %v, 期望 nil,nil", files, err)
	}
	//传入文件而非目录：同样 nil,nil
	files, err = ListFile(ctx, path.Join(dir, "a.txt"))
	if err != nil || files != nil {
		t.Errorf("传入文件 = %v, %v, 期望 nil,nil", files, err)
	}
}

func TestWriteAndReadFile(t *testing.T) {
	ctx := GenCtx()
	dir := newTestDir(t)
	filePath := path.Join(dir, "sub", "w.txt")

	//写入时自动创建父目录
	if err := WriteString2File(ctx, "hello", filePath); err != nil {
		t.Fatalf("WriteString2File 异常: %+v", err)
	}
	got, err := ReadFile2String(ctx, filePath, "")
	if err != nil {
		t.Fatalf("%+v", err)
	}
	if got != "hello" {
		t.Errorf("读取内容 = %q, 期望 hello", got)
	}

	//覆盖写：必须截断旧内容，不能残留
	if err = WriteString2File(ctx, "hi", filePath); err != nil {
		t.Fatalf("%+v", err)
	}
	if got, err = ReadFile2String(ctx, filePath, ""); err != nil || got != "hi" {
		t.Errorf("覆盖写后 = %q (err %v), 期望 hi（旧内容未被截断则会是 hillo）", got, err)
	}

	//二进制数据往返
	bin := []byte{0, 1, 2, 255, 0}
	binPath := path.Join(dir, "b.bin")
	if err = WriteData2File(ctx, bin, binPath); err != nil {
		t.Fatalf("%+v", err)
	}
	data, err := ReadFile2Data(ctx, binPath, nil)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	if string(data) != string(bin) {
		t.Errorf("二进制往返失败: %v", data)
	}

	//WriteReader2File
	readerPath := path.Join(dir, "r.txt")
	if err = WriteReader2File(ctx, strings.NewReader("from reader"), readerPath); err != nil {
		t.Fatalf("%+v", err)
	}
	if got, err = ReadFile2String(ctx, readerPath, ""); err != nil || got != "from reader" {
		t.Errorf("WriteReader2File 结果 = %q, %v", got, err)
	}

	//写入目录路径必须报错
	if err = WriteString2File(ctx, "x", dir); err == nil {
		t.Errorf("向目录写入应报错")
	}
}

// 关键回归：只读函数不得产生落盘副作用（曾创建目录+空文件）
func TestReadFileNoSideEffect(t *testing.T) {
	ctx := GenCtx()
	dir := newTestDir(t)
	missing := path.Join(dir, "sub", "missing.txt")

	//不存在时返回默认值
	got, err := ReadFile2String(ctx, missing, "默认内容")
	if err != nil {
		t.Fatalf("ReadFile2String 异常: %+v", err)
	}
	if got != "默认内容" {
		t.Errorf("默认值 = %q", got)
	}
	//绝不能因为读取而创建文件或父目录
	if GetPathInfo(ctx, missing) != nil {
		t.Errorf("只读操作创建了文件: %s", missing)
	}
	if GetPathInfo(ctx, path.Join(dir, "sub")) != nil {
		t.Errorf("只读操作创建了父目录")
	}

	//Data 版本同样不能有副作用
	data, err := ReadFile2Data(ctx, missing, []byte("def"))
	if err != nil || string(data) != "def" {
		t.Errorf("ReadFile2Data = %q, %v", data, err)
	}
	if GetPathInfo(ctx, missing) != nil {
		t.Errorf("ReadFile2Data 创建了文件")
	}

	//OpenReadFile 对不存在的文件必须报错，而非创建
	if _, err = OpenReadFile(ctx, missing); err == nil {
		t.Errorf("OpenReadFile 对不存在的文件应报错")
	}
	if GetPathInfo(ctx, missing) != nil {
		t.Errorf("OpenReadFile 创建了文件")
	}
	//OpenReadFile 对目录必须报错
	if _, err = OpenReadFile(ctx, dir); err == nil {
		t.Errorf("OpenReadFile 对目录应报错")
	}

	//空文件应返回默认值（当前实现约定）
	emptyPath := path.Join(dir, "empty.txt")
	if err = os.WriteFile(emptyPath, []byte{}, 0644); err != nil {
		t.Fatalf("%+v", err)
	}
	if got, err = ReadFile2String(ctx, emptyPath, "兜底"); err != nil || got != "兜底" {
		t.Errorf("空文件 = %q, %v, 期望兜底", got, err)
	}
}

func TestReadFile2Writer(t *testing.T) {
	ctx := GenCtx()
	dir := newTestDir(t)
	filePath := path.Join(dir, "f.txt")
	if err := os.WriteFile(filePath, []byte("content"), 0644); err != nil {
		t.Fatalf("%+v", err)
	}

	//正常读出到writer
	var buf bytes.Buffer
	if err := ReadFile2Writer(ctx, filePath, &buf, nil); err != nil {
		t.Fatalf("%+v", err)
	}
	if buf.String() != "content" {
		t.Errorf("写出内容 = %q", buf.String())
	}

	//文件不存在时写出默认值，且不创建文件
	buf.Reset()
	missing := path.Join(dir, "sub", "nope.txt")
	if err := ReadFile2Writer(ctx, missing, &buf, []byte("默认")); err != nil {
		t.Fatalf("%+v", err)
	}
	if buf.String() != "默认" {
		t.Errorf("默认值 = %q", buf.String())
	}
	if GetPathInfo(ctx, missing) != nil {
		t.Errorf("只读操作创建了文件")
	}
	//不存在且无默认值：什么都不写
	buf.Reset()
	if err := ReadFile2Writer(ctx, missing, &buf, nil); err != nil {
		t.Fatalf("%+v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("无默认值时应无输出, got %q", buf.String())
	}

	//空文件时回落默认值
	buf.Reset()
	emptyPath := path.Join(dir, "empty.txt")
	if err := os.WriteFile(emptyPath, []byte{}, 0644); err != nil {
		t.Fatalf("%+v", err)
	}
	if err := ReadFile2Writer(ctx, emptyPath, &buf, []byte("兜底")); err != nil {
		t.Fatalf("%+v", err)
	}
	if buf.String() != "兜底" {
		t.Errorf("空文件应写出默认值, got %q", buf.String())
	}
}

// 关键回归：对不存在的文件必须报错，不能返回空文件的MD5
func TestGetFileMd5(t *testing.T) {
	ctx := GenCtx()
	dir := newTestDir(t)
	filePath := path.Join(dir, "f.txt")
	if err := os.WriteFile(filePath, []byte("123456"), 0644); err != nil {
		t.Fatalf("%+v", err)
	}

	//与标准MD5一致
	got, err := GetFileMd5(ctx, filePath)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	if got != "e10adc3949ba59abbe56e057f20f883e" {
		t.Errorf("MD5 = %s, 期望 e10adc3949ba59abbe56e057f20f883e", got)
	}
	//与 EnMd5Hex 结果一致（同一份内容两条实现路径必须自洽）
	if want := EnMd5Hex("123456"); got != want {
		t.Errorf("GetFileMd5 = %s, EnMd5Hex = %s, 两者不一致", got, want)
	}

	//不存在的文件必须报错，且不得返回空文件MD5 d41d8...
	missing := path.Join(dir, "nope.txt")
	got, err = GetFileMd5(ctx, missing)
	if err == nil {
		t.Errorf("不存在的文件应返回error")
	}
	if got == "d41d8cd98f00b204e9800998ecf8427e" {
		t.Errorf("对不存在的文件返回了空文件MD5，掩盖了文件缺失")
	}
	if GetPathInfo(ctx, missing) != nil {
		t.Errorf("GetFileMd5 创建了文件")
	}

	//空文件才应返回空MD5
	emptyPath := path.Join(dir, "empty.txt")
	if err = os.WriteFile(emptyPath, []byte{}, 0644); err != nil {
		t.Fatalf("%+v", err)
	}
	if got, err = GetFileMd5(ctx, emptyPath); err != nil || got != "d41d8cd98f00b204e9800998ecf8427e" {
		t.Errorf("空文件MD5 = %s, %v", got, err)
	}
	//目录须报错
	if _, err = GetFileMd5(ctx, dir); err == nil {
		t.Errorf("对目录求MD5应报错")
	}
}

func TestRemoveFile(t *testing.T) {
	ctx := GenCtx()
	dir := newTestDir(t)

	//删除文件
	filePath := path.Join(dir, "f.txt")
	if err := os.WriteFile(filePath, []byte("x"), 0644); err != nil {
		t.Fatalf("%+v", err)
	}
	if err := RemoveFile(ctx, filePath); err != nil {
		t.Fatalf("RemoveFile 异常: %+v", err)
	}
	if GetPathInfo(ctx, filePath) != nil {
		t.Errorf("文件未被删除")
	}
	//删除不存在的文件：约定不报错（幂等）
	if err := RemoveFile(ctx, filePath); err != nil {
		t.Errorf("删除不存在的文件应幂等: %+v", err)
	}

	//删除空目录
	emptyDir := path.Join(dir, "empty")
	if err := os.Mkdir(emptyDir, 0750); err != nil {
		t.Fatalf("%+v", err)
	}
	if err := RemoveFile(ctx, emptyDir); err != nil {
		t.Errorf("删除空目录异常: %+v", err)
	}
	if GetPathInfo(ctx, emptyDir) != nil {
		t.Errorf("空目录未被删除")
	}

	//非空目录必须拒绝删除，防止误删数据
	fullDir := path.Join(dir, "full")
	if err := os.Mkdir(fullDir, 0750); err != nil {
		t.Fatalf("%+v", err)
	}
	if err := os.WriteFile(path.Join(fullDir, "inner.txt"), []byte("x"), 0644); err != nil {
		t.Fatalf("%+v", err)
	}
	if err := RemoveFile(ctx, fullDir); err == nil {
		t.Errorf("删除非空目录应报错")
	}
	if GetPathInfo(ctx, fullDir) == nil {
		t.Errorf("非空目录被误删")
	}
}

func TestRead2LogByReader(t *testing.T) {
	ctx := GenCtx()

	//多行且末行有换行
	lines, err := Read2LogByReader(ctx, bufio.NewReader(strings.NewReader("a\nb\nc\n")), true)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	if len(lines) != 3 || lines[0] != "a" || lines[2] != "c" {
		t.Errorf("行数据 = %v, 期望 [a b c]", lines)
	}

	//关键回归：末行无换行不能丢
	lines, err = Read2LogByReader(ctx, bufio.NewReader(strings.NewReader("a\nb\nlast-no-newline")), true)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	if len(lines) != 3 || lines[2] != "last-no-newline" {
		t.Errorf("末行无换行 = %v, 期望包含 last-no-newline", lines)
	}
	//单行无换行
	lines, err = Read2LogByReader(ctx, bufio.NewReader(strings.NewReader("only")), true)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	if len(lines) != 1 || lines[0] != "only" {
		t.Errorf("单行无换行 = %v", lines)
	}

	//每行前后空白被裁剪
	lines, err = Read2LogByReader(ctx, bufio.NewReader(strings.NewReader("  a  \n\tb\t\n")), true)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	if len(lines) != 2 || lines[0] != "a" || lines[1] != "b" {
		t.Errorf("空白裁剪 = %v", lines)
	}

	//save=false 时不收集内容但仍需读完不报错
	lines, err = Read2LogByReader(ctx, bufio.NewReader(strings.NewReader("a\nb\n")), false)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	if len(lines) != 0 {
		t.Errorf("save=false 仍收集了数据: %v", lines)
	}

	//空输入
	lines, err = Read2LogByReader(ctx, bufio.NewReader(strings.NewReader("")), true)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	if len(lines) != 0 {
		t.Errorf("空输入 = %v", lines)
	}
	//纯空白行会被裁成空串，不计入结果
	lines, err = Read2LogByReader(ctx, bufio.NewReader(strings.NewReader("a\n\n  \nb\n")), true)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	//空行必须被完全跳过，只保留有效内容且保持原顺序
	if len(lines) != 2 || lines[0] != "a" || lines[1] != "b" {
		t.Errorf("空行未被跳过 = %#v, 期望 [a b]", lines)
	}

	//末尾无换行时，EOF分支与正常分支须同一套空行标准：
	//"a\n\nb"（末尾无换行）与"a\n\nb\n"（末尾有换行）结果必须一致
	withNewline, err := Read2LogByReader(ctx, bufio.NewReader(strings.NewReader("a\n\nb\n")), true)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	withoutNewline, err := Read2LogByReader(ctx, bufio.NewReader(strings.NewReader("a\n\nb")), true)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	if len(withNewline) != len(withoutNewline) {
		t.Errorf("末尾换行与否结果不一致: 有换行=%#v 无换行=%#v", withNewline, withoutNewline)
	}
	for i := range withNewline {
		if i < len(withoutNewline) && withNewline[i] != withoutNewline[i] {
			t.Errorf("末尾换行与否第%d行不一致: %q vs %q", i, withNewline[i], withoutNewline[i])
		}
	}

	//仅含空白行的输入必须返回空结果，不能收集出空串
	lines, err = Read2LogByReader(ctx, bufio.NewReader(strings.NewReader("\n  \n\t\n")), true)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	if len(lines) != 0 {
		t.Errorf("纯空白输入 = %#v, 期望空", lines)
	}
}

func TestOpenWriteFile(t *testing.T) {
	ctx := GenCtx()
	dir := newTestDir(t)

	//不存在时自动创建（写语义允许创建）
	filePath := path.Join(dir, "new", "w.txt")
	file, err := OpenWriteFile(ctx, filePath)
	if err != nil {
		t.Fatalf("OpenWriteFile 异常: %+v", err)
	}
	if _, err = file.WriteString("data"); err != nil {
		t.Errorf("写入异常: %+v", err)
	}
	CloseIo(ctx, file)
	if GetFileInfo(ctx, filePath) == nil {
		t.Errorf("文件未创建")
	}

	//已存在时打开并截断
	file, err = OpenWriteFile(ctx, filePath)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	CloseIo(ctx, file)
	got, err := ReadFile2String(ctx, filePath, "")
	if err != nil {
		t.Fatalf("%+v", err)
	}
	if got != "" {
		t.Errorf("打开写文件未截断旧内容: %q", got)
	}

	//目录必须报错
	if _, err = OpenWriteFile(ctx, dir); err == nil {
		t.Errorf("OpenWriteFile 对目录应报错")
	}
}

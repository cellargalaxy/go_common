package util

import (
	"path"
	"testing"
)

func TestIO(t *testing.T) {
	ctx := GenCtx()
	var err error

	folder := `TestIO`
	filename := `TestIO.txt`
	filepath := path.Join(folder, filename)
	filepath = ClearPath(ctx, filepath)

	err = WriteString2File(ctx, `TestIO`, filepath)
	if err != nil {
		t.Errorf("%+v", err)
		return
	}

	files, err := ListFile(ctx, folder)
	if err != nil {
		t.Errorf("%+v", err)
		return
	}
	if len(files) != 1 || files[0].Name() != filename {
		t.Errorf(`if len(files) != 1 || files[0].Name() != filename {`)
		return
	}

	text, err := ReadFile2String(ctx, filepath, "")
	if err != nil {
		t.Errorf("%+v", err)
		return
	}
	if text != "TestIO" {
		t.Errorf(`if text != "TestIO" {`)
		return
	}

	err = RemoveFile(ctx, filepath)
	if err != nil {
		t.Errorf("%+v", err)
		return
	}
	err = RemoveFile(ctx, folder)
	if err != nil {
		t.Errorf("%+v", err)
		return
	}
	fol := GetPathInfo(ctx, folder)
	if fol != nil {
		t.Errorf(`if fol != nil {`)
		return
	}
	file := GetPathInfo(ctx, filepath)
	if file != nil {
		t.Errorf(`if file != nil {`)
		return
	}
}

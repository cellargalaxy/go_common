package util

import "testing"

func TestXlsx(t *testing.T) {
	ctx := GenCtx()

	type Xlsx struct {
		Id   int    `csv:"id"`
		Name string `csv:"name"`
	}

	var err error
	var ss [][]string
	var list []Xlsx

	list = append(list, Xlsx{Id: 1, Name: "a"})
	list = append(list, Xlsx{Id: 2, Name: "b"})

	ss, err = CsvStruct2Strings(ctx, list)
	if err != nil {
		t.Errorf("%+v", err)
		return
	}

	data, err := XlsxStrings2Data(ctx, ss)
	if err != nil {
		t.Errorf("%+v", err)
		return
	}

	ss, err = XlsxData2Strings(ctx, data)
	if err != nil {
		t.Errorf("%+v", err)
		return
	}
	if len(ss) != 3 {
		t.Errorf(`len(ll) != 2`)
		return
	}
	if len(ss[0]) != 2 || len(ss[1]) != 2 || len(ss[2]) != 2 {
		t.Errorf(`len(ll) != 2`)
		return
	}
	if ss[0][0] != "id" || ss[0][1] != "name" {
		t.Errorf(`len(ll) != 2`)
		return
	}
	if ss[1][0] != "1" || ss[1][1] != "a" {
		t.Errorf(`len(ll) != 2`)
		return
	}
	if ss[2][0] != "2" || ss[2][1] != "b" {
		t.Errorf(`len(ll) != 2`)
		return
	}
}

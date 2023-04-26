package util

import (
	"testing"
)

func TestCsv(t *testing.T) {
	ctx := GenCtx()

	type Csv struct {
		Id   int    `csv:"id"`
		Name string `csv:"name"`
	}
	var err error
	var data []byte
	var ss [][]string
	var s string
	var list []Csv
	var ll []Csv

	list = append(list, Csv{Id: 1, Name: "a"})
	list = append(list, Csv{Id: 2, Name: "b"})

	data, err = CsvStruct2Data(ctx, list)
	if err != nil {
		t.Errorf("%+v", err)
		return
	}
	ll = make([]Csv, 0)
	err = CsvData2Struct(ctx, data, &ll)
	if err != nil {
		t.Errorf("%+v", err)
		return
	}
	if len(ll) != 2 || ll[0].Id != 1 || ll[0].Name != "a" || ll[1].Id != 2 || ll[1].Name != "b" {
		t.Errorf(`len(ll) != 2`)
		return
	}

	ss, err = CsvStruct2Strings(ctx, list)
	if err != nil {
		t.Errorf("%+v", err)
		return
	}
	ll = make([]Csv, 0)
	err = CsvStrings2Struct(ctx, ss, &ll)
	if err != nil {
		t.Errorf("%+v", err)
		return
	}
	if len(ll) != 2 || ll[0].Id != 1 || ll[0].Name != "a" || ll[1].Id != 2 || ll[1].Name != "b" {
		t.Errorf(`len(ll) != 2`)
		return
	}

	s, err = CsvStruct2String(ctx, list)
	if err != nil {
		t.Errorf("%+v", err)
		return
	}
	ll = make([]Csv, 0)
	err = CsvString2Struct(ctx, s, &ll)
	if err != nil {
		t.Errorf("%+v", err)
		return
	}
	if len(ll) != 2 || ll[0].Id != 1 || ll[0].Name != "a" || ll[1].Id != 2 || ll[1].Name != "b" {
		t.Errorf(`len(ll) != 2`)
		return
	}

}

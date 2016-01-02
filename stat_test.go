package main

import (
	"path/filepath"
	"testing"
)

var dsi = dirStatInfo{
	Count: 100,
}

func TestDirStatInfoHumanReadableSize(t *testing.T) {
	var tests = []struct {
		input int64
		want  string
	}{
		{0, "0B"},
		{500, "500B"},
		{1024 - 1, "1023B"},
		{1024, "1KB"},
		{1024*1024 - 1, "1023KB"},
		{1024 * 1024, "1MB"},
		{1024*1024*1024 - 1, "1023MB"},
		{1024 * 1024 * 1024, "1GB"},
		{1024*1024*1024*1024 - 1, "1023GB"},
		{1024 * 1024 * 1024 * 1024, "1TB"},
	}

	for _, test := range tests {
		dsi.TotalSize = test.input
		if dsi.HumanReadableSize() != test.want {
			t.Errorf(`%d != %s`, test.input, test.want)
		}
	}
}

func TestDirStatInfoLastProcessingDateStr(t *testing.T) {
	var tm int64 = 1451711058 // 02.01.2016 15:04:18
	dsi.LastProcessingDate = tm
	if dsi.LastProcessingDateStr() != "15:04:18" {
		t.Errorf(`LastProcessingDateStr, %d != "%s"`, tm, "15:04:18")
	}
}

func TestTosserStat(t *testing.T) {
	ts := NewTosserStat("tmp.json")
	if ts == nil {
		t.Error("ts == nil")
	}
	if ts.ConfigName != "tmp.json" {
		t.Error(`ts.ConfigName != "tmp.json"`)
	}

	file := "c:\\1.txt"
	dir := filepath.Dir(file)
	ts.update(file, 100)
	for date := range ts.Dates {
		if ts.Dates[date][dir].Count != 1 {
			t.Error(`ts.Dates[date][dir].Count != 1`)
		}
		if ts.Dates[date][dir].TotalSize != 100 {
			t.Error(`ts.Dates[date][dir].TotalSize != 100`)
		}
	}
}

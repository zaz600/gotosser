package main

import (
	"testing"
	"time"
)

//CopyRule
var rule = CopyRule{
	Masks:        []string{"*.exe", "*.dll", "*.txt"},
	DstDir:       "c:\\tmp\\",
	IfExists:     "replace",
	Mode:         "move",
	ExcludeMasks: []string{"*.tmp"},
}

func TestCopyRuleMatch(t *testing.T) {
	//файл должен попасть под маску правила
	match, masks := rule.match("123.exe")
	if !match {
		t.Errorf(`rule.match("123.exe"), match != true`)
	}

	if len(masks) == 0 {
		t.Errorf(`len(masks)==0`)
	}
	//файл не должен попасть под маску правила
	match, _ = rule.match("123.xml")
	if match {
		t.Errorf(`rule.match("123.xml"), match != false`)
	}
}

func TestCopyRuleMatchExclude(t *testing.T) {
	//файл должен попасть под маску исключения
	match := rule.matchExclude("123.tmp")
	if !match {
		t.Errorf(`rule.matchExclude ("123.tmp"), match != true`)
	}
	//файл не должен попасть под маску исключения
	match = rule.matchExclude("123.xml")
	if match {
		t.Errorf(`rule.matchExclude("123.xml"), match != false`)
	}
}

//ScanGroup
var sg = ScanGroup{
	SrcDirs:      []string{"c:\\1", "c:\\2"},
	Enabled:      true,
	Rules:        map[int]CopyRule{0: rule, 1: rule},
	СreateSrc:    true,
	ExcludeMasks: []string{"*.tmp"},
}

func TestScanGroupGetRuleKeys(t *testing.T) {
	k := sg.getRuleKeys()
	if len(k) != 2 {
		t.Errorf(`len(k) != 2`)
	}

	//должны быть отсортированы
	if k[0] != 0 || k[1] != 1 {
		t.Errorf(`k[0] !=0 || k[1] != 1`)
	}
}

func TestScanGroupMatchExclude(t *testing.T) {
	//файл должен попасть под маску исключения
	match := sg.matchExclude("123.tmp")
	if !match {
		t.Errorf(`sg.matchExclude ("123.tmp"), match != true`)
	}
	//файл не должен попасть под маску исключения
	match = sg.matchExclude("123.xml")
	if match {
		t.Errorf(`sg.matchExclude("123.xml"), match != false`)
	}
}

//Config
var c = Config{
	ScanGroups:         []ScanGroup{sg},
	GlobalExcludeMasks: []string{"*.tmp"},
}

func TestConfigMatchExclude(t *testing.T) {
	//файл должен попасть под маску исключения
	match := c.matchExclude("123.tmp")
	if !match {
		t.Errorf(`c.matchExclude ("123.tmp"), match != true`)
	}
	//файл не должен попасть под маску исключения
	match = c.matchExclude("123.xml")
	if match {
		t.Errorf(`c.matchExclude("123.xml"), match != false`)
	}
}

func TestReadConfig(t *testing.T) {
	//загружаем несуществующий файл
	_, err := readConfig("123321.yaml")
	if err == nil {
		t.Error(`readConfig("123321.yaml"), err == nil`)
	}
	//загружаем существующий конфиг
	_, err = readConfig("gotosser.yaml.example")
	if err != nil {
		t.Error(`readConfig("gotosser.yaml.example"), err != nil`)
	}
}

func TestReloadConfig(t *testing.T) {
	//загружаем несуществующий файл
	_, err := reloadConfig("123321.yaml")
	if err == nil {
		t.Error(`reloadConfig("123321.yaml"), err == nil`)
	}
	//загружаем существующий конфиг первый раз
	_, err = reloadConfig("gotosser.yaml.example")
	if err != nil {
		t.Error(`reloadConfig("gotosser.yaml.example"), err != nil`)
	}
	//перезагружаем. дата файла не изменилась
	time.Sleep(100 * time.Millisecond)
	_, err = reloadConfig("gotosser.yaml.example")
	if err != errNotModified {
		t.Error(`reloadConfig("gotosser.yaml.example"), err != errNotModified`)
	}
	//меняем время изменения файла и перезагружаем его
	configModtime -= 600
	time.Sleep(100 * time.Millisecond)
	_, err = reloadConfig("gotosser.yaml.example")
	if err != nil {
		t.Error(`reloadConfig("gotosser.yaml.example"), err != nil`)
	}
}

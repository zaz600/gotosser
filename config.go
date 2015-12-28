//config.go
package main

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v2"
)

var (
	configModtime  int64
	errNotModified = errors.New("Not modified")
)

//CopyRule - хранит правила копирования для ScanDir
type CopyRule struct {
	Masks        []string `yaml:"masks"`
	DstDir       string   `yaml:"dst_dir"`
	IfExists     string   `yaml:"ifexists"`
	Mode         string   `yaml:"mode"`
	ExcludeMasks []string `yaml:"excludemasks"`
}

//ScanGroup - хранит настройки для папки, которая будет сканироваться
type ScanGroup struct {
	SrcDirs      []string         `yaml:"src_dirs"`
	Enabled      bool             `yaml:"enabled"`
	Rules        map[int]CopyRule `yaml:"rules"`
	СreateSrc    bool             `yaml:"create_src"`
	ExcludeMasks []string         `yaml:"excludemasks"`
}

// Config - структура для считывания конфигурационного файла
type Config struct {
	ScanGroups         []ScanGroup `yaml:"scangroups"`
	EnableHTTP         bool        `yaml:"enable_http"`
	Listen             string      `yaml:"listen"`
	MaxScanThreads     int         `yaml:"max_scan_threads"`
	MaxCopyThreads     int         `yaml:"max_copy_threads"`
	RescanInterval     int         `yaml:"rescaninterval"`
	LogLevel           string      `yaml:"loglevel"`
	GlobalExcludeMasks []string    `yaml:"excludemasks"`
	LogFilename        string      `yaml:"log_filename"`
	LogMaxSize         int         `yaml:"log_max_size"`
	LogMaxBackups      int         `yaml:"log_max_backups"`
	LogMaxAge          int         `yaml:"log_max_age"`
}

func readConfig(ConfigName string) (x *Config, err error) {
	var file []byte
	if file, err = ioutil.ReadFile(ConfigName); err != nil {
		return nil, err
	}
	x = new(Config)
	if err = yaml.Unmarshal(file, x); err != nil {
		return nil, err
	}
	if x.LogLevel == "" {
		x.LogLevel = "Debug"
	}
	return x, nil
}

//Проверяет время изменения конфигурационного файла
//и перезагружает его если он изменился
//Возвращает errNotModified если изменений нет
func reloadConfig(configName string) (cfg *Config, err error) {
	info, err := os.Stat(configName)
	if err != nil {
		return nil, err
	}
	if configModtime != info.ModTime().UnixNano() {
		configModtime = info.ModTime().UnixNano()
		cfg, err = readConfig(configName)
		if err != nil {
			return nil, err
		}
		return cfg, nil
	}
	return nil, errNotModified
}

func (sd ScanGroup) getRuleKeys() []int {
	var keys []int
	for k := range sd.Rules {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	return keys
}

//проверяет подходит ли файл под маски данного правила
//возвращает список масок
func (r CopyRule) match(srcFile string) (bool, []string) {
	var masks []string
	for _, mask := range r.Masks {
		matched, err := filepath.Match(strings.ToLower(mask), strings.ToLower(srcFile))
		if err != nil {
			errorf("Ошибка проверки MASK (%s). %s", mask, err)
			continue
		}
		if matched {
			masks = append(masks, mask)
		}
	}
	if len(masks) == 0 {
		return false, masks
	}
	return true, masks
}

/*проверки на исключение*/
//проверка глобального списка исключений
func (c Config) matchExclude(srcFile string) bool {
	for _, mask := range c.GlobalExcludeMasks {
		matched, err := filepath.Match(strings.ToLower(mask), strings.ToLower(srcFile))
		if err != nil {
			errorf("Ошибка проверки MASK (%s). %s", mask, err)
			continue
		}
		if matched {
			return true
		}
	}
	return false
}

//проверка списка исключений группы
func (sd ScanGroup) matchExclude(srcFile string) bool {
	for _, mask := range sd.ExcludeMasks {
		matched, err := filepath.Match(strings.ToLower(mask), strings.ToLower(srcFile))
		if err != nil {
			errorf("Ошибка проверки MASK (%s). %s", mask, err)
			continue
		}
		if matched {
			return true
		}
	}
	return false
}

//проверка списка исключений правила
func (r CopyRule) matchExclude(srcFile string) bool {
	for _, mask := range r.ExcludeMasks {
		matched, err := filepath.Match(strings.ToLower(mask), strings.ToLower(srcFile))
		if err != nil {
			errorf("Ошибка проверки MASK (%s). %s", mask, err)
			continue
		}
		if matched {
			return true
		}
	}
	return false
}

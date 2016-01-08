package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/Sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	histLength = 10
)

var (
	log              = logrus.New()
	fileLog          = logrus.New()
	lumberjackLogger *lumberjack.Logger
	errorHistory     = newErrorHistoryStore(histLength)
	initOnce         sync.Once
)

func initLogger(cfg *Config) error {
	initOnce.Do(func() {
		//лог скопированных файлов. при перезагрузке конфига не меняется
		logDir := filepath.Dir(cfg.LogFilename)
		filelogFilePath := filepath.Join(logDir, "files.log")
		if err := os.MkdirAll(logDir, os.ModeDir); err != nil {
			log.Fatalln("Ошибка создания каталога", logDir, err)
		}
		file, err := os.OpenFile(filelogFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			log.Fatalln(err)
		}
		fileLog.Level = logrus.InfoLevel
		fileLog.Out = file
	})
	switch strings.ToUpper(cfg.LogLevel) {
	case "DEBUG":
		log.Level = logrus.DebugLevel
	case "INFO":
		log.Level = logrus.InfoLevel
	case "ERROR":
		log.Level = logrus.ErrorLevel
	default:
		return fmt.Errorf("Неизвестный уровень лога, %s", cfg.LogLevel)
	}

	if lumberjackLogger != nil {
		lumberjackLogger.Close()
	}
	lumberjackLogger = &lumberjack.Logger{
		Filename:   cfg.LogFilename,
		MaxSize:    cfg.LogMaxSize, // megabytes
		MaxBackups: cfg.LogMaxBackups,
		MaxAge:     cfg.LogMaxAge, //days
		LocalTime:  true,
	}
	log.Formatter = &logrus.TextFormatter{TimestampFormat: "2006-01-02 15:04:05"}
	multi := io.MultiWriter(lumberjackLogger, os.Stderr)
	log.Out = multi

	log.Infoln("Уровень логирования", cfg.LogLevel)
	return nil
}

func errorln(v ...interface{}) {
	s := fmt.Sprintln(v...)
	log.Error(s)
	errorHistory.Add(s)
}

func errorf(format string, v ...interface{}) {
	s := fmt.Sprintf(format, v...)
	log.Error(s)
	errorHistory.Add(s)
}

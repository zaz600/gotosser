package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

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
	errorHistory     []string
	syncOnce         = new(sync.Once)
)

func initLogger(cfg *Config) error {
	syncOnce.Do(func() {
		lumberjackLogger = &lumberjack.Logger{
			Filename:   "logs/gotosser.log",
			MaxSize:    100, // megabytes
			MaxBackups: 10,
			MaxAge:     30, //days
			LocalTime:  true,
		}
		log.Formatter = &logrus.TextFormatter{TimestampFormat: "2006-01-02 15:04:05"}
		multi := io.MultiWriter(lumberjackLogger, os.Stderr)
		log.Out = multi

		//лог скопированных файлов
		file, err := os.OpenFile("logs/files.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
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
		log.Fatalln("Неизвестные уровень лога", cfg.LogLevel)
	}

	log.Infoln("Уровень логирования", cfg.LogLevel)
	return nil
}

func errorln(v ...interface{}) {
	s := fmt.Sprint(v...)
	log.Error(s)
	saveErrorHistory(s)
}

func errorf(format string, v ...interface{}) {
	s := fmt.Sprintf(format, v...)
	log.Error(s)
	saveErrorHistory(s)
}

func saveErrorHistory(s string) {
	tm := time.Now().Format("2006-01-02 15:04:05")
	errorHistory = append(errorHistory, fmt.Sprintf("%s %s", tm, s))
	if len(errorHistory) > histLength {
		errorHistory = errorHistory[1:]
	}
}

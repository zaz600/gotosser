package main

import (
	"fmt"
	"io"
	"os"
	"strings"
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
)

func init() {
	//лог скопированных файлов. при перезагрузке конфига не меняется
	file, err := os.OpenFile("logs/files.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalln(err)
	}
	fileLog.Level = logrus.InfoLevel
	fileLog.Out = file
}

func initLogger(cfg *Config) error {
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

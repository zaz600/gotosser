package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	histLength = 10
)

var (
	Info             *log.Logger
	Error            *log.Logger
	Debug            *log.Logger
	FileLog          *log.Logger
	LumberjackLogger *lumberjack.Logger
	errorHistory     []string
	syncOnce         = new(sync.Once)
)

func initLogger(cfg *Config) error {
	syncOnce.Do(func() {
		LumberjackLogger = &lumberjack.Logger{
			Filename:   "logs/gotosser.log",
			MaxSize:    30, // megabytes
			MaxBackups: 5,
			MaxAge:     30, //days
			LocalTime:  true,
		}
	})

	if err := LumberjackLogger.Rotate(); err != nil {
		return fmt.Errorf("Failed to open log file: %s", err)
	}
	if strings.ToUpper(cfg.LogLevel) == "DEBUG" {
		multi := io.MultiWriter(LumberjackLogger, os.Stdout)
		Debug = log.New(multi, "DEBUG: ", log.Ldate|log.Ltime)
		Info = log.New(multi, "INFO:  ", log.Ldate|log.Ltime)
	} else if strings.ToUpper(cfg.LogLevel) == "INFO" {
		multi := io.MultiWriter(LumberjackLogger, os.Stdout)
		Debug = log.New(ioutil.Discard, "DEBUG: ", log.Ldate|log.Ltime)
		Info = log.New(multi, "INFO:  ", log.Ldate|log.Ltime)
	} else if strings.ToUpper(cfg.LogLevel) == "ERROR" {
		Debug = log.New(ioutil.Discard, "DEBUG: ", log.Ldate|log.Ltime)
		Info = log.New(ioutil.Discard, "INFO:  ", log.Ldate|log.Ltime)
	}
	Info.Println("Уровень логирования", cfg.LogLevel)

	multi := io.MultiWriter(LumberjackLogger, os.Stderr)
	Error = log.New(multi, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)

	file, err := os.OpenFile("logs/files.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return fmt.Errorf("Failed to open files.log file: %s", err)
	}
	FileLog = log.New(file, "", log.Ldate|log.Ltime)
	return nil
}

func errorln(v ...interface{}) {
	s := fmt.Sprintln(v...)
	Error.Print(s)
	saveErrorHistory(s)
}

func errorf(format string, v ...interface{}) {
	s := fmt.Sprintf(format, v...)
	Error.Println(s)
	saveErrorHistory(s)
}

func saveErrorHistory(s string) {
	tm := time.Now().Format("2006-01-02 15:04:05")
	errorHistory = append(errorHistory, fmt.Sprintf("%s %s", tm, s))
	if len(errorHistory) > histLength {
		errorHistory = errorHistory[1:]
	}
}

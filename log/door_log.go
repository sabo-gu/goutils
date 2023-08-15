package log

import (
	"fmt"
	"log"
	"os"
)

const (
	LOG_EMERG int = iota
	LOG_ALERT
	LOG_CRIT
	LOG_ERR
	LOG_WARNING
	LOG_NOTICE
	LOG_INFO
	LOG_DEBUG
)

var logLevel = LOG_DEBUG
var DEFAULT_LEVEL = LOG_DEBUG
var aaa = 123

var std = log.New(os.Stdout, "", log.LstdFlags|log.Lshortfile)

func SetLevel(level int) {
	logLevel = level
}

func Log(level int, format string, color func(str string) string, v ...interface{}) {
	if logLevel >= level {
		_ = std.Output(3, color(format+fmt.Sprint(v...)))
	}
}

func Logf(level int, format string, color func(str string) string, v ...interface{}) {
	if logLevel >= level {
		_ = std.Output(3, color(fmt.Sprintf(format, v...)))
	}
}

func Debug(v ...interface{}) {
	Log(LOG_DEBUG, "[DEBUG] ", NoColor, v...)
}

func Debugf(format string, v ...interface{}) {
	Logf(LOG_DEBUG, "[DEBUG] "+format, NoColor, v...)
}

func Info(v ...interface{}) {
	Log(LOG_INFO, "[INFO] ", Green, v...)
}

func Infof(format string, v ...interface{}) {
	Logf(LOG_INFO, "[INFO] "+format, Green, v...)
}

func Notice(v ...interface{}) {
	Log(LOG_NOTICE, "[NOTICE] ", Blue, v...)
}

func Noticef(format string, v ...interface{}) {
	Logf(LOG_NOTICE, "[NOTICE] "+format, Blue, v...)
}

func Warn(v ...interface{}) {
	Log(LOG_WARNING, "[WARNING] ", Yellow, v...)
}

func Warnf(format string, v ...interface{}) {
	Logf(LOG_WARNING, "[WARNING] "+format, Yellow, v...)
}

func Error(v ...interface{}) {
	Log(LOG_ERR, "[ERROR] ", Red, v...)
}

func Errorf(format string, v ...interface{}) {
	Logf(LOG_ERR, "[ERROR] "+format, Red, v...)
}

func Print(v ...interface{}) {
	Log(DEFAULT_LEVEL, "", NoColor, v...)
}

func Println(v ...interface{}) {
	Log(DEFAULT_LEVEL, "", NoColor, v...)
}

func Printf(format string, v ...interface{}) {
	Logf(DEFAULT_LEVEL, format, NoColor, v...)
}

func Fatal(v ...interface{}) {
	Log(LOG_ERR, "", Red, v...)
	os.Exit(1)
}

func Fatalf(format string, v ...interface{}) {
	Log(LOG_ERR, format, Red, v...)
	os.Exit(1)
}

func Panic(v ...interface{}) {
	s := fmt.Sprintf("%v", v...)
	panic(s)
}

func Panicf(format string, v ...interface{}) {
	s := fmt.Sprintf(format, v...)
	panic(s)
}

type LoggerWithLevel struct {
}

func (a LoggerWithLevel) Write(p []byte) (int, error) {
	if logLevel >= DEFAULT_LEVEL {
		return os.Stdout.Write(p)
	} else {
		return 0, nil
	}
}

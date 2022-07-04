package logger

import (
	"errors"
	"fmt"
	"sync"
)

var (
	loggerNotDefinedErr     = errors.New("logger not defined")
	loggerAlreadyDefinedErr = errors.New("logger already defined")
)

var once sync.Once

type logger struct {
	loggerI
}

var loggerInstance *logger

//func GetInstance() *logger {
//	if loggerInstance == nil {
//		panic("logger not defined")
//		return nil
//	}
//	return loggerInstance
//}

func NewInstance(i loggerI) error {
	if loggerInstance == nil {
		once.Do(func() {
			loggerInstance = &logger{i}
		})
		return nil
	}
	loggerInstance.Error(loggerAlreadyDefinedErr.Error())
	return loggerAlreadyDefinedErr

}

func checkLoggerSetup() {
	if loggerInstance == nil {
		panic("logger not defined")
	}
}
func Debug(a ...any) {
	checkLoggerSetup()
	loggerInstance.loggerI.Debug(fmt.Sprint(a...))
}

func Info(a ...any) {
	checkLoggerSetup()
	loggerInstance.loggerI.Info(fmt.Sprint(a...))
}

func Error(a ...any) {
	checkLoggerSetup()
	loggerInstance.loggerI.Error(fmt.Sprint(a...))
}

func Fatal(a ...any) {
	checkLoggerSetup()
	loggerInstance.loggerI.Debug(fmt.Sprint(a...))
}

func DebugF(format string, a ...any) {
	checkLoggerSetup()
	loggerInstance.loggerI.Debug(fmt.Sprintf(format, a...))
}

func InfoF(format string, a ...any) {
	checkLoggerSetup()
	loggerInstance.loggerI.Info(fmt.Sprintf(format, a...))
}

func ErrorF(format string, a ...any) {
	checkLoggerSetup()
	loggerInstance.loggerI.Error(fmt.Sprintf(format, a...))
}

func FatalF(format string, a ...any) {
	checkLoggerSetup()
	loggerInstance.loggerI.Fatal(fmt.Sprintf(format, a...))
}

type loggerI interface {
	Debug(str string)
	Info(str string)
	Error(str string)
	Fatal(str string)
}

package agent

import (
	"github.com/Sirupsen/logrus"
	"os"
)

//统一日志接口
type Logger interface {
	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Printf(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Warningf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Fatalf(format string, args ...interface{})
	Panicf(format string, args ...interface{})
	Debug(args ...interface{})
	Info(args ...interface{})
	Print(args ...interface{})
	Warn(args ...interface{})
	Warning(args ...interface{})
	Error(args ...interface{})
	Fatal(args ...interface{})
	Panic(args ...interface{})
	Debugln(args ...interface{})
	Infoln(args ...interface{})
	Println(args ...interface{})
	Warnln(args ...interface{})
	Warningln(args ...interface{})
	Errorln(args ...interface{})
	Fatalln(args ...interface{})
	Panicln(args ...interface{})
}

//获取日志输出器
func GetLogger() Logger {
	return defaultLogger
}

//默认日志输出
var defaultLogger *logrus.Logger

func init() {
	defaultLogger = logrus.New()
	SetLogOut(defaultLogger, "stdout")
	SetLogLevel(defaultLogger, "debug")
	defaultLogger.Formatter = &logrus.TextFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
		FullTimestamp:   true,
	}
}

//设置日志输出方式
func SetLogOut(log *logrus.Logger, outString string) error {
	switch outString {
	case "stdout":
		log.Out = os.Stdout
	case "stderr":
		log.Out = os.Stderr
	default:
		f, err := os.OpenFile(outString, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)

		if err != nil {
			return err
		}

		log.Out = f
	}

	return nil
}

//设置日志级别
func SetLogLevel(log *logrus.Logger, levelString string) error {
	level, err := logrus.ParseLevel(levelString)

	if err != nil {
		return err
	}

	log.Level = level
	return nil
}

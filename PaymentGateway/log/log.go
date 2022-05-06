package log

import (
	"io"
	"os"

	"github.com/sirupsen/logrus"
)

type Logger interface {
	Debug(msg string)
	Info(msg string)
	Warn(msg string)
	Error(msg string)
	Errorf(msg string, args ...interface{})
	Debugf(msg string, args ...interface{})
	Fatal(msg string)
}

func init() {
	// Log as JSON instead of the default ASCII formatter.
	//logger.SetFormatter(&logrus.JSONFormatter{})

	// Output to stdout instead of the default stderr
	// Can be any io.Writer, see below for File example
	logger.SetOutput(os.Stdout)

	// Only log the warning severity or above.
	//logger.SetLevel(logrus.WarnLevel)
}

var logger *logrus.Logger = logrus.New()

func Writer() io.Writer {
	return logger.Writer()
}
func New() *logrus.Logger {

	return logger
}
func Debug(msg string) {
	logger.Debug(msg)
}

func Tracef(format string, args ...interface{}) {
	logger.Tracef(format, args)
}

func Debugf(format string, args ...interface{}) {
	logger.Debugf(format, args...)
}

func Infof(format string, args ...interface{}) {
	logger.Infof(format, args...)
}

func Warnf(format string, args ...interface{}) {
	logger.Warnf(format, args...)
}

func Warningf(format string, args ...interface{}) {
	logger.Warnf(format, args...)
}

func Errorf(format string, args ...interface{}) {
	logger.Errorf(format, args...)
}

func Fatalf(format string, args ...interface{}) {
	logger.Fatalf(format, args...)

}

func Panicf(format string, args ...interface{}) {
	logger.Panicf(format, args...)
}

func Trace(args ...interface{}) {
	logger.Trace(args...)
}

func Info(args ...interface{}) {
	logger.Info(args...)
}

func Print(args ...interface{}) {
	logger.Print(args...)
}

func Warn(args ...interface{}) {
	logger.Warn(args...)
}

func Warning(args ...interface{}) {
	logger.Warn(args...)
}

func Error(args ...interface{}) {
	logger.Error(args...)
}

func Fatal(args ...interface{}) {
	logger.Fatal(args...)
}

func Panic(args ...interface{}) {
	logger.Panic(args...)
}

func Traceln(args ...interface{}) {
	logger.Traceln(args...)
}

func Debugln(args ...interface{}) {
	logger.Debugln(args...)
}

func Infoln(args ...interface{}) {
	logger.Infoln(args...)
}

func Println(args ...interface{}) {
	logger.Println(args...)
}

func Warnln(args ...interface{}) {
	logger.Warnln(args...)
}

func Warningln(args ...interface{}) {
	logger.Warnln(args...)
}

func Errorln(args ...interface{}) {
	logger.Errorln(args...)
}

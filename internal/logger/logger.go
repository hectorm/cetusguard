package logger

import (
	"io"
	"log"
	"math"
	"os"
)

// Syslog log levels from RFC 5424,
// https://datatracker.ietf.org/doc/html/rfc5424
const (
	LvlNone     = 0
	LvlCritical = 2
	LvlError    = 3
	LvlWarning  = 4
	LvlInfo     = 6
	LvlDebug    = 7
)

var (
	lgrCritical    *log.Logger
	lgrError       *log.Logger
	lgrWarning     *log.Logger
	lgrInfo        *log.Logger
	lgrDebug       *log.Logger
	level          int
	exitOnCritical bool
)

func init() {
	lgrCritical = log.New(io.Discard, "CRITICAL: ", log.Ldate|log.Ltime)
	lgrError = log.New(io.Discard, "ERROR: ", log.Ldate|log.Ltime)
	lgrWarning = log.New(io.Discard, "WARNING: ", log.Ldate|log.Ltime)
	lgrInfo = log.New(io.Discard, "INFO: ", log.Ldate|log.Ltime)
	lgrDebug = log.New(io.Discard, "DEBUG: ", log.Ldate|log.Ltime)
	SetLevel(LvlInfo)
	exitOnCritical = true
}

func Level() int {
	return level
}

func SetLevel(val int) {
	level = int(math.Min(math.Max(float64(val), LvlNone), LvlDebug))
	if level >= LvlCritical {
		lgrCritical.SetOutput(os.Stderr)
	} else {
		lgrCritical.SetOutput(io.Discard)
	}
	if level >= LvlError {
		lgrError.SetOutput(os.Stderr)
	} else {
		lgrError.SetOutput(io.Discard)
	}
	if level >= LvlWarning {
		lgrWarning.SetOutput(os.Stderr)
	} else {
		lgrWarning.SetOutput(io.Discard)
	}
	if level >= LvlInfo {
		lgrInfo.SetOutput(os.Stdout)
	} else {
		lgrInfo.SetOutput(io.Discard)
	}
	if level >= LvlDebug {
		lgrDebug.SetOutput(os.Stdout)
	} else {
		lgrDebug.SetOutput(io.Discard)
	}
}

func LgrCritical() *log.Logger {
	return lgrCritical
}

func LgrError() *log.Logger {
	return lgrError
}

func LgrWarning() *log.Logger {
	return lgrWarning
}

func LgrInfo() *log.Logger {
	return lgrInfo
}

func LgrDebug() *log.Logger {
	return lgrDebug
}

func Critical(v ...any) {
	lgrCritical.Print(v...)
	if exitOnCritical {
		os.Exit(1)
	}
}

func Criticalf(format string, v ...any) {
	lgrCritical.Printf(format, v...)
	if exitOnCritical {
		os.Exit(1)
	}
}

func Criticalln(v ...any) {
	lgrCritical.Println(v...)
	if exitOnCritical {
		os.Exit(1)
	}
}

func Error(v ...any) {
	lgrError.Print(v...)
}

func Errorf(f string, v ...any) {
	lgrError.Printf(f, v...)
}

func Errorln(v ...any) {
	lgrError.Println(v...)
}

func Warning(v ...any) {
	lgrWarning.Print(v...)
}

func Warningf(f string, v ...any) {
	lgrWarning.Printf(f, v...)
}

func Warningln(v ...any) {
	lgrWarning.Println(v...)
}

func Info(v ...any) {
	lgrInfo.Print(v...)
}

func Infof(f string, v ...any) {
	lgrInfo.Printf(f, v...)
}

func Infoln(v ...any) {
	lgrInfo.Println(v...)
}

func Debug(v ...any) {
	lgrDebug.Print(v...)
}

func Debugf(f string, v ...any) {
	lgrDebug.Printf(f, v...)
}

func Debugln(v ...any) {
	lgrDebug.Println(v...)
}

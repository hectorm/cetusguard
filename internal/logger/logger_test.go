package logger

import (
	"bytes"
	"regexp"
	"testing"
)

func TestSetLevel(t *testing.T) {
	SetLevel(LvlNone)
	if Level() != LvlNone {
		t.Errorf("level = %d, want %d", Level(), LvlNone)
	}

	SetLevel(-99)
	if Level() != LvlNone {
		t.Errorf("level = %d, want %d", Level(), LvlNone)
	}

	SetLevel(LvlDebug)
	if Level() != LvlDebug {
		t.Errorf("level = %d, want %d", Level(), LvlDebug)
	}

	SetLevel(99)
	if Level() != LvlDebug {
		t.Errorf("level = %d, want %d", Level(), LvlDebug)
	}
}

func TestLoggerCritical(t *testing.T) {
	buf := new(bytes.Buffer)
	LgrCritical().SetOutput(buf)
	exitOnCritical = false

	Critical("FOO", "BAR")
	if !regexp.MustCompile(`^CRITICAL: .+ FOOBAR`).MatchString(buf.String()) {
		t.Errorf("unexpected Critical log output: %s", buf)
	}
}

func TestLoggerCriticalf(t *testing.T) {
	buf := new(bytes.Buffer)
	LgrCritical().SetOutput(buf)
	exitOnCritical = false

	Criticalf("%s %s", "FOO", "BAR")
	if !regexp.MustCompile(`^CRITICAL: .+ FOO BAR`).MatchString(buf.String()) {
		t.Errorf("unexpected Criticalf log output: %s", buf)
	}
}

func TestLoggerCriticalln(t *testing.T) {
	buf := new(bytes.Buffer)
	LgrCritical().SetOutput(buf)
	exitOnCritical = false

	Criticalln("FOO", "BAR")
	if !regexp.MustCompile(`^CRITICAL: .+ FOO BAR`).MatchString(buf.String()) {
		t.Errorf("unexpected Criticaln log output: %s", buf)
	}
}

func TestLoggerError(t *testing.T) {
	buf := new(bytes.Buffer)
	LgrError().SetOutput(buf)

	Error("FOO", "BAR")
	if !regexp.MustCompile(`^ERROR: .+ FOOBAR`).MatchString(buf.String()) {
		t.Errorf("unexpected Error log output: %s", buf)
	}
}

func TestLoggerErrorf(t *testing.T) {
	buf := new(bytes.Buffer)
	LgrError().SetOutput(buf)

	Errorf("%s %s", "FOO", "BAR")
	if !regexp.MustCompile(`^ERROR: .+ FOO BAR`).MatchString(buf.String()) {
		t.Errorf("unexpected Errorf log output: %s", buf)
	}
}

func TestLoggerErrorln(t *testing.T) {
	buf := new(bytes.Buffer)
	LgrError().SetOutput(buf)

	Errorln("FOO", "BAR")
	if !regexp.MustCompile(`^ERROR: .+ FOO BAR`).MatchString(buf.String()) {
		t.Errorf("unexpected Errorln log output: %s", buf)
	}
}

func TestLoggerWarning(t *testing.T) {
	buf := new(bytes.Buffer)
	LgrWarning().SetOutput(buf)

	Warning("FOO", "BAR")
	if !regexp.MustCompile(`^WARNING: .+ FOOBAR`).MatchString(buf.String()) {
		t.Errorf("unexpected Warning log output: %s", buf)
	}
}

func TestLoggerWarningf(t *testing.T) {
	buf := new(bytes.Buffer)
	LgrWarning().SetOutput(buf)

	Warningf("%s %s", "FOO", "BAR")
	if !regexp.MustCompile(`^WARNING: .+ FOO BAR`).MatchString(buf.String()) {
		t.Errorf("unexpected Warningf log output: %s", buf)
	}
}

func TestLoggerWarningln(t *testing.T) {
	buf := new(bytes.Buffer)
	LgrWarning().SetOutput(buf)

	Warningln("FOO", "BAR")
	if !regexp.MustCompile(`^WARNING: .+ FOO BAR`).MatchString(buf.String()) {
		t.Errorf("unexpected Warningln log output: %s", buf)
	}
}

func TestLoggerInfo(t *testing.T) {
	buf := new(bytes.Buffer)
	LgrInfo().SetOutput(buf)

	Info("FOO", "BAR")
	if !regexp.MustCompile(`^INFO: .+ FOOBAR`).MatchString(buf.String()) {
		t.Errorf("unexpected Info log output: %s", buf)
	}
}

func TestLoggerInfof(t *testing.T) {
	buf := new(bytes.Buffer)
	LgrInfo().SetOutput(buf)

	Infof("%s %s", "FOO", "BAR")
	if !regexp.MustCompile(`^INFO: .+ FOO BAR`).MatchString(buf.String()) {
		t.Errorf("unexpected Infof log output: %s", buf)
	}
}

func TestLoggerInfoln(t *testing.T) {
	buf := new(bytes.Buffer)
	LgrInfo().SetOutput(buf)

	Infoln("FOO", "BAR")
	if !regexp.MustCompile(`^INFO: .+ FOO BAR`).MatchString(buf.String()) {
		t.Errorf("unexpected Infoln log output: %s", buf)
	}
}

func TestLoggerDebug(t *testing.T) {
	buf := new(bytes.Buffer)
	LgrDebug().SetOutput(buf)

	Debug("FOO", "BAR")
	if !regexp.MustCompile(`^DEBUG: .+ FOOBAR`).MatchString(buf.String()) {
		t.Errorf("unexpected Debug log output: %s", buf)
	}
}

func TestLoggerDebugf(t *testing.T) {
	buf := new(bytes.Buffer)
	LgrDebug().SetOutput(buf)

	Debugf("%s %s", "FOO", "BAR")
	if !regexp.MustCompile(`^DEBUG: .+ FOO BAR`).MatchString(buf.String()) {
		t.Errorf("unexpected Debugf log output: %s", buf)
	}
}

func TestLoggerDebugln(t *testing.T) {
	buf := new(bytes.Buffer)
	LgrDebug().SetOutput(buf)

	Debugln("FOO", "BAR")
	if !regexp.MustCompile(`^DEBUG: .+ FOO BAR`).MatchString(buf.String()) {
		t.Errorf("unexpected Debugln log output: %s", buf)
	}
}

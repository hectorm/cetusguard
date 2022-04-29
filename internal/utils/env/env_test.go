package env

import (
	"testing"
)

func TestStringEnvDefault(t *testing.T) {
	val := StringEnv("BAR", "FOO")

	if val != "BAR" {
		t.Errorf("val = \"%s\", want \"%s\"", val, "BAR")
	}
}

func TestStringEnvFirst(t *testing.T) {
	t.Setenv("FOO1", "VAL1")
	t.Setenv("FOO2", "VAL2")
	t.Setenv("FOO3", "VAL3")

	val := StringEnv("BAR", "FOO1", "FOO2", "FOO3")

	if val != "VAL1" {
		t.Errorf("val = \"%s\", want \"%s\"", val, "VAL1")
	}
}

func TestStringEnvSecond(t *testing.T) {
	t.Setenv("FOO2", "VAL2")
	t.Setenv("FOO3", "VAL3")

	val := StringEnv("BAR", "FOO1", "FOO2", "FOO3")

	if val != "VAL2" {
		t.Errorf("val = \"%s\", want \"%s\"", val, "VAL2")
	}
}

func TestStringEnvEmpty(t *testing.T) {
	t.Setenv("FOO", "")

	val := StringEnv("BAR", "FOO")

	if val != "" {
		t.Errorf("val = \"%s\", want \"%s\"", val, "")
	}
}

func TestStringSliceEnvDefault(t *testing.T) {
	val := StringSliceEnv([]string{"BAR"}, "FOO")

	if len(val) != 1 || val[0] != "BAR" {
		t.Errorf("val = %v, want %v", val, []string{"BAR"})
	}
}

func TestStringSliceEnvFirst(t *testing.T) {
	t.Setenv("FOO1", "VAL1")
	t.Setenv("FOO2", "VAL2")
	t.Setenv("FOO3", "VAL3")

	val := StringSliceEnv([]string{"BAR"}, "FOO1", "FOO2", "FOO3")

	if len(val) != 1 || val[0] != "VAL1" {
		t.Errorf("val = %v, want %v", val, []string{"VAL1"})
	}
}

func TestStringSliceEnvSecond(t *testing.T) {
	t.Setenv("FOO2", "VAL2")
	t.Setenv("FOO3", "VAL3")

	val := StringSliceEnv([]string{"BAR"}, "FOO1", "FOO2", "FOO3")

	if len(val) != 1 || val[0] != "VAL2" {
		t.Errorf("val = %v, want %v", val, []string{"VAL2"})
	}
}

func TestStringSliceEnvEmpty(t *testing.T) {
	t.Setenv("FOO", "")

	val := StringSliceEnv([]string{"BAR"}, "FOO")

	if len(val) != 1 || val[0] != "" {
		t.Errorf("val = %v, want %v", val, []string{})
	}
}

func TestIntEnvDefault(t *testing.T) {
	val := IntEnv(0, "FOO")

	if val != 0 {
		t.Errorf("val = %d, want %d", val, 0)
	}
}

func TestIntEnvFirst(t *testing.T) {
	t.Setenv("FOO1", "1")
	t.Setenv("FOO2", "2")
	t.Setenv("FOO3", "3")

	val := IntEnv(0, "FOO1", "FOO2", "FOO3")

	if val != 1 {
		t.Errorf("val = %d, want %d", val, 1)
	}
}

func TestIntEnvSecond(t *testing.T) {
	t.Setenv("FOO2", "2")
	t.Setenv("FOO3", "3")

	val := IntEnv(0, "FOO1", "FOO2", "FOO3")

	if val != 2 {
		t.Errorf("val = %d, want %d", val, 2)
	}
}

func TestIntEnvWrongType(t *testing.T) {
	t.Setenv("FOO", "BAR")

	val := IntEnv(0, "FOO")

	if val != 0 {
		t.Errorf("val = %d, want %d", val, 0)
	}
}

func TestIntEnvEmpty(t *testing.T) {
	t.Setenv("FOO", "")

	val := IntEnv(0, "FOO")

	if val != 0 {
		t.Errorf("val = %d, want %d", val, 0)
	}
}

func TestBoolEnvDefault(t *testing.T) {
	val := BoolEnv(true, "FOO")

	if !val {
		t.Errorf("val = %t, want %t", val, true)
	}
}

func TestBoolEnvFirst(t *testing.T) {
	t.Setenv("FOO1", "true")
	t.Setenv("FOO2", "false")
	t.Setenv("FOO3", "false")

	val := BoolEnv(true, "FOO1", "FOO2", "FOO3")

	if !val {
		t.Errorf("val = %t, want %t", val, true)
	}
}

func TestBoolEnvSecond(t *testing.T) {
	t.Setenv("FOO2", "true")
	t.Setenv("FOO3", "false")

	val := BoolEnv(true, "FOO1", "FOO2", "FOO3")

	if !val {
		t.Errorf("val = %t, want %t", val, true)
	}
}

func TestBoolEnvEmpty(t *testing.T) {
	t.Setenv("FOO", "")

	val := BoolEnv(false, "FOO")

	if val {
		t.Errorf("val = %t, want %t", val, false)
	}
}

func TestBoolEnvWrongType(t *testing.T) {
	t.Setenv("FOO", "BAR")

	val := BoolEnv(false, "FOO")

	if val {
		t.Errorf("val = %t, want %t", val, false)
	}
}

package flagextra

import (
	"flag"
	"io"
	"reflect"
	"testing"
)

func TestFlagStringSliceValueDefault(t *testing.T) {
	fs := flag.NewFlagSet("foobar", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var result []string
	fs.Var(
		NewStringSliceValue([]string{"bar1", "bar2"}, &result),
		"foo",
		"Foo",
	)

	err := fs.Parse([]string{"arg1", "arg2"})
	if err != nil {
		t.Fatal(err)
	}

	wanted := []string{"bar1", "bar2"}

	if !reflect.DeepEqual(result, wanted) {
		t.Errorf("result = %v, want %v", result, wanted)
	}
}

func TestFlagStringSliceValueOne(t *testing.T) {
	fs := flag.NewFlagSet("foobar", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var result []string
	fs.Var(
		NewStringSliceValue([]string{}, &result),
		"foo",
		"Foo",
	)

	err := fs.Parse([]string{"-foo", "bar1", "arg1", "arg2"})
	if err != nil {
		t.Fatal(err)
	}

	wanted := []string{"bar1"}

	if !reflect.DeepEqual(result, wanted) {
		t.Errorf("result = %v, want %v", result, wanted)
	}
}

func TestFlagStringSliceValueTwo(t *testing.T) {
	fs := flag.NewFlagSet("foobar", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var result []string
	fs.Var(
		NewStringSliceValue([]string{}, &result),
		"foo",
		"Foo",
	)

	err := fs.Parse([]string{"-foo", "bar1", "-foo", "bar2", "arg1", "arg2"})
	if err != nil {
		t.Fatal(err)
	}

	wanted := []string{"bar1", "bar2"}

	if !reflect.DeepEqual(result, wanted) {
		t.Errorf("result = %v, want %v", result, wanted)
	}
}

func TestFlagStringSliceValueNoValue(t *testing.T) {
	fs := flag.NewFlagSet("foobar", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var result []string
	fs.Var(
		NewStringSliceValue([]string{}, &result),
		"foo",
		"Foo",
	)

	err := fs.Parse([]string{"-foo"})
	if err == nil {
		t.Errorf("result = %v, want an error", result)
	}
}

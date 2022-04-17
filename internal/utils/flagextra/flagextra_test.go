package flagextra

import (
	"flag"
	"io"
	"reflect"
	"testing"
)

func TestFlagStringMultiValueZero(t *testing.T) {
	fs := flag.NewFlagSet("foobar", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var result StringList
	fs.Var(
		&result,
		"foo",
		"Foo",
	)

	err := fs.Parse([]string{"arg1", "arg2"})
	if err != nil {
		t.Fatal(err)
	}

	var wanted StringList

	if !reflect.DeepEqual(result, wanted) {
		t.Errorf("result = %v, want %v", result, wanted)
	}
}

func TestFlagStringMultiValueOne(t *testing.T) {
	fs := flag.NewFlagSet("foobar", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var result StringList
	fs.Var(
		&result,
		"foo",
		"Foo",
	)

	err := fs.Parse([]string{"-foo", "bar1", "arg1", "arg2"})
	if err != nil {
		t.Fatal(err)
	}

	var wanted StringList
	_ = wanted.Set("bar1")

	if !reflect.DeepEqual(result, wanted) {
		t.Errorf("result = %v, want %v", result, wanted)
	}
}

func TestFlagStringMultiValueTwo(t *testing.T) {
	fs := flag.NewFlagSet("foobar", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var result StringList
	fs.Var(
		&result,
		"foo",
		"Foo",
	)

	err := fs.Parse([]string{"-foo", "bar1", "-foo", "bar2", "arg1", "arg2"})
	if err != nil {
		t.Fatal(err)
	}

	var wanted StringList
	_ = wanted.Set("bar1")
	_ = wanted.Set("bar2")

	if !reflect.DeepEqual(result, wanted) {
		t.Errorf("result = %v, want %v", result, wanted)
	}
}

func TestFlagStringMultiValueNoValue(t *testing.T) {
	fs := flag.NewFlagSet("foobar", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var result StringList
	fs.Var(
		&result,
		"foo",
		"Foo",
	)

	err := fs.Parse([]string{"-foo"})
	if err == nil || result != nil {
		t.Errorf("result = %v, want an error", result)
	}
}

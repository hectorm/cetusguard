package cetusguard

import (
	"log"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

func TestRuleBuiltins(t *testing.T) {
	varRegex := regexp.MustCompile(`%[a-zA-Z0-9_]+%`)
	for k, v := range ruleBuiltins {
		if varRegex.MatchString(v) {
			t.Errorf("%s builtin contains an undefined variable: %s", k, v)
			continue
		}
		if _, err := regexp.Compile("^" + v + "$"); err != nil {
			t.Errorf("%s builtin could not be compiled: %v", k, err)
			continue
		}
	}
}

func TestRuleString(t *testing.T) {
	rawRule := "GET,HEAD,POST ^/.+$"
	rule := Rule{
		Methods: map[string]bool{"POST": true, "HEAD": true, "GET": true},
		Pattern: regexp.MustCompile(`^/.+$`),
	}
	if rule.String() != rawRule {
		t.Errorf("rule = %v, want = %v", rule, rawRule)
	}
}

func TestBuildDefaultRules(t *testing.T) {
	_, err := BuildRules(strings.Join(RawDefaultRules, "\n"))
	if err != nil {
		t.Errorf("cannot build default rules: %v", err)
	}
}

func TestBuildValidRules(t *testing.T) {
	rawRules := map[string]Rule{
		"! Comment\nGET,HEAD %API_PREFIX%/test01\n": {
			Methods: map[string]bool{"GET": true, "HEAD": true},
			Pattern: regexp.MustCompile(`^(/v[0-9]+(\.[0-9]+)*)?/test01$`),
		},
		"! Comment\r\nGET,HEAD %API_PREFIX%/test02\r\n": {
			Methods: map[string]bool{"GET": true, "HEAD": true},
			Pattern: regexp.MustCompile(`^(/v[0-9]+(\.[0-9]+)*)?/test02$`),
		},
		"\n\n\n! Comment\n\n\nGET,HEAD %API_PREFIX%/test03\n\n\n": {
			Methods: map[string]bool{"GET": true, "HEAD": true},
			Pattern: regexp.MustCompile(`^(/v[0-9]+(\.[0-9]+)*)?/test03$`),
		},
		" \t ! Comment\n \t GET,HEAD \t %API_PREFIX%/test04 \t ": {
			Methods: map[string]bool{"GET": true, "HEAD": true},
			Pattern: regexp.MustCompile(`^(/v[0-9]+(\.[0-9]+)*)?/test04$`),
		},
	}

	for k, v := range rawRules {
		builtRules, err := BuildRules(k)
		if err != nil {
			t.Error(err)
			continue
		}
		wantedRules := []Rule{v}
		if !reflect.DeepEqual(builtRules, wantedRules) {
			t.Errorf("builtRules = %v, want = %v", builtRules, wantedRules)
			continue
		}
	}
}

func TestBuildInvalidRules(t *testing.T) {
	rawRules := []string{
		"%API_PREFIX%/test01",
		", %API_PREFIX%/test02",
		"GET, %API_PREFIX%/test03",
		"GET,HEAD, %API_PREFIX%/test04",
		"GET %API_PREFIX%/[9-0]+/test05",
		"GET %API_PREFIX%/\x81/test06",
		"GET\n%API_PREFIX%/test07",
		"GET\r\n%API_PREFIX%/test08",
	}

	for _, v := range rawRules {
		builtRules, err := BuildRules(v)
		if err == nil || builtRules != nil {
			t.Errorf("builtRules = %v, want an error", builtRules)
			continue
		}
	}
}

func TestBuildRulesFromFilePath(t *testing.T) {
	rawRules := []byte("\n! Comment\nGET /.+\r\nGET /.+\r\nGET /.+")

	tmpdir := t.TempDir()
	path := filepath.Join(tmpdir, "rules.list")
	if err := os.WriteFile(path, rawRules, 0644); err != nil {
		t.Fatal(err)
	}

	builtRules, err := BuildRulesFromFilePath(path)
	if err != nil {
		t.Fatal(err)
	}

	if len(builtRules) != 3 {
		t.Errorf("len(builtRules) = %d, want = %d", len(builtRules), 3)
	}
}

func TestBuildInvalidRulesFromFilePath(t *testing.T) {
	rawRules := []byte("INVALID")

	tmpdir := t.TempDir()
	path := filepath.Join(tmpdir, "rules.list")
	if err := os.WriteFile(path, rawRules, 0644); err != nil {
		t.Fatal(err)
	}

	builtRules, err := BuildRulesFromFilePath(path)
	if err == nil || builtRules != nil {
		t.Errorf("builtRules = %v, want an error", builtRules)
	}
}

func TestBuildRulesFromSymlinkPath(t *testing.T) {
	tmpdir := t.TempDir()
	path := filepath.Join(tmpdir, "rules.list")
	if err := os.WriteFile(path, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	link := filepath.Join(tmpdir, "rules-link.list")
	if err := os.Symlink(path, link); err != nil {
		t.Fatal(err)
	}

	_, err := BuildRulesFromFilePath(link)
	if err != nil {
		t.Fatal(err)
	}
}

func TestBuildRulesFromDirectoryPath(t *testing.T) {
	tmpdir := t.TempDir()
	path := filepath.Join(tmpdir, "rules")
	if err := os.Mkdir(path, 0755); err != nil {
		log.Fatal(err)
	}

	builtRules, err := BuildRulesFromFilePath(path)
	if err == nil || builtRules != nil {
		t.Errorf("builtRules = %v, want an error", builtRules)
	}
}

func TestBuildRulesFromNonexistentPath(t *testing.T) {
	tmpdir := t.TempDir()
	path := filepath.Join(tmpdir, "rules.list")

	builtRules, err := BuildRulesFromFilePath(path)
	if err == nil || builtRules != nil {
		t.Errorf("builtRules = %v, want an error", builtRules)
	}
}

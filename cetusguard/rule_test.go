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

func TestRuleVars(t *testing.T) {
	varRegex := regexp.MustCompile(`%[a-zA-Z0-9_]+%`)
	for k, v := range ruleVars {
		if varRegex.MatchString(v) {
			t.Errorf("%s variable contains an undefined variable: %s", k, v)
			continue
		}
		if _, err := regexp.Compile("^" + v + "$"); err != nil {
			t.Errorf("%s variable could not be compiled: %v", k, err)
			continue
		}
	}
}

func TestRuleString(t *testing.T) {
	rawRule := "GET,HEAD,POST ^/.+$"
	rule := Rule{
		Methods: map[string]struct{}{"POST": {}, "HEAD": {}, "GET": {}},
		Pattern: regexp.MustCompile(`^/.+$`),
	}
	if rule.String() != rawRule {
		t.Errorf("rule = %v, want = %v", rule, rawRule)
	}
}

func TestBuildBuiltinRules(t *testing.T) {
	_, err := BuildRules(strings.Join(RawBuiltinRules, "\n"))
	if err != nil {
		t.Errorf("cannot build built-in rules: %v", err)
	}
}

func TestBuildValidRules(t *testing.T) {
	rawRules := map[string]Rule{
		"! Comment\nGET,HEAD %API_PREFIX%/test01\n": {
			Methods: map[string]struct{}{"GET": {}, "HEAD": {}},
			Pattern: regexp.MustCompile(`^(?:/v[0-9]+(?:\.[0-9]+)*)?/test01$`),
		},
		"! Comment\r\nGET,HEAD %API_PREFIX%/test02\r\n": {
			Methods: map[string]struct{}{"GET": {}, "HEAD": {}},
			Pattern: regexp.MustCompile(`^(?:/v[0-9]+(?:\.[0-9]+)*)?/test02$`),
		},
		"\n\n\n! Comment\n\n\nGET,HEAD %API_PREFIX%/test03\n\n\n": {
			Methods: map[string]struct{}{"GET": {}, "HEAD": {}},
			Pattern: regexp.MustCompile(`^(?:/v[0-9]+(?:\.[0-9]+)*)?/test03$`),
		},
		" \t ! Comment\n \t GET,HEAD \t %API_PREFIX%/test04 \t ": {
			Methods: map[string]struct{}{"GET": {}, "HEAD": {}},
			Pattern: regexp.MustCompile(`^(?:/v[0-9]+(?:\.[0-9]+)*)?/test04$`),
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
	if err := os.WriteFile(path, rawRules, 0600); err != nil {
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
	if err := os.WriteFile(path, rawRules, 0600); err != nil {
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
	if err := os.WriteFile(path, []byte(""), 0600); err != nil {
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
	if err := os.Mkdir(path, 0700); err != nil {
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

func TestDomainRegex(t *testing.T) {
	re := regexp.MustCompile("^" + ruleVars["DOMAIN"] + "$")

	testCases := map[string]bool{
		"":                  false,
		"l":                 true,
		"localhost":         true,
		"-localhost":        false,
		"localhost-":        false,
		"sub.example.test":  true,
		"sub.-example.test": false,
		"sub.example-.test": false,
		"001.test":          true,
		"xn--7o8h.test":     true,
		"a.a.a.a.a.a.a":     true,
		"a.a.a...a.a.a":     false,
	}

	for input, wanted := range testCases {
		if result := re.MatchString(input); result != wanted {
			t.Errorf("\"%s\" match = %t, want = %t", input, result, wanted)
		}
	}
}

func TestIpv4Regex(t *testing.T) {
	re := regexp.MustCompile("^" + ruleVars["IPV4"] + "$")

	testCases := map[string]bool{
		"":                false,
		"0":               false,
		"0.0":             false,
		"0.0.0.0.0":       false,
		"0.0.0.0":         true,
		"255.255.255.255": true,
		"1111.0.0.0":      false,
	}

	for input, wanted := range testCases {
		if result := re.MatchString(input); result != wanted {
			t.Errorf("\"%s\" match = %t, want = %t", input, result, wanted)
		}
	}
}

func TestIpv6Regex(t *testing.T) {
	re := regexp.MustCompile("^" + ruleVars["IPV6"] + "$")

	testCases := map[string]bool{
		"":                    false,
		"::":                  false,
		"[]":                  false,
		"[::]":                true,
		"[::1]":               true,
		"[f:f:f:f:f:f:f:f]":   true,
		"[f:f:f:f:f:f:f:x]":   false,
		"[f:f:f:f:f:f:f:f:f]": false,
		"[ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff]": true,
	}

	for input, wanted := range testCases {
		if result := re.MatchString(input); result != wanted {
			t.Errorf("\"%s\" match = %t, want = %t", input, result, wanted)
		}
	}
}

func TestHostRegex(t *testing.T) {
	re := regexp.MustCompile("^" + ruleVars["HOST"] + "$")

	testCases := map[string]bool{
		"":                      false,
		"localhost":             true,
		"localhost:":            false,
		"localhost:2375":        true,
		"localhost:aaaa":        false,
		"sub.example.test":      true,
		"sub.example.test:":     false,
		"sub.example.test:2375": true,
		"sub.example.test:aaaa": false,
		"127.0.0.1":             true,
		"127.0.0.1:":            false,
		"127.0.0.1:2375":        true,
		"127.0.0.1:aaaa":        false,
		"[::1]":                 true,
		"[::1]:":                false,
		"[::1]:2375":            true,
		"[::1]:aaaa":            false,
	}

	for input, wanted := range testCases {
		if result := re.MatchString(input); result != wanted {
			t.Errorf("\"%s\" match = %t, want = %t", input, result, wanted)
		}
	}
}

func TestObjectIdRegex(t *testing.T) {
	re := regexp.MustCompile("^" + ruleVars["_OBJECT_ID"] + "$")

	testCases := map[string]bool{
		"":                 false,
		"0123456789abcdef": true,
		"0123456789x":      false,
	}

	for input, wanted := range testCases {
		if result := re.MatchString(input); result != wanted {
			t.Errorf("\"%s\" match = %t, want = %t", input, result, wanted)
		}
	}
}

func TestObjectNameRegex(t *testing.T) {
	re := regexp.MustCompile("^" + ruleVars["_OBJECT_NAME"] + "$")

	testCases := map[string]bool{
		"":        false,
		"x":       false,
		"x0":      true,
		"0x":      true,
		"xx":      true,
		"xx_.-":   true,
		"-xx":     false,
		"busybox": true,
	}

	for input, wanted := range testCases {
		if result := re.MatchString(input); result != wanted {
			t.Errorf("\"%s\" match = %t, want = %t", input, result, wanted)
		}
	}
}

func TestImageReferenceRegex(t *testing.T) {
	re := regexp.MustCompile("^" + ruleVars["IMAGE_REFERENCE"] + "$")

	testCases := map[string]bool{
		"":                        false,
		"b":                       true,
		"busybox":                 true,
		"busybox:":                false,
		"busybox:l":               true,
		"busybox:latest":          true,
		"busybox:latest@":         false,
		"busybox:latest@sha256":   false,
		"busybox:latest@sha256:":  false,
		"busybox:latest@sha256:x": false,
		"busybox:latest@sha256:09c731d73926315908778730f9e6068":                                  false,
		"busybox:latest@sha256:09c731d73926315908778730f9e60686":                                 true,
		"busybox:latest@sha256:09c731d73926315908778730f9e606864fb72f1523d5c1c81c02dc51563885ba": true,
		"busybox@sha256:09c731d73926315908778730f9e606864fb72f1523d5c1c81c02dc51563885ba":        true,
		"busybox@sha256-:09c731d73926315908778730f9e606864fb72f1523d5c1c81c02dc51563885ba":       false,
		"busybox@sha256-test:09c731d73926315908778730f9e606864fb72f1523d5c1c81c02dc51563885ba":   true,
		"busybox@sha256--test:09c731d73926315908778730f9e606864fb72f1523d5c1c81c02dc51563885ba":  false,
		"busybox@sha256-0test:09c731d73926315908778730f9e606864fb72f1523d5c1c81c02dc51563885ba":  false,
		"busybox@sha256-test0:09c731d73926315908778730f9e606864fb72f1523d5c1c81c02dc51563885ba":  true,
		"busybox@sha256_test:09c731d73926315908778730f9e606864fb72f1523d5c1c81c02dc51563885ba":   true,
		"busybox@sha256.test:09c731d73926315908778730f9e606864fb72f1523d5c1c81c02dc51563885ba":   true,
		"busybox@sha256+test:09c731d73926315908778730f9e606864fb72f1523d5c1c81c02dc51563885ba":   true,
		"docker.io/busybox:latest":              true,
		"docker.io/library/busybox:latest":      true,
		"localhost:5000/library/busybox:latest": true,
		"127.0.0.1:5000/library/busybox:latest": true,
		"[::1]:5000/library/busybox:latest":     true,
		"-busybox:latest":                       false,
		"busybox:-latest":                       false,
		"docker.io/-library/busybox:latest":     false,
		"docker.io/foo/bar/busybox:latest":      true,
		"docker.io/foo.bar/busybox:latest":      true,
		"docker.io/foo..bar/busybox:latest":     false,
		"docker.io/foo_bar/busybox:latest":      true,
		"docker.io/foo__bar/busybox:latest":     true,
		"docker.io/foo___bar/busybox:latest":    false,
		"docker.io/foo-bar/busybox:latest":      true,
		"docker.io/foo--bar/busybox:latest":     true,
		"docker.io/foo---bar/busybox:latest":    true,
	}

	for input, wanted := range testCases {
		if result := re.MatchString(input); result != wanted {
			t.Errorf("\"%s\" match = %t, want = %t", input, result, wanted)
		}
	}
}

func TestApiPrefixRegex(t *testing.T) {
	re := regexp.MustCompile("^" + ruleVars["API_PREFIX"] + "$")

	testCases := map[string]bool{
		"":           true,
		"/":          false,
		"/v":         false,
		"/v9":        true,
		"/v99":       true,
		"/v99.9":     true,
		"/v99.99":    true,
		"/v99.99.9":  true,
		"/v99.99.99": true,
		"/9.9":       false,
		"/v9.9/":     false,
		"/v9.9.":     false,
		"/v.9.9":     false,
		"/v9..9":     false,
		"/va.a":      false,
	}

	for input, wanted := range testCases {
		if result := re.MatchString(input); result != wanted {
			t.Errorf("\"%s\" match = %t, want = %t", input, result, wanted)
		}
	}
}

func TestApiPrefixPing(t *testing.T) {
	re := regexp.MustCompile("^" + ruleVars["API_PREFIX_PING"] + "$")

	testCases := map[string]bool{
		"":            false,
		"/":           false,
		"/_ping":      true,
		"/_ping/":     false,
		"/_pong":      false,
		"_ping":       false,
		"//_ping":     false,
		"/v9.9/":      false,
		"/v9.9/_ping": true,
		"v9.9/_ping":  false,
	}

	for input, wanted := range testCases {
		if result := re.MatchString(input); result != wanted {
			t.Errorf("\"%s\" match = %t, want = %t", input, result, wanted)
		}
	}
}

func TestApiPrefixLibpodRegex(t *testing.T) {
	re := regexp.MustCompile("^" + ruleVars["API_PREFIX_LIBPOD"] + "$")

	testCases := map[string]bool{
		"":                  false,
		"/":                 false,
		"/libpod":           false,
		"/v/libpod":         false,
		"/v9/libpod":        true,
		"/v99.99.99/libpod": true,
	}

	for input, wanted := range testCases {
		if result := re.MatchString(input); result != wanted {
			t.Errorf("\"%s\" match = %t, want = %t", input, result, wanted)
		}
	}
}

func TestApiPrefixLibpodPingRegex(t *testing.T) {
	re := regexp.MustCompile("^" + ruleVars["API_PREFIX_LIBPOD_PING"] + "$")

	testCases := map[string]bool{
		"":                    false,
		"/":                   false,
		"/_ping":              false,
		"/libpod/_ping":       false,
		"/v/libpod/_ping":     false,
		"/v9.9/libpod/_ping":  true,
		"/v9.9/libpod/_ping/": false,
		"/v9.9/libpod/_pong":  false,
	}

	for input, wanted := range testCases {
		if result := re.MatchString(input); result != wanted {
			t.Errorf("\"%s\" match = %t, want = %t", input, result, wanted)
		}
	}
}

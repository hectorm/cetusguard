package cetusguard

import (
	"log"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"regexp/syntax"
	"slices"
	"strings"
	"testing"
)

var expandedRuleVars = maps.Clone(ruleVars)

func init() {
	for k, v := range expandedRuleVars {
		for kk, vv := range expandedRuleVars {
			if strings.Contains(vv, "%"+k+"%") {
				expandedRuleVars[kk] = strings.ReplaceAll(vv, "%"+k+"%", v)
			}
		}
	}
}

func TestRuleVars(t *testing.T) {
	varRegex := regexp.MustCompile(`%[a-zA-Z0-9_]+%`)
	for k, v := range expandedRuleVars {
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

func TestRuleVarsExcludeQuerySeparator(t *testing.T) {
	var matchesQuestionMark func(re *syntax.Regexp) bool
	matchesQuestionMark = func(re *syntax.Regexp) bool {
		switch re.Op {
		case syntax.OpAnyChar, syntax.OpAnyCharNotNL:
			return true
		case syntax.OpLiteral:
			if slices.Contains(re.Rune, '?') {
				return true
			}
		case syntax.OpCharClass:
			for i := 0; i < len(re.Rune); i += 2 {
				if re.Rune[i] <= '?' && '?' <= re.Rune[i+1] {
					return true
				}
			}
		}
		return slices.ContainsFunc(re.Sub, matchesQuestionMark)
	}

	for k, v := range expandedRuleVars {
		re, err := syntax.Parse(v, syntax.Perl)
		if err != nil {
			t.Errorf("%s variable could not be parsed: %v", k, err)
			continue
		}
		if matchesQuestionMark(re) {
			t.Errorf("%s variable can match a query separator: %s", k, v)
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
		"GET %API_PREFIX%/test09\\?)|(.*",
	}

	for _, v := range rawRules {
		builtRules, err := BuildRules(v)
		if err == nil || builtRules != nil {
			t.Errorf("builtRules = %v, want an error", builtRules)
			continue
		}
	}
}

func TestBuildQueryRules(t *testing.T) {
	rawRules := map[string]struct {
		pattern    string
		matchQuery bool
	}{
		`GET /x`:                     {`^/x$`, false},
		`GET /x?`:                    {`^/x?$`, false},
		`GET /x[?&]`:                 {`^/x[?&]$`, false},
		`GET /x\x3F`:                 {`^/x\x3F$`, false},
		`GET /x\?a=1`:                {`^(?:/x\?a=1)$`, true},
		`GET /x\?a=.*`:               {`^(?:/x\?a=.*)$`, true},
		`GET /x[^/\?]`:               {`^(?:/x[^/\?])$`, true},
		`GET /x\\?`:                  {`^(?:/x\\?)$`, true},
		`GET /a\?x=1|/b\?y=1`:        {`^(?:/a\?x=1|/b\?y=1)$`, true},
		`GET /x(?:\?a=(?:0|false))?`: {`^(?:/x(?:\?a=(?:0|false))?)$`, true},
		`GET /x\`:                    {`^/x\$`, false},
		`GET /a|/b`:                  {`^/a|/b$`, false},
	}

	for raw, wanted := range rawRules {
		rules, err := BuildRules(raw)
		if err != nil {
			t.Errorf("\"%s\": %v", raw, err)
			continue
		}
		if len(rules) != 1 {
			t.Errorf("\"%s\": len(rules) = %d, want = %d", raw, len(rules), 1)
			continue
		}
		if rules[0].MatchQuery != wanted.matchQuery {
			t.Errorf("\"%s\": MatchQuery = %t, want = %t", raw, rules[0].MatchQuery, wanted.matchQuery)
		}
		if rules[0].Pattern.String() != wanted.pattern {
			t.Errorf("\"%s\": Pattern = \"%s\", want = \"%s\"", raw, rules[0].Pattern, wanted.pattern)
		}
	}
}

func TestBuildOverlappingVarRules(t *testing.T) {
	if !slices.IsSorted(ruleVarNames) {
		t.Fatalf("ruleVarNames = %v, want = sorted", ruleVarNames)
	}

	// "%API_PREFIX%" and "%CONTAINER_NAME%" overlap on the shared "%"
	raw := "GET %API_PREFIX%CONTAINER_NAME%0"
	wanted := `^(?:/v[0-9]+(?:\.[0-9]+)*)?CONTAINER_NAME%0$`

	for range 100 {
		rules, err := BuildRules(raw)
		if err != nil {
			t.Fatal(err)
		}
		if result := rules[0].Pattern.String(); result != wanted {
			t.Fatalf("\"%s\": Pattern = \"%s\", want = \"%s\"", raw, result, wanted)
		}
	}
}

func FuzzBuildRules(f *testing.F) {
	seeds := []string{
		`GET %API_PREFIX_CONTAINERS%/json(?:\?all=(?:0|false))?`,
		`DELETE %API_PREFIX_CONTAINERS%/%CONTAINER_ID_OR_NAME%(?:\?v=1)?`,
		`POST %API_PREFIX_IMAGES%/create\?fromImage=ghcr\.io%2Fexample%2F[^&]+&tag=[^&]+`,
		`GET /a\?x=1|/b\?y=1`,
		`GET /a\?b)|(.*`,
		`GET (?m)/x\?a=1`,
		`GET /x\?\Qa=1)|(.*\E`,
		`GET /x\?a=1\Q`,
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, raw string) {
		rules, _ := BuildRules(raw)
		for _, rule := range rules {
			if !rule.MatchQuery {
				continue
			}
			pattern := rule.Pattern.String()
			re, err := syntax.Parse(pattern, syntax.Perl)
			if err != nil {
				t.Fatalf("\"%s\": %v", raw, err)
			}
			if re.Op != syntax.OpConcat || re.Sub[0].Op != syntax.OpBeginText || re.Sub[len(re.Sub)-1].Op != syntax.OpEndText {
				t.Fatalf("\"%s\": Pattern = \"%s\", want = anchored as a whole", raw, pattern)
			}
		}
	})
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
	re := regexp.MustCompile("^" + expandedRuleVars["DOMAIN"] + "$")

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
	re := regexp.MustCompile("^" + expandedRuleVars["IPV4"] + "$")

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
	re := regexp.MustCompile("^" + expandedRuleVars["IPV6"] + "$")

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
	re := regexp.MustCompile("^" + expandedRuleVars["HOST"] + "$")

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
	re := regexp.MustCompile("^" + expandedRuleVars["_OBJECT_ID"] + "$")

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
	re := regexp.MustCompile("^" + expandedRuleVars["_OBJECT_NAME"] + "$")

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
	re := regexp.MustCompile("^" + expandedRuleVars["IMAGE_REFERENCE"] + "$")

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

func TestNetworkNameRegex(t *testing.T) {
	re := regexp.MustCompile("^" + expandedRuleVars["NETWORK_NAME"] + "$")

	testCases := map[string]bool{
		"":           false,
		"n":          true,
		"network":    true,
		"net work":   true,
		"net%3Fwork": true,
		"net/work":   false,
		"net?work":   false,
	}

	for input, wanted := range testCases {
		if result := re.MatchString(input); result != wanted {
			t.Errorf("\"%s\" match = %t, want = %t", input, result, wanted)
		}
	}
}

func TestApiPrefixRegex(t *testing.T) {
	re := regexp.MustCompile("^" + expandedRuleVars["API_PREFIX"] + "$")

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

func TestApiPrefixPingRegex(t *testing.T) {
	re := regexp.MustCompile("^" + expandedRuleVars["API_PREFIX_PING"] + "$")

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
	re := regexp.MustCompile("^" + expandedRuleVars["API_PREFIX_LIBPOD"] + "$")

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
	re := regexp.MustCompile("^" + expandedRuleVars["API_PREFIX_LIBPOD_PING"] + "$")

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

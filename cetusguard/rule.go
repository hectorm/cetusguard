package cetusguard

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/hectorm/cetusguard/internal/logger"
)

var RawDefaultRules = []string{
	// Ping
	`GET,HEAD %API_PREFIX_PING%`,
	// Get version
	`GET %API_PREFIX_VERSION%`,
	// Get system information
	`GET %API_PREFIX_INFO%`,
}

var (
	ruleLineRegex    = regexp.MustCompile(`^[\t ]*([A-Z]+(?:,[A-Z]+)*)[\t ]+(.+?)[\t ]*$`)
	commentLineRegex = regexp.MustCompile(`^[\t ]*(?:!.*)?$`)
	newLineRegex     = regexp.MustCompile(`\r?\n`)
	ruleBuiltins     = map[string]string{
		"HOST":                 `(?:[a-zA-Z0-9][a-zA-Z0-9_.-]*)`,
		"IPV4":                 `(?:[0-9]{1,3}(?:\.[0-9]{1,3}){3})`,
		"IPV6":                 `(?:\[[a-fA-F0-9]{0,4}(?::[a-fA-F0-9]{0,4}){2,7}\])`,
		"IP":                   `(?:%IPV4%|%IPV6%)`,
		"HOST_OR_IP":           `(?:%HOST%|%IP%)`,
		"HOST_OR_IP_WITH_PORT": `(?:%HOST_OR_IP%(?::[0-9]+)?)`,

		"IMAGE_ID":         `(?:(?:[a-zA-Z0-9_-]+:)?[a-fA-F0-9]+)`,
		"IMAGE_COMPONENT":  `(?:[a-zA-Z0-9][a-zA-Z0-9_.-]*)`,
		"IMAGE_TAG":        `(?:[a-zA-Z0-9_][a-zA-Z0-9_.-]{0,127})`,
		"IMAGE_NAME":       `(?:(?:%HOST_OR_IP_WITH_PORT%)?(?:/%IMAGE_COMPONENT%)+(?::%IMAGE_TAG%)?)`,
		"IMAGE_ID_OR_NAME": `(?:%IMAGE_ID%|%IMAGE_NAME%)`,

		"CONTAINER_ID":         `(?:[a-fA-F0-9]+)`,
		"CONTAINER_NAME":       `(?:[a-zA-Z0-9][a-zA-Z0-9_.-]+)`,
		"CONTAINER_ID_OR_NAME": `(?:%CONTAINER_ID%|%CONTAINER_NAME%)`,

		"VOLUME_ID":         `(?:[a-fA-F0-9]+)`,
		"VOLUME_NAME":       `(?:[a-zA-Z0-9][a-zA-Z0-9_.-]+)`,
		"VOLUME_ID_OR_NAME": `(?:%VOLUME_ID%|%VOLUME_NAME%)`,

		"NETWORK_ID":         `(?:[a-fA-F0-9]+)`,
		"NETWORK_NAME":       `(?:[^/]+)`,
		"NETWORK_ID_OR_NAME": `(?:%NETWORK_ID%|%NETWORK_NAME%)`,

		"PLUGIN_ID":         `(?:[a-fA-F0-9]+)`,
		"PLUGIN_NAME":       `%IMAGE_NAME%`,
		"PLUGIN_ID_OR_NAME": `(?:%PLUGIN_ID%|%PLUGIN_NAME%)`,

		"API_PREFIX":              `(?:/v[0-9]+(?:\.[0-9]+)*)?`,
		"API_PREFIX_AUTH":         `%API_PREFIX%/auth`,
		"API_PREFIX_BUILD":        `%API_PREFIX%/build`,
		"API_PREFIX_COMMIT":       `%API_PREFIX%/commit`,
		"API_PREFIX_CONFIGS":      `%API_PREFIX%/configs`,
		"API_PREFIX_CONTAINERS":   `%API_PREFIX%/containers`,
		"API_PREFIX_DISTRIBUTION": `%API_PREFIX%/distribution`,
		"API_PREFIX_EVENTS":       `%API_PREFIX%/events`,
		"API_PREFIX_EXEC":         `%API_PREFIX%/exec`,
		"API_PREFIX_GRPC":         `%API_PREFIX%/grpc`,
		"API_PREFIX_IMAGES":       `%API_PREFIX%/images`,
		"API_PREFIX_INFO":         `%API_PREFIX%/info`,
		"API_PREFIX_NETWORKS":     `%API_PREFIX%/networks`,
		"API_PREFIX_NODES":        `%API_PREFIX%/nodes`,
		"API_PREFIX_PING":         `%API_PREFIX%/_ping`,
		"API_PREFIX_PLUGINS":      `%API_PREFIX%/plugins`,
		"API_PREFIX_SECRETS":      `%API_PREFIX%/secrets`,
		"API_PREFIX_SERVICES":     `%API_PREFIX%/services`,
		"API_PREFIX_SESSION":      `%API_PREFIX%/session`,
		"API_PREFIX_SWARM":        `%API_PREFIX%/swarm`,
		"API_PREFIX_SYSTEM":       `%API_PREFIX%/system`,
		"API_PREFIX_TASKS":        `%API_PREFIX%/tasks`,
		"API_PREFIX_VERSION":      `%API_PREFIX%/version`,
		"API_PREFIX_VOLUMES":      `%API_PREFIX%/volumes`,
	}
)

func init() {
	for k, v := range ruleBuiltins {
		for kk, vv := range ruleBuiltins {
			ruleBuiltins[kk] = strings.ReplaceAll(vv, "%"+k+"%", v)
		}
	}
}

func BuildRules(str string) ([]Rule, error) {
	var rules []Rule

	lines := newLineRegex.Split(str, -1)
	for _, line := range lines {
		if commentLineRegex.MatchString(line) {
			continue
		}

		matches := ruleLineRegex.FindStringSubmatch(line)
		if len(matches) != 3 {
			return nil, fmt.Errorf("invalid rule line: %s", line)
		}
		methodsFrag := matches[1]
		patternFrag := matches[2]

		methods := map[string]bool{}
		for _, method := range strings.Split(methodsFrag, ",") {
			methods[method] = true
		}

		for k, v := range ruleBuiltins {
			patternFrag = strings.ReplaceAll(patternFrag, "%"+k+"%", v)
		}
		pattern, err := regexp.Compile("^" + patternFrag + "$")
		if err != nil {
			return nil, fmt.Errorf("invalid rule pattern: %s", str)
		}

		rule := Rule{methods, pattern}
		rules = append(rules, rule)

		logger.Debugf("loaded rule: %s\n", rule)
	}

	return rules, nil
}

func BuildRulesFromFilePath(path string) ([]Rule, error) {
	var rules []Rule

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = file.Close()
	}()

	fileInfo, err := file.Stat()
	if err != nil {
		return nil, err
	}

	if !fileInfo.Mode().IsRegular() {
		return nil, fmt.Errorf("open %s: not a file", path)
	}

	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		r, err := BuildRules(scanner.Text())
		if err != nil {
			return nil, err
		}

		rules = append(rules, r...)
	}

	return rules, nil
}

type Rule struct {
	Methods map[string]bool
	Pattern *regexp.Regexp
}

func (rule Rule) String() string {
	methods := make([]string, 0, len(rule.Methods))
	for k := range rule.Methods {
		methods = append(methods, k)
	}
	sort.Strings(methods)

	return fmt.Sprintf("%s %s",
		strings.Join(methods, ","),
		rule.Pattern.String(),
	)
}

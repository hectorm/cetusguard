package cetusguard

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/hectorm/cetusguard/internal/logger"
)

var RawBuiltinRules = []string{
	// Ping
	`GET,HEAD %API_PREFIX_PING%`,
	`GET,HEAD %API_PREFIX_LIBPOD_PING%`,
	// Get version
	`GET %API_PREFIX_VERSION%`,
	`GET %API_PREFIX_LIBPOD_VERSION%`,
	// Get system information
	`GET %API_PREFIX_INFO%`,
	`GET %API_PREFIX_LIBPOD_INFO%`,
}

var (
	ruleLineRegex    = regexp.MustCompile(`^[\t ]*([A-Z]+(?:,[A-Z]+)*)[\t ]+(.+?)[\t ]*$`)
	commentLineRegex = regexp.MustCompile(`^[\t ]*(?:!.*)?$`)
	newLineRegex     = regexp.MustCompile(`\r?\n`)
	ruleVars         = map[string]string{
		"DOMAIN":       `(?:[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*)`,
		"IPV4":         `(?:[0-9]{1,3}(?:\.[0-9]{1,3}){3})`,
		"IPV6":         `(?:\[[a-fA-F0-9]{0,4}(?::[a-fA-F0-9]{0,4}){2,7}(?:%[a-zA-Z0-9_]+)?\])`,
		"IP":           `(?:%IPV4%|%IPV6%)`,
		"DOMAIN_OR_IP": `(?:%DOMAIN%|%IP%)`,
		"HOST":         `(?:%DOMAIN_OR_IP%(?::[0-9]+)?)`,

		"IMAGE_ID":              `%_OBJECT_ID%`,
		"IMAGE_COMPONENT":       `(?:[a-zA-Z0-9]+(?:(?:\.|_{1,2}|-+)[a-zA-Z0-9]+)*)`,
		"IMAGE_TAG":             `(?:[a-zA-Z0-9_][a-zA-Z0-9_.-]{0,127})`,
		"IMAGE_DIGEST":          `(?:[a-zA-Z][a-zA-Z0-9]*(?:[_.+-][a-zA-Z][a-zA-Z0-9]*)*:[a-fA-F0-9]{32,})`,
		"IMAGE_NAME":            `(?:(?:%HOST%/)?%IMAGE_COMPONENT%(?:/%IMAGE_COMPONENT%)*)`,
		"IMAGE_REFERENCE":       `(?:%IMAGE_NAME%(?::%IMAGE_TAG%)?(?:@%IMAGE_DIGEST%)?)`,
		"IMAGE_ID_OR_REFERENCE": `(?:%IMAGE_ID%|%IMAGE_REFERENCE%)`,

		"CONTAINER_ID":         `%_OBJECT_ID%`,
		"CONTAINER_NAME":       `%_OBJECT_NAME%`,
		"CONTAINER_ID_OR_NAME": `(?:%CONTAINER_ID%|%CONTAINER_NAME%)`,

		"VOLUME_ID":         `%_OBJECT_ID%`,
		"VOLUME_NAME":       `%_OBJECT_NAME%`,
		"VOLUME_ID_OR_NAME": `(?:%VOLUME_ID%|%VOLUME_NAME%)`,

		"NETWORK_ID":         `%_OBJECT_ID%`,
		"NETWORK_NAME":       `(?:[^/]+)`,
		"NETWORK_ID_OR_NAME": `(?:%NETWORK_ID%|%NETWORK_NAME%)`,

		"PLUGIN_ID":         `%_OBJECT_ID%`,
		"PLUGIN_NAME":       `%IMAGE_NAME%`,
		"PLUGIN_ID_OR_NAME": `(?:%PLUGIN_ID%|%PLUGIN_NAME%)`,

		"API_PREFIX":              `%_API_VERSION%?`,
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

		"API_PREFIX_LIBPOD":            `%_API_VERSION%/libpod`,
		"API_PREFIX_LIBPOD_BUILD":      `%API_PREFIX_LIBPOD%/build`,
		"API_PREFIX_LIBPOD_COMMIT":     `%API_PREFIX_LIBPOD%/commit`,
		"API_PREFIX_LIBPOD_CONTAINERS": `%API_PREFIX_LIBPOD%/containers`,
		"API_PREFIX_LIBPOD_EVENTS":     `%API_PREFIX_LIBPOD%/events`,
		"API_PREFIX_LIBPOD_EXEC":       `%API_PREFIX_LIBPOD%/exec`,
		"API_PREFIX_LIBPOD_GENERATE":   `%API_PREFIX_LIBPOD%/generate`,
		"API_PREFIX_LIBPOD_IMAGES":     `%API_PREFIX_LIBPOD%/images`,
		"API_PREFIX_LIBPOD_INFO":       `%API_PREFIX_LIBPOD%/info`,
		"API_PREFIX_LIBPOD_MANIFESTS":  `%API_PREFIX_LIBPOD%/manifests`,
		"API_PREFIX_LIBPOD_NETWORKS":   `%API_PREFIX_LIBPOD%/networks`,
		"API_PREFIX_LIBPOD_PING":       `%API_PREFIX_LIBPOD%/_ping`,
		"API_PREFIX_LIBPOD_PLAY":       `%API_PREFIX_LIBPOD%/play`,
		"API_PREFIX_LIBPOD_PODS":       `%API_PREFIX_LIBPOD%/pods`,
		"API_PREFIX_LIBPOD_SECRETS":    `%API_PREFIX_LIBPOD%/secrets`,
		"API_PREFIX_LIBPOD_SYSTEM":     `%API_PREFIX_LIBPOD%/system`,
		"API_PREFIX_LIBPOD_VERSION":    `%API_PREFIX_LIBPOD%/version`,
		"API_PREFIX_LIBPOD_VOLUMES":    `%API_PREFIX_LIBPOD%/volumes`,

		// Private variables, may change in any version
		"_API_VERSION": `(?:/v[0-9]+(?:\.[0-9]+)*)`,
		"_OBJECT_ID":   `(?:[a-fA-F0-9]+)`,
		"_OBJECT_NAME": `(?:[a-zA-Z0-9][a-zA-Z0-9_.-]+)`,
	}
)

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

		methods := make(map[string]struct{})
		for _, method := range strings.Split(methodsFrag, ",") {
			methods[method] = struct{}{}
		}

		for {
			p := patternFrag
			for k, v := range ruleVars {
				if strings.Contains(p, "%"+k+"%") {
					p = strings.ReplaceAll(p, "%"+k+"%", v)
				}
			}
			if p == patternFrag {
				break
			}
			patternFrag = p
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

	file, err := os.Open(filepath.Clean(path))
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
	Methods map[string]struct{}
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

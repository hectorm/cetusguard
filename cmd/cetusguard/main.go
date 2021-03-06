package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/hectorm/cetusguard/cetusguard"
	"github.com/hectorm/cetusguard/internal/logger"
	"github.com/hectorm/cetusguard/internal/utils/env"
	"github.com/hectorm/cetusguard/internal/utils/flagextra"
)

var (
	version    = "dev"
	author     = "H\u00E9ctor Molinero Fern\u00E1ndez <hector@molinero.dev>"
	license    = "MIT, https://opensource.org/licenses/MIT"
	repository = "https://github.com/hectorm/cetusguard"
)

func main() {
	var backendAddr string
	flag.StringVar(
		&backendAddr,
		"backend-addr",
		env.StringEnv("unix:///var/run/docker.sock", "CETUSGUARD_BACKEND_ADDR", "CONTAINER_HOST", "DOCKER_HOST"),
		"Container daemon socket to connect to (env CETUSGUARD_BACKEND_ADDR, CONTAINER_HOST, DOCKER_HOST)",
	)

	var frontendAddr []string
	flag.Var(
		flagextra.NewStringSliceValue(env.StringSliceEnv([]string{"tcp://127.0.0.1:2375"}, "CETUSGUARD_FRONTEND_ADDR"), &frontendAddr),
		"frontend-addr",
		"Address to bind the server to, can be specified multiple times (env CETUSGUARD_FRONTEND_ADDR)",
	)

	var backendTlsCacert string
	flag.StringVar(
		&backendTlsCacert,
		"backend-tls-cacert",
		env.StringEnv("", "CETUSGUARD_BACKEND_TLS_CACERT"),
		"Path to the backend TLS certificate used to verify the daemon identity (env CETUSGUARD_BACKEND_TLS_CACERT)",
	)

	var backendTlsCert string
	flag.StringVar(
		&backendTlsCert,
		"backend-tls-cert",
		env.StringEnv("", "CETUSGUARD_BACKEND_TLS_CERT"),
		"Path to the backend TLS certificate used to authenticate with the daemon (env CETUSGUARD_BACKEND_TLS_CERT)",
	)

	var backendTlsKey string
	flag.StringVar(
		&backendTlsKey,
		"backend-tls-key",
		env.StringEnv("", "CETUSGUARD_BACKEND_TLS_KEY"),
		"Path to the backend TLS key used to authenticate with the daemon (env CETUSGUARD_BACKEND_TLS_KEY)",
	)

	var frontendTlsCacert string
	flag.StringVar(
		&frontendTlsCacert,
		"frontend-tls-cacert",
		env.StringEnv("", "CETUSGUARD_FRONTEND_TLS_CACERT"),
		"Path to the frontend TLS certificate used to verify the identity of clients (env CETUSGUARD_FRONTEND_TLS_CACERT)",
	)

	var frontendTlsCert string
	flag.StringVar(
		&frontendTlsCert,
		"frontend-tls-cert",
		env.StringEnv("", "CETUSGUARD_FRONTEND_TLS_CERT"),
		"Path to the frontend TLS certificate (env CETUSGUARD_FRONTEND_TLS_CERT)",
	)

	var frontendTlsKey string
	flag.StringVar(
		&frontendTlsKey,
		"frontend-tls-key",
		env.StringEnv("", "CETUSGUARD_FRONTEND_TLS_KEY"),
		"Path to the frontend TLS key (env CETUSGUARD_FRONTEND_TLS_KEY)",
	)

	var ruleList []string
	flag.Var(
		flagextra.NewStringSliceValue(env.StringSliceEnv(nil, "CETUSGUARD_RULES"), &ruleList),
		"rules",
		"Filter rules separated by new lines, can be specified multiple times (env CETUSGUARD_RULES)",
	)

	var ruleFileList []string
	flag.Var(
		flagextra.NewStringSliceValue(env.StringSliceEnv(nil, "CETUSGUARD_RULES_FILE"), &ruleFileList),
		"rules-file",
		"Filter rules file, can be specified multiple times (env CETUSGUARD_RULES_FILE)",
	)

	var noBuiltinRules bool
	flag.BoolVar(
		&noBuiltinRules,
		"no-builtin-rules",
		env.BoolEnv(false, "CETUSGUARD_NO_BUILTIN_RULES"),
		"Do not load the built-in rules (env CETUSGUARD_NO_BUILTIN_RULES)",
	)

	var logLevel int
	flag.IntVar(
		&logLevel,
		"log-level",
		env.IntEnv(logger.LvlInfo, "CETUSGUARD_LOG_LEVEL"),
		fmt.Sprintf("The minimum entry level to log, from %d to %d (env CETUSGUARD_LOG_LEVEL)", logger.LvlNone, logger.LvlDebug),
	)

	var printVersion bool
	flag.BoolVar(
		&printVersion,
		"version",
		false,
		"Show version number and quit",
	)

	flag.Parse()
	logger.SetLevel(logLevel)

	if printVersion {
		fmt.Printf("CetusGuard %s\n", version)
		fmt.Printf("Author: %s\n", author)
		fmt.Printf("License: %s\n", license)
		fmt.Printf("Repository: %s\n", repository)
		os.Exit(0)
	}

	var rules []cetusguard.Rule
	if !noBuiltinRules {
		rawRules := strings.Join(cetusguard.RawBuiltinRules, "\n")
		builtRules, err := cetusguard.BuildRules(rawRules)
		if err != nil {
			logger.Critical(err)
		}
		rules = append(rules, builtRules...)
	}
	for _, ruleElem := range ruleList {
		builtRules, err := cetusguard.BuildRules(ruleElem)
		if err != nil {
			logger.Critical(err)
		}
		rules = append(rules, builtRules...)
	}
	for _, ruleFileElem := range ruleFileList {
		builtRules, err := cetusguard.BuildRulesFromFilePath(ruleFileElem)
		if err != nil {
			logger.Critical(err)
		}
		rules = append(rules, builtRules...)
	}

	cg := &cetusguard.Server{
		Backend: &cetusguard.Backend{
			Addr:      backendAddr,
			TlsCacert: backendTlsCacert,
			TlsCert:   backendTlsCert,
			TlsKey:    backendTlsKey,
		},
		Frontend: &cetusguard.Frontend{
			Addr:      frontendAddr,
			TlsCacert: frontendTlsCacert,
			TlsCert:   frontendTlsCert,
			TlsKey:    frontendTlsKey,
		},
		Rules: rules,
	}

	ready := make(chan any, 1)
	err := cg.Start(ready)
	if err != nil {
		logger.Critical(err)
	}
}

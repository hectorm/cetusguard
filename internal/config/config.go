package config

import (
	"flag"
	"fmt"
	"os"

	"github.com/hectorm/cetusguard/internal/logger"
	"github.com/hectorm/cetusguard/internal/utils/env"
	"github.com/hectorm/cetusguard/internal/utils/flagextra"
)

var (
	Author     = "Héctor Molinero Fernández <hector@molinero.dev>"
	License    = "MIT, https://opensource.org/licenses/MIT"
	Repository = "https://github.com/hectorm/cetusguard"
	Version    = "dev"
)

type Config struct {
	BackendAddr       string
	FrontendAddr      []string
	BackendTLSCacert  string
	BackendTLSCert    string
	BackendTLSKey     string
	FrontendTLSCacert string
	FrontendTLSCert   string
	FrontendTLSKey    string
	Rules             []string
	RuleFiles         []string
	NoBuiltinRules    bool
	LogLevel          int
}

func NewConfig() *Config {
	config := &Config{}

	flag.StringVar(
		&config.BackendAddr,
		"backend-addr",
		env.StringEnv("unix:///var/run/docker.sock", "CETUSGUARD_BACKEND_ADDR", "CONTAINER_HOST", "DOCKER_HOST"),
		"Container daemon socket to connect to (env CETUSGUARD_BACKEND_ADDR, CONTAINER_HOST, DOCKER_HOST)",
	)

	flag.Var(
		flagextra.NewStringSliceValue(env.StringSliceEnv([]string{"tcp://127.0.0.1:2375"}, "CETUSGUARD_FRONTEND_ADDR"), &config.FrontendAddr),
		"frontend-addr",
		"Address to bind the server to, can be specified multiple times (env CETUSGUARD_FRONTEND_ADDR)",
	)

	flag.StringVar(
		&config.BackendTLSCacert,
		"backend-tls-cacert",
		env.StringEnv("", "CETUSGUARD_BACKEND_TLS_CACERT"),
		"Path to the backend TLS certificate used to verify the daemon identity (env CETUSGUARD_BACKEND_TLS_CACERT)",
	)

	flag.StringVar(
		&config.BackendTLSCert,
		"backend-tls-cert",
		env.StringEnv("", "CETUSGUARD_BACKEND_TLS_CERT"),
		"Path to the backend TLS certificate used to authenticate with the daemon (env CETUSGUARD_BACKEND_TLS_CERT)",
	)

	flag.StringVar(
		&config.BackendTLSKey,
		"backend-tls-key",
		env.StringEnv("", "CETUSGUARD_BACKEND_TLS_KEY"),
		"Path to the backend TLS key used to authenticate with the daemon (env CETUSGUARD_BACKEND_TLS_KEY)",
	)

	flag.StringVar(
		&config.FrontendTLSCacert,
		"frontend-tls-cacert",
		env.StringEnv("", "CETUSGUARD_FRONTEND_TLS_CACERT"),
		"Path to the frontend TLS certificate used to verify the identity of clients (env CETUSGUARD_FRONTEND_TLS_CACERT)",
	)

	flag.StringVar(
		&config.FrontendTLSCert,
		"frontend-tls-cert",
		env.StringEnv("", "CETUSGUARD_FRONTEND_TLS_CERT"),
		"Path to the frontend TLS certificate (env CETUSGUARD_FRONTEND_TLS_CERT)",
	)

	flag.StringVar(
		&config.FrontendTLSKey,
		"frontend-tls-key",
		env.StringEnv("", "CETUSGUARD_FRONTEND_TLS_KEY"),
		"Path to the frontend TLS key (env CETUSGUARD_FRONTEND_TLS_KEY)",
	)

	flag.Var(
		flagextra.NewStringSliceValue(env.StringSliceEnv(nil, "CETUSGUARD_RULES"), &config.Rules),
		"rules",
		"Filter rules separated by new lines, can be specified multiple times (env CETUSGUARD_RULES)",
	)

	flag.Var(
		flagextra.NewStringSliceValue(env.StringSliceEnv(nil, "CETUSGUARD_RULES_FILE"), &config.RuleFiles),
		"rules-file",
		"Filter rules file, can be specified multiple times (env CETUSGUARD_RULES_FILE)",
	)

	flag.BoolVar(
		&config.NoBuiltinRules,
		"no-builtin-rules",
		env.BoolEnv(false, "CETUSGUARD_NO_BUILTIN_RULES"),
		"Do not load the built-in rules (env CETUSGUARD_NO_BUILTIN_RULES)",
	)

	flag.IntVar(
		&config.LogLevel,
		"log-level",
		env.IntEnv(logger.LvlInfo, "CETUSGUARD_LOG_LEVEL"),
		fmt.Sprintf("The minimum entry level to log, from %d to %d (env CETUSGUARD_LOG_LEVEL)", logger.LvlNone, logger.LvlDebug),
	)

	printVersion := false
	flag.BoolVar(
		&printVersion,
		"version",
		false,
		"Show version number and quit",
	)

	flag.Parse()
	logger.SetLevel(config.LogLevel)

	if printVersion {
		fmt.Printf("CetusGuard %s\n", Version)
		fmt.Printf("Author: %s\n", Author)
		fmt.Printf("License: %s\n", License)
		fmt.Printf("Repository: %s\n", Repository)
		os.Exit(0)
	}

	return config
}

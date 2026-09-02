package main

import (
	"strings"

	"github.com/hectorm/cetusguard/cetusguard"
	"github.com/hectorm/cetusguard/internal/config"
	"github.com/hectorm/cetusguard/internal/logger"
)

func main() {
	cfg := config.NewConfig()

	var rules []cetusguard.Rule
	if !cfg.NoBuiltinRules {
		rawRules := strings.Join(cetusguard.RawBuiltinRules, "\n")
		builtRules, err := cetusguard.BuildRules(rawRules)
		if err != nil {
			logger.Critical(err)
		}
		rules = append(rules, builtRules...)
	}
	for _, ruleElem := range cfg.Rules {
		builtRules, err := cetusguard.BuildRules(ruleElem)
		if err != nil {
			logger.Critical(err)
		}
		rules = append(rules, builtRules...)
	}
	for _, ruleFileElem := range cfg.RuleFiles {
		builtRules, err := cetusguard.BuildRulesFromFilePath(ruleFileElem)
		if err != nil {
			logger.Critical(err)
		}
		rules = append(rules, builtRules...)
	}

	cg := &cetusguard.Server{
		Backend: &cetusguard.Backend{
			Addr:      cfg.BackendAddr,
			TLSCacert: cfg.BackendTLSCacert,
			TLSCert:   cfg.BackendTLSCert,
			TLSKey:    cfg.BackendTLSKey,
		},
		Frontend: &cetusguard.Frontend{
			Addr:      cfg.FrontendAddr,
			TLSCacert: cfg.FrontendTLSCacert,
			TLSCert:   cfg.FrontendTLSCert,
			TLSKey:    cfg.FrontendTLSKey,
		},
		Rules: rules,
	}

	ready := make(chan any, 1)
	err := cg.Start(ready)
	if err != nil {
		logger.Critical(err)
	}
}

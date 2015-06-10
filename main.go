package main

import (
	"fmt"
	"github.com/Matir/httpwatch/config"
	"github.com/Matir/httpwatch/httpsource"
	"github.com/Matir/httpwatch/output"
	"github.com/Matir/httpwatch/rules"
	"os"
)

func main() {
	// Setup config
	cfg := config.Config{}
	cfg.Init()
	if err := cfg.Valid(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	// Set all loggers to the same
	httpsource.SetLogger(cfg.Logger)
	output.SetLogger(cfg.Logger)
	rules.SetLogger(cfg.Logger)

	// Setup sources
	source := httpsource.NewHTTPSource()
	source.ConvertConnectionsToPairs()
	opened_any := false
	for _, iface := range cfg.Interfaces {
		if err := source.AddPCAPIface(iface); err != nil {
			cfg.Logger.Printf("Error adding interface: %s\n", err)
		} else {
			opened_any = true
		}
	}
	for _, fname := range cfg.PcapFiles {
		if err := source.AddPCAPFile(fname); err != nil {
			cfg.Logger.Printf("Error opening file: %s\n", err)
		} else {
			opened_any = true
		}
	}
	if !opened_any {
		return
	}

	// Create rule mux
	// TODO: provide option on type of Mux
	rulemux := httpsource.NewBlockingPairMux(source.Pairs)

	// Setup all rules
	ruleEngine := rules.NewRuleEngine(cfg.Rules, rulemux)

	// Setup outputs
	outputEngine := output.NewOutputEngine(ruleEngine.Matches)
	for _, o := range cfg.Outputs {
		outputEngine.AddOutput(o.Name, o.Options)
	}

	// Start all the working parts
	ruleEngine.Start()
	outputEngine.Start()

	// Wait until finished
	source.WaitUntilFinished()
	ruleEngine.WaitUntilFinished()
	outputEngine.WaitUntilFinished()
}

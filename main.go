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

	// Setup sources
	source := httpsource.NewHTTPSource()
	source.ConvertConnectionsToPairs()
	for _, iface := range cfg.Interfaces {
		source.AddPCAPIface(iface)
	}
	for _, fname := range cfg.PcapFiles {
		source.AddPCAPFile(fname)
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

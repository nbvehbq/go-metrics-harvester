package main

import (
	errname "github.com/Antonboom/errname/pkg/analyzer"
	"github.com/timakin/bodyclose/passes/bodyclose"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
	"golang.org/x/tools/go/analysis/passes/printf"
	"golang.org/x/tools/go/analysis/passes/shadow"
	"golang.org/x/tools/go/analysis/passes/structtag"
	"honnef.co/go/tools/staticcheck"

	"github.com/nbvehbq/go-metrics-harvester/pkg/exitcheckanalyzer"
)

func main() {
	checks := map[string]bool{
		"SA":     true,
		"ST1015": true,
	}
	var mychecks []*analysis.Analyzer
	for _, v := range staticcheck.Analyzers {
		if checks[v.Analyzer.Name] {
			mychecks = append(mychecks, v.Analyzer)
		}
	}

	mychecks = append(
		mychecks,
		printf.Analyzer,
		shadow.Analyzer,
		structtag.Analyzer,
		bodyclose.Analyzer,
		errname.New(),
		exitcheckanalyzer.ExitCheckAnalyzer,
	)

	multichecker.Main(mychecks...)
}

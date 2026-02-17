package main

import (
	"github.com/tigranqic/metrics-tpl/internal/analysis"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(analysis.Analyzer)
}

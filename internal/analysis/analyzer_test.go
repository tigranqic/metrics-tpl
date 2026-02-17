package analysis

import (
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestBadCode(t *testing.T) {
	data := getTestDataPath(t)
	analysistest.Run(t, data, Analyzer, "bad")
}

func TestGoodCode(t *testing.T) {
	data := getTestDataPath(t)
	analysistest.Run(t, data, Analyzer, "good")
}

func TestMainPackage(t *testing.T) {
	data := getTestDataPath(t)
	analysistest.Run(t, data, Analyzer, "mainpkg")
}

func TestMainBadCode(t *testing.T) {
	data := getTestDataPath(t)
	analysistest.Run(t, data, Analyzer, "mainbad")
}

func TestAliasImport(t *testing.T) {
	data := getTestDataPath(t)
	analysistest.Run(t, data, Analyzer, "aliasbad")
}

func TestAliasShadow(t *testing.T) {
	data := getTestDataPath(t)
	analysistest.Run(t, data, Analyzer, "aliasshadow")
}

func getTestDataPath(t *testing.T) string {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	return filepath.Join(wd, "testdata")
}

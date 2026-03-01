package main

import (
	"go/ast"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsGoSourceFile(t *testing.T) {
	assert.True(t, isGoSourceFile("main.go"))
	assert.False(t, isGoSourceFile("main_test.go"))
	assert.False(t, isGoSourceFile("main.gen.go"))
	assert.False(t, isGoSourceFile("README.md"))
}

func TestGetZeroValueForType(t *testing.T) {
	assert.Equal(t, `""`, getZeroValueForType(&ast.Ident{Name: "string"}))
	assert.Equal(t, "false", getZeroValueForType(&ast.Ident{Name: "bool"}))
	assert.Equal(t, "0", getZeroValueForType(&ast.Ident{Name: "int"}))
	assert.Equal(t, "nil", getZeroValueForType(&ast.StarExpr{}))
}

func TestParseGoFileAndBuildModel(t *testing.T) {
	dir := t.TempDir()
	source := `package test
// generate:reset
type MyStruct struct {
	A int
	B string
	C *float64
}
`
	path := filepath.Join(dir, "source.go")
	err := os.WriteFile(path, []byte(source), 0644)
	require.NoError(t, err)

	packages := make(map[string]*packageInfo)
	err = parseGoFile(path, packages)
	require.NoError(t, err)

	pkg, ok := packages[dir]
	require.True(t, ok)
	assert.Equal(t, "test", pkg.name)
	require.Len(t, pkg.structs, 1)
	assert.Equal(t, "MyStruct", pkg.structs[0].name)

	model := buildModel(pkg)
	assert.Equal(t, "test", model.PackageName)
	require.Len(t, model.Structs, 1)
	assert.Equal(t, "MyStruct", model.Structs[0].Name)
	assert.Len(t, model.Structs[0].Fields, 3)
}

func TestGenerateResetCode(t *testing.T) {
	t.Run("Map", func(t *testing.T) {
		code := generateResetCode("MyMap", &ast.MapType{})
		assert.Contains(t, code, "clear(r.MyMap)")
	})

	t.Run("Array", func(t *testing.T) {
		code := generateResetCode("MySlice", &ast.ArrayType{})
		assert.Contains(t, code, "r.MySlice[:0]")
	})
}

# Custom Analyzer (Linter)

This directory contains a custom static analyzer for Go that enforces specific code quality rules.

## Overview

The analyzer (`exitcheck`) detects two types of code issues:

### 1. Panic Detection
- **Rule**: Detects any use of the `panic()` built-in function anywhere in the code
- **Reason**: `panic()` should be avoided in production code as it causes unhandled crashes
- **Message**: `"panic() detected"`

### 2. Exit Detection (Main Package Only)
- **Rule**: Detects calls to `log.Fatal()` or `os.Exit()` outside the `main()` function in the `main` package
- **Reason**: These functions terminate the program and should only be called from the main entry point
- **Messages**: 
  - `"log.Fatal() detected outside main function"`
  - `"os.Exit() detected outside main function"`

## Usage

### Building the Linter

```bash
go build -o linter ./cmd/linter
```

### Running the Linter

```bash
./linter ./...
```

Or to analyze a specific package:

```bash
./linter ./cmd/server
```

### Expected Output

When the linter finds issues, it reports the file path, line number, and the specific problem:

```
file.go:42:3: panic() detected
main.go:10:5: log.Fatal() detected outside main function
```

## Testing

The analyzer includes comprehensive tests using `golang.org/x/tools/go/analysis/analysistest`:

```bash
go test -v ./internal/analysis
```

Test data is organized in `testdata/src/`:
- `bad/`: Code that should trigger analyzer warnings (panic usage)
- `good/`: Code that correctly avoids issues
- `mainpkg/`: Valid main package with correct main function usage
- `mainbad/`: Main package with issues outside main function

## Implementation Details

The analyzer is built using the standard `golang.org/x/tools/go/analysis` framework, which provides:
- AST (Abstract Syntax Tree) inspection
- Proper integration with Go tooling
- Compatible with other analysis tools


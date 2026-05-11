package semantic_test

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"testing"

	splapp "github.com/MDx3R/spl/internal/app/spl"
	"github.com/MDx3R/spl/internal/parser"
	"github.com/MDx3R/spl/internal/semantic"
)

// Run tests with -update to regenerate all golden files.
var update = flag.Bool("update", false, "regenerate golden files")

// parseTestFile cleans, scans, and parses the file at path, returning the AST.
func parseTestFile(t *testing.T, path string) *parser.File {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	var buf bytes.Buffer
	app := splapp.NewApp(&buf)

	cleaned, _, err := app.Cleaner.Clean(string(data))
	if err != nil {
		t.Fatalf("clean %s: %v", path, err)
	}
	buf.WriteString(cleaned)

	return app.Parse()
}

// semanticOutput runs the analyzer and triad emitter on file and returns the
// concatenated symbol table + error report + triad listing.
func semanticOutput(file *parser.File) string {
	a := semantic.NewAnalyzer()
	a.AnalyzeFile(file)

	e := semantic.NewTriadEmitter()
	e.EmitFile(file, a.Result())

	return a.FormatTable() + a.FormatErrors() + e.String()
}

// checkGolden compares got to the content of goldenPath.
// When -update is set it overwrites the golden file instead.
func checkGolden(t *testing.T, goldenPath, got string) {
	t.Helper()
	if *update {
		if err := os.WriteFile(goldenPath, []byte(got), 0644); err != nil {
			t.Fatalf("write golden %s: %v", goldenPath, err)
		}
		t.Logf("updated %s", goldenPath)
		return
	}
	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden %s: %v (run with -update to create it)", goldenPath, err)
	}
	if got != string(want) {
		t.Errorf("output does not match golden file %s\n--- want ---\n%s\n--- got ---\n%s",
			goldenPath, want, got)
	}
}

// goldenPath returns the path of the golden file for the given source file.
func goldenPath(src string) string {
	abs, _ := filepath.Abs(filepath.Join("..", "..", "testdata", filepath.Base(src)+".semantic.golden"))
	return abs
}

func TestAnalyze_TestRust(t *testing.T) {
	const src = "../../testdata/test.rust"
	file := parseTestFile(t, src)
	got := semanticOutput(file)
	checkGolden(t, goldenPath(src), got)
}

func TestAnalyze_SemanticInvalid(t *testing.T) {
	const src = "../../testdata/test.rust.semantic-invalid"
	file := parseTestFile(t, src)
	got := semanticOutput(file)
	checkGolden(t, goldenPath(src), got)
}

// TestSymbolTable_Fields validates the symbol table structure for test.rust
// without relying on a golden file – this catches regressions in individual fields.
func TestSymbolTable_Fields(t *testing.T) {
	file := parseTestFile(t, "../../testdata/test.rust")

	a := semantic.NewAnalyzer()
	a.AnalyzeFile(file)

	type wantSym struct {
		name      string
		typStr    string // String representation of the expected type
		mut       bool
		scopeName string
	}

	want := []wantSym{
		{"main", "fn() -> unit", false, "global"},
		{"x", "unknown", true, "main"},
		{"emoji", "string", false, "main"},
		{"y", "i32", false, "main"},
		{"sum", "unknown", false, "main"},
		{"difference", "unknown", false, "main"},
		{"product", "unknown", false, "main"},
		{"division", "unknown", false, "main"},
		{"is_greater", "bool", false, "main"},
		{"i", "i32", false, "for"},
		{"counter", "i32", true, "main"},
		{"factorial_result", "unknown", false, "main"},
		{"even_check", "unknown", false, "main"},
	}

	got := a.Symbols()
	if len(got) != len(want) {
		t.Fatalf("symbol count: got %d, want %d", len(got), len(want))
	}
	for i, w := range want {
		s := got[i]
		if s.Name != w.name {
			t.Errorf("[%d] Name: got %q, want %q", i, s.Name, w.name)
		}
		gotType := "unknown"
		if s.Type != nil {
			gotType = s.Type.String()
		}
		if gotType != w.typStr {
			t.Errorf("[%d] %s Type: got %s, want %s", i, s.Name, gotType, w.typStr)
		}
		if s.Mutable != w.mut {
			t.Errorf("[%d] %s Mutable: got %v, want %v", i, s.Name, s.Mutable, w.mut)
		}
		if s.ScopeName != w.scopeName {
			t.Errorf("[%d] %s ScopeName: got %q, want %q", i, s.Name, s.ScopeName, w.scopeName)
		}
		if !s.Initialized {
			t.Errorf("[%d] %s Initialized: got false, want true", i, s.Name)
		}
	}
}

func TestErrors_TestRust(t *testing.T) {
	file := parseTestFile(t, "../../testdata/test.rust")
	a := semantic.NewAnalyzer()
	a.AnalyzeFile(file)

	errs := a.Errors()
	if len(errs) != 4 {
		t.Fatalf("error count: got %d, want 4; errors: %v", len(errs), errs)
	}
	// First three should be undeclared
	for i := 0; i < 3; i++ {
		if errs[i].Kind != semantic.ErrUndeclared {
			t.Errorf("error [%d] kind: got %q, want %q", i, errs[i].Kind, semantic.ErrUndeclared)
		}
	}
	// Fourth should be type mismatch (x with unknown type being assigned i32)
	if errs[3].Kind != semantic.ErrTypeMismatch {
		t.Errorf("error [3] kind: got %q, want %q", errs[3].Kind, semantic.ErrTypeMismatch)
	}
}

func TestErrors_SemanticInvalid(t *testing.T) {
	file := parseTestFile(t, "../../testdata/test.rust.semantic-invalid")
	a := semantic.NewAnalyzer()
	a.AnalyzeFile(file)

	errs := a.Errors()
	if len(errs) != 4 {
		t.Fatalf("error count: got %d, want 4; errors: %v", len(errs), errs)
	}

	kinds := []semantic.ErrorKind{
		semantic.ErrDuplicate,
		semantic.ErrUndeclared,
		semantic.ErrImmutable,
		semantic.ErrTypeMismatch,
	}
	for i, want := range kinds {
		if errs[i].Kind != want {
			t.Errorf("[%d] Kind: got %q, want %q", i, errs[i].Kind, want)
		}
	}
}

func TestTriadCount_TestRust(t *testing.T) {
	file := parseTestFile(t, "../../testdata/test.rust")
	e := semantic.NewTriadEmitter()
	e.EmitFile(file, nil)
	if len(e.Triads()) != 73 {
		t.Errorf("triad count: got %d, want 73", len(e.Triads()))
	}
}

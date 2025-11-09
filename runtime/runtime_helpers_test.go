package runtime

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sergev/gisp/lang"
)

func TestReadFileSkippingShebang(t *testing.T) {
	dir := t.TempDir()

	withShebang := filepath.Join(dir, "script.gs")
	if err := os.WriteFile(withShebang, []byte("#!/usr/bin/env gisp\n(+ 1 2)\n"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}
	data, err := readFileSkippingShebang(withShebang)
	if err != nil {
		t.Fatalf("readFileSkippingShebang error: %v", err)
	}
	if string(data) != "(+ 1 2)\n" {
		t.Fatalf("expected shebang to be stripped, got %q", data)
	}

	onlyShebang := filepath.Join(dir, "only_shebang.gs")
	if err := os.WriteFile(onlyShebang, []byte("#!/bin/true"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}
	data, err = readFileSkippingShebang(onlyShebang)
	if err != nil {
		t.Fatalf("readFileSkippingShebang error: %v", err)
	}
	if len(data) != 0 {
		t.Fatalf("expected empty body for shebang-only script, got %q", data)
	}

	noShebang := filepath.Join(dir, "plain.gs")
	if err := os.WriteFile(noShebang, []byte("(display \"hi\")"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}
	data, err = readFileSkippingShebang(noShebang)
	if err != nil {
		t.Fatalf("readFileSkippingShebang error: %v", err)
	}
	if string(data) != "(display \"hi\")" {
		t.Fatalf("expected content unchanged, got %q", data)
	}
}

func TestEvaluateFileDispatchesByExtension(t *testing.T) {
	dir := t.TempDir()

	gispScript := filepath.Join(dir, "prog.gisp")
	gispSrc := `
func inc(n) {
	return n + 1;
}
inc(41);
`
	if err := os.WriteFile(gispScript, []byte(gispSrc), 0o600); err != nil {
		t.Fatalf("write gisp script: %v", err)
	}

	schemeScript := filepath.Join(dir, "prog.gs")
	if err := os.WriteFile(schemeScript, []byte("(* 3 4)"), 0o600); err != nil {
		t.Fatalf("write scheme script: %v", err)
	}

	ev := NewEvaluator()
	SetArgv(ev.Global, []string{})

	val, err := EvaluateFile(ev, gispScript)
	if err != nil {
		t.Fatalf("EvaluateFile gisp error: %v", err)
	}
	if val.Type != lang.TypeInt || val.Int() != 42 {
		t.Fatalf("expected 42 from gisp script, got %v", val)
	}

	val, err = EvaluateFile(ev, schemeScript)
	if err != nil {
		t.Fatalf("EvaluateFile scheme error: %v", err)
	}
	if val.Type != lang.TypeInt || val.Int() != 12 {
		t.Fatalf("expected 12 from scheme script, got %v", val)
	}
}

func TestSetArgvProducesSchemeList(t *testing.T) {
	env := lang.NewEnv(nil)
	SetArgv(env, []string{"foo", "bar"})

	val, err := env.Get("*argv*")
	if err != nil {
		t.Fatalf("Get *argv*: %v", err)
	}
	items, err := lang.ToSlice(val)
	if err != nil {
		t.Fatalf("ToSlice argv: %v", err)
	}
	if len(items) != 2 || items[0].Str() != "foo" || items[1].Str() != "bar" {
		t.Fatalf("unexpected argv contents: %v", items)
	}
}

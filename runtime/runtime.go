package runtime

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/sergev/gisp/lang"
	gispparser "github.com/sergev/gisp/parser"
	"github.com/sergev/gisp/reader"
)

// NewEvaluator constructs an evaluator with the standard runtime installed.
func NewEvaluator() *lang.Evaluator {
	ev := lang.NewEvaluator()
	installPrimitives(ev)
	if err := installLibrary(ev); err != nil {
		panic(fmt.Errorf("runtime bootstrap failed: %w", err))
	}
	return ev
}

// SetArgv stores the command-line arguments as a Scheme list in the given environment.
func SetArgv(env *lang.Env, args []string) {
	values := make([]lang.Value, len(args))
	for i, arg := range args {
		values[i] = lang.StringValue(arg)
	}
	env.Define("*argv*", lang.List(values...))
}

func installLibrary(ev *lang.Evaluator) error {
	if len(preludeForms) == 0 {
		return nil
	}
	for _, form := range preludeForms {
		forms, err := reader.ReadString(form)
		if err != nil {
			return err
		}
		for _, expr := range forms {
			if _, err := ev.Eval(expr, nil); err != nil {
				return err
			}
		}
	}
	return nil
}

func readFileSkippingShebang(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if bytes.HasPrefix(data, []byte("#!")) {
		if idx := bytes.IndexByte(data, '\n'); idx >= 0 {
			return data[idx+1:], nil
		}
		return []byte{}, nil
	}
	return data, nil
}

// EvaluateReader consumes all expressions from the reader and evaluates them.
func EvaluateReader(ev *lang.Evaluator, r io.Reader) (lang.Value, error) {
	forms, err := reader.ReadAll(r)
	if err != nil {
		return lang.Value{}, err
	}
	return ev.EvalAll(forms, nil)
}

// EvaluateGispReader parses and evaluates Gisp source from the reader.
func EvaluateGispReader(ev *lang.Evaluator, r io.Reader) (lang.Value, error) {
	forms, err := gispparser.ParseReader(r)
	if err != nil {
		return lang.Value{}, err
	}
	return ev.EvalAll(forms, nil)
}

// EvaluateGispString parses and evaluates Gisp source from a string.
func EvaluateGispString(ev *lang.Evaluator, src string) (lang.Value, error) {
	forms, err := gispparser.ParseString(src)
	if err != nil {
		return lang.Value{}, err
	}
	return ev.EvalAll(forms, nil)
}

// EvaluateFile loads and executes a Scheme file, allowing #! shebang.
func EvaluateFile(ev *lang.Evaluator, path string) (lang.Value, error) {
	data, err := readFileSkippingShebang(path)
	if err != nil {
		return lang.Value{}, err
	}
	switch filepath.Ext(path) {
	case ".gisp":
		return EvaluateGispReader(ev, bytes.NewReader(data))
	default:
		return EvaluateReader(ev, bytes.NewReader(data))
	}
}

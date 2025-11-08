package runtime

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/sergev/gisp/internal/lang"
	"github.com/sergev/gisp/internal/reader"
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

// EvaluateFile loads and executes a Scheme file, allowing #! shebang.
func EvaluateFile(ev *lang.Evaluator, path string) (lang.Value, error) {
	data, err := readFileSkippingShebang(path)
	if err != nil {
		return lang.Value{}, err
	}
	return EvaluateReader(ev, bytes.NewReader(data))
}

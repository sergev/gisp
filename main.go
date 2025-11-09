package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/peterh/liner"
	"github.com/sergev/gisp/lang"
	"github.com/sergev/gisp/runtime"
	"github.com/sergev/gisp/sexpr"
)

func main() {
	ev := runtime.NewEvaluator()
	args := os.Args[1:]
	if len(args) > 0 {
		runtime.SetArgv(ev.Global, args)
		script := args[0]
		var err error
		if script == "-" {
			_, err = runtime.EvaluateReader(ev, os.Stdin)
		} else {
			_, err = runtime.EvaluateFile(ev, script)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "gisp: %v\n", err)
			os.Exit(1)
		}
		return
	}

	runtime.SetArgv(ev.Global, []string{})
	runREPL(ev)
}

func runREPL(ev *lang.Evaluator) {
	if !isInteractive() {
		runBufferedREPL(ev, bufio.NewReader(os.Stdin))
		return
	}
	runInteractiveREPL(ev)
}

func readerPkgReadString(src string) ([]lang.Value, error) {
	return sexpr.ReadString(src)
}

func isIncomplete(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "unterminated")
}

func runBufferedREPL(ev *lang.Evaluator, reader *bufio.Reader) {
	var buffer strings.Builder

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				if buffer.Len() == 0 {
					return
				}
			} else {
				fmt.Fprintf(os.Stderr, "read error: %v\n", err)
				return
			}
		}
		buffer.WriteString(line)
		src := buffer.String()
		forms, parseErr := readerPkgReadString(src)
		if parseErr != nil {
			if isIncomplete(parseErr) && !errors.Is(err, io.EOF) {
				continue
			}
			fmt.Fprintf(os.Stderr, "parse error: %v\n", parseErr)
			buffer.Reset()
			if errors.Is(err, io.EOF) {
				return
			}
			continue
		}
		buffer.Reset()
		for _, expr := range forms {
			val, evalErr := ev.Eval(expr, nil)
			if evalErr != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", evalErr)
				break
			}
			fmt.Println(val.String())
		}
		if errors.Is(err, io.EOF) {
			return
		}
	}
}

func runInteractiveREPL(ev *lang.Evaluator) {
	state := liner.NewLiner()
	defer state.Close()
	state.SetCtrlCAborts(true)

	historyPath := replHistoryPath()
	if historyPath != "" {
		if f, err := os.Open(historyPath); err == nil {
			state.ReadHistory(f)
			f.Close()
		}
		defer func() {
			if f, err := os.Create(historyPath); err == nil {
				state.WriteHistory(f)
				f.Close()
			}
		}()
	}

	var buffer strings.Builder

	for {
		prompt := "gisp> "
		if buffer.Len() > 0 {
			prompt = ".... "
		}
		input, err := state.Prompt(prompt)
		if err != nil {
			switch {
			case errors.Is(err, liner.ErrPromptAborted):
				fmt.Println()
				buffer.Reset()
				continue
			case errors.Is(err, io.EOF):
				fmt.Println()
				return
			default:
				fmt.Fprintf(os.Stderr, "read error: %v\n", err)
				return
			}
		}
		buffer.WriteString(input)
		buffer.WriteString("\n")

		src := buffer.String()
		forms, parseErr := readerPkgReadString(src)
		if parseErr != nil {
			if isIncomplete(parseErr) {
				continue
			}
			fmt.Fprintf(os.Stderr, "parse error: %v\n", parseErr)
			buffer.Reset()
			continue
		}

		buffer.Reset()
		if trimmed := strings.TrimSpace(src); trimmed != "" {
			state.AppendHistory(trimmed)
		}
		for _, expr := range forms {
			val, evalErr := ev.Eval(expr, nil)
			if evalErr != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", evalErr)
				break
			}
			fmt.Println(val.String())
		}
	}
}

func replHistoryPath() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, ".gisp_history")
}

func isInteractive() bool {
	info, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}

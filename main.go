package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/sergev/gisp/internal/lang"
	"github.com/sergev/gisp/internal/reader"
	"github.com/sergev/gisp/internal/runtime"
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
	reader := bufio.NewReader(os.Stdin)
	var buffer strings.Builder
	interactive := isInteractive()

	for {
		if interactive {
			if buffer.Len() == 0 {
				fmt.Print("gisp> ")
			} else {
				fmt.Print(".... ")
			}
		}
		line, err := reader.ReadString('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				if buffer.Len() == 0 {
					if interactive {
						fmt.Println()
					}
					return
				}
			} else {
				fmt.Fprintf(os.Stderr, "read error: %v\n", err)
				return
			}
		}
		buffer.WriteString(line)
		forms, parseErr := readerPkgReadString(buffer.String())
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

func readerPkgReadString(src string) ([]lang.Value, error) {
	return reader.ReadString(src)
}

func isIncomplete(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "unterminated")
}

func isInteractive() bool {
	info, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}

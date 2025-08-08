package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"runtime"
	"sync"

	"github.com/WJQSERVER/wanf"

	"github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"
)

const usage = `wanflint: a tool for linting and formatting WANF files.

Usage:
  wanflint <command> [arguments]

Commands:
  lint [path ...]   lint files and report issues
  fmt [path ...]    format files
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(1)
	}

	lintCmd := flag.NewFlagSet("lint", flag.ExitOnError)
	jsonOutput := lintCmd.Bool("json", false, "Output issues in JSON format")

	fmtCmd := flag.NewFlagSet("fmt", flag.ExitOnError)
	fmtStyle := fmtCmd.String("style", "default", "Output style (default, streaming, single-line)")
	concurrent := fmtCmd.Bool("concurrent", false, "Enable experimental concurrent formatting")

	switch os.Args[1] {
	case "lint":
		lintCmd.Parse(os.Args[2:])
		paths := lintCmd.Args()
		if len(paths) == 0 {
			fmt.Fprintln(os.Stderr, "Error: missing file paths for lint command.")
			os.Exit(1)
		}
		if err := lintFiles(paths, *jsonOutput); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "fmt":
		fmtCmd.Parse(os.Args[2:])
		paths := fmtCmd.Args()
		if len(paths) == 0 {
			fmt.Fprintln(os.Stderr, "Error: missing file paths for fmt command.")
			os.Exit(1)
		}
		if err := formatFiles(paths, *fmtStyle, *concurrent); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %q\n", os.Args[1])
		fmt.Fprint(os.Stderr, usage)
		os.Exit(1)
	}
}

func lintFiles(paths []string, jsonOutput bool) error {
	var allErrors []wanf.LintError
	hasParseErrors := false

	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", path, err)
			hasParseErrors = true
			continue
		}
		_, errs := wanf.Lint(data)
		if len(errs) > 0 {
			allErrors = append(allErrors, errs...)
		}
	}

	if jsonOutput {
		err := json.MarshalWrite(os.Stdout, allErrors, jsontext.Multiline(true), jsontext.WithIndent("  "))
		if err != nil {
			return fmt.Errorf("could not marshal json: %w", err)
		}
		return nil
	}

	if len(allErrors) > 0 {
		fmt.Fprintln(os.Stderr, "Linter found issues:")
		for _, e := range allErrors {
			fmt.Fprintf(os.Stderr, "  - [%s] %s:%d:%d: %s\n", e.Level, "file", e.Line, e.Column, e.Message)
		}
		return fmt.Errorf("linting found issues")
	}

	if hasParseErrors {
		return fmt.Errorf("errors encountered during linting")
	}

	return nil
}

func formatFiles(paths []string, style string, concurrent bool) error {
	if !concurrent {
		// 顺序格式化
		for _, path := range paths {
			if err := formatFile(path, style); err != nil {
				return err
			}
		}
		return nil
	}

	// 并发格式化
	numWorkers := runtime.NumCPU()
	pathsChan := make(chan string, len(paths))
	errChan := make(chan error, len(paths))
	var wg sync.WaitGroup

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range pathsChan {
				err := formatFile(path, style)
				if err != nil {
					errChan <- err
				}
			}
		}()
	}

	for _, path := range paths {
		pathsChan <- path
	}
	close(pathsChan)

	wg.Wait()
	close(errChan)

	var allErrors []error
	for err := range errChan {
		allErrors = append(allErrors, err)
	}

	if len(allErrors) > 0 {
		return errors.Join(allErrors...)
	}

	return nil
}

func formatFile(path string, style string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("could not read file %s: %w", path, err)
	}

	program, errs := wanf.Lint(data)
	if len(errs) > 0 {
		hasFatalError := false
		for _, e := range errs {
			// A "parser error: " prefix indicates a fatal error from which the
			// parser could not recover. Formatting should be aborted.
			if len(e.Message) > 14 && e.Message[:14] == "parser error: " {
				hasFatalError = true
				break
			}
		}

		if hasFatalError {
			fmt.Fprintf(os.Stderr, "Error: found fatal errors in %s, formatting aborted:\n", path)
			for _, e := range errs {
				fmt.Fprintf(os.Stderr, "  - %s\n", e.Error())
			}
			return fmt.Errorf("fatal errors encountered in %s", path)
		}

		// For non-fatal formatting warnings, just print them.
		fmt.Fprintf(os.Stderr, "Warning: found %d issues in %s:\n", len(errs), path)
		for _, e := range errs {
			fmt.Fprintf(os.Stderr, "  - %s\n", e.Error())
		}
	}

	opts := wanf.FormatOptions{EmptyLines: true}
	switch style {
	case "streaming":
		opts.Style = wanf.StyleStreaming
	case "single-line":
		opts.Style = wanf.StyleSingleLine
		opts.EmptyLines = false
	default:
		opts.Style = wanf.StyleDefault
	}

	formatted := wanf.Format(program, opts)
	if !bytes.Equal(data, formatted) {
		if err := os.WriteFile(path, formatted, 0644); err != nil {
			return fmt.Errorf("failed to write formatted file %s: %w", path, err)
		}
		fmt.Printf("Formatted %s\n", path)
	}
	return nil
}

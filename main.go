package main

import (
	"fmt"
	"io"
	"os"
	"rune/pkg/errors"
	"rune/pkg/helpers"
	"rune/pkg/rune"
	"strings"
)

const version = "0.1"

const (
	exitCodeOk         = 0
	exitCodeError      = 1
	exitCodeParseError = 65
	exitCodeEvalError  = 70

	runeExtension = ".rn"
)

func printUsage() {
	asciiArt := `
 ____  _   _ _   _ _____ 
|  _ \| | | | \ | | ____|
| |_) | | | |  \| |  _|  
|  _ <| |_| | |\  | |___ 
|_| \_\\___/|_| \_|_____|
`
	fmt.Fprintf(os.Stderr, "%s\n", asciiArt)
	fmt.Fprintf(os.Stderr, "Rune Interpreter v%s\n", version)
	fmt.Fprintf(os.Stderr, "Copyright: Alexander Satretdinov (c), 2025\n")
	fmt.Fprintf(os.Stderr, "Based on the Lox programming language and Robert Nystrom's book.\n")
	fmt.Fprintf(os.Stderr, "A simple interpreter for processing and evaluating scripts.\n\n")
	fmt.Fprintf(os.Stderr, "Usage: rune <command> <filename>\n")
	fmt.Fprintf(os.Stderr, "Commands:\n")
	fmt.Fprintf(os.Stderr, "  tokenize  - Tokenizes the input file\n")
	fmt.Fprintf(os.Stderr, "  evaluate  - Evaluates a single expression from the input file\n")
	fmt.Fprintf(os.Stderr, "  run       - Runs the program from the input file\n")
	fmt.Fprintf(os.Stderr, "  disasm    - Prints the bytecode of the program in the input file\n")
	fmt.Fprintf(os.Stderr, "  version   - Prints the version of the interpreter\n")
	os.Exit(1)
}

var commands = map[string]func(stdout, stderr io.Writer, src []byte) int{
	"tokenize": tokenize,
	"evaluate": evaluate,
	"run":      run,
	"disasm":   disasm,
}

func tokenize(stdout, stderr io.Writer, src []byte) int {
	tokens, errs := rune.Scan(src)
	for _, token := range tokens {
		fmt.Fprintln(stdout, token)
	}

	if len(errs) == 0 {
		return exitCodeOk
	}

	printErrors(stderr, errs, src)

	return exitCodeParseError
}

func evaluate(stdout, stderr io.Writer, src []byte) int {
	tokens, errs := rune.Scan(src)
	if len(errs) > 0 {
		printErrors(stderr, errs, src)
		return exitCodeParseError
	}

	expr, err := rune.ParseExpr(tokens)
	if err != nil {
		printErrors(stderr, []error{err}, src)
		return exitCodeParseError
	}

	result, err := rune.EvaluateExpr(expr)
	if err != nil {
		printErrors(stderr, []error{err}, src)
		return exitCodeEvalError
	}

	fmt.Fprintln(stdout, helpers.Stringify(result))

	return exitCodeOk
}

func printErrors(w io.Writer, errs []error, src []byte) {
	for _, err := range errs {
		fmt.Fprint(w, errors.Report(err, string(src)))
	}
}

func run(stdout, stderr io.Writer, src []byte) int {
	prog, errs := rune.Load(src)
	if len(errs) > 0 {
		printErrors(stderr, errs, src)
		return exitCodeParseError
	}

	p := rune.NewInterpreter()
	p.SetOutput(stdout)

	if err := p.Run(prog); err != nil {
		printErrors(stderr, []error{err}, src)
		return exitCodeEvalError
	}

	return exitCodeOk
}

func disasm(stdout, stderr io.Writer, src []byte) int {
	prog, errs := rune.Load(src)
	if len(errs) > 0 {
		printErrors(stderr, errs, src)
		return exitCodeParseError
	}

	fmt.Fprint(stdout, prog.Disassemble())

	return exitCodeOk
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		return
	}

	command := os.Args[1]
	if command == "version" {
		fmt.Printf("Rune Interpreter v%s\n", version)
		return
	}

	if len(os.Args) < 3 {
		printUsage()
		return
	}

	fileName := os.Args[2]

	if !strings.HasSuffix(fileName, runeExtension) {
		fmt.Fprintf(os.Stderr, "Error: Only .rn files are supported. Provided file: %s\n", fileName)
		os.Exit(exitCodeError)
	}

	fileContents, err := os.ReadFile(fileName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(exitCodeError)
	}

	cmd, ok := commands[command]
	if !ok {
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		printUsage()
	}

	os.Exit(cmd(os.Stdout, os.Stderr, fileContents))
}

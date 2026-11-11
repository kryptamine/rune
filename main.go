package main

import (
	"fmt"
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
	fmt.Fprintf(os.Stderr, "  version   - Prints the version of the interpreter\n")
	os.Exit(1)
}

func tokenize(fileContents []byte) int {
	tokens, errs := rune.Scan(fileContents)
	for _, token := range tokens {
		fmt.Println(token)
	}

	if len(errs) == 0 {
		return exitCodeOk
	}

	printErrors(errs, fileContents)

	return exitCodeParseError
}

func evaluate(fileContents []byte) int {
	tokens, errs := rune.Scan(fileContents)
	if len(errs) > 0 {
		printErrors(errs, fileContents)
		return exitCodeParseError
	}

	expr, err := rune.ParseExpr(tokens)
	if err != nil {
		printError(err, fileContents)
		return exitCodeParseError
	}

	result, err := rune.EvaluateExpr(expr)
	if err != nil {
		printError(err, fileContents)
		return exitCodeEvalError
	}

	fmt.Println(helpers.Stringify(result))
	return exitCodeOk
}

func printErrors(errs []error, source []byte) {
	for _, err := range errs {
		printError(err, source)
	}
}

func printError(err error, source []byte) {
	fmt.Fprint(os.Stderr, errors.Report(err, string(source)))
}

func run(fileContents []byte) int {
	prog, errs := rune.Load(fileContents)
	if len(errs) > 0 {
		printErrors(errs, fileContents)
		return exitCodeParseError
	}

	if err := rune.NewInterpreter().Run(prog); err != nil {
		printError(err, fileContents)
		return exitCodeEvalError
	}

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

	switch command {
	case "tokenize":
		os.Exit(tokenize(fileContents))
	case "evaluate":
		os.Exit(evaluate(fileContents))
	case "run":
		os.Exit(run(fileContents))
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		printUsage()
	}
}

package main

import (
	"fmt"
	"os"
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
	tokens, errors := rune.Scan(fileContents)
	for _, token := range tokens {
		fmt.Println(token)
	}

	if len(errors) == 0 {
		return exitCodeOk
	}

	for _, err := range errors {
		fmt.Fprintf(os.Stderr, "%v\n", err)
	}

	return exitCodeParseError
}

func evaluate(fileContents []byte) int {
	tokens, errors := rune.Scan(fileContents)
	if len(errors) > 0 {
		return exitCodeParseError
	}

	expr, err := rune.ParseExpr(tokens)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return exitCodeParseError
	}

	result, err := rune.EvaluateExpr(expr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return exitCodeEvalError
	}

	fmt.Println(result)
	return exitCodeOk
}

func run(fileContents []byte) int {
	tokens, errors := rune.Scan(fileContents)

	if len(errors) > 0 {
		for _, err := range errors {
			fmt.Fprintf(os.Stderr, "%v\n", err)
		}

		return exitCodeParseError
	}

	stmts, err := rune.ParseStmts(tokens)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return exitCodeParseError
	}

	interpreter := rune.NewInterpreter()
	resolver := rune.NewResolver(interpreter)

	if err := resolver.ResolveStmts(stmts); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return exitCodeParseError
	}

	if err := interpreter.EvaluateStmts(stmts); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
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
		fmt.Println(fmt.Sprintf("Rune Interpreter v%s", version))
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

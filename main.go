package main

import (
	"fmt"
	"os"
	"rune/pkg/rune"
)

func printUsage() {
	fmt.Fprintf(os.Stderr, "Rune Interpreter v0.1\n")
	fmt.Fprintf(os.Stderr, "Copyright: Alexander Satretdinov (c)\n")
	fmt.Fprintf(os.Stderr, "A simple interpreter for processing and evaluating scripts.\n\n")
	fmt.Fprintf(os.Stderr, "Usage: rune <command> <filename>\n")
	fmt.Fprintf(os.Stderr, "Commands:\n")
	fmt.Fprintf(os.Stderr, "  tokenize  - Tokenizes the input file\n")
	fmt.Fprintf(os.Stderr, "  evaluate  - Evaluates a single expression from the input file\n")
	fmt.Fprintf(os.Stderr, "  run       - Runs the program from the input file\n")
	os.Exit(1)
}

func main() {
	if len(os.Args) < 3 {
		printUsage()
		return
	}

	command := os.Args[1]
	filename := os.Args[2]

	fileContents, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}

	tokens, errors := rune.Scan(fileContents)

	switch command {
	case "tokenize":
		for _, token := range tokens {
			fmt.Println(token)
		}

		if len(errors) == 0 {
			return
		}

		for _, err := range errors {
			fmt.Fprintf(os.Stderr, "%v\n", err)
		}

		os.Exit(65)

	case "evaluate":
		if len(errors) > 0 {
			os.Exit(65)
		}

		expr, err := rune.ParseExpr(tokens)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(65)
		}

		result, err := rune.EvaluateExpr(expr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(70)
		}

		fmt.Println(result)

	case "run":
		stmts, err := rune.ParseStmts(tokens)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(65)
		}

		if err := rune.EvaluateStmts(stmts); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(70)
		}
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		printUsage()
	}
}

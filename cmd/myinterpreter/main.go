package main

import (
	"fmt"
	"github.com/codecrafters-io/interpreter-starter-go/pkg/solus"
	"os"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: ./your_program.sh tokenize <filename>")
		os.Exit(1)
	}

	command := os.Args[1]
	filename := os.Args[2]

	fileContents, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
		return
	}

	tokens, errors := solus.Scan(fileContents)

	switch command {
	case "tokenize":
		for _, token := range tokens {
			fmt.Println(token)
		}

		if len(errors) > 0 {
			for _, err := range errors {
				fmt.Fprintf(os.Stderr, "%v\n", err)
			}

			os.Exit(65)
			return
		}
		break
	case "parse":
		if len(errors) > 0 {
			os.Exit(65)
			return
		}

		expr, err := solus.ParseExpr(tokens)

		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(65)
		}

		solus.PrintExpr(expr)
		break
	case "evaluate":
		if len(errors) > 0 {
			os.Exit(65)
		}

		expr, err := solus.ParseExpr(tokens)

		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(65)
		}

		result, err := solus.EvaluateExpr(expr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(70)
		}

		if result == nil {
			result = "nil"
		}

		fmt.Print(result)
		break

	case "run":
		stmts, err := solus.ParseStmts(tokens)

		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(65)
		}

		err = solus.EvaluateStmts(stmts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(70)
		}
		break
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		os.Exit(1)
		break
	}
}

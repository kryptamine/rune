package main

import (
	"fmt"
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
	}

	tokens, errors := Scan(fileContents)

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
		}
		break
	case "parse":
		if len(errors) > 0 {
			os.Exit(65)
			return
		}

		expr, err := Evaluate(tokens)

		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(65)
		}

		expr.accept(&PrintVisitor{})
		break
	case "evaluate":
		if len(errors) > 0 {
			os.Exit(65)
		}

		expr, err := Evaluate(tokens)

		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(65)
		}

		result, err := PrintExpr(expr)
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
		stmts, err := Parse(tokens)

		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(65)
		}

		err = Interpret(stmts)
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

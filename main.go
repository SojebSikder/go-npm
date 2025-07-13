package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/sojebsikder/go-npm/cmd"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage:")
		fmt.Println("  go-npm install [--package path/to/package.json]")
		fmt.Println("  go-npm init")
		fmt.Println("  go-npm add [--dev] <package[@version]> [...]")
		fmt.Println("  go-npm remove <package> [...]")
		fmt.Println("  go-npm ci")
		fmt.Println("  go-npm run <script>")
		return
	}

	cmdName := os.Args[1]
	switch cmdName {
	case "install":
		fs := flag.NewFlagSet("install", flag.ExitOnError)
		pkgPath := fs.String("package", "package.json", "Path to package.json")
		fs.Parse(os.Args[2:])
		cmd.RunInstall(*pkgPath)
	case "init":
		cmd.RunInit()
	case "add":
		cmd.RunAdd(os.Args[2:])
	case "remove":
		cmd.RunRemove(os.Args[2:])
	case "ci":
		cmd.RunCI()
	case "run":
		cmd.RunScript(os.Args[2:])
	default:
		fmt.Printf("Unknown command: %s\n", cmdName)
		fmt.Println("Available commands: install, init, add, remove, ci, run")
	}
}

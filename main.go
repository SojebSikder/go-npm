package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/sojebsikder/go-npm/cmd"
)

const (
	version = "0.1.0"
	appName = "snpm"
)

func printUsage() {
	fmt.Println("Usage:")
	fmt.Printf("%s install [--package path/to/package.json] \n", appName)
	fmt.Printf("%s init\n", appName)
	fmt.Printf("%s add [--dev] <package[@version]> [...]\n", appName)
	fmt.Printf("%s remove <package> [...] \n", appName)
	fmt.Printf("%s ci\n", appName)
	fmt.Printf("%s run <script>", appName)
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		return
	}

	cmdName := os.Args[1]
	switch cmdName {
	case "version":
		fmt.Printf("%s v%s\n", appName, version)
	case "help":
		printUsage()
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

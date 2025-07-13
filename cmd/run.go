package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/sojebsikder/go-npm/pkg"
)

func RunScript(args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: go-npm run <script>")
		return
	}

	scriptName := args[0]
	pkgJSON, err := pkg.LoadPackageJSON("package.json")
	if err != nil {
		fmt.Println("Error loading package.json:", err)
		return
	}

	command, ok := pkgJSON.Scripts[scriptName]
	if !ok {
		fmt.Printf("Script \"%s\" not found in package.json\n", scriptName)
		return
	}

	fmt.Printf("Running script \"%s\": %s\n", scriptName, command)

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/C", command)
	} else {
		cmd = exec.Command("sh", "-c", command)
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		fmt.Printf("Error running script \"%s\": %v\n", scriptName, err)
	}
}

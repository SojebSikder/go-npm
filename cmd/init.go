package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sojebsikder/go-npm/pkg"
)

func RunInit() {
	dirName := ""
	dir, err := os.Getwd()
	if err != nil {
		dirName = "app"
	} else {
		dirName = filepath.Base(dir)
	}

	defaultPkg := pkg.PackageJSON{
		Name:            dirName,
		Version:         "1.0.0",
		Dependencies:    map[string]string{},
		DevDependencies: map[string]string{},
	}

	if err := pkg.SavePackageJSON("package.json", &defaultPkg); err != nil {
		fmt.Println("Error writing package.json:", err)
		return
	}
	fmt.Println("Created package.json")
}

package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sojebsikder/go-npm/pkg"
)

func RunRemove(args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: go-npm remove <package> [...]")
		return
	}

	pkgJSON, err := pkg.LoadPackageJSON("package.json")
	if err != nil {
		fmt.Println("Error loading package.json:", err)
		return
	}

	lock, _ := pkg.LoadPackageLock("package-lock.json")

	changed := false
	for _, name := range args {
		if _, ok := pkgJSON.Dependencies[name]; ok {
			delete(pkgJSON.Dependencies, name)
			changed = true
		}
		if _, ok := pkgJSON.DevDependencies[name]; ok {
			delete(pkgJSON.DevDependencies, name)
			changed = true
		}
		if lock != nil {
			delete(lock.Lockfile, name)
			delete(lock.DevLock, name)
		}
		modPath := filepath.Join("node_modules", name)
		if err := os.RemoveAll(modPath); err != nil {
			fmt.Printf("Failed to remove %s from node_modules: %v\n", name, err)
		} else {
			fmt.Printf("Removed %s\n", name)
		}
	}

	if changed {
		pkg.SavePackageJSON("package.json", pkgJSON)
		if lock != nil {
			pkg.SavePackageLock("package-lock.json", lock)
		}
	}
}

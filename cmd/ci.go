package cmd

import (
	"fmt"
	"os"

	"github.com/sojebsikder/go-npm/pkg"
)

func RunCI() {
	lock, err := pkg.LoadPackageLock("package-lock.json")
	if err != nil {
		fmt.Println("Error loading package-lock.json:", err)
		return
	}

	err = os.RemoveAll("node_modules")
	if err != nil {
		fmt.Println("Error cleaning node_modules:", err)
		return
	}

	os.MkdirAll("node_modules", 0755)

	for name, dep := range lock.Lockfile {
		fmt.Println("Installing", name, dep.Version)
		if err := pkg.InstallPackage(name, dep.Version, lock.Lockfile, true); err != nil {
			fmt.Printf("Failed to install %s@%s: %v\n", name, dep.Version, err)
			return
		}
	}

	for name, dep := range lock.DevLock {
		fmt.Println("Installing dev dependency", name, dep.Version)
		if err := pkg.InstallPackage(name, dep.Version, lock.DevLock, true); err != nil {
			fmt.Printf("Failed to install dev dependency %s@%s: %v\n", name, dep.Version, err)
			return
		}
	}

	fmt.Println("Dependencies and devDependencies installed from package-lock.json")
}

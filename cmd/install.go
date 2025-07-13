package cmd

import (
	"fmt"
	"os"
	"sync"

	"github.com/sojebsikder/go-npm/pkg"
)

func RunInstall(pkgPath string) {
	pkgJSON, err := pkg.LoadPackageJSON(pkgPath)
	if err != nil {
		fmt.Println("Error loading package.json:", err)
		return
	}

	os.MkdirAll("node_modules", 0755)

	lock := &pkg.PackageLock{
		Name:     pkgJSON.Name,
		Version:  pkgJSON.Version,
		Lockfile: make(map[string]pkg.LockedDependency),
		DevLock:  make(map[string]pkg.LockedDependency),
	}

	var wg sync.WaitGroup
	errs := make(chan error, len(pkgJSON.Dependencies)+len(pkgJSON.DevDependencies))

	installSet := func(depMap map[string]string, lockMap map[string]pkg.LockedDependency) {
		for dep, ver := range depMap {
			wg.Add(1)
			go func(dep, ver string) {
				defer wg.Done()
				if err := pkg.InstallPackage(dep, ver, lockMap, false); err != nil {
					errs <- fmt.Errorf("error installing %s: %w", dep, err)
				}
			}(dep, ver)
		}
	}

	installSet(pkgJSON.Dependencies, lock.Lockfile)
	installSet(pkgJSON.DevDependencies, lock.DevLock)

	wg.Wait()
	close(errs)

	if len(errs) > 0 {
		fmt.Println("Errors occurred:")
		for e := range errs {
			fmt.Println("-", e)
		}
	} else {
		fmt.Println("All dependencies installed successfully!")
		pkg.SavePackageLock("package-lock.json", lock)
	}
}

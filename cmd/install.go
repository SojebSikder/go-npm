package cmd

import (
	"fmt"
	"os"
	"sync"

	"github.com/sojebsikder/go-npm/pkg"
)

// job struct to pass through the channel
type installJob struct {
	name    string
	version string
	lockMap map[string]pkg.LockedDependency
}

func RunInstall(pkgPath string) {
	pkgJSON, err := pkg.LoadPackageJSON(pkgPath)
	if err != nil {
		fmt.Printf("Error loading package.json: %v\n", err)
		return
	}

	fmt.Printf("Installing from %s\n", pkgPath)

	os.MkdirAll("node_modules", 0755)

	lock := &pkg.PackageLock{
		Name:     pkgJSON.Name,
		Version:  pkgJSON.Version,
		Lockfile: make(map[string]pkg.LockedDependency),
		DevLock:  make(map[string]pkg.LockedDependency),
	}

	// Setup Worker Pool
	const numWorkers = 5 // Limit concurrent downloads
	jobs := make(chan installJob)
	errs := make(chan error, 1000) // Buffer to prevent blocking
	var wg sync.WaitGroup

	// Start workers
	for w := 1; w <= numWorkers; w++ {
		go func() {
			for job := range jobs {
				if err := pkg.InstallPackage(job.name, job.version, job.lockMap, false); err != nil {
					errs <- fmt.Errorf("error installing %s: %w", job.name, err)
				}
				wg.Done()
			}
		}()
	}

	// Queue Top-level Dependencies
	queueDeps := func(depMap map[string]string, targetLock map[string]pkg.LockedDependency) {
		for dep, ver := range depMap {
			wg.Add(1)
			jobs <- installJob{
				name:    dep,
				version: ver,
				lockMap: targetLock,
			}
		}
	}

	fmt.Println("Resolving and installing dependencies...")
	queueDeps(pkgJSON.Dependencies, lock.Lockfile)
	queueDeps(pkgJSON.DevDependencies, lock.DevLock)

	// Cleanup
	go func() {
		wg.Wait()
		close(jobs)
		close(errs)
	}()

	// Wait for everything to finish
	wg.Wait()

	if len(errs) > 0 {
		fmt.Println("\nErrors occurred during installation:")
		for e := range errs {
			fmt.Println("-", e)
		}
	} else {
		fmt.Println("\nAll dependencies installed successfully!")
		pkg.SavePackageLock("package-lock.json", lock)
	}
}

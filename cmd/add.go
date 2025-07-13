package cmd

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/sojebsikder/go-npm/pkg"
)

func RunAdd(args []string) {
	fs := flag.NewFlagSet("add", flag.ExitOnError)
	isDev := fs.Bool("dev", false, "Add as devDependency")
	fs.Parse(args)

	pkgs := fs.Args()
	if len(pkgs) == 0 {
		fmt.Println("Usage: go-npm add [--dev] <package[@version]> [...]")
		return
	}

	pkgJSON, err := pkg.LoadPackageJSON("package.json")
	if err != nil {
		fmt.Println("Error loading package.json:", err)
		return
	}

	os.MkdirAll("node_modules", 0755)

	lock := make(map[string]pkg.LockedDependency)
	devLock := make(map[string]pkg.LockedDependency)

	for _, arg := range pkgs {
		var name, version string
		parts := strings.SplitN(arg, "@", 2)
		name = parts[0]
		if len(parts) == 2 {
			version = parts[1]
		} else {
			version = "latest"
		}

		if version == "latest" {
			meta, err := pkg.FetchPackageMeta(name)
			if err != nil {
				fmt.Printf("Error fetching %s: %v\n", name, err)
				continue
			}
			distTags, ok := meta["dist-tags"].(map[string]interface{})
			if !ok {
				fmt.Printf("dist-tags not found for %s\n", name)
				continue
			}
			latest, ok := distTags["latest"].(string)
			if !ok {
				fmt.Printf("latest tag not found for %s\n", name)
				continue
			}
			version = latest
		}

		if err := pkg.InstallPackage(name, version, lock, false); err != nil {
			fmt.Printf("Failed to install %s@%s: %v\n", name, version, err)
			continue
		}

		if *isDev {
			if pkgJSON.DevDependencies == nil {
				pkgJSON.DevDependencies = map[string]string{}
			}
			pkgJSON.DevDependencies[name] = "^" + version
			devLock[name] = lock[name]
		} else {
			if pkgJSON.Dependencies == nil {
				pkgJSON.Dependencies = map[string]string{}
			}
			pkgJSON.Dependencies[name] = "^" + version
		}
	}

	pkg.SavePackageJSON("package.json", pkgJSON)
	pkg.SavePackageLock("package-lock.json", &pkg.PackageLock{
		Name:     pkgJSON.Name,
		Version:  pkgJSON.Version,
		Lockfile: lock,
		DevLock:  devLock,
	})
}

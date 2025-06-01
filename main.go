package main

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"

	"github.com/Masterminds/semver/v3"
)

type PackageJSON struct {
	Name            string            `json:"name"`
	Version         string            `json:"version"`
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
	Scripts         map[string]string `json:"scripts"`
}

type PackageLock struct {
	Name     string                      `json:"name"`
	Version  string                      `json:"version"`
	Lockfile map[string]LockedDependency `json:"dependencies"`
	DevLock  map[string]LockedDependency `json:"devDependencies"`
}

type LockedDependency struct {
	Version  string `json:"version"`
	Resolved string `json:"resolved"`
}

func LoadPackageJSON(path string) (*PackageJSON, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var pkg PackageJSON
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, err
	}
	return &pkg, nil
}

func SavePackageJSON(path string, pkg *PackageJSON) error {
	data, err := json.MarshalIndent(pkg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func LoadPackageLock(path string) (*PackageLock, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var lock PackageLock
	if err := json.Unmarshal(data, &lock); err != nil {
		return nil, err
	}
	return &lock, nil
}

func SavePackageLock(path string, lock *PackageLock) error {
	data, err := json.MarshalIndent(lock, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func FetchPackageMeta(name string) (map[string]interface{}, error) {
	url := "https://registry.npmjs.org/" + name
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result, nil
}

func GetTarballURL(meta map[string]interface{}, version string) (string, error) {
	versions := meta["versions"].(map[string]interface{})
	verMeta, ok := versions[version].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("version %s not found", version)
	}
	dist := verMeta["dist"].(map[string]interface{})
	return dist["tarball"].(string), nil
}

func DownloadAndExtractTarball(url, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	gzr, err := gzip.NewReader(resp.Body)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		parts := strings.SplitN(hdr.Name, "/", 2)
		if len(parts) < 2 {
			continue
		}
		relPath := parts[1]
		target := filepath.Join(dest, relPath)

		switch hdr.Typeflag {
		case tar.TypeDir:
			os.MkdirAll(target, 0755)
		case tar.TypeReg:
			os.MkdirAll(filepath.Dir(target), 0755)
			outFile, err := os.Create(target)
			if err != nil {
				return err
			}
			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return err
			}
			outFile.Close()
		}
	}
	return nil
}

func InstallPackage(name, version string, lock map[string]LockedDependency, force bool) error {
	if !force {
		if _, exists := lock[name]; exists {
			return nil
		}
	}

	fmt.Println("Installing", name, version)
	meta, err := FetchPackageMeta(name)
	if err != nil {
		return err
	}

	if strings.HasPrefix(version, "^") || strings.HasPrefix(version, "~") || strings.Contains(version, ">") || strings.Contains(version, "<") {
		versions := meta["versions"].(map[string]interface{})
		var validVersions []*semver.Version
		for ver := range versions {
			v, err := semver.NewVersion(ver)
			if err == nil {
				validVersions = append(validVersions, v)
			}
		}
		sort.Sort(semver.Collection(validVersions))
		rangeConstraint, err := semver.NewConstraint(version)
		if err != nil {
			return err
		}
		for i := len(validVersions) - 1; i >= 0; i-- {
			if rangeConstraint.Check(validVersions[i]) {
				version = validVersions[i].String()
				break
			}
		}
	} else if strings.HasPrefix(version, "*") {
		// if version is "*", we need to install the latest version
		distTags, ok := meta["dist-tags"].(map[string]interface{})
		if !ok {
			return fmt.Errorf("dist-tags not found for package %s", name)
		}
		latest, ok := distTags["latest"].(string)
		if !ok {
			return fmt.Errorf("latest tag not found for package %s", name)
		}
		version = latest
	} else if version == "latest" {
		// if version is "latest", we need to install the latest version
		distTags, ok := meta["dist-tags"].(map[string]interface{})
		if !ok {
			return fmt.Errorf("dist-tags not found for package %s", name)
		}
		latest, ok := distTags["latest"].(string)
		if !ok {
			return fmt.Errorf("latest tag not found for package %s", name)
		}
		version = latest
	} else {
		// if version is a range, we need to install the latest version that satisfies the range
		versions := meta["versions"].(map[string]interface{})
		for ver := range versions {
			if strings.HasPrefix(ver, version) {
				version = ver
				break
			}
		}
	}

	tarballURL, err := GetTarballURL(meta, version)
	if err != nil {
		return err
	}

	dest := filepath.Join("node_modules", name)
	if err := DownloadAndExtractTarball(tarballURL, dest); err != nil {
		return err
	}

	lock[name] = LockedDependency{
		Version:  version,
		Resolved: tarballURL,
	}

	verMeta := meta["versions"].(map[string]interface{})[version].(map[string]interface{})
	if deps, ok := verMeta["dependencies"].(map[string]interface{}); ok {
		for dep, ver := range deps {
			if err := InstallPackage(dep, ver.(string), lock, force); err != nil {
				return err
			}
		}
	}
	return nil
}

func runInstall(pkgPath string) {
	pkg, err := LoadPackageJSON(pkgPath)
	if err != nil {
		fmt.Println("Error loading package.json:", err)
		return
	}

	os.MkdirAll("node_modules", 0755)

	lock := &PackageLock{
		Name:     pkg.Name,
		Version:  pkg.Version,
		Lockfile: make(map[string]LockedDependency),
		DevLock:  make(map[string]LockedDependency),
	}

	var wg sync.WaitGroup
	errs := make(chan error, len(pkg.Dependencies)+len(pkg.DevDependencies))

	installSet := func(depMap map[string]string, lockMap map[string]LockedDependency) {
		for dep, ver := range depMap {
			wg.Add(1)
			go func(dep, ver string) {
				defer wg.Done()
				if err := InstallPackage(dep, ver, lockMap, false); err != nil {
					errs <- fmt.Errorf("error installing %s: %w", dep, err)
				}
			}(dep, ver)
		}
	}

	installSet(pkg.Dependencies, lock.Lockfile)
	installSet(pkg.DevDependencies, lock.DevLock)

	wg.Wait()
	close(errs)

	if len(errs) > 0 {
		fmt.Println("Errors occurred:")
		for e := range errs {
			fmt.Println("-", e)
		}
	} else {
		fmt.Println("All dependencies installed successfully!")
		SavePackageLock("package-lock.json", lock)
	}
}

func runCI() {
	lock, err := LoadPackageLock("package-lock.json")
	if err != nil {
		fmt.Println("Error loading package-lock.json:", err)
		return
	}

	os.RemoveAll("node_modules")
	os.MkdirAll("node_modules", 0755)

	for name, dep := range lock.Lockfile {
		fmt.Println("Installing", name, dep.Version)
		if err := InstallPackage(name, dep.Version, lock.Lockfile, true); err != nil {
			fmt.Printf("Failed to install %s@%s: %v\n", name, dep.Version, err)
			return
		}
	}
	for name, dep := range lock.DevLock {
		fmt.Println("Installing dev dependency", name, dep.Version)
		if err := InstallPackage(name, dep.Version, lock.DevLock, true); err != nil {
			fmt.Printf("Failed to install dev dependency %s@%s: %v\n", name, dep.Version, err)
			return
		}
	}

	fmt.Println("Dependencies and devDependencies installed from package-lock.json")
}

func runInit() {
	dirName := ""
	dir, err := os.Getwd()
	if err != nil {
		dirName = "app"
	} else {
		dirName = filepath.Base(dir)
	}

	defaultPkg := PackageJSON{
		Name:            dirName,
		Version:         "1.0.0",
		Dependencies:    map[string]string{},
		DevDependencies: map[string]string{},
	}

	if err := SavePackageJSON("package.json", &defaultPkg); err != nil {
		fmt.Println("Error writing package.json:", err)
		return
	}
	fmt.Println("Created package.json")
}

func runAdd(args []string) {
	fs := flag.NewFlagSet("add", flag.ExitOnError)
	isDev := fs.Bool("dev", false, "Add as devDependency")
	fs.Parse(args)

	pkgs := fs.Args()
	if len(pkgs) == 0 {
		fmt.Println("Usage: go-npm add [--dev] <package[@version]> [...]")
		return
	}

	pkg, err := LoadPackageJSON("package.json")
	if err != nil {
		fmt.Println("Error loading package.json:", err)
		return
	}

	os.MkdirAll("node_modules", 0755)

	lock := make(map[string]LockedDependency)
	devLock := make(map[string]LockedDependency)

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
			meta, err := FetchPackageMeta(name)
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

		if err := InstallPackage(name, version, lock, false); err != nil {
			fmt.Printf("Failed to install %s@%s: %v\n", name, version, err)
			continue
		}

		if *isDev {
			if pkg.DevDependencies == nil {
				pkg.DevDependencies = map[string]string{}
			}
			pkg.DevDependencies[name] = "^" + version
			devLock[name] = lock[name]
		} else {
			if pkg.Dependencies == nil {
				pkg.Dependencies = map[string]string{}
			}
			pkg.Dependencies[name] = "^" + version
		}
	}

	SavePackageJSON("package.json", pkg)
	SavePackageLock("package-lock.json", &PackageLock{
		Name:     pkg.Name,
		Version:  pkg.Version,
		Lockfile: lock,
		DevLock:  devLock,
	})
}

func runRemove(args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: go-npm remove <package> [...]")
		return
	}

	pkg, err := LoadPackageJSON("package.json")
	if err != nil {
		fmt.Println("Error loading package.json:", err)
		return
	}

	lock, _ := LoadPackageLock("package-lock.json")

	changed := false
	for _, name := range args {
		if _, ok := pkg.Dependencies[name]; ok {
			delete(pkg.Dependencies, name)
			changed = true
		}
		if _, ok := pkg.DevDependencies[name]; ok {
			delete(pkg.DevDependencies, name)
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
		SavePackageJSON("package.json", pkg)
		if lock != nil {
			SavePackageLock("package-lock.json", lock)
		}
	}
}

func runScript(args []string) {

	if len(args) == 0 {
		fmt.Println("Usage: go-npm run <script>")
		return
	}

	scriptName := args[0]
	pkg, err := LoadPackageJSON("package.json")
	if err != nil {
		fmt.Println("Error loading package.json:", err)
		return
	}

	command, ok := pkg.Scripts[scriptName]
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

	cmd := os.Args[1]
	switch cmd {
	case "install":
		fs := flag.NewFlagSet("install", flag.ExitOnError)
		pkgPath := fs.String("package", "package.json", "Path to package.json")
		fs.Parse(os.Args[2:])
		runInstall(*pkgPath)
	case "init":
		runInit()
	case "add":
		runAdd(os.Args[2:])
	case "remove":
		runRemove(os.Args[2:])
	case "ci":
		runCI()
	case "run":
		runScript(os.Args[2:])
	default:
		fmt.Printf("Unknown command: %s\n", cmd)
		fmt.Println("Available commands: install, init, add, remove, ci, run")
	}
}

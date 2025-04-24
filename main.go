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
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/Masterminds/semver/v3"
)

type PackageJSON struct {
	Name         string            `json:"name"`
	Version      string            `json:"version"`
	Dependencies map[string]string `json:"dependencies"`
}

type PackageLock struct {
	Name     string                      `json:"name"`
	Version  string                      `json:"version"`
	Lockfile map[string]LockedDependency `json:"dependencies"`
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
	}

	var wg sync.WaitGroup
	errs := make(chan error, len(pkg.Dependencies))

	for dep, ver := range pkg.Dependencies {
		wg.Add(1)
		go func(dep, ver string) {
			defer wg.Done()
			if err := InstallPackage(dep, ver, lock.Lockfile, false); err != nil {
				errs <- fmt.Errorf("error installing %s: %w", dep, err)
			}
		}(dep, ver)
	}

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

	fmt.Println("Dependencies installed from package-lock.json")
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
		Name:         dirName,
		Version:      "1.0.0",
		Dependencies: map[string]string{},
	}

	if err := SavePackageJSON("package.json", &defaultPkg); err != nil {
		fmt.Println("Error writing package.json:", err)
		return
	}
	fmt.Println("Created package.json")
}

func runAdd(args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: go-npm add <package[@version]> [...]")
		return
	}

	pkg, err := LoadPackageJSON("package.json")
	if err != nil {
		fmt.Println("Error loading package.json:", err)
		return
	}

	os.MkdirAll("node_modules", 0755)

	lock := make(map[string]LockedDependency)

	for _, arg := range args {
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
			if latest, ok := meta["dist-tags"].(map[string]interface{})["latest"].(string); ok {
				version = latest
			} else {
				fmt.Printf("Could not determine latest version of %s\n", name)
				continue
			}
		}

		if err := InstallPackage(name, version, lock, false); err != nil {
			fmt.Printf("Failed to install %s@%s: %v\n", name, version, err)
			continue
		}

		if pkg.Dependencies == nil {
			pkg.Dependencies = map[string]string{}
		}
		pkg.Dependencies[name] = "^" + version
	}

	SavePackageJSON("package.json", pkg)
	SavePackageLock("package-lock.json", &PackageLock{
		Name:     pkg.Name,
		Version:  pkg.Version,
		Lockfile: lock,
	})
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage:")
		fmt.Println("  go-npm install [--package path/to/package.json]")
		fmt.Println("  go-npm init")
		fmt.Println("  go-npm add <package[@version]> [...]")
		fmt.Println("  go-npm ci")
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
	case "ci":
		runCI()
	default:
		fmt.Printf("Unknown command: %s\n", cmd)
		fmt.Println("Available commands: install, init, add, ci")
	}
}

package pkg

import (
	"fmt"
	"path/filepath"
	"sort"

	"github.com/Masterminds/semver/v3"
)

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

	resolvedVersion, err := resolveVersion(meta, version)
	if err != nil {
		return fmt.Errorf("error resolving %s@%s: %w", name, version, err)
	}
	version = resolvedVersion

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

	// Create .bin executables
	if err := CreateBinLinks(dest); err != nil {
		return err
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

func resolveVersion(meta map[string]interface{}, constraintStr string) (string, error) {
	versionsMap := meta["versions"].(map[string]interface{})

	// Handle "latest" or "*"
	if constraintStr == "latest" || constraintStr == "*" || constraintStr == "" {
		distTags := meta["dist-tags"].(map[string]interface{})
		return distTags["latest"].(string), nil
	}

	// Try to parse as a constraint (handles ^, ~, >, <, and .x)
	c, err := semver.NewConstraint(constraintStr)
	if err == nil {
		var validVersions []*semver.Version
		for ver := range versionsMap {
			v, err := semver.NewVersion(ver)
			if err == nil {
				validVersions = append(validVersions, v)
			}
		}
		sort.Sort(semver.Collection(validVersions))

		// Find the highest version that matches the constraint
		for i := len(validVersions) - 1; i >= 0; i-- {
			if c.Check(validVersions[i]) {
				return validVersions[i].String(), nil
			}
		}
	}

	// Fallback: Exact match
	if _, exists := versionsMap[constraintStr]; exists {
		return constraintStr, nil
	}

	return "", fmt.Errorf("version %s not found", constraintStr)
}

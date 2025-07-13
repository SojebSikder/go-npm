package pkg

import (
	"fmt"
	"sort"
	"strings"

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
		distTags := meta["dist-tags"].(map[string]interface{})
		version = distTags["latest"].(string)
	} else if version == "latest" {
		distTags := meta["dist-tags"].(map[string]interface{})
		version = distTags["latest"].(string)
	} else {
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

	dest := "node_modules/" + name
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

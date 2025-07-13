package pkg

import (
	"encoding/json"
	"os"
)

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

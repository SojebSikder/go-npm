package pkg_test

import (
	"os"
	"testing"

	"github.com/sojebsikder/go-npm/pkg"
)

func TestPackageLockReadWrite(t *testing.T) {
	filename := "test_package-lock.json"
	original := &pkg.PackageLock{
		Name:    "test",
		Version: "1.0.0",
		Lockfile: map[string]pkg.LockedDependency{
			"axios": {
				Version:  "1.2.0",
				Resolved: "https://registry.npmjs.org/axios/-/axios-1.2.0.tgz",
			},
		},
		DevLock: map[string]pkg.LockedDependency{
			"eslint": {
				Version:  "8.0.0",
				Resolved: "https://registry.npmjs.org/eslint/-/eslint-8.0.0.tgz",
			},
		},
	}

	err := pkg.SavePackageLock(filename, original)
	if err != nil {
		t.Fatalf("Failed to save package-lock.json: %v", err)
	}
	defer os.Remove(filename)

	loaded, err := pkg.LoadPackageLock(filename)
	if err != nil {
		t.Fatalf("Failed to load package-lock.json: %v", err)
	}

	if loaded.Lockfile["axios"].Version != "1.2.0" {
		t.Errorf("Lockfile entry not loaded correctly")
	}
	if loaded.DevLock["eslint"].Resolved != original.DevLock["eslint"].Resolved {
		t.Errorf("DevLock entry not loaded correctly")
	}
}

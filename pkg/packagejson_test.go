package pkg_test

import (
	"os"
	"testing"

	"github.com/sojebsikder/go-npm/pkg"
)

func TestPackageJSONReadWrite(t *testing.T) {
	filename := "test_package.json"
	original := &pkg.PackageJSON{
		Name:    "test",
		Version: "1.0.0",
		Dependencies: map[string]string{
			"left-pad": "^1.3.0",
		},
		DevDependencies: map[string]string{
			"jest": "^29.0.0",
		},
		Scripts: map[string]string{
			"test": "jest",
		},
	}

	err := pkg.SavePackageJSON(filename, original)
	if err != nil {
		t.Fatalf("Failed to save package.json: %v", err)
	}
	defer os.Remove(filename)

	loaded, err := pkg.LoadPackageJSON(filename)
	if err != nil {
		t.Fatalf("Failed to load package.json: %v", err)
	}

	if loaded.Name != original.Name || loaded.Version != original.Version {
		t.Errorf("Loaded package.json doesn't match original")
	}
	if loaded.Scripts["test"] != "jest" {
		t.Errorf("Loaded script doesn't match")
	}
}

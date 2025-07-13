package pkg_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sojebsikder/go-npm/pkg"
)

func TestInstallPackage(t *testing.T) {
	tempDir := t.TempDir()
	os.MkdirAll(filepath.Join(tempDir, "node_modules"), 0755)
	os.Chdir(tempDir)

	lock := make(map[string]pkg.LockedDependency)
	err := pkg.InstallPackage("is-sorted", "1.0.5", lock, false)
	if err != nil {
		t.Fatalf("Failed to install package: %v", err)
	}

	if _, ok := lock["is-sorted"]; !ok {
		t.Errorf("Package not added to lock")
	}

	// Check extracted files
	pkgPath := filepath.Join(tempDir, "node_modules", "is-sorted")
	if _, err := os.Stat(pkgPath); os.IsNotExist(err) {
		t.Errorf("Package directory not found: %v", pkgPath)
	}
}

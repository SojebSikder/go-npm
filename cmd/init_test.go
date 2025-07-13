package cmd_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/sojebsikder/go-npm/cmd"
	"github.com/sojebsikder/go-npm/pkg"
)

func TestRunInit(t *testing.T) {
	tempDir := t.TempDir()
	os.Chdir(tempDir)

	cmd.RunInit()

	data, err := os.ReadFile("package.json")
	if err != nil {
		t.Fatalf("package.json not created: %v", err)
	}

	var pkgJson pkg.PackageJSON
	if err := json.Unmarshal(data, &pkgJson); err != nil {
		t.Fatalf("Invalid package.json format: %v", err)
	}

	if pkgJson.Name == "" || pkgJson.Version != "1.0.0" {
		t.Errorf("Unexpected package.json values: %+v", pkgJson)
	}
}

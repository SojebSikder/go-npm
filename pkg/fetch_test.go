package pkg_test

import (
	"testing"

	"github.com/sojebsikder/go-npm/pkg"
)

func TestFetchPackageMeta(t *testing.T) {
	meta, err := pkg.FetchPackageMeta("is-even")
	if err != nil {
		t.Fatalf("Failed to fetch package metadata: %v", err)
	}
	if _, ok := meta["name"]; !ok {
		t.Errorf("Package metadata does not contain 'name'")
	}
}

func TestGetTarballURL(t *testing.T) {
	meta, err := pkg.FetchPackageMeta("lodash")
	if err != nil {
		t.Fatalf("Failed to fetch package metadata: %v", err)
	}
	version := "1.0.0"
	url, err := pkg.GetTarballURL(meta, version)
	if err != nil {
		t.Fatalf("Failed to get tarball URL: %v", err)
	}
	if url == "" {
		t.Errorf("Empty tarball URL returned")
	}
}

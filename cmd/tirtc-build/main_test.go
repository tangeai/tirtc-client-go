package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCopyTreePreservesLicenseClosure(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	target := filepath.Join(root, "target")
	if err := os.MkdirAll(filepath.Join(source, "tirtc", "third_party"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "tirtc", "third_party", "LICENSE"), []byte("license\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := copyTree(source, target); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(target, "tirtc", "third_party", "LICENSE"))
	if err != nil || string(data) != "license\n" {
		t.Fatalf("copied license = %q, %v", data, err)
	}
	if err := copyTree(source, target); err != nil {
		t.Fatalf("identical closure is not idempotent: %v", err)
	}
	if err := os.WriteFile(filepath.Join(source, "tirtc", "third_party", "LICENSE"), []byte("different\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := copyTree(source, target); err == nil {
		t.Fatal("copyTree accepted a conflicting license closure")
	}
}

func TestPlatformLibrariesRejectsUnsupportedTuple(t *testing.T) {
	if _, _, err := librariesForPlatform("windows", "amd64"); err == nil {
		t.Fatal("unsupported tuple was accepted")
	}
}

func TestGoBuildEnvironmentPinsMacOSDeploymentTarget(t *testing.T) {
	environ := []string{"PATH=/usr/bin", "MACOSX_DEPLOYMENT_TARGET=26.0", "VALUE=kept"}
	got := goBuildEnvironment("darwin", environ)
	want := []string{"PATH=/usr/bin", "VALUE=kept", "MACOSX_DEPLOYMENT_TARGET=11.5"}
	if len(got) != len(want) {
		t.Fatalf("environment length = %d, want %d: %v", len(got), len(want), got)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("environment[%d] = %q, want %q", index, got[index], want[index])
		}
	}
	if linux := goBuildEnvironment("linux", environ); &linux[0] != &environ[0] {
		t.Fatal("non-Darwin build environment was copied or changed")
	}
}

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
)

const modulePath = "github.com/tangeai/tirtc-client-go/v2"

const macOSDeploymentTarget = "11.5"

type moduleInfo struct {
	Path    string
	Version string
	Dir     string
}

func main() {
	if len(os.Args) < 2 || os.Args[1] != "build" {
		fatal("usage: tirtc-build build --output <executable> <package>")
	}
	flags := flag.NewFlagSet("build", flag.ExitOnError)
	output := flags.String("output", "", "output executable path")
	_ = flags.Parse(os.Args[2:])
	if *output == "" || flags.NArg() != 1 {
		fatal("build requires --output and one package")
	}
	selected, err := selectedModule()
	if err != nil {
		fatal(err.Error())
	}
	if version := ownVersion(); version != "(devel)" && version != selected.Version {
		fatal(fmt.Sprintf("tirtc-build %s does not match selected SDK %s", version, selected.Version))
	}
	absolute, err := filepath.Abs(*output)
	if err != nil {
		fatal(err.Error())
	}
	if err := os.MkdirAll(filepath.Dir(absolute), 0o755); err != nil {
		fatal(err.Error())
	}
	command := exec.Command("go", "build", "-o", absolute, flags.Arg(0))
	command.Env = goBuildEnvironment(runtime.GOOS, os.Environ())
	command.Stdout, command.Stderr = os.Stdout, os.Stderr
	if err := command.Run(); err != nil {
		fatal(err.Error())
	}
	libDir := filepath.Join(filepath.Dir(filepath.Dir(absolute)), "lib", "tirtc")
	if err := os.MkdirAll(libDir, 0o755); err != nil {
		fatal(err.Error())
	}
	tuple, libraries, err := platformLibraries()
	if err != nil {
		fatal(err.Error())
	}
	for _, name := range libraries {
		if err := copyFile(filepath.Join(selected.Dir, "internal", "native", "artifacts", tuple, "lib", name), filepath.Join(libDir, name)); err != nil {
			fatal(err.Error())
		}
	}
	licenseSource := filepath.Join(selected.Dir, "internal", "native", "artifacts", tuple, "share", "licenses")
	if err := copyTree(licenseSource, filepath.Join(filepath.Dir(filepath.Dir(absolute)), "share", "licenses")); err != nil {
		fatal(fmt.Sprintf("copy Runtime license closure: %s", err))
	}
	if err := setRelativeRPath(absolute, filepath.Join(selected.Dir, "internal", "native", "artifacts", tuple, "lib")); err != nil {
		fatal(err.Error())
	}
}

func goBuildEnvironment(goos string, environ []string) []string {
	if goos != "darwin" {
		return environ
	}
	const key = "MACOSX_DEPLOYMENT_TARGET="
	result := make([]string, 0, len(environ)+1)
	for _, entry := range environ {
		if !strings.HasPrefix(entry, key) {
			result = append(result, entry)
		}
	}
	return append(result, key+macOSDeploymentTarget)
}

func selectedModule() (moduleInfo, error) {
	command := exec.Command("go", "list", "-m", "-json", modulePath)
	data, err := command.Output()
	if err != nil {
		return moduleInfo{}, fmt.Errorf("resolve selected SDK: %w", err)
	}
	var value moduleInfo
	if err := json.Unmarshal(data, &value); err != nil {
		return value, err
	}
	if value.Path != modulePath || value.Dir == "" {
		return value, errors.New("selected SDK module directory is unavailable")
	}
	return value, nil
}

func ownVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "(devel)"
	}
	return info.Main.Version
}

func platformLibraries() (string, []string, error) {
	return librariesForPlatform(runtime.GOOS, runtime.GOARCH)
}

func librariesForPlatform(goos, goarch string) (string, []string, error) {
	switch goos + "/" + goarch {
	case "darwin/arm64":
		return "darwin-arm64", []string{"libtirtc_media.dylib", "libTiRTC.dylib", "libtgrtc.dylib"}, nil
	case "linux/amd64":
		return "linux-amd64", []string{"libtirtc_media.so", "libTiRTC.so", "libwebrtc.so"}, nil
	default:
		return "", nil, fmt.Errorf("unsupported platform %s/%s", goos, goarch)
	}
}

func setRelativeRPath(executable, sourceRPath string) error {
	var command *exec.Cmd
	if runtime.GOOS == "darwin" {
		remove := exec.Command("install_name_tool", "-delete_rpath", sourceRPath, executable)
		if output, err := remove.CombinedOutput(); err != nil {
			return fmt.Errorf("remove module-cache runtime path: %s: %w", output, err)
		}
		command = exec.Command("install_name_tool", "-add_rpath", "@executable_path/../lib/tirtc", executable)
	} else {
		path, err := exec.LookPath("patchelf")
		if err != nil {
			return errors.New("patchelf is required for linux-amd64")
		}
		command = exec.Command(path, "--set-rpath", "$ORIGIN/../lib/tirtc", executable)
	}
	output, err := command.CombinedOutput()
	if err != nil {
		return fmt.Errorf("set relative runtime path: %s: %w", output, err)
	}
	if runtime.GOOS == "darwin" {
		output, err = exec.Command("codesign", "--force", "--sign", "-", executable).CombinedOutput()
		if err != nil {
			return fmt.Errorf("ad-hoc sign relocated executable: %s: %w", output, err)
		}
	}
	return nil
}

func copyFile(source, target string) error {
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()
	output, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
	if err != nil {
		return err
	}
	if _, err := io.Copy(output, input); err != nil {
		_ = output.Close()
		return err
	}
	return output.Close()
}

func copyTree(source, target string) error {
	info, err := os.Stat(source)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("source is not a directory: %s", source)
	}
	return filepath.WalkDir(source, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		if relative == "." {
			return os.MkdirAll(target, 0o755)
		}
		if strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.IsAbs(relative) {
			return errors.New("license path escaped its closure")
		}
		destination := filepath.Join(target, relative)
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("license closure contains a symlink: %s", relative)
		}
		if entry.IsDir() {
			return os.MkdirAll(destination, 0o755)
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("license closure contains a non-regular file: %s", relative)
		}
		return copyRegularFile(path, destination)
	})
}

func copyRegularFile(source, target string) error {
	if existing, err := os.ReadFile(target); err == nil {
		candidate, readErr := os.ReadFile(source)
		if readErr != nil {
			return readErr
		}
		if bytes.Equal(existing, candidate) {
			return nil
		}
		return fmt.Errorf("existing license differs: %s", target)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()
	output, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	if _, err := io.Copy(output, input); err != nil {
		_ = output.Close()
		return err
	}
	return output.Close()
}

func fatal(message string) { fmt.Fprintln(os.Stderr, message); os.Exit(2) }

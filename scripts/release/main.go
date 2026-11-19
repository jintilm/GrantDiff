// Command release builds and packages the supported targets without publishing.
package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

func main() {
	version := flag.String("version", "0.5.0-preview", "release version")
	flag.Parse()
	if err := run(*version); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(version string) error {
	if !regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+(?:-[A-Za-z0-9.-]+)?$`).MatchString(version) {
		return fmt.Errorf("invalid version")
	}
	if err := os.MkdirAll("dist", 0755); err != nil {
		return err
	}
	if err := dependencyLicenses(); err != nil {
		return err
	}
	checksums := strings.Builder{}
	for _, target := range []string{"windows", "linux"} {
		name := "grantdiff"
		if target == "windows" {
			name += ".exe"
		}
		directory := filepath.Join("dist", target+"-amd64")
		if err := os.MkdirAll(directory, 0755); err != nil {
			return err
		}
		binary := filepath.Join(directory, name)
		cmd := exec.Command("go", "build", "-trimpath", "-ldflags=-s -w -X github.com/jintilm/GrantDiff/internal/cli.Version="+version, "-o", binary, "./cmd/grantdiff")
		cmd.Env = append(os.Environ(), "GOOS="+target, "GOARCH=amd64", "CGO_ENABLED=0")
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
		if err := cmd.Run(); err != nil {
			return err
		}
		files := []struct{ name, path string }{{name, binary}, {"README.md", "README.md"}, {"README.zh-CN.md", "README.zh-CN.md"}, {"CONTRIBUTING.md", "CONTRIBUTING.md"}, {"CHANGELOG.md", "CHANGELOG.md"}, {"LICENSE", "LICENSE"}, {"NOTICE", "NOTICE"}, {"THIRD_PARTY_LICENSES.txt", "dist/THIRD_PARTY_LICENSES.txt"}}
		for _, root := range []string{"examples", "docs"} {
			if err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if !entry.IsDir() {
					files = append(files, struct{ name, path string }{filepath.ToSlash(path), path})
				}
				return nil
			}); err != nil {
				return err
			}
		}
		var data bytes.Buffer
		archiveName := fmt.Sprintf("grantdiff_%s_%s_amd64", version, target)
		if target == "windows" {
			archiveName += ".zip"
			writer := zip.NewWriter(&data)
			for _, item := range files {
				content, err := os.ReadFile(item.path)
				if err != nil {
					return err
				}
				entry, err := writer.Create(item.name)
				if err != nil {
					return err
				}
				if _, err := entry.Write(content); err != nil {
					return err
				}
			}
			if err := writer.Close(); err != nil {
				return err
			}
		} else {
			archiveName += ".tar.gz"
			compressed := gzip.NewWriter(&data)
			writer := tar.NewWriter(compressed)
			for _, item := range files {
				content, err := os.ReadFile(item.path)
				if err != nil {
					return err
				}
				mode := int64(0644)
				if item.name == name {
					mode = 0755
				}
				if err := writer.WriteHeader(&tar.Header{Name: item.name, Size: int64(len(content)), Mode: mode}); err != nil {
					return err
				}
				if _, err := writer.Write(content); err != nil {
					return err
				}
			}
			if err := writer.Close(); err != nil {
				return err
			}
			if err := compressed.Close(); err != nil {
				return err
			}
		}
		if err := os.WriteFile(filepath.Join("dist", archiveName), data.Bytes(), 0644); err != nil {
			return err
		}
		fmt.Fprintf(&checksums, "%x  %s\n", sha256.Sum256(data.Bytes()), archiveName)
		fmt.Println(archiveName)
	}
	return os.WriteFile(filepath.Join("dist", "SHA256SUMS"), []byte(checksums.String()), 0644)
}

func dependencyLicenses() error {
	cmd := exec.Command("go", "list", "-deps", "-json", "./cmd/grantdiff")
	cmd.Stderr = os.Stderr
	data, err := cmd.Output()
	if err != nil {
		return err
	}
	type module struct {
		Path, Version, Dir string
		Main               bool
	}
	modules := map[string]module{}
	decoder := json.NewDecoder(bytes.NewReader(data))
	for {
		var dependency struct{ Module *module }
		if err := decoder.Decode(&dependency); err == io.EOF {
			break
		} else if err != nil {
			return err
		}
		if dependency.Module != nil && !dependency.Module.Main {
			modules[dependency.Module.Path] = *dependency.Module
		}
	}
	paths := []string{}
	for path := range modules {
		paths = append(paths, path)
	}
	slices.Sort(paths)
	var result bytes.Buffer
	for _, path := range paths {
		dependency := modules[path]
		licenses, err := filepath.Glob(filepath.Join(dependency.Dir, "LICENSE*"))
		if err != nil {
			return err
		}
		if len(licenses) == 0 {
			return fmt.Errorf("license file missing for %s", path)
		}
		notices, err := filepath.Glob(filepath.Join(dependency.Dir, "NOTICE*"))
		if err != nil {
			return err
		}
		for _, file := range append(licenses, notices...) {
			text, err := os.ReadFile(file)
			if err != nil {
				return err
			}
			fmt.Fprintf(&result, "%s %s — %s\n\n%s\n\n", path, dependency.Version, filepath.Base(file), text)
		}
	}
	return os.WriteFile("dist/THIRD_PARTY_LICENSES.txt", result.Bytes(), 0644)
}

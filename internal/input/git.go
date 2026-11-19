package input

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"path"
	"strings"
	"time"

	"github.com/jintilm/GrantDiff/internal/model"
)

func Git(repo, ref, directory string, opts Options) (*model.State, error) {
	if ref == "" || strings.HasPrefix(ref, "-") {
		return nil, fmt.Errorf("invalid Git revision")
	}
	directory = path.Clean(strings.ReplaceAll(directory, "\\", "/"))
	if path.IsAbs(directory) || directory == ".." || strings.HasPrefix(directory, "../") || strings.Contains(directory, ":") {
		return nil, fmt.Errorf("Git path must be relative to repository")
	}
	commit, err := gitOutput(repo, "rev-parse", "--verify", "--end-of-options", ref+"^{commit}")
	if err != nil {
		return nil, err
	}
	sha := strings.TrimSpace(string(commit))
	listing, err := gitOutput(repo, "ls-tree", "-rz", "--full-tree", sha, "--", directory)
	if err != nil {
		return nil, err
	}
	files := []File{}
	total := 0
	for _, record := range bytes.Split(listing, []byte{0}) {
		if len(record) == 0 {
			continue
		}
		header, name, ok := strings.Cut(string(record), "\t")
		if !ok {
			return nil, fmt.Errorf("invalid git ls-tree response")
		}
		parts := strings.Fields(header)
		if len(parts) != 3 {
			return nil, fmt.Errorf("invalid git ls-tree header")
		}
		if !IsManifest(name) {
			continue
		}
		if parts[0] != "100644" && parts[0] != "100755" {
			return nil, fmt.Errorf("manifest must be a regular Git blob: %s", name)
		}
		if len(files) >= MaxFiles {
			return nil, fmt.Errorf("Git input exceeds file limit")
		}
		data, err := gitOutput(repo, "cat-file", "blob", parts[2])
		if err != nil {
			return nil, err
		}
		total += len(data)
		if total > MaxBytes {
			return nil, fmt.Errorf("Git input exceeds byte limit")
		}
		files = append(files, File{Name: name, Data: data})
	}
	return Parse(files, "git:"+sha+":"+directory, opts)
}

func gitOutput(repo string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", repo}, args...)...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	data, readErr := io.ReadAll(io.LimitReader(stdout, MaxBytes+1))
	if len(data) > MaxBytes {
		cancel()
	}
	err = cmd.Wait()
	if len(data) > MaxBytes {
		return nil, fmt.Errorf("Git output exceeds byte limit")
	}
	if readErr != nil {
		return nil, readErr
	}
	if err != nil {
		return nil, fmt.Errorf("git %s: %w: %s", args[0], err, strings.TrimSpace(stderr.String()))
	}
	return data, nil
}

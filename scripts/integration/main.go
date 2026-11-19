// Command integration owns a disposable kind cluster and runs the API oracle tests.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

func main() {
	kind := flag.String("kind", "kind", "kind executable")
	image := flag.String("image", "kindest/node:v1.37.0@sha256:a1ed56cfb0e7b93589bdf97c8cd566405a265939e3620fc4f5de89adff580ae5", "pinned kind node image")
	result := flag.String("result", "", "write verification JSON to path")
	flag.Parse()
	if err := run(*kind, *image, *result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(kind, image, result string) error {
	name := fmt.Sprintf("grantdiff-%d", time.Now().UnixNano())
	directory, err := os.MkdirTemp("", "grantdiff-integration-")
	if err != nil {
		return err
	}
	kubeconfig := filepath.Join(directory, "kubeconfig")
	defer func() {
		_ = os.Remove(kubeconfig)
		_ = os.Remove(directory)
	}()
	create := exec.Command(kind, "create", "cluster", "--name", name, "--image", image, "--wait", "120s", "--kubeconfig", kubeconfig)
	create.Stdout, create.Stderr = os.Stdout, os.Stderr
	// Cleanup is tied to the unique name generated in this invocation.
	defer func() {
		cleanup := exec.Command(kind, "delete", "cluster", "--name", name)
		cleanup.Stdout, cleanup.Stderr = os.Stdout, os.Stderr
		if err := cleanup.Run(); err != nil {
			fmt.Fprintln(os.Stderr, "cluster cleanup failed:", err)
		}
	}()
	if err := create.Run(); err != nil {
		return err
	}
	if result != "" {
		result, err = filepath.Abs(result)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(result), 0700); err != nil {
			return err
		}
	}
	test := exec.Command("go", "test", "-count=1", "-v", "./integration")
	test.Env = append(os.Environ(), "GRANTDIFF_TEST_KUBECONFIG="+kubeconfig, "GRANTDIFF_TEST_CONTEXT=kind-"+name, "GRANTDIFF_TEST_RESULT="+result)
	test.Stdout, test.Stderr = os.Stdout, os.Stderr
	return test.Run()
}

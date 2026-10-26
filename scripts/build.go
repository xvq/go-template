//go:build ignore

// Run from the repository root: go run ./scripts/build.go
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

func output(name string, args ...string) (string, error) {
	b, err := exec.Command(name, args...).Output()
	return strings.TrimSpace(string(b)), err
}

func run() error {
	targetOS := flag.String("os", runtime.GOOS, "target GOOS")
	targetArch := flag.String("arch", runtime.GOARCH, "target GOARCH")
	destination := flag.String("o", "", "output file; defaults to bin/app[-OS-ARCH][.exe]")
	flag.Parse()
	if flag.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments")
	}
	if _, err := os.Stat("go.mod"); err != nil {
		return fmt.Errorf("run from the project root: %w", err)
	}
	module, err := output("go", "list", "-m")
	if err != nil {
		return fmt.Errorf("read module: %w", err)
	}
	version, commit := "dev", "unknown"
	root, gitErr := output("git", "rev-parse", "--show-toplevel")
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	rootInfo, rootErr := os.Stat(root)
	cwdInfo, cwdErr := os.Stat(cwd)
	if gitErr == nil && rootErr == nil && cwdErr == nil && os.SameFile(rootInfo, cwdInfo) {
		commit, err = output("git", "rev-parse", "--verify", "HEAD")
		if err != nil {
			return fmt.Errorf("repository needs an initial commit: %w", err)
		}
		version, err = output("git", "describe", "--tags", "--match", "v[0-9]*", "--abbrev=7", "HEAD")
		if err != nil {
			version = "dev-" + commit[:7]
		}
		status, err := output("git", "status", "--porcelain", "--untracked-files=normal")
		if err != nil {
			return fmt.Errorf("read working tree status: %w", err)
		}
		if status != "" {
			version += "-dirty"
		}
	}
	// Keep Git-derived text out of the linker flag parser unless it is safe.
	if !regexp.MustCompile(`^[A-Za-z0-9._+/-]+$`).MatchString(version) {
		return fmt.Errorf("version tag contains unsupported characters: %q", version)
	}
	if *destination == "" {
		*destination = "bin/app"
		if *targetOS != runtime.GOOS || *targetArch != runtime.GOARCH {
			*destination += "-" + *targetOS + "-" + *targetArch
		}
		if *targetOS == "windows" {
			*destination += ".exe"
		}
	}
	if err := os.MkdirAll(filepath.Dir(*destination), 0755); err != nil {
		return err
	}
	pkg := module + "/internal/core/infrastructure/cli"
	stamp := time.Now().UTC().Format(time.RFC3339)
	flags := fmt.Sprintf("-s -w -X %s.version=%s -X %s.commit=%s -X %s.buildTime=%s", pkg, version, pkg, commit, pkg, stamp)
	cmd := exec.Command("go", "build", "-trimpath", "-buildvcs=false", "-ldflags", flags, "-o", *destination, "./cmd")
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "GOOS="+*targetOS, "GOARCH="+*targetArch, "CGO_ENABLED=0")
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}
	fmt.Printf("Built %s (%s, %s)\n", *destination, version, commit)
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

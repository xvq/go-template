package cli

import (
	"bytes"
	"runtime"
	"strings"
	"testing"
)

func TestVersionWithoutConfig(t *testing.T) {
	root := NewRoot()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"-c", "not-existing.yaml", "version"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Version:    " + version, "Commit:     " + commit, "Build time: " + buildTime, runtime.Version()} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("missing %q: %s", want, out.String())
		}
	}
}

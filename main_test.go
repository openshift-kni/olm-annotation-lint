package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

var testBinary string

func TestMain(m *testing.M) {
	bin := filepath.Join(os.TempDir(), "olm-annotation-lint-test")
	cmd := exec.Command("go", "build", "-o", bin, ".") //nolint:gosec // builds from local source for testing
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to build test binary: %v\n%s", err, out)
		os.Exit(1)
	}
	testBinary = bin
	code := m.Run()
	_ = os.Remove(bin)
	os.Exit(code)
}

func TestSplitAndTrim(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{"comma separated", "a, b, c", []string{"a", "b", "c"}},
		{"single value", "single", []string{"single"}},
		{"empty string", "", []string{""}},
		{"extra spaces", " spaces , everywhere ", []string{"spaces", "everywhere"}},
		{"no spaces", "a,b,c", []string{"a", "b", "c"}},
		{"trailing comma", "a,b,", []string{"a", "b", ""}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitAndTrim(tt.input)
			if len(result) != len(tt.expected) {
				t.Fatalf("expected %d parts, got %d: %v", len(tt.expected), len(result), result)
			}
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("index %d: expected %q, got %q", i, tt.expected[i], result[i])
				}
			}
		})
	}
}

func TestStringOrListUnmarshalYAMLScalar(t *testing.T) {
	input := `"a, b, c"`
	var s stringOrList
	if err := yaml.Unmarshal([]byte(input), &s); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(s) != 3 || s[0] != "a" || s[1] != "b" || s[2] != "c" {
		t.Errorf("expected [a b c], got %v", s)
	}
}

func TestStringOrListUnmarshalYAMLSequence(t *testing.T) {
	input := "- x\n- y\n- z"
	var s stringOrList
	if err := yaml.Unmarshal([]byte(input), &s); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(s) != 3 || s[0] != "x" || s[1] != "y" || s[2] != "z" {
		t.Errorf("expected [x y z], got %v", s)
	}
}

func TestStringOrListUnmarshalYAMLInvalidSequence(t *testing.T) {
	input := "- 1\n- true\n- key: value"
	var s stringOrList
	err := yaml.Unmarshal([]byte(input), &s)
	if err == nil {
		t.Error("expected error for invalid sequence contents")
	}
}

func TestLoadConfig(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		dir := t.TempDir()
		cfgPath := filepath.Join(dir, "config.yaml")
		content := "path: testdata/valid\nexclude: vendor,testdata\nallow: olm.custom\nstrict: true\n"
		if err := os.WriteFile(cfgPath, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		cfg, err := loadConfig(cfgPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(cfg.Path) != 1 || cfg.Path[0] != "testdata/valid" {
			t.Errorf("expected path [testdata/valid], got %v", cfg.Path)
		}
		if len(cfg.Exclude) != 2 {
			t.Errorf("expected 2 exclude paths, got %v", cfg.Exclude)
		}
		if len(cfg.Allow) != 1 || cfg.Allow[0] != "olm.custom" {
			t.Errorf("expected allow [olm.custom], got %v", cfg.Allow)
		}
		if cfg.Strict == nil || !*cfg.Strict {
			t.Error("expected strict to be true")
		}
	})

	t.Run("config with list syntax", func(t *testing.T) {
		dir := t.TempDir()
		cfgPath := filepath.Join(dir, "config.yaml")
		content := "path:\n  - dir1\n  - dir2\nexclude:\n  - vendor\n"
		if err := os.WriteFile(cfgPath, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		cfg, err := loadConfig(cfgPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(cfg.Path) != 2 || cfg.Path[0] != "dir1" || cfg.Path[1] != "dir2" {
			t.Errorf("expected paths [dir1 dir2], got %v", cfg.Path)
		}
	})

	t.Run("invalid YAML", func(t *testing.T) {
		dir := t.TempDir()
		cfgPath := filepath.Join(dir, "config.yaml")
		if err := os.WriteFile(cfgPath, []byte(":::invalid"), 0o600); err != nil {
			t.Fatal(err)
		}
		_, err := loadConfig(cfgPath)
		if err == nil {
			t.Fatal("expected error for invalid YAML")
		}
		if !strings.Contains(err.Error(), "parsing config") {
			t.Errorf("expected 'parsing config' in error, got: %v", err)
		}
	})

	t.Run("nonexistent file", func(t *testing.T) {
		_, err := loadConfig("/nonexistent/config.yaml")
		if err == nil {
			t.Fatal("expected error for nonexistent file")
		}
		if !errors.Is(err, os.ErrNotExist) {
			t.Errorf("expected os.ErrNotExist, got: %v", err)
		}
	})
}

func TestDiscoverConfig(t *testing.T) {
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	t.Run("no config file", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.Chdir(dir); err != nil {
			t.Fatal(err)
		}
		cfg, err := discoverConfig()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg != nil {
			t.Error("expected nil config when no file exists")
		}
	})

	t.Run("valid config file", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.Chdir(dir); err != nil {
			t.Fatal(err)
		}
		content := "path: .\nstrict: false\n"
		if err := os.WriteFile(".olm-lint.yaml", []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		cfg, err := discoverConfig()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg == nil {
			t.Fatal("expected non-nil config")
		}
		if len(cfg.Path) != 1 || cfg.Path[0] != "." {
			t.Errorf("expected path [.], got %v", cfg.Path)
		}
	})

	t.Run("invalid config file", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.Chdir(dir); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(".olm-lint.yaml", []byte(":::bad"), 0o600); err != nil {
			t.Fatal(err)
		}
		_, err := discoverConfig()
		if err == nil {
			t.Fatal("expected error for invalid config file")
		}
	})
}

func TestMainVersion(t *testing.T) {
	out, err := exec.Command(testBinary, "--version").Output() //nolint:gosec // test runs locally built binary
	if err != nil {
		t.Fatalf("--version failed: %v", err)
	}
	if strings.TrimSpace(string(out)) == "" {
		t.Error("expected version output, got empty string")
	}
}

func TestMainListRules(t *testing.T) {
	bin := testBinary
	out, err := exec.Command(bin, "--list-rules").Output() //nolint:gosec // test runs locally built binary
	if err != nil {
		t.Fatalf("--list-rules failed: %v", err)
	}
	if !strings.Contains(string(out), "User-settable annotations") {
		t.Error("expected 'User-settable annotations' in --list-rules output")
	}
}

func TestMainExitCodes(t *testing.T) {
	bin := testBinary

	tests := []struct {
		name     string
		args     []string
		exitCode int
	}{
		{"valid files exit 0", []string{"--path", "testdata/valid"}, 0},
		{"invalid files strict exit 1", []string{"--path", "testdata/invalid/unknown_olm_annotation.yaml", "--strict"}, 1},
		{"nonexistent path exit 2", []string{"--path", "/nonexistent/path"}, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command(bin, tt.args...) //nolint:gosec // test runs locally built binary
			err := cmd.Run()
			if tt.exitCode == 0 {
				if err != nil {
					t.Errorf("expected exit 0, got error: %v", err)
				}
				return
			}
			var exitErr *exec.ExitError
			if !errors.As(err, &exitErr) {
				t.Fatalf("expected ExitError, got: %v", err)
			}
			if exitErr.ExitCode() != tt.exitCode {
				t.Errorf("expected exit code %d, got %d", tt.exitCode, exitErr.ExitCode())
			}
		})
	}
}

func TestMainOutputFormats(t *testing.T) {
	bin := testBinary

	tests := []struct {
		name     string
		format   string
		contains string
	}{
		{"text format", "text", "[ERROR]"},
		{"json format", "json", `"severity"`},
		{"github format", "github", "::error file="},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, _ := exec.Command(bin, "--path", "testdata/invalid/unknown_olm_annotation.yaml", "--format", tt.format).CombinedOutput() //nolint:gosec // test runs locally built binary
			if !strings.Contains(string(out), tt.contains) {
				t.Errorf("expected %q in %s output, got: %s", tt.contains, tt.format, out)
			}
		})
	}
}

func TestMainConfigFile(t *testing.T) {
	bin := testBinary

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "test-config.yaml")
	content := "path: testdata/valid\n"
	if err := os.WriteFile(cfgPath, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(bin, "--config", cfgPath) //nolint:gosec // test runs locally built binary
	if err := cmd.Run(); err != nil {
		t.Errorf("expected exit 0 with valid config pointing to valid testdata, got: %v", err)
	}
}

func TestMainConfigFlagOverride(t *testing.T) {
	bin := testBinary

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "test-config.yaml")
	content := "path: /nonexistent/should/be/overridden\n"
	if err := os.WriteFile(cfgPath, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(bin, "--config", cfgPath, "--path", "testdata/valid") //nolint:gosec // test runs locally built binary
	if err := cmd.Run(); err != nil {
		t.Errorf("CLI --path should override config path, got: %v", err)
	}
}

func TestMainInvalidConfig(t *testing.T) {
	bin := testBinary

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "bad-config.yaml")
	if err := os.WriteFile(cfgPath, []byte(":::invalid"), 0o600); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(bin, "--config", cfgPath) //nolint:gosec // test runs locally built binary
	err := cmd.Run()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected ExitError, got: %v", err)
	}
	if exitErr.ExitCode() != 2 {
		t.Errorf("expected exit code 2 for invalid config, got %d", exitErr.ExitCode())
	}
}

func TestGitHubOutputs(t *testing.T) {
	bin := testBinary

	tests := []struct {
		name        string
		args        []string
		wantErrors  string
		wantHasErrs string
	}{
		{
			name:        "valid files produce zero counts",
			args:        []string{"--path", "testdata/valid"},
			wantErrors:  "error-count=0",
			wantHasErrs: "has-errors=false",
		},
		{
			name:        "invalid files produce error counts",
			args:        []string{"--path", "testdata/invalid/unknown_olm_annotation.yaml"},
			wantErrors:  "error-count=1",
			wantHasErrs: "has-errors=true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile, err := os.CreateTemp(t.TempDir(), "github-output-*")
			if err != nil {
				t.Fatal(err)
			}
			_ = tmpFile.Close()

			cmd := exec.Command(bin, tt.args...) //nolint:gosec // test runs locally built binary
			cmd.Env = append(os.Environ(), "GITHUB_OUTPUT="+tmpFile.Name())
			_ = cmd.Run()

			data, err := os.ReadFile(tmpFile.Name())
			if err != nil {
				t.Fatalf("failed to read GITHUB_OUTPUT file: %v", err)
			}
			output := string(data)

			if !strings.Contains(output, tt.wantErrors) {
				t.Errorf("expected %q in output, got:\n%s", tt.wantErrors, output)
			}
			if !strings.Contains(output, tt.wantHasErrs) {
				t.Errorf("expected %q in output, got:\n%s", tt.wantHasErrs, output)
			}
			if !strings.Contains(output, "warning-count=") {
				t.Errorf("expected warning-count in output, got:\n%s", output)
			}
			if !strings.Contains(output, "total-count=") {
				t.Errorf("expected total-count in output, got:\n%s", output)
			}
		})
	}
}

func TestGitHubOutputsNotWrittenWithoutEnv(t *testing.T) {
	bin := testBinary

	cmd := exec.Command(bin, "--path", "testdata/valid") //nolint:gosec // test runs locally built binary
	cmd.Env = []string{"PATH=" + os.Getenv("PATH"), "HOME=" + os.Getenv("HOME")}
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("unexpected error: %v\n%s", err, out)
	}
	if strings.Contains(string(out), "error-count=") {
		t.Error("expected no GitHub output lines in stdout/stderr when GITHUB_OUTPUT is not set")
	}
}

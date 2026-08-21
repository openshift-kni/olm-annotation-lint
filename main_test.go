package main

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

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

	t.Run("config with format", func(t *testing.T) {
		dir := t.TempDir()
		cfgPath := filepath.Join(dir, "config.yaml")
		content := "path: testdata/valid\nformat: junit\n"
		if err := os.WriteFile(cfgPath, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		cfg, err := loadConfig(cfgPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.Format != "junit" {
			t.Errorf("expected format junit, got %q", cfg.Format)
		}
	})

	t.Run("config with rules", func(t *testing.T) {
		dir := t.TempDir()
		cfgPath := filepath.Join(dir, "config.yaml")
		content := "rules:\n  olm.operatorGroup:\n    enabled: false\n  unknown-annotation:\n    severity: warning\n"
		if err := os.WriteFile(cfgPath, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		cfg, err := loadConfig(cfgPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		rc, ok := cfg.Rules["olm.operatorGroup"]
		if !ok || rc.Enabled == nil || *rc.Enabled {
			t.Errorf("expected olm.operatorGroup enabled=false, got %+v", rc)
		}
		if cfg.Rules["unknown-annotation"].Severity != "warning" {
			t.Errorf("expected unknown-annotation severity warning, got %q", cfg.Rules["unknown-annotation"].Severity)
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

func TestRunVersion(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"--version"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if strings.TrimSpace(stdout.String()) == "" {
		t.Error("expected version output, got empty string")
	}
}

func TestDisplayVersion(t *testing.T) {
	info := &debug.BuildInfo{Main: debug.Module{Version: "v1.2.3"}}
	tests := []struct {
		name   string
		ldflag string
		info   *debug.BuildInfo
		ok     bool
		want   string
	}{
		{"ldflags win", "v9.9.9", info, true, "v9.9.9"},
		{"go install fallback", "dev", info, true, "v1.2.3"},
		{"empty ldflag fallback", "", info, true, "v1.2.3"},
		{"devel keeps ldflag", "dev", &debug.BuildInfo{Main: debug.Module{Version: "(devel)"}}, true, "dev"},
		{"missing build info", "dev", nil, false, "dev"},
		{"empty module version", "dev", &debug.BuildInfo{}, true, "dev"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := displayVersion(tt.ldflag, tt.info, tt.ok)
			if got != tt.want {
				t.Errorf("displayVersion(%q) = %q, want %q", tt.ldflag, got, tt.want)
			}
		})
	}
}

func TestRunListRules(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"--list-rules"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if !strings.Contains(stdout.String(), "User-settable annotations") {
		t.Error("expected 'User-settable annotations' in --list-rules output")
	}
}

func TestRunOutputFlag(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "results.json")
	var stdout, stderr bytes.Buffer
	code := run([]string{"--path", "testdata/invalid/unknown_olm_annotation.yaml", "--format", "json", "--output", outPath}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("expected exit 1, got %d: %s", code, stderr.String())
	}
	if strings.Contains(stdout.String(), `"violations"`) {
		t.Errorf("JSON should be written to the file, not stdout: %s", stdout.String())
	}
	data, err := os.ReadFile(outPath) //nolint:gosec // test-controlled temp file
	if err != nil {
		t.Fatalf("expected output file: %v", err)
	}
	if !strings.Contains(string(data), `"violations"`) {
		t.Errorf("expected JSON report in output file, got: %s", data)
	}
	if !strings.Contains(stderr.String(), "error(s)") {
		t.Errorf("expected summary on stderr, got: %s", stderr.String())
	}
}

func TestRunOutputFlagUnwritable(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"--path", "testdata/valid", "--output", "/no/such/dir/results.json"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("expected exit 2 for unwritable output, got %d", code)
	}
}

func TestRunInvalidRuleSeverity(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	content := "path: testdata/valid\nrules:\n  olm.skipRange:\n    severity: banana\n"
	if err := os.WriteFile(cfgPath, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := run([]string{"--config", cfgPath}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("expected exit 2 for invalid severity, got %d: %s", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "severity") {
		t.Errorf("expected severity error, got: %s", stderr.String())
	}
}

func TestRunTimeoutFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"--timeout", "1h", "--path", "testdata/valid"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit 0 with long timeout, got %d: %s", code, stderr.String())
	}
}

func TestRunTimeoutFlagInvalid(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"--timeout", "not-a-duration"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("expected exit 2 for invalid timeout, got %d", code)
	}
}

func TestRunExitCodes(t *testing.T) {
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
			var stdout, stderr bytes.Buffer
			code := run(tt.args, &stdout, &stderr)
			if code != tt.exitCode {
				t.Errorf("expected exit code %d, got %d\nstdout: %s\nstderr: %s", tt.exitCode, code, stdout.String(), stderr.String())
			}
		})
	}
}

func TestRunOutputFormats(t *testing.T) {
	tests := []struct {
		name     string
		format   string
		contains string
	}{
		{"text format", "text", "[ERROR]"},
		{"json format", "json", `"severity"`},
		{"github format", "github", "::error file="},
		{"junit format", "junit", "<testsuites"},
		{"sarif format", "sarif", `"$schema"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			run([]string{"--path", "testdata/invalid/unknown_olm_annotation.yaml", "--format", tt.format}, &stdout, &stderr)
			combined := stdout.String() + stderr.String()
			if !strings.Contains(combined, tt.contains) {
				t.Errorf("expected %q in %s output, got: %s", tt.contains, tt.format, combined)
			}
		})
	}
}

func TestRunUnknownFormat(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"--path", "testdata/valid", "--format", "xml"}, &stdout, &stderr)
	if code != 2 {
		t.Errorf("expected exit code 2 for unknown format, got %d", code)
	}
	combined := stdout.String() + stderr.String()
	if !strings.Contains(combined, `unknown format "xml"`) {
		t.Errorf("expected unknown format error, got: %s", combined)
	}
	if !strings.Contains(combined, "github, json, junit, sarif, text") {
		t.Errorf("expected supported formats listed, got: %s", combined)
	}
}

func TestRunConfigFile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "test-config.yaml")
	content := "path: testdata/valid\n"
	if err := os.WriteFile(cfgPath, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := run([]string{"--config", cfgPath}, &stdout, &stderr)
	if code != 0 {
		t.Errorf("expected exit 0 with valid config pointing to valid testdata, got: %d\nstderr: %s", code, stderr.String())
	}
}

func TestRunConfigFlagOverride(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "test-config.yaml")
	content := "path: /nonexistent/should/be/overridden\n"
	if err := os.WriteFile(cfgPath, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := run([]string{"--config", cfgPath, "--path", "testdata/valid"}, &stdout, &stderr)
	if code != 0 {
		t.Errorf("CLI --path should override config path, got exit %d\nstderr: %s", code, stderr.String())
	}
}

func TestRunConfigFormat(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "test-config.yaml")
	content := "path: testdata/invalid/unknown_olm_annotation.yaml\nformat: json\n"
	if err := os.WriteFile(cfgPath, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	run([]string{"--config", cfgPath}, &stdout, &stderr)
	if !strings.Contains(stdout.String(), `"severity"`) {
		t.Errorf("expected JSON output from config format, got: %s", stdout.String())
	}
}

func TestRunConfigFormatCLIOverride(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "test-config.yaml")
	content := "path: testdata/invalid/unknown_olm_annotation.yaml\nformat: json\n"
	if err := os.WriteFile(cfgPath, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	run([]string{"--config", cfgPath, "--format", "text"}, &stdout, &stderr)
	combined := stdout.String() + stderr.String()
	if !strings.Contains(combined, "[ERROR]") {
		t.Errorf("expected text output when CLI overrides config, got: %s", combined)
	}
	if strings.Contains(stdout.String(), `"severity"`) {
		t.Errorf("expected CLI --format to override config format")
	}
}

func TestRunConfigFormatShortFlagOverride(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "test-config.yaml")
	content := "path: testdata/invalid/unknown_olm_annotation.yaml\nformat: json\n"
	if err := os.WriteFile(cfgPath, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	run([]string{"--config", cfgPath, "-f", "text"}, &stdout, &stderr)
	combined := stdout.String() + stderr.String()
	if !strings.Contains(combined, "[ERROR]") {
		t.Errorf("expected text output when -f overrides config, got: %s", combined)
	}
	if strings.Contains(stdout.String(), `"severity"`) {
		t.Errorf("expected -f flag to override config format")
	}
}

func TestRunConfigFormatInvalid(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "test-config.yaml")
	content := "path: testdata/valid\nformat: xml\n"
	if err := os.WriteFile(cfgPath, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := run([]string{"--config", cfgPath}, &stdout, &stderr)
	if code != 2 {
		t.Errorf("expected exit code 2, got %d", code)
	}
	combined := stdout.String() + stderr.String()
	if !strings.Contains(combined, `unknown format "xml"`) {
		t.Errorf("expected unknown format error, got: %s", combined)
	}
}

func TestRunInvalidConfig(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "bad-config.yaml")
	if err := os.WriteFile(cfgPath, []byte(":::invalid"), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := run([]string{"--config", cfgPath}, &stdout, &stderr)
	if code != 2 {
		t.Errorf("expected exit code 2 for invalid config, got %d", code)
	}
}

func TestRunInvalidFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"--nonexistent-flag"}, &stdout, &stderr)
	if code != 2 {
		t.Errorf("expected exit code 2 for invalid flag, got %d", code)
	}
}

func TestRunWarningOnlyExitCode(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"--path", "testdata/invalid/controller_managed_annotation.yaml"}, &stdout, &stderr)
	if code != 0 {
		t.Errorf("expected exit 0 for warning-only violations, got %d", code)
	}
	if !strings.Contains(stderr.String(), "warning(s)") {
		t.Errorf("expected warning count in stderr, got: %s", stderr.String())
	}
}

func TestRunExcludeFlag(t *testing.T) {
	dir := t.TempDir()
	subdir := filepath.Join(dir, "excluded")
	if err := os.MkdirAll(subdir, 0o750); err != nil { //nolint:gosec // test directory
		t.Fatal(err)
	}
	data, err := os.ReadFile("testdata/invalid/controller_managed_annotation.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subdir, "test.yaml"), data, 0o600); err != nil { //nolint:gosec // test writes to temp dir
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	run([]string{"--path", dir, "--exclude", "excluded"}, &stdout, &stderr)
	if strings.Contains(stdout.String(), "controller-managed") {
		t.Error("excluded directory should not produce violations")
	}
}

func TestRunAllowFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	run([]string{"--path", "testdata/invalid/unknown_olm_annotation.yaml", "--allow", "olm.operatorframework.io/bundle-install-timeout"}, &stdout, &stderr)
	if !strings.Contains(stdout.String(), "allowed via user override") {
		t.Errorf("expected allowed-override message, got: %s", stdout.String())
	}
}

func TestRunConfigMergeExclude(t *testing.T) {
	dir := t.TempDir()
	subdir := filepath.Join(dir, "skipme")
	if err := os.MkdirAll(subdir, 0o750); err != nil { //nolint:gosec // test directory
		t.Fatal(err)
	}
	data, err := os.ReadFile("testdata/invalid/controller_managed_annotation.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subdir, "test.yaml"), data, 0o600); err != nil { //nolint:gosec // test writes to temp dir
		t.Fatal(err)
	}

	cfgPath := filepath.Join(dir, "test-config.yaml")
	content := "path: " + dir + "\nexclude:\n  - skipme\n"
	if err := os.WriteFile(cfgPath, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	run([]string{"--config", cfgPath}, &stdout, &stderr)
	if strings.Contains(stdout.String(), "controller-managed") {
		t.Error("config exclude should prevent controller-managed violations")
	}
}

func TestRunConfigMergeAllow(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "test-config.yaml")
	content := "path: testdata/invalid/unknown_olm_annotation.yaml\nallow:\n  - olm.operatorframework.io/bundle-install-timeout\n"
	if err := os.WriteFile(cfgPath, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	run([]string{"--config", cfgPath}, &stdout, &stderr)
	if !strings.Contains(stdout.String(), "allowed via user override") {
		t.Errorf("expected allowed-override from config allow, got: %s", stdout.String())
	}
}

func TestRunConfigMergeStrict(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "test-config.yaml")
	content := "path: testdata/invalid/controller_managed_annotation.yaml\nstrict: true\n"
	if err := os.WriteFile(cfgPath, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := run([]string{"--config", cfgPath}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("expected exit 1 when config strict escalates warnings, got %d", code)
	}
}

func TestGitHubOutputs(t *testing.T) {
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

			t.Setenv("GITHUB_OUTPUT", tmpFile.Name())

			var stdout, stderr bytes.Buffer
			run(tt.args, &stdout, &stderr)

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
	t.Setenv("GITHUB_OUTPUT", "")

	var stdout, stderr bytes.Buffer
	run([]string{"--path", "testdata/valid"}, &stdout, &stderr)
	combined := stdout.String() + stderr.String()
	if strings.Contains(combined, "error-count=") {
		t.Error("expected no GitHub output lines in stdout/stderr when GITHUB_OUTPUT is not set")
	}
}

func TestMainBinaryExitCode(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping binary test in short mode")
	}

	bin := filepath.Join(t.TempDir(), "olm-annotation-lint-test")
	cmd := exec.Command("go", "build", "-o", bin, ".") //nolint:gosec // builds from local source for testing
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to build test binary: %v\n%s", err, out)
	}

	tests := []struct {
		name     string
		args     []string
		exitCode int
	}{
		{"exit 0", []string{"--path", "testdata/valid"}, 0},
		{"exit 1", []string{"--path", "testdata/invalid/unknown_olm_annotation.yaml", "--strict"}, 1},
		{"exit 2", []string{"--path", "/nonexistent/path"}, 2},
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

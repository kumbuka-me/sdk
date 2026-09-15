package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	sdk "github.com/kumbuka-me/sdk"
	"github.com/kumbuka-me/sdk/internal/build"
)

const sdkModule = "github.com/kumbuka-me/sdk"

var (
	projectName = regexp.MustCompile(`^[a-z][a-z0-9-]{0,62}$`)
	versionName = regexp.MustCompile(`^v[0-9][a-zA-Z0-9.+-]*$`)
)

type initConfig struct {
	Name          string
	SDKPath       string
	SDKVersion    string
	SDKPathSet    bool
	SDKVersionSet bool
}

func initialize(ctx context.Context, cfg initConfig, stdout, stderr io.Writer) error {
	if !projectName.MatchString(cfg.Name) {
		return fmt.Errorf("plugin name must match %s", projectName.String())
	}

	version, replacement, err := resolveSDK(ctx, cfg)
	if err != nil {
		return err
	}
	if err := os.Mkdir(cfg.Name, 0o755); err != nil {
		return fmt.Errorf("create project: %w", err)
	}

	files := map[string]string{
		"go.mod":       pluginGoMod(cfg.Name, version, replacement),
		"plugin.yaml":  pluginManifest(cfg.Name),
		"main.go":      sampleSource,
		"main_test.go": sampleTest,
		".gitignore":   "/dist/\n*.wasm\n",
		"README.md":    pluginREADME(cfg.Name),
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(cfg.Name, name), []byte(content), 0o644); err != nil {
			return err
		}
	}

	command := exec.CommandContext(ctx, "go", "mod", "tidy")
	command.Dir = cfg.Name
	command.Stdout = stdout
	command.Stderr = stderr
	if err := command.Run(); err != nil {
		return fmt.Errorf("resolve SDK in %s: %w (project retained)", cfg.Name, err)
	}

	_, err = fmt.Fprintf(stdout, "Created %s. Run: cd %s && kumbuka-plugin test && kumbuka-plugin build\n", cfg.Name, cfg.Name)
	return err
}

func resolveSDK(ctx context.Context, cfg initConfig) (string, string, error) {
	path := cfg.SDKPath
	version := cfg.SDKVersion

	if !cfg.SDKPathSet && !cfg.SDKVersionSet {
		path = developmentCheckout()
	}
	if path != "" {
		absolute, err := filepath.Abs(path)
		if err != nil {
			return "", "", err
		}
		if err := validateSDKCheckout(absolute); err != nil {
			return "", "", err
		}
		return "v0.0.0", fmt.Sprintf("replace %s => %s", sdkModule, filepath.ToSlash(absolute)), nil
	}

	if version == "latest" {
		command := exec.CommandContext(ctx, "go", "list", "-m", "-f", "{{.Version}}", sdkModule+"@latest")
		value, err := command.Output()
		if err != nil {
			return "", "", fmt.Errorf("resolve SDK version: %w", err)
		}
		version = strings.TrimSpace(string(value))
	}
	if !versionName.MatchString(version) {
		return "", "", fmt.Errorf("invalid SDK version %q", version)
	}
	return version, "", nil
}

func validateSDKCheckout(directory string) error {
	module, err := os.ReadFile(filepath.Join(directory, "go.mod"))
	if err != nil {
		return fmt.Errorf("SDK checkout: %w", err)
	}
	if !strings.HasPrefix(string(module), "module "+sdkModule+"\n") {
		return fmt.Errorf("SDK checkout has unexpected module path")
	}
	if _, err := os.Stat(filepath.Join(directory, "plugin.go")); err != nil {
		return fmt.Errorf("SDK checkout: %w", err)
	}
	return nil
}

func developmentCheckout() string {
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		return ""
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(source), "../.."))
	if validateSDKCheckout(root) != nil {
		return ""
	}
	return root
}

func testProject(ctx context.Context, stdout, stderr io.Writer) error {
	directory, err := os.Getwd()
	if err != nil {
		return err
	}
	manifest, err := build.Validate(directory)
	if err != nil {
		return err
	}
	if !manifest.RequiresWASM() {
		return nil
	}

	command := exec.CommandContext(ctx, "go", "test", "./...")
	command.Dir = directory
	command.Stdout = stdout
	command.Stderr = stderr
	if err := command.Run(); err != nil {
		return fmt.Errorf("go tests: %w", err)
	}
	return nil
}

func buildProject(ctx context.Context, stdout io.Writer) error {
	directory, err := os.Getwd()
	if err != nil {
		return err
	}
	destination := filepath.Join(directory, "dist", filepath.Base(directory)+".kumbukaplugin")
	if err := build.Build(ctx, directory, destination); err != nil {
		return err
	}
	_, err = fmt.Fprintln(stdout, destination)
	return err
}

func pluginGoMod(name, version, replacement string) string {
	content := fmt.Sprintf("module example.com/%s\n\ngo 1.27.0\n\nrequire %s %s\n", name, sdkModule, version)
	if replacement != "" {
		content += "\n" + replacement + "\n"
	}
	return content
}

func pluginManifest(name string) string {
	return fmt.Sprintf("api_version: %d\nprovider: Example\nid: com.example.%s\nname: %s\nversion: 1.0.0\ndefault_enabled: true\nmodules:\n  - type: macro\n    id: greeting\n    name: greeting\npermissions: []\n", sdk.Version, name, name)
}

func pluginREADME(name string) string {
	return fmt.Sprintf("# %s\n\nA Go/WASI Kumbuka plugin.\n\nUse `{{greeting}}` in a Kumbuka page. Run `kumbuka-plugin test`, then `kumbuka-plugin build`. Install `dist/%s.kumbukaplugin` from Administration → Plugins.\n", name, name)
}

const sampleSource = `package main

import (
	"strings"

	plugin "github.com/kumbuka-me/sdk"
)

func main() {}

func init() {
	plugin.RegisterMacro("greeting", parseGreeting, renderGreeting)
}

func parseGreeting(source string) (struct{}, bool) {
	return struct{}{}, strings.TrimSpace(source) == "{{greeting}}"
}

func renderGreeting(_ struct{}) (plugin.Result, error) {
	return plugin.Text("<p>Hello from a Go plugin!</p>"), nil
}
`

const sampleTest = `package main

import "testing"

func TestGreeting(t *testing.T) {
	if _, matched := parseGreeting("{{greeting}}"); !matched {
		t.Fatal("greeting should match")
	}
	if _, matched := parseGreeting("ordinary text"); matched {
		t.Fatal("ordinary text should not match")
	}
	result, err := renderGreeting(struct{}{})
	if err != nil || len(result.Parts) != 1 {
		t.Fatalf("render: %+v, %v", result, err)
	}
}
`

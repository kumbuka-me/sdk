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

// initConfig contains the resolved inputs for creating one plugin project.
type initConfig struct {
	// Name is the new project directory and plugin name.
	Name string
	// SDKPath is an optional local SDK checkout.
	SDKPath string
	// SDKVersion is the requested SDK module version when no local checkout is used.
	SDKVersion string
	// SDKPathSet reports whether SDKPath was explicitly supplied.
	SDKPathSet bool
	// SDKVersionSet reports whether SDKVersion was explicitly supplied.
	SDKVersionSet bool
}

// initialize creates a plugin project, resolves its SDK dependency, and runs go mod tidy.
func initialize(ctx context.Context, config initConfig, stdout, stderr io.Writer) error {
	if !projectName.MatchString(config.Name) {
		return fmt.Errorf("plugin name must match %s", projectName.String())
	}
	version, replacement, err := resolveSDK(ctx, config)
	if err != nil {
		return err
	}

	if err := os.Mkdir(config.Name, 0o755); err != nil {
		return fmt.Errorf("create project: %w", err)
	}
	if err := writeProjectFiles(config.Name, generatedProjectFiles(config.Name, version, replacement)); err != nil {
		return err
	}
	if err := runGo(ctx, config.Name, stdout, stderr, "mod", "tidy"); err != nil {
		return fmt.Errorf("resolve SDK in %s: %w (project retained)", config.Name, err)
	}

	_, err = fmt.Fprintf(
		stdout,
		"Created %s. Run: cd %s && kumbuka-plugin test && kumbuka-plugin build\n",
		config.Name,
		config.Name,
	)
	return err
}

// generatedProjectFiles returns the initial files written by the init command.
func generatedProjectFiles(name, version, replacement string) map[string]string {
	return map[string]string{
		"go.mod":       pluginGoMod(name, version, replacement),
		"plugin.yaml":  pluginManifest(name),
		"main.go":      sampleSource,
		"main_test.go": sampleTest,
		".gitignore":   "/dist/\n*.wasm\n",
		"README.md":    pluginREADME(name),
	}
}

// writeProjectFiles writes generated project files beneath one new project directory.
func writeProjectFiles(directory string, files map[string]string) error {
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(directory, name), []byte(content), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", name, err)
		}
	}
	return nil
}

// resolveSDK selects an explicit checkout, the development checkout, or a module version.
func resolveSDK(ctx context.Context, config initConfig) (string, string, error) {
	path := config.SDKPath
	if !config.SDKPathSet && !config.SDKVersionSet {
		path = developmentCheckout()
	}
	if path != "" {
		return resolveSDKPath(path)
	}
	return resolveSDKVersion(ctx, config.SDKVersion)
}

// resolveSDKPath validates a local SDK checkout and returns a go.mod replacement.
func resolveSDKPath(path string) (string, string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", "", err
	}
	if err := validateSDKCheckout(absolute); err != nil {
		return "", "", err
	}
	return "v0.0.0", fmt.Sprintf("replace %s => %s", sdkModule, filepath.ToSlash(absolute)), nil
}

// resolveSDKVersion validates or resolves the requested SDK module version.
func resolveSDKVersion(ctx context.Context, version string) (string, string, error) {
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

// validateSDKCheckout verifies the expected module path and public plugin entry point.
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

// developmentCheckout returns the SDK repository containing this CLI when available.
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

// testProject validates the current plugin and runs Go tests for executable plugins.
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
	if err := runGo(ctx, directory, stdout, stderr, "test", "./..."); err != nil {
		return fmt.Errorf("go tests: %w", err)
	}
	return nil
}

// buildProject builds the current plugin and prints the resulting package path.
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

// runGo runs one Go command in a project directory with caller-provided streams.
func runGo(ctx context.Context, directory string, stdout, stderr io.Writer, args ...string) error {
	command := exec.CommandContext(ctx, "go", args...)
	command.Dir = directory
	command.Stdout = stdout
	command.Stderr = stderr
	return command.Run()
}

// pluginGoMod renders the generated plugin module file.
func pluginGoMod(name, version, replacement string) string {
	content := fmt.Sprintf("module example.com/%s\n\ngo 1.27.0\n\nrequire %s %s\n", name, sdkModule, version)
	if replacement != "" {
		content += "\n" + replacement + "\n"
	}
	return content
}

// pluginManifest renders the starter plugin manifest.
func pluginManifest(name string) string {
	return fmt.Sprintf("api_version: %d\nprovider: Example\nid: com.example.%s\nname: %s\nversion: 1.0.0\ndefault_enabled: true\nmodules:\n  - type: macro\n    id: greeting\n    name: greeting\npermissions: []\n", sdk.Version, name, name)
}

// pluginREADME renders the starter project README.
func pluginREADME(name string) string {
	return fmt.Sprintf("# %s\n\nA Go/WASI Kumbuka plugin.\n\nUse `{{greeting}}` in a Kumbuka page. Run `kumbuka-plugin test`, then `kumbuka-plugin build`. Install `dist/%s.kumbukaplugin` from Administration → Plugins.\n", name, name)
}

const sampleSource = `package main

import (
	"strings"

	plugin "github.com/kumbuka-me/sdk"
)

// main is intentionally empty because Kumbuka invokes registered WASM handlers.
func main() {}

// init registers the starter greeting macro.
func init() {
	plugin.RegisterMacro("greeting", parseGreeting, renderGreeting)
}

// parseGreeting recognizes the starter greeting macro.
func parseGreeting(source string) (struct{}, bool) {
	return struct{}{}, strings.TrimSpace(source) == "{{greeting}}"
}

// renderGreeting returns the starter greeting markup.
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

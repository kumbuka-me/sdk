// Package cli implements the kumbuka-plugin command.
package cli

import (
	"context"
	"fmt"
	"io"

	"github.com/containeroo/tinyflags"
)

// Run parses and executes one kumbuka-plugin subcommand.
func Run(ctx context.Context, args []string, version string, stdout, stderr io.Writer) error {
	root := tinyflags.NewCommand("kumbuka-plugin", tinyflags.ContinueOnError).RequireCommand()
	root.Version(version)

	initCommand := root.Command("init", "Create a new Kumbuka plugin project")
	resolveInitConfig := bindInitFlags(initCommand.FlagSet, version)
	initCommand.Run(func(ctx context.Context) error {
		return initialize(ctx, resolveInitConfig(), stdout, stderr)
	})

	testCommand := root.Command("test", "Validate and test the current plugin")
	testCommand.Run(func(ctx context.Context) error {
		return testProject(ctx, stdout, stderr)
	})

	buildCommand := root.Command("build", "Build the current plugin package")
	buildCommand.Run(func(ctx context.Context) error {
		return buildProject(ctx, stdout)
	})

	runner, err := root.ParseRunner(args)
	if err != nil {
		return handleParseError(err, stdout, stderr)
	}
	return runner.Run(ctx)
}

// bindInitFlags configures init arguments and returns the resolved command configuration.
func bindInitFlags(flags *tinyflags.FlagSet, version string) func() initConfig {
	flags.RequirePositional(1)
	sdkPath := flags.String("sdk-path", "", "Local Kumbuka SDK checkout").
		OneOfGroup("sdk-source").
		Placeholder("DIR")
	sdkVersion := flags.String("sdk-version", version, "Kumbuka SDK module version").
		OneOfGroup("sdk-source").
		Placeholder("VERSION")
	flags.Validate(func() error {
		if len(flags.Args()) != 1 {
			return fmt.Errorf("init requires exactly one plugin name")
		}
		return nil
	})

	return func() initConfig {
		name, _ := flags.Arg(0)
		return initConfig{
			Name:          name,
			SDKPath:       *sdkPath.Value(),
			SDKVersion:    *sdkVersion.Value(),
			SDKPathSet:    sdkPath.Changed(),
			SDKVersionSet: sdkVersion.Changed(),
		}
	}
}

// handleParseError renders expected CLI control-flow errors and returns real failures.
func handleParseError(err error, stdout, stderr io.Writer) error {
	switch {
	case tinyflags.IsHelpRequested(err), tinyflags.IsVersionRequested(err):
		_, _ = fmt.Fprint(stdout, err.Error())
		return nil
	case tinyflags.IsCommandRequired(err):
		help, _ := tinyflags.HelpText(err)
		_, _ = fmt.Fprint(stderr, help)
		return nil
	default:
		return err
	}
}

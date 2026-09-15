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
	initFlags := initCommand.FlagSet
	initFlags.RequirePositional(1)
	sdkPath := initFlags.String("sdk-path", "", "Local Kumbuka SDK checkout").
		OneOfGroup("sdk-source").
		Placeholder("DIR")
	sdkVersion := initFlags.String("sdk-version", version, "Kumbuka SDK module version").
		OneOfGroup("sdk-source").
		Placeholder("VERSION")
	initFlags.Validate(func() error {
		if len(initFlags.Args()) != 1 {
			return fmt.Errorf("init requires exactly one plugin name")
		}
		return nil
	})
	initCommand.Run(func(ctx context.Context) error {
		name, _ := initFlags.Arg(0)
		return initialize(ctx, initConfig{
			Name:          name,
			SDKPath:       *sdkPath.Value(),
			SDKVersion:    *sdkVersion.Value(),
			SDKPathSet:    sdkPath.Changed(),
			SDKVersionSet: sdkVersion.Changed(),
		}, stdout, stderr)
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

	return runner.Run(ctx)
}

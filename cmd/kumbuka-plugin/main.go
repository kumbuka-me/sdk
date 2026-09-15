// Command kumbuka-plugin develops Go plugins for Kumbuka's WASI runtime.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/kumbuka-me/sdk/internal/cli"
)

var (
	Version = "dev"
	Commit  = "none"
)

func main() {
	if err := cli.Run(
		context.Background(),
		os.Args[1:],
		Version,
		os.Stdout,
		os.Stderr,
	); err != nil {
		fmt.Fprintln(os.Stderr, "kumbuka-plugin:", err)
		os.Exit(1)
	}
}

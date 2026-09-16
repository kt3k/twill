// Command twill is the Twill utility-first CSS compiler (SPEC §14).
//
//	twill -i input.css -o output.css --watch
package main

import (
	"context"
	"os"

	"github.com/kt3k/twill/impl/go/cli"
)

func main() {
	os.Exit(cli.Main(context.Background(), os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

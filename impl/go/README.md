# twill (Go)

A Go implementation of the [Twill](../../SPEC.md) utility-first CSS
compiler. It depends only on the standard library.

## CLI

```sh
go run github.com/kt3k/twill/impl/go/cmd/twill@latest -i input.css -o output.css
go run github.com/kt3k/twill/impl/go/cmd/twill@latest -i input.css -o output.css --watch
go run github.com/kt3k/twill/impl/go/cmd/twill@latest -i input.css -o output.css --poll 250
go run github.com/kt3k/twill/impl/go/cmd/twill@latest -i input.css --minify > output.css
```

Run with `--help` for the full option list. Watch mode detects change
batches by comparing modification times (the standard library has no
filesystem event API); `--poll` selects the polling algorithm of SPEC §14.4.

## Library

```go
import twill "github.com/kt3k/twill/impl/go"

compiler, err := twill.Compile(`@import "twill";`, twill.CompileOptions{})
if err != nil {
	log.Fatal(err)
}
css := compiler.Build([]string{"flex", "p-4", "hover:underline"})
```

`CompileOptions.LoadStylesheet` resolves user imports; the built-in
stylesheets are embedded and need no loader. `twill.NewScanner` and
`twill.ExtractCandidates` implement source scanning (SPEC §13).

## Development

```sh
gofmt -l .
go vet ./...
go test ./...
```

`css/theme.css` and `css/preflight.css` are ported from
[Tailwind CSS](https://github.com/tailwindlabs/tailwindcss) (MIT License,
Copyright (c) Tailwind Labs, Inc.).

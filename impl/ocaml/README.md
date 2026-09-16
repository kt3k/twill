# twill (OCaml)

An OCaml implementation of the [Twill](../../SPEC.md) utility-first CSS
compiler. It depends only on the standard library plus `str` and `unix`.

## CLI

```sh
dune build
./_build/default/bin/main.exe -i input.css -o output.css
./_build/default/bin/main.exe -i input.css -o output.css --watch
./_build/default/bin/main.exe -i input.css -o output.css --poll 250
./_build/default/bin/main.exe -i input.css --minify > output.css
```

Run with `--help` for the full option list. Watch mode detects change
batches by comparing modification times; `--poll` selects the polling
algorithm of SPEC §14.4.

## Library

```ocaml
let compiler = Twill.Compile.compile {|@import "twill";|} in
let css = Twill.Compile.build compiler [ "flex"; "p-4"; "hover:underline" ]
```

`Compile.compile` takes an optional `~load` for user imports; the built-in
stylesheets are embedded and need no loader. `Scanner` and
`Extract.extract_candidates` implement source scanning (SPEC §13). Errors
are raised as `Twill_error.Error`.

## Development

```sh
dune build
dune test
```

Requires OCaml 4.14 or later and dune 3. `conformance/run.ml` is the runner
used by the repository-level [conformance comparison](../../conformance).

`css/theme.css` and `css/preflight.css` are ported from
[Tailwind CSS](https://github.com/tailwindlabs/tailwindcss) (MIT License,
Copyright (c) Tailwind Labs, Inc.).

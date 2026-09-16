# Conformance comparison

Runs the same inputs through the Deno, Go, and OCaml implementations and
diffs the outputs against the Deno output. Requires `deno`, `go`, and
`dune` (with OCaml) on the path.

```sh
conformance/compare.sh       # library API: cases/*/input.css + candidates.txt
conformance/compare_cli.sh   # CLI on project/: build, --minify, default input
```

- `cases/<name>/input.css` is the entry stylesheet, `candidates.txt` lists one
  candidate per line, and an optional `files/` directory holds stylesheets
  imported relatively from `input.css`. Each case is also built a second time
  with the candidates reversed to check order independence.
- `project/` is a small source tree with a `.gitignore`, `node_modules`, and
  `@source` directives so that scanning and extraction are compared too.
  The fixture files that its `.gitignore` ignores are tracked with `git add -f`.
- The runners are `run_deno.ts`, `run_go/`, and `impl/ocaml/conformance/`
  (the OCaml runner lives inside the dune project so it can link the library).
- Outputs, diffs, and the Go and OCaml binaries are written to `out/`, which
  is ignored.

Both scripts exit non-zero when any output differs.

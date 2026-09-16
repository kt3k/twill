# Conformance comparison

Runs the same inputs through the Deno and Go implementations and diffs the
outputs. Requires `deno` and `go` on the path.

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
- Outputs, diffs, and the Go binary are written to `out/`, which is ignored.

Both scripts exit non-zero when any output differs.

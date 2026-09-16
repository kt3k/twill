#!/usr/bin/env bash
# Compares the Deno, Go, and OCaml CLIs on the sample project in project/
# (scanner, extraction, minification). Builds the Go and OCaml binaries into
# out/, which is ignored.
set -euo pipefail
cd "$(dirname "$0")"
root=$PWD
mkdir -p out/cli
go build -C ../impl/go -o "$root/out/cli/twill-go" ./cmd/twill
(cd ../impl/ocaml && dune build --no-print-directory bin/main.exe)
cp ../impl/ocaml/_build/default/bin/main.exe out/cli/twill-ocaml
status=0
run() {
  local label=$1; shift
  (cd ../impl/deno && deno run -A cli.ts --cwd "$root/project" --silent "$@") > "out/cli/$label.deno.css"
  out/cli/twill-go --cwd project --silent "$@" > "out/cli/$label.go.css"
  out/cli/twill-ocaml --cwd project --silent "$@" > "out/cli/$label.ocaml.css"
  for impl in go ocaml; do
    if ! diff -u "out/cli/$label.deno.css" "out/cli/$label.$impl.css" > "out/cli/$label.$impl.diff"; then
      echo "DIFF: $label ($impl; see conformance/out/cli/$label.$impl.diff)"; status=1
    else
      rm "out/cli/$label.$impl.diff"; echo "same: $label $impl ($(wc -c < "out/cli/$label.deno.css") bytes)"
    fi
  done
}
run build -i input.css
run minify -i input.css --minify
run default-input < /dev/null
exit $status

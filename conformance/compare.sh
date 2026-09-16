#!/usr/bin/env bash
# Compares the Deno, Go, and OCaml implementations on the shared conformance
# cases (library API: compile + build). Outputs go to out/, which is ignored.
set -euo pipefail
cd "$(dirname "$0")"
root=$PWD
rm -rf out/deno out/go out/ocaml
(cd ../impl/deno && deno run -A "$root/run_deno.ts")
(cd run_go && go run . "$root")
(cd ../impl/ocaml && dune build --no-print-directory conformance/run.exe && ./_build/default/conformance/run.exe "$root")
status=0
for f in out/deno/*.css; do
  name=$(basename "$f")
  for impl in go ocaml; do
    if ! diff -u "out/deno/$name" "out/$impl/$name" > "out/$name.$impl.diff"; then
      echo "DIFF: $name ($impl; see conformance/out/$name.$impl.diff)"
      status=1
    else
      rm "out/$name.$impl.diff"
    fi
  done
done
echo "compared $(ls out/deno | wc -l) cases"
exit $status

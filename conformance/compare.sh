#!/usr/bin/env bash
# Compares the Deno and Go implementations on the shared conformance cases
# (library API: compile + build). Outputs go to out/, which is ignored.
set -euo pipefail
cd "$(dirname "$0")"
root=$PWD
rm -rf out/deno out/go
(cd ../impl/deno && deno run -A "$root/run_deno.ts")
(cd run_go && go run . "$root")
status=0
for f in out/deno/*.css; do
  name=$(basename "$f")
  if ! diff -u "out/deno/$name" "out/go/$name" > "out/$name.diff"; then
    echo "DIFF: $name (see conformance/out/$name.diff)"
    status=1
  else
    rm "out/$name.diff"
  fi
done
echo "compared $(ls out/deno | wc -l) cases"
exit $status

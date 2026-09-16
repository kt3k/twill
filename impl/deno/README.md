# @kt3k/twill

A Deno implementation of the [Twill](../../SPEC.md) utility-first CSS compiler.

## CLI

```sh
deno run -A jsr:@kt3k/twill/cli -i input.css -o output.css
deno run -A jsr:@kt3k/twill/cli -i input.css -o output.css --watch
deno run -A jsr:@kt3k/twill/cli -i input.css -o output.css --poll 250
deno run -A jsr:@kt3k/twill/cli -i input.css --minify > output.css
```

Run with `--help` for the full option list.

## Library

```ts
import { compile } from "jsr:@kt3k/twill";

const compiler = await compile(`@import "twill";`);
const css = compiler.build(["flex", "p-4", "hover:underline"]);
```

`compile` accepts a `loadStylesheet(id, base)` callback for resolving user
imports; the built-in stylesheets are embedded and need no loader.

## Development

```sh
deno task ok    # fmt --check, lint, check, test
deno task gen   # regenerate src/bundled_css.ts from css/
```

`css/theme.css` and `css/preflight.css` are ported from
[Tailwind CSS](https://github.com/tailwindlabs/tailwindcss) (MIT License,
Copyright (c) Tailwind Labs, Inc.).

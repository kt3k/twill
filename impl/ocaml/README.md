# twill (OCaml)

An OCaml implementation of the [Twill](../../SPEC.md) utility-first CSS
compiler. It depends only on the standard library plus `str` and `unix`.

## Development

```sh
dune build
dune test
```

Requires OCaml 4.14 or later and dune 3.

`css/theme.css` and `css/preflight.css` are ported from
[Tailwind CSS](https://github.com/tailwindlabs/tailwindcss) (MIT License,
Copyright (c) Tailwind Labs, Inc.).

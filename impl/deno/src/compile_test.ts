import { assertEquals, assertRejects, assertStringIncludes } from "@std/assert";
import { compile } from "./compile.ts";
import { TwillError } from "./error.ts";
import { Features } from "./features.ts";
import { memoryLoader } from "./testing.ts";

/** Compiles and builds, returning the CSS. */
async function run(
  css: string,
  candidates: string[] = [],
  files: Record<string, string> = {},
): Promise<string> {
  const compiler = await compile(css, {
    base: "/root",
    loadStylesheet: memoryLoader(files),
  });
  return compiler.build(candidates);
}

/** Like `run` but drops the leading `:root, :host { ... }` block. */
async function runUtilities(
  css: string,
  candidates: string[],
): Promise<string> {
  const output = await run(css, candidates);
  return output.replace(/^:root, :host \{\n(?:[ ]{2}.*\n)*\}\n/, "");
}

const THEME = `@import "twill/theme.css";\n@twill utilities;\n`;

Deno.test("SPEC §16.1 basic document", async () => {
  const output = await run(
    `@theme {
  --color-black: #000;
  --breakpoint-md: 768px;
}
@layer utilities {
  @twill utilities;
}`,
    ["dark:bg-black", "hover:underline", "md:grid", "flex"],
  );
  assertEquals(
    output,
    `:root, :host {
  --color-black: #000;
}
@layer utilities {
  .flex {
    display: flex;
  }
  @media (hover: hover) {
    .hover\\:underline:hover {
      text-decoration-line: underline;
    }
  }
  @media (width >= 768px) {
    .md\\:grid {
      display: grid;
    }
  }
  @media (prefers-color-scheme: dark) {
    .dark\\:bg-black {
      background-color: var(--color-black);
    }
  }
}
`,
  );
});

Deno.test("SPEC §16.2 value forms", async () => {
  const cases: [string, string][] = [
    ["p-4", "padding: calc(var(--spacing) * 4);"],
    ["p-1", "padding: var(--spacing);"],
    ["p-0", "padding: 0px;"],
    ["p-px", "padding: 1px;"],
    ["-mt-2", "margin-top: calc(var(--spacing) * -2);"],
    ["w-1/2", "width: calc(1 / 2 * 100%);"],
    ["w-[13px]", "width: 13px;"],
    ["w-(--my-w)", "width: var(--my-w);"],
    ["max-w-md", "max-width: var(--container-md);"],
    ["bg-red-500", "background-color: var(--color-red-500);"],
    [
      "bg-red-500/50",
      "background-color: color-mix(in oklab, var(--color-red-500) 50%, transparent);",
    ],
    ["bg-[#0088cc]", "background-color: #0088cc;"],
    ["bg-[url(/a_b.png)]", "background-image: url(/a_b.png);"],
    ["bg-[length:10px_20px]", "background-size: 10px 20px;"],
    [
      "text-lg",
      "font-size: var(--text-lg);\n  line-height: var(--tw-leading, var(--text-lg--line-height));",
    ],
    [
      "text-lg/8",
      "font-size: var(--text-lg);\n  line-height: calc(var(--spacing) * 8);",
    ],
    ["text-red-500", "color: var(--color-red-500);"],
    [
      "font-bold",
      "--tw-font-weight: var(--font-weight-bold);\n  font-weight: var(--font-weight-bold);",
    ],
    ["rounded-lg", "border-radius: var(--radius-lg);"],
    ["rounded-full", "border-radius: calc(infinity * 1px);"],
    ["border", "border-style: var(--tw-border-style);\n  border-width: 1px;"],
    ["border-2", "border-style: var(--tw-border-style);\n  border-width: 2px;"],
    ["z-10", "z-index: 10;"],
    ["-z-10", "z-index: calc(10 * -1);"],
    ["flex-1", "flex: 1;"],
    ["opacity-50", "opacity: 50%;"],
    ["[mask-type:luminance]", "mask-type: luminance;"],
    ["[--my-var:1px]", "--my-var: 1px;"],
    ["underline!", "text-decoration-line: underline !important;"],
  ];
  for (const [candidate, declarations] of cases) {
    const output = await runUtilities(THEME, [candidate]);
    const escaped = candidate.replace(/[^a-zA-Z0-9_-]/g, (c) => `\\${c}`);
    assertStringIncludes(
      output,
      `.${escaped} {\n  ${declarations}\n}\n`,
      candidate,
    );
  }

  const fontBold = await runUtilities(THEME, ["font-bold"]);
  assertStringIncludes(
    fontBold,
    `@property --tw-font-weight {\n  syntax: "*";\n  inherits: false;\n}\n`,
  );
  const border = await runUtilities(THEME, ["border"]);
  assertStringIncludes(
    border,
    `@property --tw-border-style {\n  syntax: "*";\n  inherits: false;\n  initial-value: solid;\n}\n`,
  );
});

Deno.test("SPEC §16.3 variant forms", async () => {
  const cases: [string, string][] = [
    [
      "hover:flex",
      "@media (hover: hover) {\n  .hover\\:flex:hover {\n    display: flex;\n  }\n}\n",
    ],
    ["focus:flex", ".focus\\:flex:focus {\n  display: flex;\n}\n"],
    [
      "sm:flex",
      "@media (width >= 40rem) {\n  .sm\\:flex {\n    display: flex;\n  }\n}\n",
    ],
    [
      "max-md:flex",
      "@media (width < 48rem) {\n  .max-md\\:flex {\n    display: flex;\n  }\n}\n",
    ],
    [
      "min-[600px]:flex",
      "@media (width >= 600px) {\n  .min-\\[600px\\]\\:flex {\n    display: flex;\n  }\n}\n",
    ],
    [
      "@md:flex",
      "@container (width >= 28rem) {\n  .\\@md\\:flex {\n    display: flex;\n  }\n}\n",
    ],
    [
      "@md/main:flex",
      "@container main (width >= 28rem) {\n  .\\@md\\/main\\:flex {\n    display: flex;\n  }\n}\n",
    ],
    [
      "group-hover:flex",
      "@media (hover: hover) {\n  .group-hover\\:flex:is(:where(.group):hover *) {\n    display: flex;\n  }\n}\n",
    ],
    [
      "group-hover/item:flex",
      "@media (hover: hover) {\n  .group-hover\\/item\\:flex:is(:where(.group\\/item):hover *) {\n    display: flex;\n  }\n}\n",
    ],
    [
      "peer-checked:flex",
      ".peer-checked\\:flex:is(:where(.peer):checked ~ *) {\n  display: flex;\n}\n",
    ],
    [
      "has-[>img]:flex",
      ".has-\\[\\>img\\]\\:flex:has(> img) {\n  display: flex;\n}\n",
    ],
    [
      "in-data-visible:flex",
      ":where(*[data-visible]) .in-data-visible\\:flex {\n  display: flex;\n}\n",
    ],
    [
      "not-hover:flex",
      ".not-hover\\:flex:not(:hover) {\n  display: flex;\n}\n@media not all and (hover: hover) {\n  .not-hover\\:flex {\n    display: flex;\n  }\n}\n",
    ],
    [
      "not-supports-grid:flex",
      "@supports not (grid: var(--tw)) {\n  .not-supports-grid\\:flex {\n    display: flex;\n  }\n}\n",
    ],
    [
      "data-[state=open]:flex",
      '.data-\\[state\\=open\\]\\:flex[data-state="open"] {\n  display: flex;\n}\n',
    ],
    [
      "aria-checked:flex",
      '.aria-checked\\:flex[aria-checked="true"] {\n  display: flex;\n}\n',
    ],
    ["nth-3:flex", ".nth-3\\:flex:nth-child(3) {\n  display: flex;\n}\n"],
    ["[&_p]:flex", ".\\[\\&_p\\]\\:flex p {\n  display: flex;\n}\n"],
    [
      "[@media(width>=100px)]:flex",
      "@media (width>=100px) {\n  .\\[\\@media\\(width\\>\\=100px\\)\\]\\:flex {\n    display: flex;\n  }\n}\n",
    ],
    ["*:flex", ":is(.\\*\\:flex > *) {\n  display: flex;\n}\n"],
    [
      "before:block",
      ".before\\:block::before {\n  content: var(--tw-content);\n  display: block;\n}\n",
    ],
    [
      "dark:hover:flex",
      "@media (prefers-color-scheme: dark) {\n  @media (hover: hover) {\n    .dark\\:hover\\:flex:hover {\n      display: flex;\n    }\n  }\n}\n",
    ],
  ];
  for (const [candidate, expected] of cases) {
    const output = await runUtilities(THEME, [candidate]);
    assertStringIncludes(output, expected, candidate);
  }
});

Deno.test("SPEC §16.4 custom utilities and @apply", async () => {
  const output = await runUtilities(
    `@import "twill/theme.css";
@utility tab-* {
  tab-size: --value(integer);
  tab-size: --value(--tab-size-*);
  tab-size: --value([integer]);
}
@utility content-auto {
  content-visibility: auto;
}
.btn {
  @apply rounded-lg px-4 py-2 hover:bg-red-500;
}
@twill utilities;`,
    ["tab-4", "tab-[8]", "content-auto"],
  );
  // `tab-size` has a position in the global property order while
  // `content-visibility` does not, so `tab-*` sorts before `content-auto`
  // per SPEC §11.3.
  assertEquals(
    output,
    `.btn {
  border-radius: var(--radius-lg);
  padding-inline: calc(var(--spacing) * 4);
  padding-block: calc(var(--spacing) * 2);
}
@media (hover: hover) {
  .btn:hover {
    background-color: var(--color-red-500);
  }
}
.tab-4 {
  tab-size: 4;
}
.tab-\\[8\\] {
  tab-size: 8;
}
.content-auto {
  content-visibility: auto;
}
`,
  );
});

Deno.test("SPEC §16.5 theme customization", async () => {
  const css = `@import "twill";
@theme {
  --color-*: initial;
  --color-primary: oklch(0.6 0.2 250);
  --breakpoint-3xl: 120rem;
  --font-display: "Inter", sans-serif;
}
@custom-variant dark (&:where(.dark, .dark *));`;
  const output = await run(css, [
    "bg-primary",
    "3xl:flex",
    "font-display",
    "bg-red-500",
    "dark:flex",
  ]);
  assertStringIncludes(
    output,
    ".bg-primary {\n    background-color: var(--color-primary);\n  }",
  );
  assertStringIncludes(
    output,
    "@media (width >= 120rem) {\n    .\\33 xl\\:flex {\n      display: flex;\n    }\n  }",
  );
  assertStringIncludes(
    output,
    ".font-display {\n    font-family: var(--font-display);\n  }",
  );
  assertStringIncludes(
    output,
    ".dark\\:flex:where(.dark, .dark *) {\n    display: flex;\n  }",
  );
  assertEquals(output.includes("bg-red-500"), false);
  assertEquals(output.includes("--color-red-500"), false);
  assertStringIncludes(output, "--color-primary: oklch(0.6 0.2 250);");
});

Deno.test("build: output is independent of candidate order and stable", async () => {
  const css = THEME;
  const a = await run(css, ["p-4", "hover:flex", "sm:p-2", "flex", "m-1"]);
  const b = await run(css, ["m-1", "flex", "sm:p-2", "hover:flex", "p-4"]);
  assertEquals(a, b);
  const compiler = await compile(css);
  const first = compiler.build(["flex", "p-4"]);
  const second = compiler.build(["p-4"]);
  assertEquals(first === second, true);
  const third = compiler.build(["flex", "nope"]);
  assertEquals(third, first);
  const fourth = compiler.build(["m-2"]);
  assertEquals(fourth !== first, true);
  assertStringIncludes(fourth, ".m-2");
});

Deno.test("build: candidates accumulate across builds", async () => {
  const compiler = await compile(THEME);
  compiler.build(["flex"]);
  const output = compiler.build(["hidden"]);
  assertStringIncludes(output, ".flex {");
  assertStringIncludes(output, ".hidden {");
});

Deno.test("build: features", async () => {
  const compiler = await compile(".a { color: red; }");
  assertEquals(compiler.features, Features.NONE);
  assertEquals(compiler.build(["flex"]), ".a { color: red; }");

  const themed = await compile("@theme { --a: 1; } .a { color: var(--a); }");
  assertEquals(themed.features & Features.AT_THEME, Features.AT_THEME);
  assertEquals(
    themed.build(["flex"]),
    ":root, :host {\n  --a: 1;\n}\n.a {\n  color: var(--a);\n}\n",
  );

  const full = await compile(
    `@import "twill"; .a { @apply flex; } .b { @variant hover { color: red; } }`,
  );
  assertEquals(full.features & Features.AT_IMPORT, Features.AT_IMPORT);
  assertEquals(full.features & Features.AT_APPLY, Features.AT_APPLY);
  assertEquals(full.features & Features.VARIANTS, Features.VARIANTS);
  assertEquals(full.features & Features.UTILITIES, Features.UTILITIES);
  assertEquals(
    full.features & Features.THEME_FUNCTION,
    Features.THEME_FUNCTION,
  );
});

Deno.test("important: stylesheet flag and candidate marker", async () => {
  const output = await run(
    `@import "twill/theme.css" important;\n@twill utilities;`,
    [
      "flex",
      "font-bold",
    ],
  );
  assertStringIncludes(output, "display: flex !important;");
  assertStringIncludes(
    output,
    "font-weight: var(--font-weight-bold) !important;",
  );
  // Registrations are never marked important.
  assertStringIncludes(output, "inherits: false;\n");
  assertEquals(output.includes("inherits: false !important"), false);

  const applied = await run(
    `@import "twill/theme.css" important;\n.a { @apply flex underline!; }\n@twill utilities;`,
  );
  assertStringIncludes(
    applied,
    ".a {\n  display: flex;\n  text-decoration-line: underline !important;\n}",
  );
});

Deno.test("theme functions", async () => {
  const output = await run(`@import "twill/theme.css";
.a {
  padding: --spacing(4);
  margin: --spacing(0) --spacing(1);
  color: --alpha(var(--color-red-500) / 50%);
  width: --theme(--container-md);
  height: --theme(--container-md inline);
  gap: --theme(--nope, 1px, 2px);
  font: theme(--text-lg);
  border-color: --theme(--color-red-500/0.5);
}
@media (width >= --theme(--breakpoint-md)) {
  .b { color: red; }
}`);
  assertStringIncludes(output, "padding: calc(var(--spacing) * 4);");
  assertStringIncludes(output, "margin: 0px var(--spacing);");
  assertStringIncludes(
    output,
    "color: color-mix(in oklab, var(--color-red-500) 50%, transparent);",
  );
  assertStringIncludes(output, "width: var(--container-md);");
  assertStringIncludes(output, "height: 28rem;");
  assertStringIncludes(output, "gap: 1px, 2px;");
  assertStringIncludes(output, "font: 1.125rem;");
  assertStringIncludes(
    output,
    "border-color: color-mix(in oklab, var(--color-red-500) 50%, transparent);",
  );
  assertStringIncludes(output, "@media (width >= 48rem) {");
  // The variables referenced through `var(...)` survive pruning.
  assertStringIncludes(output, "--container-md: 28rem;");
});

Deno.test("theme functions: fallback injection and errors", async () => {
  const output = await run(`@theme { --a: var(--b); --c: initial; --d: 1px; }
.x { color: --theme(--a, red); background: --theme(--a inline, red); width: --theme(--d, initial); height: --theme(--c, 2px); }`);
  assertStringIncludes(output, "color: var(--a, red);");
  assertStringIncludes(output, "background: var(--b, red);");
  assertStringIncludes(output, "width: var(--d);");
  assertStringIncludes(output, "height: 2px;");

  await assertRejects(
    () => run(".a { padding: --spacing(); }"),
    TwillError,
    "--spacing",
  );
  await assertRejects(
    () => run(".a { padding: --spacing(1, 2); }"),
    TwillError,
    "--spacing",
  );
  await assertRejects(
    () => run(".a { padding: --spacing(4); }"),
    TwillError,
    "--spacing",
  );
  await assertRejects(
    () => run(".a { color: --alpha(red); }"),
    TwillError,
    "--alpha",
  );
  await assertRejects(
    () => run(".a { color: --theme(spacing); }"),
    TwillError,
    "--theme",
  );
  await assertRejects(
    () => run(".a { color: --theme(--nope); }"),
    TwillError,
    "resolve",
  );
  await assertRejects(
    () => run(".a { color: theme(--nope); }"),
    TwillError,
    "resolve",
  );
});

Deno.test("theme functions: failure in a candidate makes it invalid", async () => {
  // No `--spacing` in the theme: `p-4` has no spacing multiplier and is skipped.
  const output = await run(`@theme { --color-a: red; }\n@twill utilities;`, [
    "p-4",
    "bg-a",
  ]);
  assertEquals(output.includes("p-4"), false);
  assertStringIncludes(output, ".bg-a {");
});

Deno.test("@apply: variants, errors, mixins", async () => {
  const output = await run(`@import "twill/theme.css";
.a { @apply flex hover:underline sm:p-2; }
.b { @apply --my-mixin; }
@apply flex;`);
  assertStringIncludes(
    output,
    `.a {
  display: flex;
}
@media (hover: hover) {
  .a:hover {
    text-decoration-line: underline;
  }
}
@media (width >= 40rem) {
  .a {
    padding: calc(var(--spacing) * 2);
  }
}
.b {
  @apply --my-mixin;
}
@apply flex;
`,
  );

  await assertRejects(() => run(`.a { @apply --x flex; }`), TwillError, "mix");
  await assertRejects(
    () => run(`.a { @apply flex { color: red; } }`),
    TwillError,
    "body",
  );
  await assertRejects(
    () => run(`@keyframes x { to { @apply flex; } }`),
    TwillError,
    "@keyframes",
  );
  await assertRejects(() => run(`.a { @apply nope; }`), TwillError, "empty");
  await assertRejects(
    () => run(`@import "twill/theme.css"; .a { @apply nope; }`),
    TwillError,
    "unknown utility",
  );
  await assertRejects(
    () => run(`@import "twill/theme.css"; .a { @apply nope:flex; }`),
    TwillError,
    "unknown variant",
  );
  await assertRejects(
    () => run(`@import "twill/theme.css" prefix(tw); .a { @apply flex; }`),
    TwillError,
    "prefix",
  );
  await assertRejects(
    () =>
      run(
        `@import "twill/theme.css"; @source not inline("flex"); .a { @apply flex; }`,
      ),
    TwillError,
    "disabled",
  );
});

Deno.test("@apply: custom utilities in dependency order and cycles", async () => {
  const output = await run(
    `@import "twill/theme.css";
@utility foo { @apply bar p-1; }
@utility bar { color: red; }
.x { @apply foo; }
@twill utilities;`,
    ["foo"],
  );
  assertStringIncludes(
    output,
    ".x {\n  padding: var(--spacing);\n  color: red;\n}",
  );
  assertStringIncludes(
    output,
    ".foo {\n  padding: var(--spacing);\n  color: red;\n}",
  );

  await assertRejects(
    () => run(`@utility foo { @apply bar; }\n@utility bar { @apply foo; }`),
    TwillError,
    "circular",
  );
  await assertRejects(
    () => run(`@utility foo { @apply foo; }`),
    TwillError,
    "circular",
  );
});

Deno.test("@utility: static and functional forms", async () => {
  const output = await run(
    `@import "twill/theme.css";
@theme { --tab-size-github: 8; --leading-tight: 1.25; }
@utility tab-* {
  tab-size: --value(integer);
  tab-size: --value(--tab-size-*);
  tab-size: --value([integer]);
}
@utility aspect-* {
  aspect-ratio: --value(ratio, --aspect-*, [ratio]);
  width: --value(number);
}
@utility opacity-* {
  opacity: --value(percentage);
  opacity: --modifier(number);
}
@utility text-* {
  font-size: --value(--text-*);
  line-height: --value(--text-*--line-height);
  line-height: --modifier(--leading-*);
}
@utility lit-* {
  color: --value("red", 'blue');
}
@utility any-* {
  color: --value([*]);
}
@utility content-auto { content-visibility: auto; }
@utility foo-1\\/2 { color: red; }
@twill utilities;`,
    [
      "tab-4",
      "tab-github",
      "tab-[8]",
      "tab-[foo]",
      "tab-x",
      "aspect-16/9",
      "aspect-video",
      "aspect-[4/3]",
      "aspect-3",
      "aspect-16/9/2",
      "opacity-50%",
      "opacity-50%/2",
      "opacity-50%/foo",
      "text-lg",
      "text-lg/tight",
      "lit-red",
      "lit-blue",
      "lit-green",
      "any-[foo]",
      "content-auto",
      "foo-1/2",
    ],
  );
  assertStringIncludes(output, ".tab-4 {\n  tab-size: 4;\n}");
  assertStringIncludes(
    output,
    ".tab-github {\n  tab-size: var(--tab-size-github);\n}",
  );
  assertStringIncludes(output, ".tab-\\[8\\] {\n  tab-size: 8;\n}");
  assertEquals(output.includes("tab-\\[foo\\]"), false);
  assertEquals(output.includes("tab-x"), false);
  // A ratio value removes the non-ratio declarations.
  assertStringIncludes(output, ".aspect-16\\/9 {\n  aspect-ratio: 16 / 9;\n}");
  assertStringIncludes(
    output,
    ".aspect-video {\n  aspect-ratio: var(--aspect-video);\n}",
  );
  assertStringIncludes(
    output,
    ".aspect-\\[4\\/3\\] {\n  aspect-ratio: 4/3;\n}",
  );
  assertStringIncludes(output, ".aspect-3 {\n  width: 3;\n}");
  assertEquals(output.includes("aspect-16\\/9\\/2"), false);
  assertStringIncludes(output, ".opacity-50\\% {\n  opacity: 50%;\n}");
  assertStringIncludes(
    output,
    ".opacity-50\\%\\/2 {\n  opacity: 50%;\n  opacity: 2;\n}",
  );
  assertEquals(output.includes("opacity-50\\%\\/foo"), false);
  assertStringIncludes(
    output,
    ".text-lg {\n  font-size: var(--text-lg);\n  line-height: var(--text-lg--line-height);\n}",
  );
  assertStringIncludes(
    output,
    ".text-lg\\/tight {\n  font-size: var(--text-lg);\n  line-height: var(--text-lg--line-height);\n  line-height: var(--leading-tight);\n}",
  );
  assertStringIncludes(output, ".lit-red {\n  color: red;\n}");
  assertStringIncludes(output, ".lit-blue {\n  color: blue;\n}");
  assertEquals(output.includes("lit-green"), false);
  assertStringIncludes(output, ".any-\\[foo\\] {\n  color: foo;\n}");
  assertStringIncludes(
    output,
    ".content-auto {\n  content-visibility: auto;\n}",
  );
  assertStringIncludes(output, ".foo-1\\/2 {\n  color: red;\n}");
});

Deno.test("@utility: errors", async () => {
  await assertRejects(() => run(`@utility foo {}`), TwillError, "empty");
  await assertRejects(
    () => run(`@utility foo* { color: red; }`),
    TwillError,
    "-*",
  );
  await assertRejects(
    () => run(`@utility fo*o { color: red; }`),
    TwillError,
    "end",
  );
  await assertRejects(
    () => run(`@utility Foo { color: red; }`),
    TwillError,
    "invalid utility name",
  );
  await assertRejects(
    () => run(`@utility foo- { color: red; }`),
    TwillError,
    "invalid utility name",
  );
  await assertRejects(
    () => run(`.a { @utility foo { color: red; } }`),
    TwillError,
    "nested",
  );
});

Deno.test("@custom-variant: selector form", async () => {
  const output = await run(
    `@import "twill/theme.css";
@custom-variant dark (&:where(.dark, .dark *));
@custom-variant hocus (&:hover, &:focus);
@custom-variant wide (@media (width >= 100px), @supports (display: grid));
@custom-variant mixed (&:hover, @media print);
@twill utilities;`,
    ["dark:flex", "hocus:flex", "wide:flex", "mixed:flex", "not-hocus:flex"],
  );
  assertStringIncludes(
    output,
    ".dark\\:flex:where(.dark, .dark *) {\n  display: flex;\n}",
  );
  assertStringIncludes(
    output,
    ".hocus\\:flex:hover, .hocus\\:flex:focus {\n  display: flex;\n}",
  );
  assertStringIncludes(
    output,
    "@media (width >= 100px) {\n  .wide\\:flex {\n    display: flex;\n  }\n}\n@supports (display: grid) {\n  .wide\\:flex {\n    display: flex;\n  }\n}",
  );
  assertStringIncludes(
    output,
    ".mixed\\:flex:hover {\n  display: flex;\n}\n@media print {\n  .mixed\\:flex {\n    display: flex;\n  }\n}",
  );
  assertStringIncludes(
    output,
    ".not-hocus\\:flex:not(:hover), .not-hocus\\:flex:not(:focus) {",
  );

  await assertRejects(
    () => run(`@custom-variant foo (&:hover) { color: red; }`),
    TwillError,
    "both",
  );
  await assertRejects(
    () => run(`@custom-variant foo;`),
    TwillError,
    "no selector",
  );
  await assertRejects(
    () => run(`@custom-variant Foo (&:hover);`),
    TwillError,
    "invalid variant name",
  );
  await assertRejects(
    () => run(`@custom-variant foo (&:hover,);`),
    TwillError,
    "empty",
  );
  await assertRejects(
    () => run(`.a { @custom-variant foo (&:hover); }`),
    TwillError,
    "nested",
  );
});

Deno.test("@custom-variant: body form, dependencies, cycles", async () => {
  const output = await run(
    `@import "twill/theme.css";
@custom-variant theme-dark {
  &:where([data-theme="dark"], [data-theme="dark"] *) {
    @slot;
  }
}
@custom-variant dark-hover {
  @variant theme-dark {
    &:hover {
      @slot;
    }
  }
}
@custom-variant animated {
  @keyframes spin-custom { to { transform: rotate(1turn); } }
  animation: spin-custom 1s;
  @slot;
}
@twill utilities;`,
    ["theme-dark:flex", "dark-hover:flex", "animated:flex"],
  );
  assertStringIncludes(
    output,
    '.theme-dark\\:flex:where([data-theme="dark"], [data-theme="dark"] *) {\n  display: flex;\n}',
  );
  assertStringIncludes(
    output,
    '.dark-hover\\:flex:where([data-theme="dark"], [data-theme="dark"] *):hover {\n  display: flex;\n}',
  );
  assertStringIncludes(
    output,
    ".animated\\:flex {\n  animation: spin-custom 1s;\n  display: flex;\n}",
  );
  assertStringIncludes(
    output,
    "@keyframes spin-custom {\n  to {\n    transform: rotate(1turn);\n  }\n}",
  );

  await assertRejects(
    () =>
      run(
        `@custom-variant a { @variant b { @slot; } }\n@custom-variant b { @variant a { @slot; } }`,
      ),
    TwillError,
    "circular",
  );
});

Deno.test("@variant compatibility forms", async () => {
  const output = await run(
    `@import "twill/theme.css";
@variant hocus (&:hover, &:focus);
@variant dark { &:where(.dark, .dark *) { @slot; } }
@twill utilities;`,
    ["hocus:flex", "dark:flex"],
  );
  assertStringIncludes(output, ".hocus\\:flex:hover, .hocus\\:flex:focus {");
  assertStringIncludes(output, ".dark\\:flex:where(.dark, .dark *) {");
  assertEquals(output.includes("@variant"), false);
});

Deno.test("nested @variant", async () => {
  const output = await run(`@import "twill/theme.css";
.a {
  color: red;
  @variant hover { color: blue; }
  @variant dark:hover, sm { color: green; }
}`);
  assertEquals(
    output,
    `.a {
  color: red;
}
@media (hover: hover) {
  .a:hover {
    color: blue;
  }
}
@media (prefers-color-scheme: dark) {
  @media (hover: hover) {
    .a:hover {
      color: green;
    }
  }
}
@media (width >= 40rem) {
  .a {
    color: green;
  }
}
`,
  );
  await assertRejects(
    () => run(`.a { @variant nope { color: red; } }`),
    TwillError,
    "unknown variant",
  );
  await assertRejects(
    () =>
      run(`@import "twill/theme.css"; .a { @variant hover: { color: red; } }`),
    TwillError,
    "empty",
  );
});

Deno.test("optimizer: registrations print once and are hoisted", async () => {
  const output = await run(THEME, ["font-bold", "font-thin", "leading-6"]);
  assertEquals(output.split("@property --tw-font-weight").length, 2);
  const registration = output.indexOf("@property");
  const lastRule = output.lastIndexOf(".leading-6");
  assertEquals(registration > lastRule, true);
  assertEquals(output.trimEnd().endsWith("}"), true);
});

Deno.test("optimizer: unused theme values and keyframes are pruned", async () => {
  const css = `@import "twill/theme.css";
@theme { --color-keep: red; --keep-static: 1px; --chain-a: var(--chain-b); --chain-b: 2px; }
@theme static { --always: 3px; }
@twill utilities;`;
  const output = await run(css, ["bg-keep", "[width:var(--chain-a)]"]);
  assertStringIncludes(output, "--color-keep: red;");
  assertStringIncludes(output, "--chain-a: var(--chain-b);");
  assertStringIncludes(output, "--chain-b: 2px;");
  assertStringIncludes(output, "--always: 3px;");
  assertEquals(output.includes("--keep-static"), false);
  assertEquals(output.includes("--color-red-500"), false);
  assertEquals(output.includes("@keyframes"), false);

  const spin = await run(css, ["animate-spin"]);
  assertStringIncludes(spin, "@keyframes spin {");
  assertEquals(spin.includes("@keyframes ping"), false);
  assertStringIncludes(spin, "--animate-spin: spin 1s linear infinite;");

  // A `--key` candidate keeps the variable.
  const marked = await run(css, ["--color-blue-500"]);
  assertStringIncludes(marked, "--color-blue-500:");

  // An empty root rule and its layer disappear.
  const empty = await run(
    `@layer theme, base;
@import "twill/theme.css" layer(theme);
@twill utilities;`,
    [],
  );
  assertEquals(empty, "@layer theme, base;\n");
});

Deno.test("optimizer: nesting is flattened", async () => {
  const output = await run(`@import "twill/theme.css";
.a, .b {
  color: red;
  &:hover { color: blue; }
  .c & { color: green; }
  > .d { color: purple; }
  @media (x) { color: orange; .e { color: black; } }
  @supports (y) { @media (z) { & .f { color: white; } } }
}
@keyframes k { 50% { opacity: 0; } }`);
  assertEquals(
    output,
    `.a, .b {
  color: red;
}
:is(.a, .b):hover {
  color: blue;
}
.c :is(.a, .b) {
  color: green;
}
:is(.a, .b) > .d {
  color: purple;
}
@media (x) {
  .a, .b {
    color: orange;
  }
  :is(.a, .b).e {
    color: black;
  }
}
@supports (y) {
  @media (z) {
    :is(.a, .b) .f {
      color: white;
    }
  }
}
@keyframes k {
  50% {
    opacity: 0;
  }
}
`,
  );
});

Deno.test("reference imports produce no output", async () => {
  const output = await run(`@reference "twill";\n.a { @apply flex p-4; }`);
  assertEquals(
    output,
    ".a {\n  display: flex;\n  padding: calc(var(--spacing, 0.25rem) * 4);\n}\n",
  );
});

Deno.test("prefix", async () => {
  const output = await run(
    `@import "twill/theme.css" prefix(tw);\n@twill utilities;`,
    [
      "tw:flex",
      "tw:bg-red-500",
      "tw:group-hover:flex",
      "flex",
    ],
  );
  assertStringIncludes(
    output,
    "--tw-color-red-500: oklch(63.7% 0.237 25.331);",
  );
  assertStringIncludes(output, ".tw\\:flex {\n  display: flex;\n}");
  assertStringIncludes(
    output,
    ".tw\\:bg-red-500 {\n  background-color: var(--tw-color-red-500);\n}",
  );
  assertStringIncludes(
    output,
    ".tw\\:group-hover\\:flex:is(:where(.tw\\:group):hover *)",
  );
  assertEquals(output.includes("\n.flex"), false);
});

Deno.test("@source inline and not inline", async () => {
  const output = await run(
    `@import "twill/theme.css";
@source inline("p-{1,2}");
@source not inline("flex");
@twill utilities;`,
    ["flex", "hidden"],
  );
  assertStringIncludes(output, ".p-1 {");
  assertStringIncludes(output, ".p-2 {");
  assertStringIncludes(output, ".hidden {");
  assertEquals(output.includes(".flex"), false);
});

Deno.test("license comments and external imports survive", async () => {
  const output = await run(
    `/*! license */\n@import url(x.css);\n@import "https://x/y.css";\n@theme { --a: 1; }\n.a { color: var(--a); }`,
  );
  assertEquals(
    output,
    `/*! license */\n@import url(x.css);\n@import "https://x/y.css";\n:root, :host {\n  --a: 1;\n}\n.a {\n  color: var(--a);\n}\n`,
  );
});

Deno.test("user files through the loader", async () => {
  const output = await run(
    `@import "./theme.css";\n@import "twill/theme.css";\n@twill utilities;`,
    ["bg-brand"],
    {
      "/root/theme.css": "@theme { --color-brand: blue; }",
    },
  );
  assertStringIncludes(
    output,
    ".bg-brand {\n  background-color: var(--color-brand);\n}",
  );
});

import { assertEquals, assertThrows } from "@std/assert";
import { collectDirectives } from "./directives.ts";
import { parse } from "./parser.ts";
import { serialize } from "./serializer.ts";
import { Theme, ThemeOptions } from "./theme.ts";
import { TwillError } from "./error.ts";
import { substituteAtImports } from "./import.ts";
import { type LoadedStylesheet, resolveBuiltin } from "./builtin.ts";
import { type AstNode, context, decl, styleRule } from "./ast.ts";
import { Features } from "./features.ts";

function collect(css: string, base = "/root") {
  const theme = new Theme();
  const ast: AstNode[] = [context({ base }, parse(css))];
  const state = collectDirectives(ast, theme);
  return { theme, ast, state };
}

Deno.test("@theme: registers values and is replaced by :root, :host", () => {
  const { theme, ast, state } = collect(`
    @theme {
      --color-black: #000;
      /* comment */
      --breakpoint-md: 768px;
      @keyframes spin { to { transform: rotate(360deg) } }
    }
    @theme { --color-white: #fff; }
  `);
  assertEquals(theme.resolveValue("black", ["--color"]), "#000");
  assertEquals(theme.resolveValue("white", ["--color"]), "#fff");
  assertEquals(theme.getKeyframes().length, 1);
  assertEquals(serialize(ast), ":root, :host {\n}\n");
  assertEquals(state.firstThemeRule?.selector, ":root, :host");
  assertEquals(state.features & Features.AT_THEME, Features.AT_THEME);
});

Deno.test("@theme: options", () => {
  const { theme } = collect(`
    @theme reference { --a: 1; }
    @theme inline { --b: 2; }
    @theme default { --c: 3; }
    @theme static { --d: 4; }
    @theme prefix(tw) { --e: 5; }
  `);
  assertEquals(theme.getOptions("--a"), ThemeOptions.REFERENCE);
  assertEquals(theme.getOptions("--b"), ThemeOptions.INLINE);
  assertEquals(theme.getOptions("--c"), ThemeOptions.DEFAULT);
  assertEquals(theme.getOptions("--d"), ThemeOptions.STATIC);
  assertEquals(theme.prefix, "tw");
  assertThrows(() => collect(`@theme prefix(Tw) {}`), TwillError, "prefix");
  assertThrows(() => collect(`@theme prefix(1) {}`), TwillError, "prefix");
});

Deno.test("@theme: a reference theme first does not create the root rule", () => {
  const { ast } = collect(`@theme reference { --a: 1; }\n@theme { --b: 2; }`);
  assertEquals(serialize(ast), ":root, :host {\n}\n");
});

Deno.test("@theme: rejects non-custom-property children with a snippet", () => {
  const err = assertThrows(
    () => collect(`@theme { .foo { color: red; } }`),
    TwillError,
  );
  assertEquals(err.message.includes(".foo {"), true);
  assertThrows(() => collect(`@theme { color: red; }`), TwillError);
});

Deno.test("@theme: escaped keys are unescaped", () => {
  const { theme } = collect(`@theme { --container-1\\.5: 1.5rem; }`);
  assertEquals(theme.has("--container-1.5"), true);
});

Deno.test("@source: globs, negation, inline, brace expansion", () => {
  const { state, ast } = collect(`
    @source "./src/**/*.html";
    @source not '../vendor/**';
    @source inline("p-{1..3} {hover:,}flex");
    @source not inline("bg-red-{100,200}");
  `);
  assertEquals(state.sources, [
    { base: "/root", pattern: "./src/**/*.html", negated: false },
    { base: "/root", pattern: "../vendor/**", negated: true },
  ]);
  assertEquals(state.inlineCandidates, [
    "p-1",
    "p-2",
    "p-3",
    "hover:flex",
    "flex",
  ]);
  assertEquals(state.ignoredCandidates, ["bg-red-100", "bg-red-200"]);
  assertEquals(serialize(ast), "");
});

Deno.test("@source: errors", () => {
  assertThrows(
    () => collect(`@source "./a" { color: red; }`),
    TwillError,
    "body",
  );
  assertThrows(() => collect(`.a { @source "./a"; }`), TwillError, "nested");
  assertThrows(
    () => collect(`@media x { @source "./a"; }`),
    TwillError,
    "nested",
  );
  assertThrows(() => collect(`@source ./a;`), TwillError, "quoted");
  assertThrows(() => collect(`@source inline(p-4);`), TwillError, "quoted");
});

Deno.test("@twill utilities: first occurrence kept, source() parsed", () => {
  const { state, ast } = collect(`@twill utilities;\n@twill utilities;`);
  assertEquals(serialize(ast), "@twill utilities;\n");
  assertEquals(state.utilitiesNode?.name, "@twill");
  assertEquals(state.root, null);
  assertEquals(state.features & Features.UTILITIES, Features.UTILITIES);

  assertEquals(collect(`@twill utilities source(none);`).state.root, "none");
  assertEquals(collect(`@twill utilities source("../app");`).state.root, {
    base: "/root",
    pattern: "../app",
  });
  assertThrows(
    () => collect(`@twill utilities source(../app);`),
    TwillError,
    "quoted",
  );
});

Deno.test("@twill utilities: ignored inside reference context", () => {
  const { state, ast } = collect(`@media reference { @twill utilities; }`);
  assertEquals(state.utilitiesNode, null);
  assertEquals(serialize(ast), "");
});

function makeLoader(files: Record<string, string>) {
  return (id: string, base: string): LoadedStylesheet => {
    const builtin = resolveBuiltin(id, base);
    if (builtin !== null) return builtin;
    const path = `${base}/${id.replace(/^\.\//, "")}`;
    if (!(path in files)) throw new TwillError(`Cannot find ${path}`);
    return {
      path,
      base: path.slice(0, path.lastIndexOf("/")),
      content: files[path],
    };
  };
}

async function importAndCollect(
  css: string,
  files: Record<string, string> = {},
) {
  const theme = new Theme();
  const ast: AstNode[] = [context({ base: "/root" }, parse(css))];
  await substituteAtImports(ast, "/root", makeLoader(files));
  const state = collectDirectives(ast, theme);
  return { theme, ast, state };
}

Deno.test("import parameters: reference", async () => {
  const { theme, ast } = await importAndCollect(`@reference "./a.css";`, {
    "/root/a.css": `@theme { --a: 1; } .x { color: red; }`,
  });
  assertEquals(
    theme.getOptions("--a") & ThemeOptions.REFERENCE,
    ThemeOptions.REFERENCE,
  );
  // The @media wrapper is gone; the content sits in a reference context.
  assertEquals(ast, [
    context({ base: "/root" }, [
      context({}, [
        context({ reference: true }, [
          context({ base: "/root" }, [styleRule(".x", [decl("color", "red")])]),
        ]),
      ]),
    ]),
  ]);
});

Deno.test("import parameters: theme(...), prefix(...), important", async () => {
  const { theme, state } = await importAndCollect(
    `@import "./a.css" theme(static) prefix(tw) important;`,
    { "/root/a.css": `@theme { --a: 1; }` },
  );
  assertEquals(theme.getOptions("--a"), ThemeOptions.STATIC);
  assertEquals(theme.prefix, "tw");
  assertEquals(state.important, true);
});

Deno.test("import parameters: theme(reference) rejects non-theme rules", async () => {
  let error: unknown;
  try {
    await importAndCollect(`@import "./a.css" theme(reference);`, {
      "/root/a.css": `@theme { --a: 1; } .x { color: red; }`,
    });
  } catch (e) {
    error = e;
  }
  assertEquals(error instanceof TwillError, true);
});

Deno.test("import parameters: source(...)", async () => {
  const { state } = await importAndCollect(
    `@import "./sub/a.css" source("../app");`,
    {
      "/root/sub/a.css": `@twill utilities;`,
    },
  );
  assertEquals(state.root, { base: "/root", pattern: "../app" });
  assertEquals(state.utilitiesNode?.params, `utilities source("../app")`);
});

Deno.test("import parameters: unconsumed media queries are kept", async () => {
  const { ast } = await importAndCollect(`@import "./a.css" print important;`, {
    "/root/a.css": `.x { color: red; }`,
  });
  assertEquals(
    serialize(ast),
    "@media print {\n  .x {\n    color: red;\n  }\n}\n",
  );
});

Deno.test("built-in theme through @import 'twill'", async () => {
  const { theme, state, ast } = await importAndCollect(`
    @import "twill";
    @theme {
      --color-*: initial;
      --color-primary: oklch(0.6 0.2 250);
      --breakpoint-3xl: 120rem;
    }
  `);
  assertEquals(theme.resolveValue(null, ["--spacing"]), "0.25rem");
  assertEquals(theme.resolveValue("md", ["--breakpoint"]), "48rem");
  assertEquals(theme.resolveValue("3xl", ["--breakpoint"]), "120rem");
  assertEquals(theme.resolveValue("md", ["--container"]), "28rem");
  assertEquals(theme.resolveValue("lg", ["--text"]), "1.125rem");
  assertEquals(theme.resolveValue("bold", ["--font-weight"]), "700");
  assertEquals(theme.resolveValue("lg", ["--radius"]), "0.5rem");
  assertEquals(
    theme.resolveValue(null, ["--default-transition-duration"]),
    "150ms",
  );
  assertEquals(theme.resolveValue("red-500", ["--color"]), null);
  assertEquals(
    theme.resolveValue("primary", ["--color"]),
    "oklch(0.6 0.2 250)",
  );
  assertEquals(theme.getKeyframes().map((k) => k.params), [
    "spin",
    "ping",
    "pulse",
    "bounce",
  ]);
  assertEquals(state.utilitiesNode !== null, true);
  const css = serialize(ast);
  assertEquals(css.includes("@theme"), false);
  assertEquals(css.includes("@layer theme {\n  :root, :host {\n  }\n}"), true);
});

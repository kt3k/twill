import { assertEquals, assertRejects, assertThrows } from "@std/assert";
import { parseImportParams, substituteAtImports } from "./import.ts";
import { parse } from "./parser.ts";
import { serialize } from "./serializer.ts";
import { TwillError } from "./error.ts";
import { Features } from "./features.ts";
import { type LoadedStylesheet, resolveBuiltin } from "./builtin.ts";
import { atRule, context, decl, styleRule } from "./ast.ts";

Deno.test("parseImportParams", () => {
  assertEquals(parseImportParams(`"a.css"`), {
    uri: "a.css",
    layer: null,
    media: null,
    supports: null,
  });
  assertEquals(parseImportParams(`'a.css' layer(base)`), {
    uri: "a.css",
    layer: "base",
    media: null,
    supports: null,
  });
  assertEquals(
    parseImportParams(
      `"a.css" layer(base) supports(display: grid) screen and (min-width: 1px)`,
    ),
    {
      uri: "a.css",
      layer: "base",
      media: "screen and (min-width: 1px)",
      supports: "display: grid",
    },
  );
  assertEquals(
    parseImportParams(`"twill" important theme(reference) prefix(tw)`),
    {
      uri: "twill",
      layer: null,
      media: "important theme(reference) prefix(tw)",
      supports: null,
    },
  );
  assertEquals(parseImportParams(`url(a.css)`), null);
  assertEquals(parseImportParams(`"data:text/css,a"`), null);
  assertEquals(parseImportParams(`"https://example.com/a.css"`), null);
  assertEquals(parseImportParams(`a.css`), null);
  assertThrows(
    () => parseImportParams(`"a.css" supports(x) layer(base)`),
    TwillError,
  );
  assertThrows(
    () => parseImportParams(`"a.css" screen layer(base)`),
    TwillError,
  );
});

function makeLoader(files: Record<string, string>) {
  const loaded: string[] = [];
  const load = (id: string, base: string): LoadedStylesheet => {
    const builtin = resolveBuiltin(id, base);
    if (builtin !== null) return builtin;
    const path = `${base}/${id.replace(/^\.\//, "")}`;
    loaded.push(path);
    if (!(path in files)) throw new TwillError(`Cannot find ${path}`);
    const dir = path.slice(0, path.lastIndexOf("/"));
    return { path, base: dir, content: files[path] };
  };
  return { load, loaded };
}

Deno.test("substituteAtImports: wraps content in context, layer, media, supports", async () => {
  const { load } = makeLoader({ "/root/a.css": ".a { color: red }" });
  const ast = parse(
    `@import "./a.css" layer(base) supports(display: grid) print;`,
  );
  const features = await substituteAtImports(ast, "/root", load);
  assertEquals(features, Features.AT_IMPORT);
  assertEquals(ast, [
    context({}, [
      atRule("@supports", "(display: grid)", [
        atRule("@media", "print", [
          atRule("@layer", "base", [
            context({ base: "/root" }, [
              styleRule(".a", [decl("color", "red")]),
            ]),
          ]),
        ]),
      ]),
    ]),
  ]);
});

Deno.test("substituteAtImports: recursive imports resolve against the imported base", async () => {
  const { load, loaded } = makeLoader({
    "/root/a.css": `@import "./sub/b.css";`,
    "/root/sub/b.css": `@import "./c.css";`,
    "/root/sub/c.css": ".c { color: red }",
  });
  const ast = parse(`@import "./a.css";`);
  await substituteAtImports(ast, "/root", load);
  assertEquals(loaded, ["/root/a.css", "/root/sub/b.css", "/root/sub/c.css"]);
  assertEquals(serialize(ast), ".c {\n  color: red;\n}\n");
});

Deno.test("substituteAtImports: untouched imports", async () => {
  const { load } = makeLoader({});
  const css =
    `@import url(a.css);\n@import "https://example.com/a.css";\n@import "data:text/css,a";\n`;
  const ast = parse(css);
  const features = await substituteAtImports(ast, "/root", load);
  assertEquals(features, Features.NONE);
  assertEquals(serialize(ast), css);
});

Deno.test("substituteAtImports: @reference is @import ... reference", async () => {
  const { load } = makeLoader({ "/root/a.css": ".a { color: red }" });
  const ast = parse(`@reference "./a.css";`);
  await substituteAtImports(ast, "/root", load);
  assertEquals(ast, [
    context({}, [
      atRule("@media", "reference", [
        context({ base: "/root" }, [styleRule(".a", [decl("color", "red")])]),
      ]),
    ]),
  ]);
});

Deno.test("substituteAtImports: recursion limit", async () => {
  const { load } = makeLoader({ "/root/a.css": `@import "./a.css";` });
  const ast = parse(`@import "./a.css";`);
  await assertRejects(
    () => substituteAtImports(ast, "/root", load),
    TwillError,
    "recursion",
  );
});

Deno.test("substituteAtImports: built-in stylesheets", async () => {
  const { load, loaded } = makeLoader({});
  const ast = parse(`@import "twill";`);
  await substituteAtImports(ast, "/root", load);
  // Nothing was loaded from the filesystem.
  assertEquals(loaded, []);
  const css = serialize(ast);
  assertEquals(
    css.startsWith(
      "@layer theme, base, components, utilities;\n@layer theme {\n",
    ),
    true,
  );
  assertEquals(
    css.includes("--color-red-500: oklch(63.7% 0.237 25.331);"),
    true,
  );
  assertEquals(
    css.includes("@layer utilities {\n  @twill utilities;\n}\n"),
    true,
  );
});

Deno.test("resolveBuiltin: user relative imports never resolve to embedded resources", () => {
  assertEquals(resolveBuiltin("./theme.css", "/root"), null);
  assertEquals(
    resolveBuiltin("./theme.css", "twill:")?.path,
    "twill:theme.css",
  );
  assertEquals(
    resolveBuiltin("twill/preflight.css", "/root")?.path,
    "twill:preflight.css",
  );
  assertThrows(() => resolveBuiltin("./nope.css", "twill:"), TwillError);
});

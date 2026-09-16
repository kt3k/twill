import { assertEquals, assertThrows } from "@std/assert";
import { Theme, ThemeOptions } from "./theme.ts";
import { TwillError } from "./error.ts";
import { atRule } from "./ast.ts";

function makeTheme(): Theme {
  const theme = new Theme();
  theme.add("--color-red-500", "red");
  theme.add("--color-blue-500", "blue");
  theme.add("--spacing", "0.25rem");
  theme.add("--text-lg", "1.125rem");
  theme.add("--text-lg--line-height", "calc(1.75 / 1.125)");
  theme.add("--text-shadow-sm", "0 1px 1px black");
  theme.add("--font-sans", "ui-sans-serif");
  theme.add("--font-weight-bold", "700");
  theme.add("--breakpoint-md", "48rem");
  theme.add("--container-1_5", "1.5rem");
  return theme;
}

Deno.test("Theme: resolve returns var() references", () => {
  const theme = makeTheme();
  assertEquals(theme.resolve("red-500", ["--color"]), "var(--color-red-500)");
  assertEquals(theme.resolve(null, ["--spacing"]), "var(--spacing)");
  assertEquals(theme.resolve("missing", ["--color"]), null);
  assertEquals(
    theme.resolve("red-500", ["--text-color", "--color"]),
    "var(--color-red-500)",
  );
  assertEquals(theme.resolveValue("red-500", ["--color"]), "red");
  assertEquals(theme.resolve("1.5", ["--container"]), "var(--container-1_5)");
});

Deno.test("Theme: inline and reference modes", () => {
  const theme = new Theme();
  theme.add("--a", "1", ThemeOptions.INLINE);
  theme.add("--b", "2", ThemeOptions.REFERENCE);
  theme.add("--c", "3");
  assertEquals(theme.resolve(null, ["--a"]), "1");
  assertEquals(theme.resolve(null, ["--b"]), "var(--b, 2)");
  assertEquals(theme.resolve(null, ["--c"]), "var(--c)");
  assertEquals(theme.resolve(null, ["--c"], ThemeOptions.INLINE), "3");
});

Deno.test("Theme: default entries lose to author entries regardless of order", () => {
  const theme = new Theme();
  theme.add("--color-a", "author");
  theme.add("--color-a", "default", ThemeOptions.DEFAULT);
  assertEquals(theme.resolveValue("a", ["--color"]), "author");

  theme.add("--color-b", "default", ThemeOptions.DEFAULT);
  theme.add("--color-b", "author");
  assertEquals(theme.resolveValue("b", ["--color"]), "author");

  theme.add("--color-c", "default-1", ThemeOptions.DEFAULT);
  theme.add("--color-c", "default-2", ThemeOptions.DEFAULT);
  assertEquals(theme.resolveValue("c", ["--color"]), "default-2");
});

Deno.test("Theme: initial deletes, namespace clearing keeps ignored sub-namespaces", () => {
  const theme = makeTheme();
  theme.add("--color-red-500", "initial");
  assertEquals(theme.has("--color-red-500"), false);
  assertEquals(theme.has("--color-blue-500"), true);

  theme.add("--color-*", "initial");
  assertEquals(theme.has("--color-blue-500"), false);
  assertEquals(theme.has("--spacing"), true);

  theme.add("--text-*", "initial");
  assertEquals(theme.has("--text-lg"), false);
  assertEquals(theme.has("--text-lg--line-height"), false);
  assertEquals(theme.has("--text-shadow-sm"), true);

  theme.add("--font-*", "initial");
  assertEquals(theme.has("--font-sans"), false);
  assertEquals(theme.has("--font-weight-bold"), true);

  assertThrows(() => theme.add("--color-*", "red"), TwillError);

  theme.add("--*", "initial");
  assertEquals(theme.size, 0);
});

Deno.test("Theme: ignored sub-namespaces during resolution", () => {
  const theme = makeTheme();
  assertEquals(theme.resolve("shadow-sm", ["--text"]), null);
  assertEquals(theme.resolve("weight-bold", ["--font"]), null);
  assertEquals(
    theme.resolve("bold", ["--font-weight"]),
    "var(--font-weight-bold)",
  );
});

Deno.test("Theme: resolveWith nested keys", () => {
  const theme = makeTheme();
  assertEquals(
    theme.resolveWith("lg", ["--text"], ["--line-height", "--letter-spacing"]),
    [
      "var(--text-lg)",
      { "--line-height": "var(--text-lg--line-height)" },
    ],
  );
  assertEquals(theme.resolveWith("xl", ["--text"], ["--line-height"]), null);
});

Deno.test("Theme: namespace and keysInNamespaces", () => {
  const theme = makeTheme();
  assertEquals(
    theme.namespace("--text"),
    new Map<string | null, string>([
      ["lg", "1.125rem"],
      ["lg--line-height", "calc(1.75 / 1.125)"],
      ["shadow-sm", "0 1px 1px black"],
    ]),
  );
  assertEquals(theme.namespace("--spacing"), new Map([[null, "0.25rem"]]));
  assertEquals(theme.keysInNamespaces(["--text"]), ["lg"]);
  assertEquals(theme.keysInNamespaces(["--color", "--breakpoint"]), [
    "red-500",
    "blue-500",
    "md",
  ]);
});

Deno.test("Theme: prefix", () => {
  const theme = makeTheme();
  theme.prefix = "tw";
  assertEquals(
    theme.resolve("red-500", ["--color"]),
    "var(--tw-color-red-500)",
  );
  assertEquals(theme.prefixKey("--spacing"), "--tw-spacing");
  assertEquals(theme.markUsedVariable("--tw-color-red-500"), true);
  assertEquals(
    theme.getOptions("--color-red-500") & ThemeOptions.USED,
    ThemeOptions.USED,
  );
});

Deno.test("Theme: markUsedVariable", () => {
  const theme = makeTheme();
  assertEquals(theme.markUsedVariable("--color-red-500"), true);
  assertEquals(theme.markUsedVariable("--color-red-500"), false);
  assertEquals(theme.markUsedVariable("--missing"), false);
  assertEquals(theme.markUsedVariable("--container-1\\.5"), false);
  assertEquals(theme.markUsedVariable("--container-1_5"), true);
});

Deno.test("Theme: resolveThemeValue", () => {
  const theme = makeTheme();
  assertEquals(theme.resolveThemeValue("--color-red-500"), "red");
  assertEquals(
    theme.resolveThemeValue("--color-red-500", false),
    "var(--color-red-500)",
  );
  assertEquals(
    theme.resolveThemeValue("--color-red-500/0.5"),
    "color-mix(in oklab, red 50%, transparent)",
  );
  assertEquals(theme.resolveThemeValue("--color-red-500 / 100%"), "red");
  assertEquals(theme.resolveThemeValue("--missing"), null);
});

Deno.test("Theme: get and keyframes", () => {
  const theme = makeTheme();
  assertEquals(theme.get(["--missing", "--spacing"]), "0.25rem");
  assertEquals(theme.get(["--missing"]), null);
  const kf = atRule("@keyframes", "spin", []);
  theme.addKeyframes(kf);
  theme.addKeyframes(kf);
  assertEquals(theme.getKeyframes(), [kf]);
});

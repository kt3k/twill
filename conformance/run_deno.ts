// Runs every conformance case through the Deno implementation and writes
// the output to out/deno/<case>.css.
import { dirname, join, resolve } from "jsr:@std/path@^1";
import { compile } from "../impl/deno/src/compile.ts";
import {
  resolveBuiltin,
  type StylesheetLoader,
} from "../impl/deno/src/builtin.ts";

const here = dirname(new URL(import.meta.url).pathname);
const casesDir = join(here, "cases");
const outDir = join(here, "out", "deno");
await Deno.mkdir(outDir, { recursive: true });

const loader: StylesheetLoader = async (id, base) => {
  const builtin = resolveBuiltin(id, base);
  if (builtin !== null) return builtin;
  const path = resolve(base, id);
  return { path, base: dirname(path), content: await Deno.readTextFile(path) };
};

const names: string[] = [];
for await (const entry of Deno.readDir(casesDir)) {
  if (entry.isDirectory) names.push(entry.name);
}
names.sort();
for (const name of names) {
  const dir = join(casesDir, name);
  const css = await Deno.readTextFile(join(dir, "input.css"));
  const candidates = (await Deno.readTextFile(join(dir, "candidates.txt")))
    .split("\n").filter((l) => l !== "");
  let output: string;
  try {
    const compiler = await compile(css, { base: dir, loadStylesheet: loader });
    output = compiler.build(candidates);
    // Build twice with the same candidates to check caching stability.
    const again = compiler.build([...candidates].reverse());
    if (again !== output) {
      output += "\n/* UNSTABLE: second build differs */\n" + again;
    }
  } catch (error) {
    output = `ERROR: ${(error as Error).message}\n`;
  }
  await Deno.writeTextFile(join(outDir, `${name}.css`), output);
}
console.log(`deno: ${names.length} cases`);

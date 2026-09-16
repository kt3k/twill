import { assertEquals, assertStringIncludes, assertThrows } from "@std/assert";
import { join } from "@std/path";
import { assembleSources, minifyCss, parseCliArgs } from "./cli.ts";

const CLI = new URL("./cli.ts", import.meta.url).pathname;

async function runCli(
  args: string[],
  options: { cwd?: string; stdin?: string } = {},
): Promise<{ code: number; stdout: string; stderr: string }> {
  const command = new Deno.Command(Deno.execPath(), {
    args: ["run", "-A", CLI, ...args],
    cwd: options.cwd,
    stdin: options.stdin === undefined ? "null" : "piped",
    stdout: "piped",
    stderr: "piped",
  });
  const child = command.spawn();
  if (options.stdin !== undefined) {
    const writer = child.stdin.getWriter();
    await writer.write(new TextEncoder().encode(options.stdin));
    await writer.close();
  }
  const output = await child.output();
  return {
    code: output.code,
    stdout: new TextDecoder().decode(output.stdout),
    stderr: new TextDecoder().decode(output.stderr),
  };
}

async function project(): Promise<string> {
  const root = await Deno.makeTempDir({ prefix: "twill-cli-" });
  await Deno.writeTextFile(
    join(root, "input.css"),
    `@import "twill/theme.css";\n@source not "dist";\n@twill utilities;\n`,
  );
  await Deno.writeTextFile(
    join(root, "index.html"),
    `<div class="flex p-4 hover:underline">`,
  );
  await Deno.mkdir(join(root, "dist"));
  await Deno.writeTextFile(
    join(root, "dist", "old.html"),
    `<div class="hidden">`,
  );
  return root;
}

Deno.test("parseCliArgs", () => {
  assertEquals(parseCliArgs([]), {
    input: null,
    output: "-",
    watch: false,
    poll: null,
    minify: false,
    optimize: false,
    cwd: ".",
    silent: false,
    help: false,
  });
  const full = parseCliArgs([
    "build",
    "-i",
    "a.css",
    "-o",
    "b.css",
    "--watch",
    "always",
    "--poll",
    "500",
    "-m",
    "--optimize",
    "--cwd",
    "x",
    "--silent",
  ]);
  assertEquals(full, {
    input: "a.css",
    output: "b.css",
    watch: "always",
    poll: 500,
    minify: true,
    optimize: true,
    cwd: "x",
    silent: true,
    help: false,
  });
  assertEquals(parseCliArgs(["-w", "-i", "a.css"]).watch, true);
  assertEquals(parseCliArgs(["--watch=always"]).watch, "always");
  assertEquals(parseCliArgs(["--poll"]).poll, 250);
  assertEquals(parseCliArgs(["--poll=100"]).poll, 100);
  assertEquals(parseCliArgs(["-h"]).help, true);
  assertThrows(() => parseCliArgs(["--poll=0"]), Error, "positive");
  assertThrows(() => parseCliArgs(["--nope"]), Error, "Unknown option");
});

Deno.test("assembleSources", () => {
  const compiler = {
    sources: [{ base: "/p", pattern: "./src/**/*", negated: false }],
    root: null,
    features: 0,
    designSystem: null as unknown as never,
    build: () => "",
  };
  assertEquals(
    assembleSources(compiler, {
      cwd: "/p",
      inputPath: "/p/in.css",
      executable: "/bin/twill",
    }),
    [
      { base: "/p", pattern: "**/*", negated: false },
      { base: "/p", pattern: "./src/**/*", negated: false },
      { base: "/bin", pattern: "twill", negated: true },
      { base: "/p", pattern: "in.css", negated: false },
    ],
  );
  assertEquals(
    assembleSources({ ...compiler, root: "none", sources: [] }, {
      cwd: "/p",
      inputPath: null,
      executable: null,
    }),
    [],
  );
  assertEquals(
    assembleSources({
      ...compiler,
      root: { base: "/p", pattern: "../app" },
      sources: [],
    }, {
      cwd: "/p",
      inputPath: null,
      executable: null,
    }),
    [{ base: "/p", pattern: "../app", negated: false }],
  );
});

Deno.test("minifyCss", () => {
  assertEquals(
    minifyCss(
      ".a {\n  color: red !important;\n}\n@media (x) {\n  .b {\n    y: z;\n  }\n}\n",
    ),
    ".a{color:red!important;}@media (x){.b{y:z;}}",
  );
});

Deno.test("cli: single build to stdout and to a file", async () => {
  const root = await project();
  try {
    const result = await runCli(["-i", "input.css"], { cwd: root });
    assertEquals(result.code, 0, result.stderr);
    assertStringIncludes(result.stdout, ".flex {\n  display: flex;\n}");
    assertStringIncludes(
      result.stdout,
      ".p-4 {\n  padding: calc(var(--spacing) * 4);\n}",
    );
    assertStringIncludes(result.stdout, ".hover\\:underline:hover");
    assertEquals(result.stdout.includes(".hidden"), false);
    assertStringIncludes(result.stderr, "twill");
    assertStringIncludes(result.stderr, "Done in");

    const toFile = await runCli([
      "-i",
      "input.css",
      "-o",
      "out/build.css",
      "--silent",
    ], { cwd: root });
    assertEquals(toFile.code, 0, toFile.stderr);
    assertEquals(toFile.stdout, "");
    assertEquals(toFile.stderr, "");
    const written = await Deno.readTextFile(join(root, "out", "build.css"));
    assertStringIncludes(written, ".flex {");

    const minified = await runCli(["-i", "input.css", "--minify", "--silent"], {
      cwd: root,
    });
    assertEquals(minified.code, 0, minified.stderr);
    assertStringIncludes(minified.stdout, ".flex{display:flex;}");
  } finally {
    await Deno.remove(root, { recursive: true });
  }
});

Deno.test("cli: stdin input and default input", async () => {
  const root = await project();
  try {
    const result = await runCli(["-i", "-", "--silent"], {
      cwd: root,
      stdin: `@import "twill/theme.css";\n@twill utilities;`,
    });
    assertEquals(result.code, 0, result.stderr);
    assertStringIncludes(result.stdout, ".flex {");

    const defaulted = await runCli(["--silent"], { cwd: root, stdin: "" });
    assertEquals(defaulted.code, 0, defaulted.stderr);
    assertStringIncludes(
      defaulted.stdout,
      "@layer theme, base, components, utilities;",
    );
    assertStringIncludes(defaulted.stdout, ".flex {");
  } finally {
    await Deno.remove(root, { recursive: true });
  }
});

Deno.test("cli: errors exit with status 1", async () => {
  const root = await project();
  try {
    const missing = await runCli(["-i", "nope.css"], { cwd: root });
    assertEquals(missing.code, 1);
    assertStringIncludes(missing.stderr, "does not exist");

    const same = await runCli(["-i", "input.css", "-o", "input.css"], {
      cwd: root,
    });
    assertEquals(same.code, 1);
    assertStringIncludes(same.stderr, "identical");

    await Deno.writeTextFile(join(root, "bad.css"), ".a { color: red;");
    const bad = await runCli(["-i", "bad.css", "--silent"], { cwd: root });
    assertEquals(bad.code, 1);
    assertStringIncludes(bad.stderr, "Missing closing");
  } finally {
    await Deno.remove(root, { recursive: true });
  }
});

Deno.test("cli: --help", async () => {
  const result = await runCli(["--help"]);
  assertEquals(result.code, 0);
  assertStringIncludes(result.stderr, "Usage: twill");
});

async function waitFor(
  check: () => Promise<boolean>,
  timeout = 10000,
): Promise<void> {
  const start = Date.now();
  while (Date.now() - start < timeout) {
    if (await check()) return;
    await new Promise((r) => setTimeout(r, 100));
  }
  throw new Error("Timed out waiting for the watcher");
}

async function fileIncludes(path: string, text: string): Promise<boolean> {
  try {
    return (await Deno.readTextFile(path)).includes(text);
  } catch {
    return false;
  }
}

Deno.test("cli: polling mode rebuilds on changes", async () => {
  const root = await project();
  const output = join(root, "out.css");
  const command = new Deno.Command(Deno.execPath(), {
    args: [
      "run",
      "-A",
      CLI,
      "-i",
      "input.css",
      "-o",
      "out.css",
      "--poll",
      "100",
      "--silent",
    ],
    cwd: root,
    stdin: "null",
    stdout: "null",
    stderr: "piped",
  });
  const child = command.spawn();
  try {
    await waitFor(() => fileIncludes(output, ".flex {"));
    // A source change is picked up incrementally.
    await Deno.writeTextFile(join(root, "page.html"), `<i class="hidden">`);
    await waitFor(() => fileIncludes(output, ".hidden {"));
    // A stylesheet change triggers a full rebuild.
    await Deno.writeTextFile(
      join(root, "input.css"),
      `@import "twill/theme.css";\n@theme { --color-brand: blue; }\n@twill utilities;\n`,
    );
    await Deno.writeTextFile(
      join(root, "page.html"),
      `<i class="hidden bg-brand">`,
    );
    await waitFor(() => fileIncludes(output, ".bg-brand {"));
  } finally {
    child.kill("SIGTERM");
    await child.status;
    await Deno.remove(root, { recursive: true });
  }
});

Deno.test("cli: watch mode rebuilds on changes and exits on stdin EOF", async () => {
  const root = await project();
  const output = join(root, "out.css");
  const command = new Deno.Command(Deno.execPath(), {
    args: [
      "run",
      "-A",
      CLI,
      "-i",
      "input.css",
      "-o",
      "out.css",
      "--watch",
      "--silent",
    ],
    cwd: root,
    stdin: "piped",
    stdout: "null",
    stderr: "piped",
  });
  const child = command.spawn();
  let stderr = "";
  const reading = (async () => {
    for await (const chunk of child.stderr) {
      stderr += new TextDecoder().decode(chunk);
    }
  })();
  try {
    await waitFor(() => fileIncludes(output, ".flex {"));
    await Deno.writeTextFile(join(root, "page.html"), `<i class="hidden">`);
    await waitFor(() => fileIncludes(output, ".hidden {"));
    await Deno.writeTextFile(
      join(root, "input.css"),
      `@import "twill/theme.css";\n@theme static { --color-brand: blue; }\n@twill utilities;\n`,
    );
    await waitFor(() => fileIncludes(output, "--color-brand"));
    // A broken stylesheet reports an error and keeps watching. The file is
    // written twice so that a rebuild sees the broken content even if the
    // first event fired between truncation and write.
    await Deno.writeTextFile(join(root, "input.css"), `.a { color: red;`);
    await new Promise((r) => setTimeout(r, 300));
    await Deno.writeTextFile(join(root, "input.css"), `.a { color: red;`);
    await waitFor(() => Promise.resolve(stderr.includes("Missing closing")));
    await Deno.writeTextFile(
      join(root, "input.css"),
      `@import "twill/theme.css";\n@theme static { --color-other: red; }\n@twill utilities;\n`,
    );
    await waitFor(() => fileIncludes(output, "--color-other"));
  } finally {
    await child.stdin.close();
    const status = await Promise.race([
      child.status,
      new Promise<null>((r) => setTimeout(() => r(null), 5000)),
    ]);
    if (status === null) child.kill("SIGTERM");
    await child.status;
    await reading;
    await Deno.remove(root, { recursive: true });
  }
});

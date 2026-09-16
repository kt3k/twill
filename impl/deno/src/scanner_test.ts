import { assertEquals } from "@std/assert";
import { join } from "@std/path";
import { Scanner } from "./scanner.ts";

async function fixture(): Promise<string> {
  const root = await Deno.makeTempDir({ prefix: "twill-scan-" });
  await Deno.mkdir(join(root, "src", "sub"), { recursive: true });
  await Deno.mkdir(join(root, "node_modules", "pkg"), { recursive: true });
  await Deno.mkdir(join(root, "vendor"), { recursive: true });
  await Deno.mkdir(join(root, "ignored"), { recursive: true });
  await Deno.writeTextFile(join(root, ".gitignore"), "ignored/\n*.tmp\n");
  await Deno.writeTextFile(
    join(root, "index.html"),
    `<div class="flex p-4"></div>`,
  );
  await Deno.writeTextFile(
    join(root, "src", "app.js"),
    `el.className = "hover:underline";`,
  );
  await Deno.writeTextFile(
    join(root, "src", "sub", "page.html"),
    `<p class="md:grid">`,
  );
  await Deno.writeTextFile(join(root, "src", "notes.tmp"), `tmp-only`);
  await Deno.writeTextFile(join(root, "styles.scss"), `.scss-only {}`);
  await Deno.writeTextFile(join(root, "logo.png"), `png-only`);
  await Deno.writeTextFile(join(root, "package-lock.json"), `lock-only`);
  await Deno.writeTextFile(join(root, ".env"), `env-only`);
  await Deno.writeTextFile(
    join(root, "node_modules", "pkg", "x.js"),
    `nm-only`,
  );
  await Deno.writeTextFile(join(root, "vendor", "v.html"), `vendor-only`);
  await Deno.writeTextFile(join(root, "ignored", "i.html"), `ignored-only`);
  return root;
}

Deno.test("scanner: auto-detection respects ignore rules", async () => {
  const root = await fixture();
  try {
    const scanner = new Scanner([{
      base: root,
      pattern: "**/*",
      negated: false,
    }]);
    const candidates = await scanner.scan();
    for (const expected of ["flex", "p-4", "hover:underline", "md:grid"]) {
      assertEquals(candidates.includes(expected), true, expected);
    }
    for (
      const unexpected of [
        "tmp-only",
        "scss-only",
        "png-only",
        "lock-only",
        "env-only",
        "nm-only",
        "ignored-only",
      ]
    ) {
      assertEquals(candidates.includes(unexpected), false, unexpected);
    }
    assertEquals(candidates.includes("vendor-only"), true);
    assertEquals(scanner.scannedFiles.length, 4);
  } finally {
    await Deno.remove(root, { recursive: true });
  }
});

Deno.test("scanner: explicit globs bypass ignore rules; negated sources exclude", async () => {
  const root = await fixture();
  try {
    const scanner = new Scanner([
      { base: root, pattern: "**/*", negated: false },
      { base: root, pattern: "node_modules/**/*.js", negated: false },
      { base: root, pattern: "./ignored/*.html", negated: false },
      { base: root, pattern: "vendor", negated: true },
    ]);
    const candidates = await scanner.scan();
    assertEquals(candidates.includes("nm-only"), true);
    assertEquals(candidates.includes("ignored-only"), true);
    assertEquals(candidates.includes("vendor-only"), false);
    assertEquals(candidates.includes("flex"), true);
  } finally {
    await Deno.remove(root, { recursive: true });
  }
});

Deno.test("scanner: directory and file sources", async () => {
  const root = await fixture();
  try {
    const scanner = new Scanner([
      { base: root, pattern: "src", negated: false },
      { base: root, pattern: "./index.html", negated: false },
      { base: root, pattern: "src/sub/*.html", negated: true },
    ]);
    const candidates = await scanner.scan();
    assertEquals(candidates.includes("hover:underline"), true);
    assertEquals(candidates.includes("flex"), true);
    assertEquals(candidates.includes("md:grid"), false);
    assertEquals(candidates.includes("vendor-only"), false);
  } finally {
    await Deno.remove(root, { recursive: true });
  }
});

Deno.test("scanner: incremental scans and scanFiles", async () => {
  const root = await fixture();
  try {
    const scanner = new Scanner([{
      base: root,
      pattern: "**/*",
      negated: false,
    }]);
    await scanner.scan();
    const again = await scanner.scan();
    assertEquals(scanner.scannedFiles, []);
    assertEquals(again.includes("flex"), true);

    const changed = join(root, "src", "app.js");
    await Deno.writeTextFile(
      changed,
      `el.className = "hover:underline new-one";`,
    );
    await Deno.utime(
      changed,
      new Date(Date.now() + 5000),
      new Date(Date.now() + 5000),
    );
    const third = await scanner.scan();
    assertEquals(scanner.scannedFiles, [changed]);
    assertEquals(third.includes("new-one"), true);

    const fresh = join(root, "fresh.html");
    await Deno.writeTextFile(fresh, `<i class="fresh-one flex">`);
    assertEquals(await scanner.scanFiles([fresh, join(root, "missing.html")]), [
      "fresh-one",
    ]);
    assertEquals(await scanner.scanFiles([fresh]), []);
    assertEquals(scanner.candidates.includes("fresh-one"), true);
  } finally {
    await Deno.remove(root, { recursive: true });
  }
});

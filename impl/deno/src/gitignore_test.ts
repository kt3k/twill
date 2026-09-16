import { assertEquals } from "@std/assert";
import { Gitignore } from "./gitignore.ts";

Deno.test("gitignore: basic patterns", () => {
  const ignore = new Gitignore(`
# comment
*.log
build/
/dist
docs/*.md
!keep.log
**/generated
temp?
  `);
  assertEquals(ignore.ignores("a.log", false), true);
  assertEquals(ignore.ignores("deep/a.log", false), true);
  assertEquals(ignore.ignores("keep.log", false), false);
  assertEquals(ignore.ignores("build", true), true);
  assertEquals(ignore.ignores("build", false), false);
  assertEquals(ignore.ignores("src/build", true), true);
  assertEquals(ignore.ignores("dist", true), true);
  assertEquals(ignore.ignores("src/dist", true), false);
  assertEquals(ignore.ignores("docs/a.md", false), true);
  assertEquals(ignore.ignores("docs/sub/a.md", false), false);
  assertEquals(ignore.ignores("x/generated", true), true);
  assertEquals(ignore.ignores("generated", false), true);
  assertEquals(ignore.ignores("temp1", false), true);
  assertEquals(ignore.ignores("temp12", false), false);
  assertEquals(ignore.ignores("other.txt", false), false);
});

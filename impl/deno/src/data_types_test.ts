import { assertEquals } from "@std/assert";
import { inferDataType } from "./data_types.ts";

Deno.test("inferDataType", () => {
  assertEquals(inferDataType("red", ["color"]), "color");
  assertEquals(inferDataType("#fff", ["color"]), "color");
  assertEquals(inferDataType("#ffff", ["color"]), "color");
  assertEquals(inferDataType("#ggg", ["color"]), null);
  assertEquals(inferDataType("rgb(0 0 0)", ["color"]), "color");
  assertEquals(inferDataType("oklch(0.5 0.1 20)", ["color"]), "color");
  assertEquals(
    inferDataType("color-mix(in oklab, red, blue)", ["color"]),
    "color",
  );
  assertEquals(inferDataType("currentcolor", ["color"]), "color");
  assertEquals(inferDataType("var(--x)", ["color"]), null);

  assertEquals(inferDataType("10px", ["length"]), "length");
  assertEquals(inferDataType("1.5rem", ["length"]), "length");
  assertEquals(inferDataType("0", ["length"]), "length");
  assertEquals(inferDataType("100svh", ["length"]), "length");
  assertEquals(inferDataType("calc(1px + 2px)", ["length"]), "length");
  assertEquals(inferDataType("--spacing(4)", ["length"]), "length");
  assertEquals(inferDataType("10", ["length"]), null);
  assertEquals(inferDataType("10%", ["percentage"]), "percentage");
  assertEquals(inferDataType("10", ["number"]), "number");
  assertEquals(inferDataType("-1.5", ["number"]), "number");
  assertEquals(inferDataType(".5", ["number"]), "number");
  assertEquals(inferDataType("10", ["integer"]), "integer");
  assertEquals(inferDataType("1.5", ["integer"]), null);
  assertEquals(inferDataType("16/9", ["ratio"]), "ratio");
  assertEquals(inferDataType("16 / 9", ["ratio"]), "ratio");
  assertEquals(inferDataType("url(/a.png)", ["url"]), "url");
  assertEquals(inferDataType("linear-gradient(red, blue)", ["image"]), "image");
  assertEquals(
    inferDataType("repeating-radial-gradient(red, blue)", ["image"]),
    "image",
  );
  assertEquals(inferDataType("image-set(a.png 1x)", ["image"]), "image");
  assertEquals(inferDataType("url(/a.png)", ["image"]), null);
  assertEquals(inferDataType("center top", ["position"]), "position");
  assertEquals(inferDataType("10px 20px", ["position"]), "position");
  assertEquals(inferDataType("left 10px top 20px", ["position"]), "position");
  assertEquals(inferDataType("foo", ["position"]), null);
  assertEquals(inferDataType("cover", ["bg-size"]), "bg-size");
  assertEquals(inferDataType("10px auto", ["bg-size"]), "bg-size");
  assertEquals(inferDataType("thin", ["line-width"]), "line-width");
  assertEquals(inferDataType("2px", ["line-width"]), "line-width");
  assertEquals(inferDataType("x-large", ["absolute-size"]), "absolute-size");
  assertEquals(inferDataType("larger", ["relative-size"]), "relative-size");
  assertEquals(inferDataType("Inter", ["family-name"]), "family-name");
  assertEquals(
    inferDataType("'Segoe UI', Roboto", ["family-name"]),
    "family-name",
  );
  assertEquals(inferDataType("700", ["family-name"]), null);
  assertEquals(inferDataType("sans-serif", ["generic-name"]), "generic-name");
  assertEquals(inferDataType("45deg", ["angle"]), "angle");
  assertEquals(inferDataType("0.5turn", ["angle"]), "angle");
  assertEquals(inferDataType("1 0 0", ["vector"]), "vector");
  assertEquals(inferDataType("1 0", ["vector"]), null);

  // The first matching type wins.
  assertEquals(inferDataType("10px", ["color", "length"]), "length");
  assertEquals(
    inferDataType("10px 20px", ["percentage", "position", "bg-size"]),
    "position",
  );
  assertEquals(inferDataType("700", ["number", "family-name"]), "number");
});

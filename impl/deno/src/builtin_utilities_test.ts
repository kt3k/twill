import { assertEquals } from "@std/assert";
import { compileRaw, designSystemFor } from "./testing.ts";
import type { DesignSystem } from "./design_system.ts";
import { escape } from "./utils.ts";

let ds: DesignSystem;

async function setup(): Promise<DesignSystem> {
  ds ??= await designSystemFor();
  return ds;
}

/** Expects `raw` to compile to exactly the given declarations. */
function expectDecls(
  ds: DesignSystem,
  raw: string,
  declarations: string[],
): void {
  const body = declarations.map((d) => `  ${d}`).join("\n");
  assertEquals(compileRaw(ds, raw), `.${escape(raw)} {\n${body}\n}\n`, raw);
}

function expectInvalid(ds: DesignSystem, ...raws: string[]): void {
  for (const raw of raws) assertEquals(compileRaw(ds, raw), "", raw);
}

const BORDER_STYLE_PROPERTY = `@property --tw-border-style {
    syntax: "*";
    inherits: false;
    initial-value: solid;
  }`;

const FONT_WEIGHT_PROPERTY = `@property --tw-font-weight {
    syntax: "*";
    inherits: false;
  }`;

Deno.test("utilities: SPEC §16.2 value forms", async () => {
  const ds = await setup();
  expectDecls(ds, "p-4", ["padding: --spacing(4);"]);
  expectDecls(ds, "p-1", ["padding: --spacing(1);"]);
  expectDecls(ds, "p-0", ["padding: --spacing(0);"]);
  expectDecls(ds, "p-px", ["padding: 1px;"]);
  expectDecls(ds, "-mt-2", ["margin-top: --spacing(-2);"]);
  expectDecls(ds, "w-1/2", ["width: calc(1 / 2 * 100%);"]);
  expectDecls(ds, "w-[13px]", ["width: 13px;"]);
  expectDecls(ds, "w-(--my-w)", ["width: var(--my-w);"]);
  expectDecls(ds, "max-w-md", ["max-width: var(--container-md);"]);
  expectDecls(ds, "bg-red-500", ["background-color: var(--color-red-500);"]);
  expectDecls(ds, "bg-red-500/50", [
    "background-color: color-mix(in oklab, var(--color-red-500) 50%, transparent);",
  ]);
  expectDecls(ds, "bg-[#0088cc]", ["background-color: #0088cc;"]);
  expectDecls(ds, "bg-[url(/a_b.png)]", ["background-image: url(/a_b.png);"]);
  expectDecls(ds, "bg-[length:10px_20px]", ["background-size: 10px 20px;"]);
  expectDecls(ds, "text-lg", [
    "font-size: var(--text-lg);",
    "line-height: var(--tw-leading, var(--text-lg--line-height));",
  ]);
  expectDecls(ds, "text-lg/8", [
    "font-size: var(--text-lg);",
    "line-height: --spacing(8);",
  ]);
  expectDecls(ds, "text-red-500", ["color: var(--color-red-500);"]);
  expectDecls(ds, "font-bold", [
    FONT_WEIGHT_PROPERTY,
    "--tw-font-weight: var(--font-weight-bold);",
    "font-weight: var(--font-weight-bold);",
  ]);
  expectDecls(ds, "rounded-lg", ["border-radius: var(--radius-lg);"]);
  expectDecls(ds, "rounded-full", ["border-radius: calc(infinity * 1px);"]);
  expectDecls(ds, "border", [
    BORDER_STYLE_PROPERTY,
    "border-style: var(--tw-border-style);",
    "border-width: 1px;",
  ]);
  expectDecls(ds, "border-2", [
    BORDER_STYLE_PROPERTY,
    "border-style: var(--tw-border-style);",
    "border-width: 2px;",
  ]);
  expectDecls(ds, "z-10", ["z-index: 10;"]);
  expectDecls(ds, "-z-10", ["z-index: calc(10 * -1);"]);
  expectDecls(ds, "flex-1", ["flex: 1;"]);
  expectDecls(ds, "opacity-50", ["opacity: 50%;"]);
  expectDecls(ds, "[mask-type:luminance]", ["mask-type: luminance;"]);
  expectDecls(ds, "[--my-var:1px]", ["--my-var: 1px;"]);
  expectDecls(ds, "[color:red]/50", [
    "color: color-mix(in oklab, red 50%, transparent);",
  ]);
});

Deno.test("utilities: layout", async () => {
  const ds = await setup();
  expectDecls(ds, "flex", ["display: flex;"]);
  expectDecls(ds, "hidden", ["display: none;"]);
  expectDecls(ds, "absolute", ["position: absolute;"]);
  expectDecls(ds, "invisible", ["visibility: hidden;"]);
  expectDecls(ds, "overflow-x-auto", ["overflow-x: auto;"]);
  expectDecls(ds, "float-start", ["float: inline-start;"]);
  expectDecls(ds, "inset-0", ["inset: --spacing(0);"]);
  expectDecls(ds, "inset-x-4", ["inset-inline: --spacing(4);"]);
  expectDecls(ds, "-inset-1", ["inset: --spacing(-1);"]);
  expectDecls(ds, "top-1/2", ["top: calc(1 / 2 * 100%);"]);
  expectDecls(ds, "top-full", ["top: 100%;"]);
  expectDecls(ds, "-top-full", ["top: -100%;"]);
  expectDecls(ds, "left-auto", ["left: auto;"]);
  expectDecls(ds, "left-[10%]", ["left: 10%;"]);
  expectDecls(ds, "z-auto", ["z-index: auto;"]);
  expectDecls(ds, "order-first", ["order: -9999;"]);
  expectDecls(ds, "order-2", ["order: 2;"]);
  expectDecls(ds, "-order-2", ["order: calc(2 * -1);"]);
  expectInvalid(
    ds,
    "z-1.5",
    "z-10/50",
    "p",
    "p-foo",
    "p-4/2",
    "inset-x",
    "overflow-nope",
  );
});

Deno.test("utilities: flexbox and grid", async () => {
  const ds = await setup();
  expectDecls(ds, "flex-col", ["flex-direction: column;"]);
  expectDecls(ds, "flex-wrap", ["flex-wrap: wrap;"]);
  expectDecls(ds, "flex-auto", ["flex: auto;"]);
  expectDecls(ds, "flex-initial", ["flex: 0 auto;"]);
  expectDecls(ds, "flex-1/2", ["flex: calc(1 / 2 * 100%);"]);
  expectDecls(ds, "flex-[2_2_0%]", ["flex: 2 2 0%;"]);
  expectDecls(ds, "grow", ["flex-grow: 1;"]);
  expectDecls(ds, "grow-0", ["flex-grow: 0;"]);
  expectDecls(ds, "shrink", ["flex-shrink: 1;"]);
  expectDecls(ds, "basis-1/3", ["flex-basis: calc(1 / 3 * 100%);"]);
  expectDecls(ds, "basis-md", ["flex-basis: var(--container-md);"]);
  expectDecls(ds, "basis-4", ["flex-basis: --spacing(4);"]);
  expectDecls(ds, "grid-cols-3", [
    "grid-template-columns: repeat(3, minmax(0, 1fr));",
  ]);
  expectDecls(ds, "grid-cols-none", ["grid-template-columns: none;"]);
  expectDecls(ds, "grid-cols-[1fr_2fr]", ["grid-template-columns: 1fr 2fr;"]);
  expectDecls(ds, "grid-rows-subgrid", ["grid-template-rows: subgrid;"]);
  expectDecls(ds, "col-span-2", ["grid-column: span 2 / span 2;"]);
  expectDecls(ds, "col-span-full", ["grid-column: 1 / -1;"]);
  expectDecls(ds, "col-start-1", ["grid-column-start: 1;"]);
  expectDecls(ds, "-col-end-1", ["grid-column-end: calc(1 * -1);"]);
  expectDecls(ds, "row-3", ["grid-row: 3;"]);
  expectDecls(ds, "grid-flow-row-dense", ["grid-auto-flow: row dense;"]);
  expectDecls(ds, "auto-cols-fr", ["grid-auto-columns: minmax(0, 1fr);"]);
  expectDecls(ds, "gap-4", ["gap: --spacing(4);"]);
  expectDecls(ds, "gap-x-2", ["column-gap: --spacing(2);"]);
  expectDecls(ds, "gap-y-px", ["row-gap: 1px;"]);
  expectDecls(ds, "justify-between", ["justify-content: space-between;"]);
  expectDecls(ds, "justify-items-center", ["justify-items: center;"]);
  expectDecls(ds, "justify-self-end", ["justify-self: end;"]);
  expectDecls(ds, "items-start", ["align-items: flex-start;"]);
  expectDecls(ds, "content-evenly", ["align-content: space-evenly;"]);
  expectDecls(ds, "self-center", ["align-self: center;"]);
  expectDecls(ds, "place-content-between", ["place-content: space-between;"]);
  expectDecls(ds, "place-content-start", ["place-content: start;"]);
  expectDecls(ds, "place-items-center", ["place-items: center;"]);
  expectDecls(ds, "place-self-auto", ["place-self: auto;"]);
  expectInvalid(ds, "flex-1.5", "grow-x", "col-span-x", "grid-cols-0.5");
});

Deno.test("utilities: spacing", async () => {
  const ds = await setup();
  expectDecls(ds, "px-4", ["padding-inline: --spacing(4);"]);
  expectDecls(ds, "py-2", ["padding-block: --spacing(2);"]);
  expectDecls(ds, "ps-1", ["padding-inline-start: --spacing(1);"]);
  expectDecls(ds, "pt-0.5", ["padding-top: --spacing(0.5);"]);
  expectDecls(ds, "pl-[3px]", ["padding-left: 3px;"]);
  expectDecls(ds, "m-auto", ["margin: auto;"]);
  expectDecls(ds, "mx-auto", ["margin-inline: auto;"]);
  expectDecls(ds, "-mx-px", ["margin-inline: -1px;"]);
  expectDecls(ds, "-m-[2px]", ["margin: calc(2px * -1);"]);
  expectDecls(ds, "mb-3", ["margin-bottom: --spacing(3);"]);
  expectInvalid(ds, "-p-4", "p-0.3", "m-4/2");
});

Deno.test("utilities: sizing", async () => {
  const ds = await setup();
  expectDecls(ds, "w-full", ["width: 100%;"]);
  expectDecls(ds, "w-screen", ["width: 100vw;"]);
  expectDecls(ds, "w-fit", ["width: fit-content;"]);
  expectDecls(ds, "w-4", ["width: --spacing(4);"]);
  expectDecls(ds, "w-xl", ["width: var(--container-xl);"]);
  expectDecls(ds, "min-w-0", ["min-width: --spacing(0);"]);
  expectDecls(ds, "max-w-none", ["max-width: none;"]);
  expectDecls(ds, "max-w-full", ["max-width: 100%;"]);
  expectDecls(ds, "h-screen", ["height: 100vh;"]);
  expectDecls(ds, "h-dvh", ["height: 100dvh;"]);
  expectDecls(ds, "h-1/3", ["height: calc(1 / 3 * 100%);"]);
  expectDecls(ds, "min-h-full", ["min-height: 100%;"]);
  expectDecls(ds, "max-h-[50vh]", ["max-height: 50vh;"]);
  expectDecls(ds, "size-4", [
    "--tw-sort: size;",
    "width: --spacing(4);",
    "height: --spacing(4);",
  ]);
  expectDecls(ds, "size-full", [
    "--tw-sort: size;",
    "width: 100%;",
    "height: 100%;",
  ]);
  expectInvalid(ds, "w-1/1.5", "w-4/foo", "h-md");
});

Deno.test("utilities: typography", async () => {
  const ds = await setup();
  assertEquals(
    compileRaw(ds, "font-sans").includes("font-family: var(--font-sans);"),
    true,
  );
  expectDecls(ds, "font-[Inter]", ["font-family: Inter;"]);
  expectDecls(ds, "font-[700]", [
    FONT_WEIGHT_PROPERTY,
    "--tw-font-weight: 700;",
    "font-weight: 700;",
  ]);
  expectDecls(ds, "font-[family-name:foo]", ["font-family: foo;"]);
  expectDecls(ds, "text-center", ["text-align: center;"]);
  expectDecls(ds, "text-[14px]", ["font-size: 14px;"]);
  expectDecls(ds, "text-[14px]/6", [
    "font-size: 14px;",
    "line-height: --spacing(6);",
  ]);
  expectDecls(ds, "text-[red]", ["color: red;"]);
  expectDecls(ds, "text-[color:var(--x)]", ["color: var(--x);"]);
  expectDecls(ds, "text-lg/tight", [
    "font-size: var(--text-lg);",
    "line-height: var(--leading-tight);",
  ]);
  expectDecls(ds, "text-lg/none", [
    "font-size: var(--text-lg);",
    "line-height: 1;",
  ]);
  expectDecls(ds, "text-lg/[1.2]", [
    "font-size: var(--text-lg);",
    "line-height: 1.2;",
  ]);
  expectDecls(ds, "text-red-500/50", [
    "color: color-mix(in oklab, var(--color-red-500) 50%, transparent);",
  ]);
  expectDecls(ds, "text-current", ["color: currentcolor;"]);
  expectDecls(ds, "text-transparent", ["color: transparent;"]);
  expectDecls(ds, "leading-none", [
    `@property --tw-leading {
    syntax: "*";
    inherits: false;
  }`,
    "--tw-leading: 1;",
    "line-height: 1;",
  ]);
  expectDecls(ds, "leading-6", [
    `@property --tw-leading {
    syntax: "*";
    inherits: false;
  }`,
    "--tw-leading: --spacing(6);",
    "line-height: --spacing(6);",
  ]);
  expectDecls(ds, "tracking-tight", [
    `@property --tw-tracking {
    syntax: "*";
    inherits: false;
  }`,
    "--tw-tracking: var(--tracking-tight);",
    "letter-spacing: var(--tracking-tight);",
  ]);
  expectDecls(ds, "-tracking-tight", [
    `@property --tw-tracking {
    syntax: "*";
    inherits: false;
  }`,
    "--tw-tracking: calc(var(--tracking-tight) * -1);",
    "letter-spacing: calc(var(--tracking-tight) * -1);",
  ]);
  expectDecls(ds, "uppercase", ["text-transform: uppercase;"]);
  expectDecls(ds, "italic", ["font-style: italic;"]);
  expectDecls(ds, "underline", ["text-decoration-line: underline;"]);
  expectDecls(ds, "truncate", [
    "overflow: hidden;",
    "text-overflow: ellipsis;",
    "white-space: nowrap;",
  ]);
  expectDecls(ds, "whitespace-pre-wrap", ["white-space: pre-wrap;"]);
  expectDecls(ds, "break-all", ["word-break: break-all;"]);
  expectDecls(ds, "list-disc", ["list-style-type: disc;"]);
  expectDecls(ds, "antialiased", [
    "-webkit-font-smoothing: antialiased;",
    "-moz-osx-font-smoothing: grayscale;",
  ]);
  expectDecls(ds, "underline-offset-2", ["text-underline-offset: 2px;"]);
  expectDecls(ds, "-underline-offset-2", ["text-underline-offset: -2px;"]);
  expectDecls(ds, "underline-offset-auto", ["text-underline-offset: auto;"]);
  expectDecls(ds, "indent-4", ["text-indent: --spacing(4);"]);
  expectDecls(ds, "-indent-4", ["text-indent: --spacing(-4);"]);
  expectInvalid(
    ds,
    "text-lg/nope",
    "text-nope",
    "font-nope",
    "font-bold/50",
    "text-center/50",
  );
});

Deno.test("utilities: backgrounds", async () => {
  const ds = await setup();
  expectDecls(ds, "bg-[10px_20px]", ["background-position: 10px 20px;"]);
  expectDecls(ds, "bg-[50%]", ["background-position: 50%;"]);
  expectDecls(ds, "bg-[center_top]", ["background-position: center top;"]);
  expectDecls(ds, "bg-[cover]", ["background-size: cover;"]);
  expectDecls(ds, "bg-[linear-gradient(red,blue)]", [
    "background-image: linear-gradient(red,blue);",
  ]);
  expectDecls(ds, "bg-[color:var(--x)]", ["background-color: var(--x);"]);
  expectDecls(ds, "bg-[var(--x)]", ["background-color: var(--x);"]);
  expectDecls(ds, "bg-[var(--x)]/50", [
    "background-color: color-mix(in oklab, var(--x) 50%, transparent);",
  ]);
  expectDecls(ds, "bg-[#0088cc]/[0.3]", [
    "background-color: color-mix(in oklab, #0088cc 30%, transparent);",
  ]);
  expectDecls(ds, "bg-current", ["background-color: currentcolor;"]);
  expectDecls(ds, "bg-cover", ["background-size: cover;"]);
  expectDecls(ds, "bg-fixed", ["background-attachment: fixed;"]);
  expectDecls(ds, "bg-center", ["background-position: center;"]);
  expectDecls(ds, "bg-no-repeat", ["background-repeat: no-repeat;"]);
  expectDecls(ds, "bg-none", ["background-image: none;"]);
  expectDecls(ds, "bg-clip-text", ["background-clip: text;"]);
  expectDecls(ds, "bg-clip-padding", ["background-clip: padding-box;"]);
  expectDecls(ds, "bg-origin-content", ["background-origin: content-box;"]);
  expectInvalid(ds, "bg-[10px_20px]/50", "bg-red-500/30.3", "bg-nope", "bg");
});

Deno.test("utilities: borders", async () => {
  const ds = await setup();
  expectDecls(ds, "border-red-500", ["border-color: var(--color-red-500);"]);
  expectDecls(ds, "border-t-red-500/50", [
    "border-top-color: color-mix(in oklab, var(--color-red-500) 50%, transparent);",
  ]);
  expectDecls(ds, "border-t-2", [
    BORDER_STYLE_PROPERTY,
    "border-style: var(--tw-border-style);",
    "border-top-width: 2px;",
  ]);
  expectDecls(ds, "border-x", [
    BORDER_STYLE_PROPERTY,
    "border-style: var(--tw-border-style);",
    "border-inline-width: 1px;",
  ]);
  expectDecls(ds, "border-[3px]", [
    BORDER_STYLE_PROPERTY,
    "border-style: var(--tw-border-style);",
    "border-width: 3px;",
  ]);
  expectDecls(ds, "border-[thin]", [
    BORDER_STYLE_PROPERTY,
    "border-style: var(--tw-border-style);",
    "border-width: thin;",
  ]);
  expectDecls(ds, "border-[#fff]", ["border-color: #fff;"]);
  expectDecls(ds, "border-[length:var(--w)]", [
    BORDER_STYLE_PROPERTY,
    "border-style: var(--tw-border-style);",
    "border-width: var(--w);",
  ]);
  expectDecls(ds, "border-dashed", [
    "--tw-border-style: dashed;",
    "border-style: dashed;",
  ]);
  expectDecls(ds, "border-none", [
    "--tw-border-style: none;",
    "border-style: none;",
  ]);
  expectDecls(ds, "rounded-t-lg", [
    "border-top-left-radius: var(--radius-lg);",
    "border-top-right-radius: var(--radius-lg);",
  ]);
  expectDecls(ds, "rounded-ss-none", ["border-start-start-radius: 0;"]);
  expectDecls(ds, "rounded-[4px]", ["border-radius: 4px;"]);
  expectDecls(ds, "rounded", ["border-radius: 0.25rem;"]);
  expectInvalid(ds, "border-2/50", "border-nope", "border/50");
});

Deno.test("utilities: default border width from the theme", async () => {
  const ds = await designSystemFor(
    '@import "twill"; @theme { --default-border-width: 2px; }',
  );
  expectDecls(ds, "border", [
    BORDER_STYLE_PROPERTY,
    "border-style: var(--tw-border-style);",
    "border-width: 2px;",
  ]);
});

Deno.test("utilities: effects, transitions, interactivity", async () => {
  const ds = await setup();
  expectDecls(ds, "opacity-[.5]", ["opacity: .5;"]);
  expectDecls(ds, "opacity-75", ["opacity: 75%;"]);
  expectInvalid(ds, "opacity-33.3", "opacity-50/50");
  assertEquals(
    compileRaw(ds, "transition").includes(
      "transition-property: color, background-color, border-color, outline-color, text-decoration-color, fill, stroke, --tw-gradient-from, --tw-gradient-via, --tw-gradient-to, opacity, box-shadow, transform, translate, scale, rotate, filter, -webkit-backdrop-filter, backdrop-filter, display, content-visibility, overlay, pointer-events;",
    ),
    true,
  );
  expectDecls(ds, "transition-none", ["transition-property: none;"]);
  expectDecls(ds, "transition-opacity", [
    "transition-property: opacity;",
    "transition-timing-function: var(--default-transition-timing-function);",
    "transition-duration: var(--default-transition-duration);",
  ]);
  expectDecls(ds, "duration-300", ["transition-duration: 300ms;"]);
  expectDecls(ds, "delay-[1s]", ["transition-delay: 1s;"]);
  expectDecls(ds, "ease-in-out", [
    "transition-timing-function: var(--ease-in-out);",
  ]);
  expectDecls(ds, "ease-linear", ["transition-timing-function: linear;"]);
  expectDecls(ds, "animate-spin", ["animation: var(--animate-spin);"]);
  expectDecls(ds, "animate-none", ["animation: none;"]);
  expectDecls(ds, "cursor-pointer", ["cursor: pointer;"]);
  expectDecls(ds, "cursor-[grab]", ["cursor: grab;"]);
  expectDecls(ds, "select-none", [
    "-webkit-user-select: none;",
    "user-select: none;",
  ]);
  expectDecls(ds, "pointer-events-none", ["pointer-events: none;"]);
  expectDecls(ds, "resize", ["resize: both;"]);
  expectDecls(ds, "resize-y", ["resize: vertical;"]);
  expectDecls(ds, "appearance-none", ["appearance: none;"]);
  expectDecls(ds, "scroll-smooth", ["scroll-behavior: smooth;"]);
  expectDecls(ds, "will-change-transform", ["will-change: transform;"]);
  expectDecls(ds, "will-change-[top]", ["will-change: top;"]);
  expectDecls(ds, "content-['hi']", [
    `@property --tw-content {
    syntax: "*";
    inherits: false;
    initial-value: "";
  }`,
    "--tw-content: 'hi';",
    "content: var(--tw-content);",
  ]);
  expectDecls(ds, "aspect-square", ["aspect-ratio: 1 / 1;"]);
  expectDecls(ds, "aspect-video", ["aspect-ratio: var(--aspect-video);"]);
  expectDecls(ds, "aspect-16/9", ["aspect-ratio: 16 / 9;"]);
  expectDecls(ds, "aspect-[4/3]", ["aspect-ratio: 4/3;"]);
  expectDecls(ds, "columns-3", ["columns: 3;"]);
  expectDecls(ds, "columns-md", ["columns: var(--container-md);"]);
  expectDecls(ds, "object-cover", ["object-fit: cover;"]);
  expectDecls(ds, "accent-red-500", ["accent-color: var(--color-red-500);"]);
  expectDecls(ds, "caret-[#fff]", ["caret-color: #fff;"]);
  expectDecls(ds, "fill-current", ["fill: currentcolor;"]);
  expectDecls(ds, "stroke-red-500/50", [
    "stroke: color-mix(in oklab, var(--color-red-500) 50%, transparent);",
  ]);
  expectInvalid(
    ds,
    "content-hi",
    "aspect-16/x",
    "columns-1.5",
    "fill",
    "accent-nope",
  );
});

Deno.test("utilities: opacity modifier through the --opacity namespace", async () => {
  const ds = await designSystemFor(
    '@import "twill"; @theme { --opacity-half: 50%; }',
  );
  expectDecls(ds, "bg-red-500/half", [
    "background-color: color-mix(in oklab, var(--color-red-500) var(--opacity-half), transparent);",
  ]);
});

Deno.test("utilities: theme customization (SPEC §16.5)", async () => {
  const ds = await designSystemFor(`
    @import "twill";
    @theme {
      --color-*: initial;
      --color-primary: oklch(0.6 0.2 250);
      --breakpoint-3xl: 120rem;
      --font-display: "Inter", sans-serif;
    }
  `);
  expectDecls(ds, "bg-primary", ["background-color: var(--color-primary);"]);
  expectDecls(ds, "font-display", ["font-family: var(--font-display);"]);
  assertEquals(
    compileRaw(ds, "3xl:flex"),
    ".\\33 xl\\:flex {\n  @media (width >= 120rem) {\n    display: flex;\n  }\n}\n",
  );
  expectInvalid(ds, "bg-red-500");
});

Deno.test("utilities: multiple interpretations emit every match", async () => {
  const ds = await setup();
  // `border-t-2` matches `border-t` + `2`; `border` + `t-2` produces nothing.
  expectDecls(ds, "border-t-2", [
    BORDER_STYLE_PROPERTY,
    "border-style: var(--tw-border-style);",
    "border-top-width: 2px;",
  ]);
});

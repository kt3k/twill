open Twill
open Harness
open Testing

let border_style_property = "@property --tw-border-style {\n    syntax: \"*\";\n    inherits: false;\n    initial-value: solid;\n  }"
let font_weight_property = "@property --tw-font-weight {\n    syntax: \"*\";\n    inherits: false;\n  }"
let leading_property = "@property --tw-leading {\n    syntax: \"*\";\n    inherits: false;\n  }"
let tracking_property = "@property --tw-tracking {\n    syntax: \"*\";\n    inherits: false;\n  }"

let reg ?(syntax = "*") name initial =
  let out = "@property " ^ name ^ " {\n    syntax: \"" ^ syntax ^ "\";\n    inherits: false;\n" in
  let out = if initial <> "" then out ^ "    initial-value: " ^ initial ^ ";\n" else out in
  out ^ "  }"

let registrations names initial = List.map (fun n -> reg n initial) names

let spec_examples () =
  let ds = design_system_for "" in
  let d = expect_decls ds in
  d "p-4" [ "padding: --spacing(4);" ];
  d "p-1" [ "padding: --spacing(1);" ];
  d "p-0" [ "padding: --spacing(0);" ];
  d "p-px" [ "padding: 1px;" ];
  d "-mt-2" [ "margin-top: --spacing(-2);" ];
  d "w-1/2" [ "width: calc(1 / 2 * 100%);" ];
  d "w-[13px]" [ "width: 13px;" ];
  d "w-(--my-w)" [ "width: var(--my-w);" ];
  d "max-w-md" [ "max-width: var(--container-md);" ];
  d "bg-red-500" [ "background-color: var(--color-red-500);" ];
  d "bg-red-500/50" [ "background-color: color-mix(in oklab, var(--color-red-500) 50%, transparent);" ];
  d "bg-[#0088cc]" [ "background-color: #0088cc;" ];
  d "bg-[url(/a_b.png)]" [ "background-image: url(/a_b.png);" ];
  d "bg-[length:10px_20px]" [ "background-size: 10px 20px;" ];
  d "text-lg" [ "font-size: var(--text-lg);"; "line-height: var(--tw-leading, var(--text-lg--line-height));" ];
  d "text-lg/8" [ "font-size: var(--text-lg);"; "line-height: --spacing(8);" ];
  d "text-red-500" [ "color: var(--color-red-500);" ];
  d "font-bold" [ font_weight_property; "--tw-font-weight: var(--font-weight-bold);"; "font-weight: var(--font-weight-bold);" ];
  d "rounded-lg" [ "border-radius: var(--radius-lg);" ];
  d "rounded-full" [ "border-radius: calc(infinity * 1px);" ];
  d "border" [ border_style_property; "border-style: var(--tw-border-style);"; "border-width: 1px;" ];
  d "border-2" [ border_style_property; "border-style: var(--tw-border-style);"; "border-width: 2px;" ];
  d "z-10" [ "z-index: 10;" ];
  d "-z-10" [ "z-index: calc(10 * -1);" ];
  d "flex-1" [ "flex: 1;" ];
  d "opacity-50" [ "opacity: 50%;" ];
  d "[mask-type:luminance]" [ "mask-type: luminance;" ];
  d "[--my-var:1px]" [ "--my-var: 1px;" ];
  d "[color:red]/50" [ "color: color-mix(in oklab, red 50%, transparent);" ]

let layout_flex_grid () =
  let ds = design_system_for "" in
  let d = expect_decls ds in
  d "flex" [ "display: flex;" ];
  d "hidden" [ "display: none;" ];
  d "absolute" [ "position: absolute;" ];
  d "invisible" [ "visibility: hidden;" ];
  d "overflow-x-auto" [ "overflow-x: auto;" ];
  d "float-start" [ "float: inline-start;" ];
  d "inset-x-4" [ "inset-inline: --spacing(4);" ];
  d "-inset-1" [ "inset: --spacing(-1);" ];
  d "top-1/2" [ "top: calc(1 / 2 * 100%);" ];
  d "-top-full" [ "top: -100%;" ];
  d "left-[10%]" [ "left: 10%;" ];
  d "order-first" [ "order: -9999;" ];
  d "-order-2" [ "order: calc(2 * -1);" ];
  expect_invalid ds [ "z-1.5"; "z-10/50"; "p"; "p-foo"; "p-4/2"; "inset-x"; "overflow-nope" ];
  d "flex-col" [ "flex-direction: column;" ];
  d "flex-initial" [ "flex: 0 auto;" ];
  d "flex-1/2" [ "flex: calc(1 / 2 * 100%);" ];
  d "flex-[2_2_0%]" [ "flex: 2 2 0%;" ];
  d "grow" [ "flex-grow: 1;" ];
  d "grow-0" [ "flex-grow: 0;" ];
  d "basis-1/3" [ "flex-basis: calc(1 / 3 * 100%);" ];
  d "basis-md" [ "flex-basis: var(--container-md);" ];
  d "grid-cols-3" [ "grid-template-columns: repeat(3, minmax(0, 1fr));" ];
  d "grid-cols-[1fr_2fr]" [ "grid-template-columns: 1fr 2fr;" ];
  d "col-span-2" [ "grid-column: span 2 / span 2;" ];
  d "col-span-full" [ "grid-column: 1 / -1;" ];
  d "-col-end-1" [ "grid-column-end: calc(1 * -1);" ];
  d "row-3" [ "grid-row: 3;" ];
  d "auto-cols-fr" [ "grid-auto-columns: minmax(0, 1fr);" ];
  d "gap-x-2" [ "column-gap: --spacing(2);" ];
  d "gap-y-px" [ "row-gap: 1px;" ];
  d "justify-between" [ "justify-content: space-between;" ];
  d "items-start" [ "align-items: flex-start;" ];
  d "content-evenly" [ "align-content: space-evenly;" ];
  d "place-content-start" [ "place-content: start;" ];
  d "place-self-auto" [ "place-self: auto;" ];
  expect_invalid ds [ "flex-1.5"; "grow-x"; "col-span-x"; "grid-cols-0.5" ]

let spacing_sizing () =
  let ds = design_system_for "" in
  let d = expect_decls ds in
  d "px-4" [ "padding-inline: --spacing(4);" ];
  d "pt-0.5" [ "padding-top: --spacing(0.5);" ];
  d "pl-[3px]" [ "padding-left: 3px;" ];
  d "mx-auto" [ "margin-inline: auto;" ];
  d "-mx-px" [ "margin-inline: -1px;" ];
  d "-m-[2px]" [ "margin: calc(2px * -1);" ];
  expect_invalid ds [ "-p-4"; "p-0.3"; "m-4/2" ];
  d "w-full" [ "width: 100%;" ];
  d "w-screen" [ "width: 100vw;" ];
  d "w-xl" [ "width: var(--container-xl);" ];
  d "max-w-none" [ "max-width: none;" ];
  d "h-dvh" [ "height: 100dvh;" ];
  d "h-1/3" [ "height: calc(1 / 3 * 100%);" ];
  d "max-h-[50vh]" [ "max-height: 50vh;" ];
  d "size-4" [ "--tw-sort: size;"; "width: --spacing(4);"; "height: --spacing(4);" ];
  d "size-full" [ "--tw-sort: size;"; "width: 100%;"; "height: 100%;" ];
  expect_invalid ds [ "w-1/1.5"; "w-4/foo"; "h-md" ]

let typography () =
  let ds = design_system_for "" in
  let d = expect_decls ds in
  includes "font-sans" (compile_raw ds "font-sans") "font-family: var(--font-sans);";
  d "font-[Inter]" [ "font-family: Inter;" ];
  d "font-[700]" [ font_weight_property; "--tw-font-weight: 700;"; "font-weight: 700;" ];
  d "font-[family-name:foo]" [ "font-family: foo;" ];
  d "text-center" [ "text-align: center;" ];
  d "text-[14px]" [ "font-size: 14px;" ];
  d "text-[14px]/6" [ "font-size: 14px;"; "line-height: --spacing(6);" ];
  d "text-[red]" [ "color: red;" ];
  d "text-[color:var(--x)]" [ "color: var(--x);" ];
  d "text-lg/tight" [ "font-size: var(--text-lg);"; "line-height: var(--leading-tight);" ];
  d "text-lg/none" [ "font-size: var(--text-lg);"; "line-height: 1;" ];
  d "text-lg/[1.2]" [ "font-size: var(--text-lg);"; "line-height: 1.2;" ];
  d "text-current" [ "color: currentcolor;" ];
  d "leading-none" [ leading_property; "--tw-leading: 1;"; "line-height: 1;" ];
  d "leading-6" [ leading_property; "--tw-leading: --spacing(6);"; "line-height: --spacing(6);" ];
  d "tracking-tight" [ tracking_property; "--tw-tracking: var(--tracking-tight);"; "letter-spacing: var(--tracking-tight);" ];
  d "-tracking-tight"
    [ tracking_property; "--tw-tracking: calc(var(--tracking-tight) * -1);"; "letter-spacing: calc(var(--tracking-tight) * -1);" ];
  d "truncate" [ "overflow: hidden;"; "text-overflow: ellipsis;"; "white-space: nowrap;" ];
  d "break-all" [ "word-break: break-all;" ];
  d "antialiased" [ "-webkit-font-smoothing: antialiased;"; "-moz-osx-font-smoothing: grayscale;" ];
  d "underline-offset-2" [ "text-underline-offset: 2px;" ];
  d "-underline-offset-2" [ "text-underline-offset: -2px;" ];
  d "-indent-4" [ "text-indent: --spacing(-4);" ];
  expect_invalid ds [ "text-lg/nope"; "text-nope"; "font-nope"; "font-bold/50"; "text-center/50" ]

let backgrounds_borders () =
  let ds = design_system_for "" in
  let d = expect_decls ds in
  d "bg-[10px_20px]" [ "background-position: 10px 20px;" ];
  d "bg-[50%]" [ "background-position: 50%;" ];
  d "bg-[cover]" [ "background-size: cover;" ];
  d "bg-[linear-gradient(red,blue)]" [ "background-image: linear-gradient(red,blue);" ];
  d "bg-[var(--x)]/50" [ "background-color: color-mix(in oklab, var(--x) 50%, transparent);" ];
  d "bg-[#0088cc]/[0.3]" [ "background-color: color-mix(in oklab, #0088cc 30%, transparent);" ];
  d "bg-current" [ "background-color: currentcolor;" ];
  d "bg-no-repeat" [ "background-repeat: no-repeat;" ];
  d "bg-clip-text" [ "background-clip: text;" ];
  d "bg-origin-content" [ "background-origin: content-box;" ];
  expect_invalid ds [ "bg-[10px_20px]/50"; "bg-red-500/30.3"; "bg-nope"; "bg" ];
  d "border-red-500" [ "border-color: var(--color-red-500);" ];
  d "border-t-red-500/50" [ "border-top-color: color-mix(in oklab, var(--color-red-500) 50%, transparent);" ];
  d "border-t-2" [ border_style_property; "border-style: var(--tw-border-style);"; "border-top-width: 2px;" ];
  d "border-x" [ border_style_property; "border-style: var(--tw-border-style);"; "border-inline-width: 1px;" ];
  d "border-[thin]" [ border_style_property; "border-style: var(--tw-border-style);"; "border-width: thin;" ];
  d "border-[#fff]" [ "border-color: #fff;" ];
  d "border-[length:var(--w)]" [ border_style_property; "border-style: var(--tw-border-style);"; "border-width: var(--w);" ];
  d "border-dashed" [ "--tw-border-style: dashed;"; "border-style: dashed;" ];
  d "rounded-t-lg" [ "border-top-left-radius: var(--radius-lg);"; "border-top-right-radius: var(--radius-lg);" ];
  d "rounded-ss-none" [ "border-start-start-radius: 0;" ];
  d "rounded" [ "border-radius: 0.25rem;" ];
  expect_invalid ds [ "border-2/50"; "border-nope"; "border/50" ];
  let custom = design_system_for "@import \"twill\"; @theme { --default-border-width: 2px; --opacity-half: 50%; }" in
  expect_decls custom "border" [ border_style_property; "border-style: var(--tw-border-style);"; "border-width: 2px;" ];
  expect_decls custom "bg-red-500/half"
    [ "background-color: color-mix(in oklab, var(--color-red-500) var(--opacity-half), transparent);" ]

let shadow_registrations () =
  List.map
    (fun (n, v) -> reg n v)
    [ ("--tw-shadow", "0 0 #0000"); ("--tw-shadow-color", ""); ("--tw-inset-shadow", "0 0 #0000"); ("--tw-inset-shadow-color", "");
      ("--tw-ring-color", ""); ("--tw-ring-shadow", "0 0 #0000"); ("--tw-inset-ring-color", ""); ("--tw-inset-ring-shadow", "0 0 #0000");
      ("--tw-ring-inset", ""); ("--tw-ring-offset-width", "0px"); ("--tw-ring-offset-color", "#fff"); ("--tw-ring-offset-shadow", "0 0 #0000") ]

let box_shadow_decl =
  "box-shadow: var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(--tw-shadow);"

let shadows_and_rings () =
  let ds = design_system_for "@import \"twill\";\n    @theme { --ring-width-thick: 4px; --box-shadow-color-glow: #ff0; }" in
  let shadow raw decls = expect_decls ds raw (shadow_registrations () @ decls) in
  shadow "shadow"
    [ "--tw-shadow: 0 1px 3px 0 var(--tw-shadow-color, rgb(0 0 0 / 0.1)), 0 1px 2px -1px var(--tw-shadow-color, rgb(0 0 0 / 0.1));";
      box_shadow_decl ];
  shadow "shadow-lg"
    [ "--tw-shadow: 0 10px 15px -3px var(--tw-shadow-color, rgb(0 0 0 / 0.1)), 0 4px 6px -4px var(--tw-shadow-color, rgb(0 0 0 / 0.1));";
      box_shadow_decl ];
  shadow "shadow-2xl/50"
    [ "--tw-shadow: 0 25px 50px -12px var(--tw-shadow-color, color-mix(in oklab, rgb(0 0 0 / 0.25) 50%, transparent));"; box_shadow_decl ];
  shadow "shadow-none" [ "--tw-shadow: 0 0 #0000;"; box_shadow_decl ];
  shadow "shadow-red-500" [ "--tw-shadow-color: var(--color-red-500);" ];
  shadow "shadow-red-500/50" [ "--tw-shadow-color: color-mix(in oklab, var(--color-red-500) 50%, transparent);" ];
  shadow "shadow-glow" [ "--tw-shadow-color: var(--box-shadow-color-glow);" ];
  shadow "shadow-current" [ "--tw-shadow-color: currentcolor;" ];
  shadow "shadow-[0_0_3px_red,0_0_6px]"
    [ "--tw-shadow: 0 0 3px var(--tw-shadow-color, red), 0 0 6px var(--tw-shadow-color, currentcolor);"; box_shadow_decl ];
  shadow "shadow-[#fff]" [ "--tw-shadow-color: #fff;" ];
  shadow "shadow-[color:var(--c)]" [ "--tw-shadow-color: var(--c);" ];
  shadow "shadow-[var(--s)]" [ "--tw-shadow: var(--s);"; box_shadow_decl ];
  shadow "inset-shadow-sm" [ "--tw-inset-shadow: inset 0 2px 4px var(--tw-inset-shadow-color, rgb(0 0 0 / 0.05));"; box_shadow_decl ];
  shadow "inset-shadow-[0_2px_4px_red]" [ "--tw-inset-shadow: inset 0 2px 4px var(--tw-inset-shadow-color, red);"; box_shadow_decl ];
  shadow "inset-shadow-[inset_0_2px_red]" [ "--tw-inset-shadow: inset 0 2px var(--tw-inset-shadow-color, red);"; box_shadow_decl ];
  shadow "inset-shadow-none" [ "--tw-inset-shadow: 0 0 #0000;"; box_shadow_decl ];
  shadow "inset-shadow-red-500" [ "--tw-inset-shadow-color: var(--color-red-500);" ];
  shadow "ring"
    [ "--tw-ring-shadow: var(--tw-ring-inset,) 0 0 0 calc(1px + var(--tw-ring-offset-width)) var(--tw-ring-color, currentcolor);";
      box_shadow_decl ];
  shadow "ring-2"
    [ "--tw-ring-shadow: var(--tw-ring-inset,) 0 0 0 calc(2px + var(--tw-ring-offset-width)) var(--tw-ring-color, currentcolor);";
      box_shadow_decl ];
  shadow "ring-thick"
    [ "--tw-ring-shadow: var(--tw-ring-inset,) 0 0 0 calc(var(--ring-width-thick) + var(--tw-ring-offset-width)) var(--tw-ring-color, currentcolor);";
      box_shadow_decl ];
  shadow "ring-[3px]"
    [ "--tw-ring-shadow: var(--tw-ring-inset,) 0 0 0 calc(3px + var(--tw-ring-offset-width)) var(--tw-ring-color, currentcolor);";
      box_shadow_decl ];
  shadow "ring-red-500/30" [ "--tw-ring-color: color-mix(in oklab, var(--color-red-500) 30%, transparent);" ];
  shadow "ring-[#000]" [ "--tw-ring-color: #000;" ];
  shadow "ring-inset" [ "--tw-ring-inset: inset;" ];
  shadow "inset-ring" [ "--tw-inset-ring-shadow: inset 0 0 0 1px var(--tw-inset-ring-color, currentcolor);"; box_shadow_decl ];
  shadow "inset-ring-2" [ "--tw-inset-ring-shadow: inset 0 0 0 2px var(--tw-inset-ring-color, currentcolor);"; box_shadow_decl ];
  shadow "inset-ring-blue-500" [ "--tw-inset-ring-color: var(--color-blue-500);" ];
  shadow "ring-offset-2"
    [ "--tw-ring-offset-width: 2px;";
      "--tw-ring-offset-shadow: var(--tw-ring-inset,) 0 0 0 var(--tw-ring-offset-width) var(--tw-ring-offset-color);" ];
  shadow "ring-offset-[3px]"
    [ "--tw-ring-offset-width: 3px;";
      "--tw-ring-offset-shadow: var(--tw-ring-inset,) 0 0 0 var(--tw-ring-offset-width) var(--tw-ring-offset-color);" ];
  shadow "ring-offset-white" [ "--tw-ring-offset-color: var(--color-white);" ];
  expect_invalid ds
    [ "shadow-nope"; "shadow-lg/foo"; "inset-shadow"; "ring-x"; "ring-2/50"; "ring-[3px]/50"; "ring-offset"; "ring-offset-2/50"; "inset-ring-x" ];
  let wrap c = "<" ^ c ^ ">" in
  let r = Shadows.replace_shadow_colors in
  equal "replace shadow colors" (r "0 0 3px red, inset 0 1px" wrap false) "0 0 3px <red>, inset 0 1px <currentcolor>";
  equal "replace none" (r "none" wrap false) "none";
  equal "replace var" (r "var(--x)" wrap false) "var(--x)";
  equal "replace inset" (r "0 2px 4px rgb(0 0 0 / 0.1)" wrap true) "inset 0 2px 4px <rgb(0 0 0 / 0.1)>";
  equal "replace has inset" (r "inset 0 2px 4px #000" wrap true) "inset 0 2px 4px <#000>";
  equal "replace whitespace" (r "0px  1px\n  0px #000" wrap false) "0px 1px 0px <#000>"

let transforms () =
  let ds = design_system_for "" in
  let d = expect_decls ds in
  let transform_regs = registrations [ "--tw-rotate-x"; "--tw-rotate-y"; "--tw-rotate-z"; "--tw-skew-x"; "--tw-skew-y" ] "" in
  let translate_regs = registrations [ "--tw-translate-x"; "--tw-translate-y"; "--tw-translate-z" ] "0" in
  let scale_regs = registrations [ "--tw-scale-x"; "--tw-scale-y"; "--tw-scale-z" ] "1" in
  let transform_decl = "transform: var(--tw-rotate-x,) var(--tw-rotate-y,) var(--tw-rotate-z,) var(--tw-skew-x,) var(--tw-skew-y,);" in
  d "transform-none" [ "transform: none;" ];
  d "transform-gpu"
    (transform_regs @ [ "transform: translateZ(0) var(--tw-rotate-x,) var(--tw-rotate-y,) var(--tw-rotate-z,) var(--tw-skew-x,) var(--tw-skew-y,);" ]);
  d "transform-[matrix(1,0,0,1,0,0)]" [ "transform: matrix(1,0,0,1,0,0);" ];
  d "transform-3d" [ "transform-style: preserve-3d;" ];
  d "backface-hidden" [ "backface-visibility: hidden;" ];
  d "perspective-near" [ "perspective: var(--perspective-near);" ];
  d "perspective-origin-top-left" [ "perspective-origin: top left;" ];
  d "origin-[10px_20px]" [ "transform-origin: 10px 20px;" ];
  d "translate-4"
    (translate_regs @ [ "--tw-translate-x: --spacing(4);"; "--tw-translate-y: --spacing(4);"; "translate: var(--tw-translate-x) var(--tw-translate-y);" ]);
  d "-translate-x-1/2"
    (translate_regs @ [ "--tw-translate-x: calc(calc(1 / 2 * 100%) * -1);"; "translate: var(--tw-translate-x) var(--tw-translate-y);" ]);
  d "-translate-y-full" (translate_regs @ [ "--tw-translate-y: -100%;"; "translate: var(--tw-translate-x) var(--tw-translate-y);" ]);
  d "translate-z-px"
    (translate_regs @ [ "--tw-translate-z: 1px;"; "translate: var(--tw-translate-x) var(--tw-translate-y) var(--tw-translate-z);" ]);
  d "translate-none" [ "translate: none;" ];
  d "scale-50"
    (scale_regs @ [ "--tw-scale-x: 50%;"; "--tw-scale-y: 50%;"; "--tw-scale-z: 50%;"; "scale: var(--tw-scale-x) var(--tw-scale-y);" ]);
  d "-scale-x-75" (scale_regs @ [ "--tw-scale-x: calc(75% * -1);"; "scale: var(--tw-scale-x) var(--tw-scale-y);" ]);
  d "scale-z-150" (scale_regs @ [ "--tw-scale-z: 150%;"; "scale: var(--tw-scale-x) var(--tw-scale-y) var(--tw-scale-z);" ]);
  d "scale-[1.5]" [ "scale: 1.5;" ];
  d "rotate-45" [ "rotate: 45deg;" ];
  d "-rotate-45" [ "rotate: calc(45deg * -1);" ];
  d "rotate-[30deg]" [ "rotate: 30deg;" ];
  d "rotate-x-30" (transform_regs @ [ "--tw-rotate-x: rotateX(30deg);"; transform_decl ]);
  d "-skew-6" (transform_regs @ [ "--tw-skew-x: skewX(calc(6deg * -1));"; "--tw-skew-y: skewY(calc(6deg * -1));"; transform_decl ]);
  d "skew-y-[10deg]" (transform_regs @ [ "--tw-skew-y: skewY(10deg);"; transform_decl ]);
  expect_invalid ds
    [ "transform"; "origin-nope"; "translate-z-1/2"; "scale-1.5"; "scale-50/2"; "-scale-[1.5]"; "rotate-1.5"; "rotate-45/2"; "skew-x"; "perspective-500" ]

let filters () =
  let ds = design_system_for "" in
  let d = expect_decls ds in
  let filter_regs =
    registrations
      [ "--tw-blur"; "--tw-brightness"; "--tw-contrast"; "--tw-grayscale"; "--tw-hue-rotate"; "--tw-invert"; "--tw-saturate"; "--tw-sepia"; "--tw-drop-shadow" ]
      ""
  in
  let backdrop_regs =
    registrations
      [ "--tw-backdrop-blur"; "--tw-backdrop-brightness"; "--tw-backdrop-contrast"; "--tw-backdrop-grayscale"; "--tw-backdrop-hue-rotate";
        "--tw-backdrop-invert"; "--tw-backdrop-opacity"; "--tw-backdrop-saturate"; "--tw-backdrop-sepia" ]
      ""
  in
  let filter_decl =
    "filter: var(--tw-blur,) var(--tw-brightness,) var(--tw-contrast,) var(--tw-grayscale,) var(--tw-hue-rotate,) var(--tw-invert,) var(--tw-saturate,) var(--tw-sepia,) var(--tw-drop-shadow,);"
  in
  let backdrop_value =
    "var(--tw-backdrop-blur,) var(--tw-backdrop-brightness,) var(--tw-backdrop-contrast,) var(--tw-backdrop-grayscale,) var(--tw-backdrop-hue-rotate,) var(--tw-backdrop-invert,) var(--tw-backdrop-opacity,) var(--tw-backdrop-saturate,) var(--tw-backdrop-sepia,)"
  in
  let filter raw decls = d raw (filter_regs @ decls @ [ filter_decl ]) in
  let backdrop raw decls =
    d raw (backdrop_regs @ decls @ [ "-webkit-backdrop-filter: " ^ backdrop_value ^ ";"; "backdrop-filter: " ^ backdrop_value ^ ";" ])
  in
  d "filter-none" [ "filter: none;" ];
  d "filter-[blur(2px)]" [ "filter: blur(2px);" ];
  filter "filter" [];
  filter "blur" [ "--tw-blur: blur(8px);" ];
  filter "blur-sm" [ "--tw-blur: blur(var(--blur-sm));" ];
  filter "blur-[2px]" [ "--tw-blur: blur(2px);" ];
  filter "blur-none" [ "--tw-blur: ;" ];
  filter "brightness-50" [ "--tw-brightness: brightness(50%);" ];
  filter "contrast-[.5]" [ "--tw-contrast: contrast(.5);" ];
  filter "saturate-150" [ "--tw-saturate: saturate(150%);" ];
  filter "grayscale" [ "--tw-grayscale: grayscale(100%);" ];
  filter "grayscale-0" [ "--tw-grayscale: grayscale(0%);" ];
  filter "invert-[.25]" [ "--tw-invert: invert(.25);" ];
  filter "sepia" [ "--tw-sepia: sepia(100%);" ];
  filter "hue-rotate-90" [ "--tw-hue-rotate: hue-rotate(90deg);" ];
  filter "-hue-rotate-90" [ "--tw-hue-rotate: hue-rotate(calc(90deg * -1));" ];
  filter "drop-shadow-lg" [ "--tw-drop-shadow: drop-shadow(0 4px 4px var(--tw-drop-shadow-color, rgb(0 0 0 / 0.15)));" ];
  filter "drop-shadow-lg/50"
    [ "--tw-drop-shadow: drop-shadow(0 4px 4px var(--tw-drop-shadow-color, color-mix(in oklab, rgb(0 0 0 / 0.15) 50%, transparent)));" ];
  filter "drop-shadow-[0_0_3px_red,0_0_6px]"
    [ "--tw-drop-shadow: drop-shadow(0 0 3px var(--tw-drop-shadow-color, red)) drop-shadow(0 0 6px var(--tw-drop-shadow-color, currentcolor));" ];
  filter "drop-shadow-none" [ "--tw-drop-shadow: ;" ];
  d "drop-shadow-red-500" [ reg "--tw-drop-shadow-color" ""; "--tw-drop-shadow-color: var(--color-red-500);" ];
  d "backdrop-filter-none" [ "-webkit-backdrop-filter: none;"; "backdrop-filter: none;" ];
  backdrop "backdrop-filter" [];
  backdrop "backdrop-blur-sm" [ "--tw-backdrop-blur: blur(var(--blur-sm));" ];
  backdrop "backdrop-grayscale" [ "--tw-backdrop-grayscale: grayscale(100%);" ];
  backdrop "backdrop-opacity-50" [ "--tw-backdrop-opacity: opacity(50%);" ];
  backdrop "-backdrop-hue-rotate-15" [ "--tw-backdrop-hue-rotate: hue-rotate(calc(15deg * -1));" ];
  expect_invalid ds
    [ "blur-4"; "brightness"; "brightness-1.5"; "hue-rotate-1.5"; "drop-shadow-lg/foo"; "drop-shadow-nope"; "backdrop-opacity";
      "backdrop-drop-shadow-lg"; "blur-sm/50" ]

let gradients () =
  let ds = design_system_for "" in
  let d = expect_decls ds in
  let gradient_regs =
    [ reg "--tw-gradient-position" ""; reg ~syntax:"<color>" "--tw-gradient-from" "#0000"; reg ~syntax:"<color>" "--tw-gradient-via" "#0000";
      reg ~syntax:"<color>" "--tw-gradient-to" "#0000"; reg "--tw-gradient-stops" ""; reg "--tw-gradient-via-stops" "";
      reg ~syntax:"<length-percentage>" "--tw-gradient-from-position" "0%"; reg ~syntax:"<length-percentage>" "--tw-gradient-via-position" "50%";
      reg ~syntax:"<length-percentage>" "--tw-gradient-to-position" "100%" ]
  in
  let stops_decl =
    "--tw-gradient-stops: var(--tw-gradient-via-stops, var(--tw-gradient-position), var(--tw-gradient-from) var(--tw-gradient-from-position), var(--tw-gradient-to) var(--tw-gradient-to-position));"
  in
  let with_regs decls = gradient_regs @ decls in
  let linear = "background-image: linear-gradient(var(--tw-gradient-stops));" in
  d "bg-linear-to-r" [ "--tw-gradient-position: to right in oklab;"; linear ];
  d "bg-linear-to-tl/srgb" [ "--tw-gradient-position: to top left in srgb;"; linear ];
  d "bg-linear-to-r/longer" [ "--tw-gradient-position: to right in oklch longer hue;"; linear ];
  d "bg-linear-45/[in_hsl]" [ "--tw-gradient-position: 45deg in hsl;"; linear ];
  d "-bg-linear-45" [ "--tw-gradient-position: calc(45deg * -1) in oklab;"; linear ];
  d "bg-linear-[30deg]" [ "--tw-gradient-position: 30deg in oklab;"; linear ];
  d "bg-linear-[to_right,red,blue]" [ "background-image: linear-gradient(to right,red,blue);" ];
  d "bg-gradient-to-b" [ "--tw-gradient-position: to bottom in oklab;"; linear ];
  d "bg-radial" [ "--tw-gradient-position: in oklab;"; "background-image: radial-gradient(var(--tw-gradient-stops));" ];
  d "bg-radial-[at_center]" [ "--tw-gradient-position: at center;"; "background-image: radial-gradient(var(--tw-gradient-stops));" ];
  d "bg-conic-90/hsl" [ "--tw-gradient-position: from 90deg in hsl;"; "background-image: conic-gradient(var(--tw-gradient-stops));" ];
  d "from-red-500" (with_regs [ "--tw-gradient-from: var(--color-red-500);"; stops_decl ]);
  d "from-red-500/50" (with_regs [ "--tw-gradient-from: color-mix(in oklab, var(--color-red-500) 50%, transparent);"; stops_decl ]);
  d "from-10%" (with_regs [ "--tw-gradient-from-position: 10%;" ]);
  d "via-blue-500"
    (with_regs
       [ "--tw-gradient-via: var(--color-blue-500);";
         "--tw-gradient-via-stops: var(--tw-gradient-position), var(--tw-gradient-from) var(--tw-gradient-from-position), var(--tw-gradient-via) var(--tw-gradient-via-position), var(--tw-gradient-to) var(--tw-gradient-to-position);";
         "--tw-gradient-stops: var(--tw-gradient-via-stops);" ]);
  d "to-[2rem]" (with_regs [ "--tw-gradient-to-position: 2rem;" ]);
  d "to-[#fff]" (with_regs [ "--tw-gradient-to: #fff;"; stops_decl ]);
  expect_invalid ds
    [ "bg-linear"; "bg-linear-to-x"; "bg-linear-to-r/nope"; "bg-linear-1.5"; "bg-gradient-45"; "bg-radial-x"; "bg-radial-[at_center]/srgb";
      "bg-conic-x"; "from-10"; "from-10%/50"; "from-nope"; "to-nope" ]

let dividers () =
  let ds = design_system_for "" in
  let d = expect_decls ds in
  let reverse name = reg name "0" in
  let nested decls = ":where(& > :not(:last-child)) {\n" ^ String.concat "" (List.map (fun x -> "    " ^ x ^ "\n") decls) ^ "  }" in
  d "space-x-4"
    [ reverse "--tw-space-x-reverse";
      nested
        [ "--tw-space-x-reverse: 0;"; "margin-inline-start: calc(--spacing(4) * var(--tw-space-x-reverse));";
          "margin-inline-end: calc(--spacing(4) * calc(1 - var(--tw-space-x-reverse)));" ] ];
  d "-space-y-px"
    [ reverse "--tw-space-y-reverse";
      nested
        [ "--tw-space-y-reverse: 0;"; "margin-block-start: calc(-1px * var(--tw-space-y-reverse));";
          "margin-block-end: calc(-1px * calc(1 - var(--tw-space-y-reverse)));" ] ];
  d "space-x-reverse" [ reverse "--tw-space-x-reverse"; nested [ "--tw-space-x-reverse: 1;" ] ];
  d "divide-x-2"
    [ reverse "--tw-divide-x-reverse"; border_style_property;
      nested
        [ "--tw-divide-x-reverse: 0;"; "border-inline-style: var(--tw-border-style);";
          "border-inline-start-width: calc(2px * var(--tw-divide-x-reverse));";
          "border-inline-end-width: calc(2px * calc(1 - var(--tw-divide-x-reverse)));" ] ];
  d "divide-y"
    [ reverse "--tw-divide-y-reverse"; border_style_property;
      nested
        [ "--tw-divide-y-reverse: 0;"; "border-block-style: var(--tw-border-style);";
          "border-block-start-width: calc(1px * var(--tw-divide-y-reverse));";
          "border-block-end-width: calc(1px * calc(1 - var(--tw-divide-y-reverse)));" ] ];
  d "divide-y-reverse" [ reverse "--tw-divide-y-reverse"; nested [ "--tw-divide-y-reverse: 1;" ] ];
  d "divide-red-500/50" [ nested [ "border-color: color-mix(in oklab, var(--color-red-500) 50%, transparent);" ] ];
  d "divide-dashed" [ nested [ "--tw-border-style: dashed;"; "border-style: dashed;" ] ];
  let outline_style = reg "--tw-outline-style" "solid" in
  d "outline" [ outline_style; "outline-style: var(--tw-outline-style);"; "outline-width: 1px;" ];
  d "outline-[3px]" [ outline_style; "outline-style: var(--tw-outline-style);"; "outline-width: 3px;" ];
  d "outline-red-500" [ "outline-color: var(--color-red-500);" ];
  d "outline-[#fff]/50" [ "outline-color: color-mix(in oklab, #fff 50%, transparent);" ];
  d "outline-none" [ "--tw-outline-style: none;"; "outline-style: none;" ];
  d "outline-hidden"
    [ "--tw-outline-style: none;"; "outline-style: none;";
      "@media (forced-colors: active) {\n    outline: 2px solid transparent;\n    outline-offset: 2px;\n  }" ];
  d "outline-dotted" [ "--tw-outline-style: dotted;"; "outline-style: dotted;" ];
  d "-outline-offset-2" [ "outline-offset: calc(2px * -1);" ];
  expect_invalid ds
    [ "space-x"; "space-x-1/2"; "divide-x-2/50"; "divide-x-nope"; "divide-nope"; "outline-2/50"; "outline-nope"; "outline-offset"; "outline-offset-x" ]

let extras () =
  let ds = design_system_for "" in
  let d = expect_decls ds in
  d "line-clamp-3" [ "overflow: hidden;"; "display: -webkit-box;"; "-webkit-box-orient: vertical;"; "-webkit-line-clamp: 3;" ];
  d "line-clamp-none" [ "overflow: visible;"; "display: block;"; "-webkit-box-orient: horizontal;"; "-webkit-line-clamp: unset;" ];
  d "decoration-red-500/50" [ "text-decoration-color: color-mix(in oklab, var(--color-red-500) 50%, transparent);" ];
  d "decoration-wavy" [ "text-decoration-style: wavy;" ];
  d "decoration-2" [ "text-decoration-thickness: 2px;" ];
  d "decoration-[10%]" [ "text-decoration-thickness: 10%;" ];
  d "hyphens-auto" [ "-webkit-hyphens: auto;"; "hyphens: auto;" ];
  d "tabular-nums"
    [ reg "--tw-ordinal" ""; reg "--tw-slashed-zero" ""; reg "--tw-numeric-figure" ""; reg "--tw-numeric-spacing" "";
      reg "--tw-numeric-fraction" ""; "--tw-numeric-spacing: tabular-nums;";
      "font-variant-numeric: var(--tw-ordinal,) var(--tw-slashed-zero,) var(--tw-numeric-figure,) var(--tw-numeric-spacing,) var(--tw-numeric-fraction,);" ];
  d "normal-nums" [ "font-variant-numeric: normal;" ];
  d "align-text-top" [ "vertical-align: text-top;" ];
  d "font-stretch-50%" [ "font-stretch: 50%;" ];
  d "font-stretch-condensed" [ "font-stretch: condensed;" ];
  d "text-shadow-2xs" [ "text-shadow: 0px 1px 0px var(--tw-text-shadow-color, rgb(0 0 0 / 0.15));" ];
  d "text-shadow-red-500" [ reg "--tw-text-shadow-color" ""; "--tw-text-shadow-color: var(--color-red-500);" ];
  d "wrap-anywhere" [ "overflow-wrap: anywhere;" ];
  d "content-none" [ reg "--tw-content" "\"\""; "--tw-content: none;"; "content: none;" ];
  d "bg-blend-multiply" [ "background-blend-mode: multiply;" ];
  d "mix-blend-plus-lighter" [ "mix-blend-mode: plus-lighter;" ];
  d "container"
    [ "width: 100%;"; "@media (width >= 40rem) {\n    max-width: 40rem;\n  }"; "@media (width >= 48rem) {\n    max-width: 48rem;\n  }";
      "@media (width >= 64rem) {\n    max-width: 64rem;\n  }"; "@media (width >= 80rem) {\n    max-width: 80rem;\n  }";
      "@media (width >= 96rem) {\n    max-width: 96rem;\n  }" ];
  d "break-inside-avoid-column" [ "break-inside: avoid-column;" ];
  d "box-decoration-clone" [ "box-decoration-break: clone;" ];
  d "object-[10px_20px]" [ "object-position: 10px 20px;" ];
  d "-start-4" [ "inset-inline-start: --spacing(-4);" ];
  d "end-auto" [ "inset-inline-end: auto;" ];
  d "border-spacing-x-2"
    [ reg "--tw-border-spacing-x" "0"; reg "--tw-border-spacing-y" "0"; "--tw-border-spacing-x: --spacing(2);";
      "border-spacing: var(--tw-border-spacing-x) var(--tw-border-spacing-y);" ];
  d "table-fixed" [ "table-layout: fixed;" ];
  d "-scroll-mt-4" [ "scroll-margin-top: --spacing(-4);" ];
  d "scroll-px-[3px]" [ "scroll-padding-inline: 3px;" ];
  d "snap-both" [ reg "--tw-scroll-snap-strictness" "proximity"; "scroll-snap-type: both var(--tw-scroll-snap-strictness);" ];
  d "snap-mandatory" [ reg "--tw-scroll-snap-strictness" "proximity"; "--tw-scroll-snap-strictness: mandatory;" ];
  d "touch-pinch-zoom"
    [ reg "--tw-pan-x" ""; reg "--tw-pan-y" ""; reg "--tw-pinch-zoom" ""; "--tw-pinch-zoom: pinch-zoom;";
      "touch-action: var(--tw-pan-x,) var(--tw-pan-y,) var(--tw-pinch-zoom,);" ];
  d "stroke-2" [ "stroke-width: 2;" ];
  d "stroke-[2px]" [ "stroke-width: 2px;" ];
  d "stroke-red-500/50" [ "stroke: color-mix(in oklab, var(--color-red-500) 50%, transparent);" ];
  d "scheme-light-dark" [ "color-scheme: light dark;" ];
  d "field-sizing-content" [ "field-sizing: content;" ];
  expect_invalid ds
    [ "line-clamp"; "line-clamp-1.5"; "decoration-2/50"; "decoration-nope"; "align-nope"; "font-stretch-50"; "text-shadow-nope";
      "bg-blend-plus-lighter"; "-scroll-p-4"; "-border-spacing-2"; "stroke-1.5"; "stroke-2/50" ]

let masks () =
  let ds = design_system_for "" in
  let d = expect_decls ds in
  let stops name =
    [ reg ("--tw-mask-" ^ name ^ "-from-color") "black"; reg ("--tw-mask-" ^ name ^ "-from-position") "0%";
      reg ("--tw-mask-" ^ name ^ "-to-color") "transparent"; reg ("--tw-mask-" ^ name ^ "-to-position") "100%" ]
  in
  let white = "linear-gradient(#fff, #fff)" in
  let regs =
    List.concat_map (fun edge -> reg ("--tw-mask-" ^ edge) white :: stops edge) [ "top"; "right"; "bottom"; "left" ]
    @ [ reg "--tw-mask-linear" white; reg "--tw-mask-linear-position" "0deg" ]
    @ stops "linear"
    @ [ reg "--tw-mask-radial" white; reg "--tw-mask-radial-shape" "ellipse"; reg "--tw-mask-radial-size" "farthest-corner";
        reg "--tw-mask-radial-position" "center" ]
    @ stops "radial"
    @ [ reg "--tw-mask-conic" white; reg "--tw-mask-conic-position" "0deg" ]
    @ stops "conic"
  in
  equal "mask registration count" (string_of_int (List.length regs)) "40";
  let composed decls =
    regs @ [ "mask-image: var(--tw-mask-linear), var(--tw-mask-radial), var(--tw-mask-conic);"; "mask-composite: intersect;" ] @ decls
  in
  let edge_list = "--tw-mask-linear: var(--tw-mask-left), var(--tw-mask-right), var(--tw-mask-bottom), var(--tw-mask-top);" in
  let edge e =
    "--tw-mask-" ^ e ^ ": linear-gradient(to " ^ e ^ ", var(--tw-mask-" ^ e ^ "-from-color) var(--tw-mask-" ^ e ^ "-from-position), var(--tw-mask-" ^ e
    ^ "-to-color) var(--tw-mask-" ^ e ^ "-to-position));"
  in
  let linear =
    "--tw-mask-linear: linear-gradient(var(--tw-mask-linear-position), var(--tw-mask-linear-from-color) var(--tw-mask-linear-from-position), var(--tw-mask-linear-to-color) var(--tw-mask-linear-to-position));"
  in
  let radial =
    "--tw-mask-radial: radial-gradient(var(--tw-mask-radial-shape) var(--tw-mask-radial-size) at var(--tw-mask-radial-position), var(--tw-mask-radial-from-color) var(--tw-mask-radial-from-position), var(--tw-mask-radial-to-color) var(--tw-mask-radial-to-position));"
  in
  let conic =
    "--tw-mask-conic: conic-gradient(from var(--tw-mask-conic-position), var(--tw-mask-conic-from-color) var(--tw-mask-conic-from-position), var(--tw-mask-conic-to-color) var(--tw-mask-conic-to-position));"
  in
  d "mask-none" [ "mask-image: none;" ];
  d "mask-[url(x.svg)]" [ "mask-image: url(x.svg);" ];
  d "mask-intersect" [ "mask-composite: intersect;" ];
  d "mask-match" [ "mask-mode: match-source;" ];
  d "mask-type-luminance" [ "mask-type: luminance;" ];
  d "mask-cover" [ "mask-size: cover;" ];
  d "mask-clip-border" [ "mask-clip: border-box;" ];
  d "mask-no-clip" [ "mask-clip: no-clip;" ];
  d "mask-origin-view" [ "mask-origin: view-box;" ];
  d "mask-bottom-right" [ "mask-position: bottom right;" ];
  d "mask-repeat-space" [ "mask-repeat: space;" ];
  d "mask-t-from-50%" (composed [ edge_list; edge "top"; "--tw-mask-top-from-position: 50%;" ]);
  d "mask-b-to-4" (composed [ edge_list; edge "bottom"; "--tw-mask-bottom-to-position: --spacing(4);" ]);
  d "mask-x-from-red-500/50"
    (composed
       [ edge_list; edge "left"; edge "right";
         "--tw-mask-left-from-color: color-mix(in oklab, var(--color-red-500) 50%, transparent);";
         "--tw-mask-right-from-color: color-mix(in oklab, var(--color-red-500) 50%, transparent);" ]);
  d "-mask-linear-45" (composed [ linear; "--tw-mask-linear-position: calc(45deg * -1);" ]);
  d "mask-linear-from-[3rem]" (composed [ linear; "--tw-mask-linear-from-position: 3rem;" ]);
  d "mask-radial-[100px_50px]" (composed [ radial; "--tw-mask-radial-size: 100px 50px;" ]);
  d "mask-radial-to-transparent" (composed [ radial; "--tw-mask-radial-to-color: transparent;" ]);
  d "mask-circle" [ "--tw-mask-radial-shape: circle;" ];
  d "mask-radial-at-top-left" [ "--tw-mask-radial-position: top left;" ];
  d "mask-conic-[0.5turn]" (composed [ conic; "--tw-mask-conic-position: 0.5turn;" ]);
  d "mask-conic-from-10%" (composed [ conic; "--tw-mask-conic-from-position: 10%;" ]);
  expect_invalid ds
    [ "mask-t-from"; "mask-t-from-nope"; "mask-t-from-50%/50"; "mask-l-from-1/2"; "mask-linear-[30px]"; "mask-linear-x"; "mask-radial-x"; "mask-conic-x" ]

let effects () =
  let ds = design_system_for "" in
  let d = expect_decls ds in
  d "opacity-[.5]" [ "opacity: .5;" ];
  d "opacity-75" [ "opacity: 75%;" ];
  expect_invalid ds [ "opacity-33.3"; "opacity-50/50" ];
  includes "transition" (compile_raw ds "transition")
    "transition-property: color, background-color, border-color, outline-color, text-decoration-color, fill, stroke, --tw-gradient-from, --tw-gradient-via, --tw-gradient-to, opacity, box-shadow, transform, translate, scale, rotate, filter, -webkit-backdrop-filter, backdrop-filter, display, content-visibility, overlay, pointer-events;";
  d "transition-opacity"
    [ "transition-property: opacity;"; "transition-timing-function: var(--default-transition-timing-function);";
      "transition-duration: var(--default-transition-duration);" ];
  d "duration-300" [ "transition-duration: 300ms;" ];
  d "ease-in-out" [ "transition-timing-function: var(--ease-in-out);" ];
  d "animate-spin" [ "animation: var(--animate-spin);" ];
  d "cursor-pointer" [ "cursor: pointer;" ];
  d "select-none" [ "-webkit-user-select: none;"; "user-select: none;" ];
  d "resize-y" [ "resize: vertical;" ];
  d "will-change-[top]" [ "will-change: top;" ];
  d "content-['hi']" [ reg "--tw-content" "\"\""; "--tw-content: 'hi';"; "content: var(--tw-content);" ];
  d "aspect-video" [ "aspect-ratio: var(--aspect-video);" ];
  d "aspect-16/9" [ "aspect-ratio: 16 / 9;" ];
  d "columns-3" [ "columns: 3;" ];
  d "columns-md" [ "columns: var(--container-md);" ];
  d "object-cover" [ "object-fit: cover;" ];
  d "accent-red-500" [ "accent-color: var(--color-red-500);" ];
  d "fill-current" [ "fill: currentcolor;" ];
  d "stroke-red-500/50" [ "stroke: color-mix(in oklab, var(--color-red-500) 50%, transparent);" ];
  expect_invalid ds [ "content-hi"; "aspect-16/x"; "columns-1.5"; "fill"; "accent-nope" ];
  let custom =
    design_system_for
      "@import \"twill\";\n    @theme { --color-*: initial; --color-primary: oklch(0.6 0.2 250); --breakpoint-3xl: 120rem; --font-display: \"Inter\", sans-serif; }"
  in
  expect_decls custom "bg-primary" [ "background-color: var(--color-primary);" ];
  expect_decls custom "font-display" [ "font-family: var(--font-display);" ];
  equal "3xl variant" (compile_raw custom "3xl:flex") ".\\33 xl\\:flex {\n  @media (width >= 120rem) {\n    display: flex;\n  }\n}\n";
  expect_invalid custom [ "bg-red-500" ]

let infer_data_type () =
  List.iter
    (fun (value, types, want) -> equal ("infer " ^ value) (Data_types.infer_data_type value types) want)
    [ ("red", [ "color" ], "color"); ("#ffff", [ "color" ], "color"); ("#ggg", [ "color" ], ""); ("color-mix(in oklab, red, blue)", [ "color" ], "color");
      ("var(--x)", [ "color" ], ""); ("1.5rem", [ "length" ], "length"); ("0", [ "length" ], "length"); ("calc(1px + 2px)", [ "length" ], "length");
      ("--spacing(4)", [ "length" ], "length"); ("10", [ "length" ], ""); ("10%", [ "percentage" ], "percentage"); (".5", [ "number" ], "number");
      ("1.5", [ "integer" ], ""); ("16 / 9", [ "ratio" ], "ratio"); ("url(/a.png)", [ "url" ], "url");
      ("repeating-radial-gradient(red, blue)", [ "image" ], "image"); ("url(/a.png)", [ "image" ], "");
      ("left 10px top 20px", [ "position" ], "position"); ("foo", [ "position" ], ""); ("10px auto", [ "bg-size" ], "bg-size");
      ("thin", [ "line-width" ], "line-width"); ("x-large", [ "absolute-size" ], "absolute-size");
      ("'Segoe UI', Roboto", [ "family-name" ], "family-name"); ("700", [ "family-name" ], ""); ("sans-serif", [ "generic-name" ], "generic-name");
      ("0.5turn", [ "angle" ], "angle"); ("1 0 0", [ "vector" ], "vector"); ("1 0", [ "vector" ], ""); ("10px", [ "color"; "length" ], "length");
      ("10px 20px", [ "percentage"; "position"; "bg-size" ], "position"); ("700", [ "number"; "family-name" ], "number") ]

let memoization () =
  let ds = design_system_for "" in
  let count raw = List.length (Design_system.parse_candidate ds raw) in
  check "parse counts" (count "flex" = 2 && count "block" = 1 && count "nope" = 0);
  check "memoization"
    (Option.get (Design_system.parse_variant ds "hover") == Option.get (Design_system.parse_variant ds "hover")
    && Design_system.parse_variant ds "nope" = None);
  let prefixed = design_system_for "@import \"twill\" prefix(tw);" in
  check "prefixed" (List.length (Design_system.parse_candidate prefixed "flex") = 0);
  check "prefixed block" (List.length (Design_system.parse_candidate prefixed "tw:block") = 1);
  check "important" (design_system_for "@import \"twill\" important;").important

let run () =
  spec_examples ();
  layout_flex_grid ();
  spacing_sizing ();
  typography ();
  backgrounds_borders ();
  shadows_and_rings ();
  transforms ();
  filters ();
  gradients ();
  dividers ();
  extras ();
  masks ();
  effects ();
  infer_data_type ();
  memoization ()

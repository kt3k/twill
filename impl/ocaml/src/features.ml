(* Feature flags reported by compile (SPEC §4.1.11). *)

let at_apply = 1
let at_import = 2
let theme_function = 8
let utilities = 16
let variants = 32
let at_theme = 64
let has features flag = features land flag <> 0

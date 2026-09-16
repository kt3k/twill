(* Data type inference for arbitrary values (SPEC §10.7). *)

open Utils

let named_colors =
  let tbl = Hashtbl.create 200 in
  List.iter (fun c -> Hashtbl.replace tbl c ())
    (fields
       "aliceblue antiquewhite aqua aquamarine azure beige bisque black blanchedalmond blue blueviolet brown burlywood \
        cadetblue chartreuse chocolate coral cornflowerblue cornsilk crimson cyan darkblue darkcyan darkgoldenrod \
        darkgray darkgreen darkgrey darkkhaki darkmagenta darkolivegreen darkorange darkorchid darkred darksalmon \
        darkseagreen darkslateblue darkslategray darkslategrey darkturquoise darkviolet deeppink deepskyblue dimgray \
        dimgrey dodgerblue firebrick floralwhite forestgreen fuchsia gainsboro ghostwhite gold goldenrod gray green \
        greenyellow grey honeydew hotpink indianred indigo ivory khaki lavender lavenderblush lawngreen lemonchiffon \
        lightblue lightcoral lightcyan lightgoldenrodyellow lightgray lightgreen lightgrey lightpink lightsalmon \
        lightseagreen lightskyblue lightslategray lightslategrey lightsteelblue lightyellow lime limegreen linen magenta \
        maroon mediumaquamarine mediumblue mediumorchid mediumpurple mediumseagreen mediumslateblue mediumspringgreen \
        mediumturquoise mediumvioletred midnightblue mintcream mistyrose moccasin navajowhite navy oldlace olive \
        olivedrab orange orangered orchid palegoldenrod palegreen paleturquoise palevioletred papayawhip peachpuff peru \
        pink plum powderblue purple rebeccapurple red rosybrown royalblue saddlebrown salmon sandybrown seagreen \
        seashell sienna silver skyblue slateblue slategray slategrey snow springgreen steelblue tan teal thistle tomato \
        turquoise violet wheat white whitesmoke yellow yellowgreen transparent currentcolor");
  tbl

let matches re s = Str.string_match re s 0
let number_body = "\\([0-9]+\\.?[0-9]*\\|\\.[0-9]+\\)"
let color_function_re = Str.regexp "^\\(rgba?\\|hsla?\\|hwb\\|oklch\\|oklab\\|lab\\|lch\\|color\\|color-mix\\|light-dark\\)("

let length_units =
  "cm\\|mm\\|Q\\|in\\|pt\\|pc\\|px\\|em\\|rem\\|ex\\|rex\\|cap\\|rcap\\|ch\\|rch\\|ic\\|ric\\|lh\\|rlh\\|vw\\|svw\\|lvw\\|dvw\\|vh\\|svh\\|lvh\\|dvh\\|vi\\|svi\\|lvi\\|dvi\\|vb\\|svb\\|lvb\\|dvb\\|vmin\\|svmin\\|lvmin\\|dvmin\\|vmax\\|svmax\\|lvmax\\|dvmax\\|cqw\\|cqh\\|cqi\\|cqb\\|cqmin\\|cqmax"

let length_re = Str.regexp ("^[+-]?" ^ number_body ^ "\\([eE][+-]?[0-9]+\\)?\\(" ^ length_units ^ "\\)$")

let math_function_re =
  Str.regexp
    "^\\(calc\\|min\\|max\\|clamp\\|round\\|mod\\|rem\\|sin\\|cos\\|tan\\|asin\\|acos\\|atan\\|atan2\\|pow\\|sqrt\\|hypot\\|log\\|exp\\|abs\\|sign\\)("

let percentage_re = Str.regexp ("^[+-]?" ^ number_body ^ "%$")
let number_re = Str.regexp ("^[+-]?" ^ number_body ^ "\\([eE][+-]?[0-9]+\\)?$")
let integer_re = Str.regexp "^[0-9]+$"
let ratio_re = Str.regexp ("^" ^ number_body ^ "[ \t]*/[ \t]*" ^ number_body ^ "$")
let image_re = Str.regexp "^\\(image\\|image-set\\|cross-fade\\|element\\)("
let gradient_re = Str.regexp "^[a-z-]*gradient("
let family_name_re = Str.regexp "^[a-zA-Z_][a-zA-Z0-9_ -]*$"
let angle_re = Str.regexp ("^[+-]?" ^ number_body ^ "\\(deg\\|rad\\|grad\\|turn\\)$")
let position_keywords = [ "top"; "right"; "bottom"; "left"; "center" ]
let absolute_sizes = [ "xx-small"; "x-small"; "small"; "medium"; "large"; "x-large"; "xx-large"; "xxx-large" ]

let generic_names =
  [ "serif"; "sans-serif"; "monospace"; "cursive"; "fantasy"; "system-ui"; "ui-serif"; "ui-sans-serif"; "ui-monospace";
    "ui-rounded"; "math"; "emoji"; "fangsong" ]

let is_hex_color v =
  let n = String.length v in
  n > 1 && v.[0] = '#' && (n = 4 || n = 5 || n = 7 || n = 9) && for_all_chars is_hex (after v 1)

let is_color v =
  let lower = String.lowercase_ascii v in
  Hashtbl.mem named_colors lower || is_hex_color v || matches color_function_re lower

let is_length v = v = "0" || matches length_re v || matches math_function_re v || has_prefix v "--spacing("
let is_percentage v = matches percentage_re v || matches math_function_re v
let is_number v = matches number_re v
let is_integer v = matches integer_re v
let is_ratio v = matches ratio_re v
let is_url v = has_prefix v "url("
let is_image v = matches image_re v || matches gradient_re v

let is_position v =
  let parts = fields v in
  let n = List.length parts in
  n > 0 && n <= 4 && List.for_all (fun p -> List.mem p position_keywords || is_length p || is_percentage p) parts

let is_bg_size v =
  v = "cover" || v = "contain" || v = "auto"
  ||
  let parts = fields v in
  let n = List.length parts in
  n > 0 && n <= 2 && List.for_all (fun p -> p = "auto" || is_length p || is_percentage p) parts

let is_line_width v = v = "thin" || v = "medium" || v = "thick" || is_length v

let is_family_name v =
  let parts = segment v ',' in
  parts <> []
  && List.for_all
       (fun part ->
         let part = String.trim part in
         part <> ""
         && ((has_prefix part "\"" && has_suffix part "\"")
            || (has_prefix part "'" && has_suffix part "'")
            || matches family_name_re part))
       parts

let is_vector v =
  let parts = fields v in
  List.length parts = 3 && List.for_all is_number parts

let checks =
  [ ("color", is_color); ("length", is_length); ("percentage", is_percentage); ("number", is_number);
    ("integer", is_integer); ("ratio", is_ratio); ("url", is_url); ("image", is_image); ("position", is_position);
    ("bg-size", is_bg_size); ("line-width", is_line_width); ("absolute-size", fun v -> List.mem v absolute_sizes);
    ("relative-size", fun v -> v = "larger" || v = "smaller"); ("family-name", is_family_name);
    ("generic-name", fun v -> List.mem v generic_names); ("angle", matches angle_re); ("vector", is_vector) ]

let is_data_type name = List.mem_assoc name checks

(* Returns the first type whose predicate accepts [value], or "". A value
   starting with var( never matches. *)
let infer_data_type value types =
  if has_prefix value "var(" then ""
  else
    match List.find_opt (fun t -> match List.assoc_opt t checks with Some check -> check value | None -> false) types with
    | Some t -> t
    | None -> ""

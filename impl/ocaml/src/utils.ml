(* String helpers (SPEC §4.2, §8, §10). *)

let has_prefix = Parser.has_prefix
let has_suffix = Parser.has_suffix

let contains s sub =
  let n = String.length s and m = String.length sub in
  let rec go i = i + m <= n && (String.sub s i m = sub || go (i + 1)) in
  go 0

let index_of s sub =
  let n = String.length s and m = String.length sub in
  let rec go i = if i + m > n then -1 else if String.sub s i m = sub then i else go (i + 1) in
  go 0

let last_index_of s sub =
  let n = String.length s and m = String.length sub in
  let rec go i = if i < 0 then -1 else if String.sub s i m = sub then i else go (i - 1) in
  go (n - m)

let trim_prefix s p = if has_prefix s p then String.sub s (String.length p) (String.length s - String.length p) else s
let trim_suffix s p = if has_suffix s p then String.sub s 0 (String.length s - String.length p) else s
let after s i = String.sub s i (String.length s - i)

let replace_all s pat rep =
  if pat = "" then s
  else
    let b = Buffer.create (String.length s) in
    let n = String.length s and m = String.length pat in
    let i = ref 0 in
    while !i < n do
      if !i + m <= n && String.sub s !i m = pat then (Buffer.add_string b rep; i := !i + m)
      else (Buffer.add_char b s.[!i]; incr i)
    done;
    Buffer.contents b

let split_char s c = String.split_on_char c s

(* Splits on runs of whitespace, dropping empty fields. *)
let fields s =
  let n = String.length s in
  let out = ref [] in
  let start = ref (-1) in
  for i = 0 to n do
    let ws = i = n || Ast.is_space s.[i] in
    if ws then begin
      if !start >= 0 then (out := String.sub s !start (i - !start) :: !out; start := -1)
    end
    else if !start < 0 then start := i
  done;
  List.rev !out

let is_digit c = c >= '0' && c <= '9'
let is_letter c = (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
let is_lower c = c >= 'a' && c <= 'z'
let is_alnum c = is_letter c || is_digit c
let is_hex c = is_digit c || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
let for_all_chars f s = try String.iter (fun c -> if not (f c) then raise Exit) s; true with Exit -> false

(* Named values and named modifiers must match `[a-zA-Z0-9_.%-]+`. *)
let is_named_value v = v <> "" && for_all_chars (fun c -> is_alnum c || c = '_' || c = '.' || c = '%' || c = '-') v

(* Custom variant names match `@?[a-z0-9][a-zA-Z0-9_-]*` and do not end in `_` or `-`. *)
let is_valid_variant_name name =
  let body = trim_prefix name "@" in
  body <> ""
  && (is_lower body.[0] || is_digit body.[0])
  && for_all_chars (fun c -> is_alnum c || c = '_' || c = '-') body
  &&
  let last = name.[String.length name - 1] in
  last <> '_' && last <> '-'

let is_valid_prefix p = p <> "" && for_all_chars is_lower p

(* Escapes a string for use in a class selector, matching CSS.escape. *)
let escape value =
  let b = Buffer.create (String.length value * 2) in
  let length = String.length value in
  let first = if length > 0 then value.[0] else '\000' in
  for i = 0 to length - 1 do
    let c = value.[i] in
    let code = Char.code c in
    if code = 0 then Buffer.add_string b "\xEF\xBF\xBD"
    else if (code >= 1 && code <= 0x1f) || code = 0x7f || (i = 0 && is_digit c) || (i = 1 && is_digit c && first = '-')
    then Buffer.add_string b (Printf.sprintf "\\%x " code)
    else if i = 0 && length = 1 && c = '-' then Buffer.add_string b "\\-"
    else if code >= 0x80 || c = '-' || c = '_' || is_alnum c then Buffer.add_char b c
    else (Buffer.add_char b '\\'; Buffer.add_char b c)
  done;
  Buffer.contents b

(* Reverses [escape]. *)
let unescape value =
  if not (String.contains value '\\') then value
  else
    let b = Buffer.create (String.length value) in
    let n = String.length value in
    let i = ref 0 in
    while !i < n do
      let c = value.[!i] in
      if c = '\\' && !i + 1 < n then begin
        let start = !i + 1 in
        if is_hex value.[start] then begin
          let j = ref start in
          while !j < n && !j - start < 6 && is_hex value.[!j] do incr j done;
          let hex = String.sub value start (!j - start) in
          if !j < n && Ast.is_space value.[!j] then incr j;
          (match int_of_string_opt ("0x" ^ hex) with
          | Some code when Uchar.is_valid code -> Buffer.add_utf_8_uchar b (Uchar.of_int code)
          | _ -> Buffer.add_string b hex);
          i := !j
        end
        else (Buffer.add_char b value.[start]; i := start + 1)
      end
      else if c = '\\' then incr i
      else (Buffer.add_char b c; incr i)
    done;
    Buffer.contents b

(* Splits [input] on a separator, ignoring separators inside brackets,
   quotes, and after a backslash. *)
let segment input sep =
  let n = String.length input in
  let parts = ref [] in
  let stack = Buffer.create 8 in
  let last = ref 0 in
  let i = ref 0 in
  while !i < n do
    let c = input.[!i] in
    if c = '\\' then i := !i + 2
    else if c = '"' || c = '\'' then begin
      incr i;
      let closed = ref false in
      while (not !closed) && !i < n do
        let d = input.[!i] in
        if d = '\\' then i := !i + 2 else if d = c then closed := true else incr i
      done;
      incr i
    end
    else begin
      (if c = '(' then Buffer.add_char stack ')'
       else if c = '[' then Buffer.add_char stack ']'
       else if c = '{' then Buffer.add_char stack '}'
       else if c = ')' || c = ']' || c = '}' then begin
         let l = Buffer.length stack in
         if l > 0 && Buffer.nth stack (l - 1) = c then Buffer.truncate stack (l - 1)
       end
       else if c = sep && Buffer.length stack = 0 then begin
         parts := String.sub input !last (!i - !last) :: !parts;
         last := !i + 1
       end);
      incr i
    end
  done;
  List.rev (String.sub input !last (n - !last) :: !parts)

(* Checks bracket balance of an arbitrary value. *)
let is_valid_arbitrary input =
  let n = String.length input in
  let stack = Buffer.create 8 in
  let i = ref 0 in
  let ok = ref true in
  while !ok && !i < n do
    let c = input.[!i] in
    if c = '\\' then i := !i + 2
    else if c = '"' || c = '\'' then begin
      incr i;
      let closed = ref false in
      while (not !closed) && !i < n do
        let d = input.[!i] in
        if d = '\\' then i := !i + 2 else if d = c then closed := true else incr i
      done;
      incr i
    end
    else begin
      (if c = '(' then Buffer.add_char stack ')'
       else if c = '[' then Buffer.add_char stack ']'
       else if c = ')' || c = ']' || c = '}' then begin
         let l = Buffer.length stack in
         if l = 0 || Buffer.nth stack (l - 1) <> c then ok := false else Buffer.truncate stack (l - 1)
       end
       else if c = ';' && Buffer.length stack = 0 then ok := false);
      incr i
    end
  done;
  !ok

(* A natural string comparison: digit runs compare numerically. *)
let compare_natural a z =
  let la = String.length a and lz = String.length z in
  let rec go i j =
    if i < la && j < lz then begin
      let ca = a.[i] and cz = z.[j] in
      if is_digit ca && is_digit cz then begin
        let ei = ref i in
        while !ei < la && is_digit a.[!ei] do incr ei done;
        let ej = ref j in
        while !ej < lz && is_digit z.[!ej] do incr ej done;
        let ra = String.sub a i (!ei - i) and rz = String.sub z j (!ej - j) in
        let strip s =
          let k = ref 0 in
          while !k < String.length s - 1 && s.[!k] = '0' do incr k done;
          let t = after s !k in
          if t = "" then "0" else t
        in
        let na = strip ra and nz = strip rz in
        if String.length na <> String.length nz then String.length na - String.length nz
        else if na <> nz then compare na nz
        else if ra <> rz then compare ra rz
        else go !ei !ej
      end
      else if ca <> cz then Char.code ca - Char.code cz
      else go (i + 1) (j + 1)
    end
    else (la - i) - (lz - j)
  in
  go 0 0

let is_positive_integer v =
  match int_of_string_opt v with Some n -> n >= 0 && string_of_int n = v | None -> false

let is_strict_positive_integer v =
  match int_of_string_opt v with Some n -> n > 0 && string_of_int n = v | None -> false

let quarter_re = Str.regexp "^-?\\(0\\|[1-9][0-9]*\\)\\(\\.[0-9]*[1-9]\\)?$"

(* Whether [v] is a multiple of 0.25 without redundant zeros. *)
let is_multiple_of_quarter v =
  Str.string_match quarter_re v 0
  && match float_of_string_opt v with Some n -> Float.is_finite n && Float.rem n 0.25 = 0. | None -> false

let range_re = Str.regexp "^\\(-?[0-9]+\\)\\.\\.\\(-?[0-9]+\\)\\(\\.\\.\\(-?[0-9]+\\)\\)?$"

(* Brace expansion: `{a,b}` enumerates and `{1..5}` produces ranges. *)
let rec expand_braces pattern =
  let n = String.length pattern in
  let depth = ref 0 and open_ = ref (-1) and close = ref (-1) in
  let i = ref 0 in
  while !i < n && !close = -1 do
    let c = pattern.[!i] in
    if c = '\\' then i := !i + 2
    else begin
      if c = '{' then begin
        if !depth = 0 then open_ := !i;
        incr depth
      end
      else if c = '}' then begin
        if !depth = 0 then Twill_error.failf "Unbalanced braces in `%s`" pattern;
        decr depth;
        if !depth = 0 then close := !i
      end;
      incr i
    end
  done;
  if !depth <> 0 then Twill_error.failf "Unbalanced braces in `%s`" pattern;
  if !open_ = -1 then [ pattern ]
  else
    let prefix = String.sub pattern 0 !open_ in
    let inner = String.sub pattern (!open_ + 1) (!close - !open_ - 1) in
    let suffix = after pattern (!close + 1) in
    let items =
      if Str.string_match range_re inner 0 then begin
        let start = int_of_string (Str.matched_group 1 inner) in
        let end_ = int_of_string (Str.matched_group 2 inner) in
        let step = try abs (int_of_string (Str.matched_group 4 inner)) with Not_found -> 1 in
        if step = 0 then Twill_error.failf "Step cannot be zero in `%s`" pattern;
        let rec up k acc = if k > end_ then List.rev acc else up (k + step) (string_of_int k :: acc) in
        let rec down k acc = if k < end_ then List.rev acc else down (k - step) (string_of_int k :: acc) in
        if end_ < start then down start [] else up start []
      end
      else segment inner ','
    in
    List.concat_map (fun item -> List.map (fun e -> prefix ^ e) (expand_braces (item ^ suffix))) items

(* Strips one pair of surrounding quotes. *)
let unquote value =
  let n = String.length value in
  if n < 2 then None
  else
    let first = value.[0] and last = value.[n - 1] in
    if (first = '"' || first = '\'') && last = first then Some (String.sub value 1 (n - 2)) else None

let math_functions =
  [ "calc"; "min"; "max"; "clamp"; "round"; "mod"; "rem"; "sin"; "cos"; "tan"; "asin"; "acos"; "atan";
    "atan2"; "pow"; "sqrt"; "hypot"; "log"; "exp"; "abs"; "sign" ]

let is_math_function name = List.mem name math_functions

let replace_underscores input =
  if not (String.contains input '_') then input
  else
    let b = Buffer.create (String.length input) in
    let n = String.length input in
    let i = ref 0 in
    while !i < n do
      let c = input.[!i] in
      if c = '\\' && !i + 1 < n && input.[!i + 1] = '_' then (Buffer.add_char b '_'; i := !i + 2)
      else if c = '_' then (Buffer.add_char b ' '; incr i)
      else (Buffer.add_char b c; incr i)
    done;
    Buffer.contents b

let rec decode_value_nodes nodes =
  List.iter
    (function
      | Value.Word w -> w.w <- replace_underscores w.w
      | Value.Sep _ -> ()
      | Value.Fn f ->
          let name = f.fname in
          if name = "url" || has_suffix name "_url" then ()
          else if name = "var" || name = "theme" || name = "--theme" then begin
            let rec split before = function
              | [] -> (List.rev before, [])
              | (Value.Sep "," :: _) as rest -> (List.rev before, rest)
              | x :: rest -> split (x :: before) rest
            in
            let head, tail = split [] f.fnodes in
            List.iter (function Value.Word w -> w.w <- replace_all w.w "\\_" "_" | _ -> ()) head;
            decode_value_nodes tail
          end
          else decode_value_nodes f.fnodes)
    nodes

let is_ident_byte c = is_alnum c || c = '_' || c = '-'

(* Whether the operator at [index] follows a value. *)
let follows_value input index =
  let i = ref (index - 1) in
  while !i >= 0 && input.[!i] = ' ' do decr i done;
  if !i < 0 then false
  else
    let c = input.[!i] in
    if is_digit c || c = '%' || c = ')' then true
    else if is_letter c then begin
      while !i >= 0 && is_letter input.[!i] do decr i done;
      !i >= 0 && (is_digit input.[!i] || input.[!i] = '.')
    end
    else false

(* Whether the input contains a math function call. *)
let has_math_call input =
  let n = String.length input in
  let found = ref false in
  for i = 0 to n - 1 do
    if (not !found) && input.[i] = '(' then begin
      let start = ref i in
      while !start > 0 && is_ident_byte input.[!start - 1] do decr start done;
      if is_math_function (String.sub input !start (i - !start)) then found := true
    end
  done;
  !found

(* Inserts spaces around operators inside math functions. *)
let add_whitespace_around_math_operators input =
  if not (has_math_call input) then input
  else begin
    let b = Buffer.create (String.length input + 8) in
    let n = String.length input in
    let stack = ref [] in
    let in_math () = match !stack with x :: _ -> x | [] -> false in
    let i = ref 0 in
    while !i < n do
      let c = input.[!i] in
      if c = '\\' then begin
        let end_ = min (!i + 2) n in
        Buffer.add_string b (String.sub input !i (end_ - !i));
        i := !i + 2
      end
      else if c = '"' || c = '\'' then begin
        let start = !i in
        incr i;
        let closed = ref false in
        while (not !closed) && !i < n do
          if input.[!i] = '\\' then i := !i + 2 else if input.[!i] = c then closed := true else incr i
        done;
        let end_ = min (!i + 1) n in
        Buffer.add_string b (String.sub input start (end_ - start));
        i := end_
      end
      else if c = '(' then begin
        let start = ref !i in
        while !start > 0 && is_ident_byte input.[!start - 1] do decr start done;
        let name = String.sub input !start (!i - !start) in
        let name = if has_prefix name "-" && not (has_prefix name "--") then after name 1 else name in
        stack := (if name = "" then in_math () else is_math_function name) :: !stack;
        Buffer.add_char b c;
        incr i
      end
      else if c = ')' then begin
        (match !stack with _ :: rest -> stack := rest | [] -> ());
        Buffer.add_char b c;
        incr i
      end
      else if not (in_math ()) then (Buffer.add_char b c; incr i)
      else begin
        let is_operator =
          if c = '*' || c = '/' then true
          else if c = '+' || c = '-' then begin
            let next = ref (!i + 1) in
            while !next < n && input.[!next] = ' ' do incr next done;
            follows_value input !i && !next < n && input.[!next] <> ')'
          end
          else false
        in
        if is_operator then begin
          let len = Buffer.length b in
          if not (len > 0 && Buffer.nth b (len - 1) = ' ') then Buffer.add_char b ' ';
          Buffer.add_char b c;
          Buffer.add_char b ' ';
          incr i;
          while !i < n && input.[!i] = ' ' do incr i done
        end
        else (Buffer.add_char b c; incr i)
      end
    done;
    Buffer.contents b
  end

(* Decodes underscores and spaces math operators. *)
let decode_arbitrary_value input =
  if not (String.contains input '(') then replace_underscores input
  else begin
    let ast = Value.parse input in
    decode_value_nodes ast;
    add_whitespace_around_math_operators (Value.to_css ast)
  end

(* Candidate extraction from arbitrary text (SPEC §13.4). *)

let is_whitespace c = c = ' ' || c = '\t' || c = '\n' || c = '\r' || c = '\012'
let is_quote c = c = '"' || c = '\'' || c = '`'
let is_start_boundary c = is_whitespace c || is_quote c || c = '.' || c = '}' || c = '>'
let is_end_boundary c = is_whitespace c || is_quote c || c = ']' || c = '{' || c = '=' || c = '\\' || c = '<'
let is_letter = Utils.is_letter
let is_digit = Utils.is_digit
let is_name_char c = is_letter c || is_digit c || c = '_' || c = '-'

(* The byte at [i], or NUL past the end. *)
let at text i = if i < 0 || i >= String.length text then '\000' else text.[i]

(* Consumes a balanced bracket group starting at [i] (the opener). Returns
   the index after the closer, or -1 when unbalanced or interrupted by a
   line break. *)
let consume_balanced text i op cl =
  let n = String.length text in
  let rec go j depth =
    if j >= n then -1
    else
      let c = text.[j] in
      if c = '\n' then -1
      else if c = '\\' then go (j + 2) depth
      else if c = op then go (j + 1) (depth + 1)
      else if c = cl then if depth - 1 = 0 then j + 1 else go (j + 1) (depth - 1)
      else go (j + 1) depth
  in
  go i 0

let consume_bracket_suffix text i =
  match at text i with '[' -> consume_balanced text i '[' ']' | '(' -> consume_balanced text i '(' ')' | _ -> -1

(* Consumes `/modifier` at [i] (the slash) and returns the end. *)
let consume_modifier text i =
  let c = at text (i + 1) in
  if c = '[' || c = '(' then
    let e = consume_bracket_suffix text (i + 1) in
    if e = -1 then i else e
  else begin
    let n = String.length text in
    let j = ref (i + 1) in
    while !j < n && (is_name_char text.[!j] || text.[!j] = '.' || text.[!j] = '%') do incr j done;
    if !j = i + 1 then i
    else
      let last = text.[!j - 1] in
      if last = '-' || last = '_' then i else !j
  end

(* Consumes a variant at [i] and returns the index after it (before the
   `:`), or -1. *)
let consume_variant text i =
  let c = at text i in
  if c = '[' then consume_balanced text i '[' ']'
  else if (not (is_letter c)) && c <> '@' then -1
  else begin
    let n = String.length text in
    let j = ref (i + 1) in
    let failed = ref false in
    let continue = ref true in
    while !continue && (not !failed) && !j < n do
      let d = text.[!j] in
      if is_name_char d then incr j
      else if (d = '[' || d = '(') && text.[!j - 1] = '-' then begin
        let e = consume_bracket_suffix text !j in
        if e = -1 then failed := true else j := e
      end
      else continue := false
    done;
    if !failed then -1
    else if !j = i + 1 && c = '@' then -1
    else
      let last = text.[!j - 1] in
      if last = '-' || last = '_' then -1 else if at text !j = '/' then consume_modifier text !j else !j
  end

(* Consumes a utility at [i] and returns the index after it, or -1. *)
let consume_utility text i =
  let n = String.length text in
  let j = ref i in
  if at text !j = '!' then incr j;
  let c = at text !j in
  let ok =
    if c = '[' then begin
      let e = consume_balanced text !j '[' ']' in
      if e = -1 then false
      else
        let found = ref false in
        for k = !j to e - 1 do if text.[k] = ':' then found := true done;
        if not !found then false else (j := e; true)
    end
    else begin
      let start_ok =
        if c = '-' then begin
          let next = at text (!j + 1) in
          if (not (is_letter next)) && not (is_digit next) then false else (incr j; true)
        end
        else is_letter c || c = '@'
      in
      if not start_ok then false
      else begin
        incr j;
        let failed = ref false and continue = ref true in
        while !continue && (not !failed) && !j < n do
          let d = text.[!j] in
          if is_name_char d || d = '%' then incr j
          else if d = '.' then
            if is_digit text.[!j - 1] && is_digit (at text (!j + 1)) then incr j else continue := false
          else if (d = '[' || d = '(') && text.[!j - 1] = '-' then begin
            let e = consume_bracket_suffix text !j in
            if e = -1 then failed := true else j := e
          end
          else continue := false
        done;
        if !failed then false
        else
          let last = text.[!j - 1] in
          not (last = '-' || last = '_')
      end
    end
  in
  if not ok then -1
  else begin
    if at text !j = '/' then j := consume_modifier text !j;
    if at text !j = '!' then incr j;
    !j
  end

(* Consumes a full candidate `(variant ":")* utility` at [i]. *)
let consume_candidate text i =
  let rec go j =
    let e = consume_variant text j in
    if e <> -1 && at text e = ':' then go (e + 1) else j
  in
  consume_utility text (go i)

(* Consumes a `--name` custom property reference at [i]. *)
let consume_variable text i =
  let n = String.length text in
  let j = ref (i + 2) in
  while !j < n && is_name_char text.[!j] do incr j done;
  if !j = i + 2 then -1 else !j

(* An insertion-ordered set of candidates. *)
type candidate_set = { seen : (string, unit) Hashtbl.t; mutable list_rev : string list; mutable size : int }

let new_candidate_set () = { seen = Hashtbl.create 256; list_rev = []; size = 0 }

let add set c =
  if Hashtbl.mem set.seen c then false
  else begin
    Hashtbl.replace set.seen c ();
    set.list_rev <- c :: set.list_rev;
    set.size <- set.size + 1;
    true
  end

let has set c = Hashtbl.mem set.seen c
let to_list set = List.rev set.list_rev

let extract_into text into =
  let length = String.length text in
  let i = ref 0 in
  while !i < length do
    let c = text.[!i] in
    if !i > 0 && not (is_start_boundary text.[!i - 1]) then incr i
    else begin
      let e =
        if c = '-' && at text (!i + 1) = '-' then consume_variable text !i
        else if is_letter c || c = '@' || c = '-' || c = '!' || c = '[' then consume_candidate text !i
        else -1
      in
      if e <> -1 && e > !i && (e = length || is_end_boundary text.[e]) then begin
        ignore (add into (String.sub text !i (e - !i)));
        i := e
      end
      else begin
        (* Skip to the next boundary so that substrings are never emitted. *)
        incr i;
        while !i < length && not (is_start_boundary text.[!i - 1]) do incr i done
      end
    end
  done

(* Extracts every candidate-like span from text in order of first
   appearance. Recall is favored over precision: unknown candidates are
   harmless. *)
let extract_candidates text =
  let set = new_candidate_set () in
  extract_into text set;
  to_list set

/**
 * A small `.gitignore` matcher for source auto-detection (SPEC §13.2).
 *
 * Supports blank lines, `#` comments, `!` negation, trailing `/` for
 * directories, anchoring with `/`, and the `*`, `**`, and `?` wildcards.
 *
 * @module
 */

interface Rule {
  pattern: RegExp;
  negated: boolean;
  directoryOnly: boolean;
  /** Anchored patterns match the full path relative to the ignore file. */
  anchored: boolean;
}

function globToRegExp(glob: string): RegExp {
  let source = "";
  for (let i = 0; i < glob.length; i++) {
    const c = glob[i];
    if (c === "*") {
      if (glob[i + 1] === "*") {
        i++;
        if (glob[i + 1] === "/") {
          i++;
          source += "(?:.*/)?";
        } else {
          source += ".*";
        }
      } else {
        source += "[^/]*";
      }
    } else if (c === "?") {
      source += "[^/]";
    } else if (c === "\\" && i + 1 < glob.length) {
      source += `\\${glob[++i]}`;
    } else if (/[.+^${}()|[\]]/.test(c)) {
      source += `\\${c}`;
    } else {
      source += c;
    }
  }
  return new RegExp(`^${source}$`);
}

export class Gitignore {
  readonly #rules: Rule[] = [];

  /** Parses the content of a `.gitignore` file. */
  constructor(content: string) {
    for (let line of content.split("\n")) {
      line = line.replace(/\r$/, "");
      if (line.trim() === "" || line.startsWith("#")) continue;
      line = line.replace(/(?<!\\)\s+$/, "");
      let negated = false;
      if (line.startsWith("!")) {
        negated = true;
        line = line.slice(1);
      } else if (line.startsWith("\\!") || line.startsWith("\\#")) {
        line = line.slice(1);
      }
      let directoryOnly = false;
      if (line.endsWith("/")) {
        directoryOnly = true;
        line = line.slice(0, -1);
      }
      let anchored = false;
      if (line.startsWith("/")) {
        anchored = true;
        line = line.slice(1);
      } else if (line.includes("/")) {
        anchored = true;
      }
      if (line === "") continue;
      this.#rules.push({
        pattern: globToRegExp(line),
        negated,
        directoryOnly,
        anchored,
      });
    }
  }

  get size(): number {
    return this.#rules.length;
  }

  /**
   * Whether `relativePath` (relative to the directory of the ignore file,
   * using `/` separators) is ignored.
   */
  ignores(relativePath: string, isDirectory: boolean): boolean {
    let ignored = false;
    const basename = relativePath.slice(relativePath.lastIndexOf("/") + 1);
    for (const rule of this.#rules) {
      if (rule.directoryOnly && !isDirectory) continue;
      let matched: boolean;
      if (rule.anchored) {
        matched = rule.pattern.test(relativePath);
      } else {
        matched = rule.pattern.test(basename);
      }
      if (matched) ignored = !rule.negated;
    }
    return ignored;
  }
}

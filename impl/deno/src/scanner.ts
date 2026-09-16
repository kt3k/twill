/**
 * Source scanning: file discovery, candidate extraction, and incremental
 * scans (SPEC §13).
 *
 * @module
 */

import {
  globToRegExp,
  isGlob,
  join,
  normalize,
  relative,
  resolve,
  SEPARATOR,
} from "@std/path";
import type { SourceEntry } from "./directives.ts";
import { extractCandidates } from "./extract.ts";
import { Gitignore } from "./gitignore.ts";

export type { SourceEntry };

const IGNORED_DIRECTORIES = new Set([
  ".git",
  ".hg",
  ".jj",
  ".next",
  ".parcel-cache",
  ".pnpm-store",
  ".svelte-kit",
  ".svn",
  ".turbo",
  ".venv",
  ".vercel",
  ".yarn",
  "__pycache__",
  "node_modules",
  "venv",
]);

const IGNORED_EXTENSIONS = new Set([
  "less",
  "lock",
  "sass",
  "scss",
  "styl",
  "log",
  // Images
  "png",
  "jpg",
  "jpeg",
  "gif",
  "webp",
  "avif",
  "ico",
  "bmp",
  "tif",
  "tiff",
  "heic",
  "psd",
  // Audio
  "mp3",
  "wav",
  "ogg",
  "flac",
  "aac",
  "m4a",
  // Video
  "mp4",
  "webm",
  "mov",
  "avi",
  "mkv",
  "m4v",
  // Archives
  "zip",
  "gz",
  "tar",
  "tgz",
  "bz2",
  "xz",
  "7z",
  "rar",
  // Fonts
  "woff",
  "woff2",
  "ttf",
  "otf",
  "eot",
  // Executables and binaries
  "exe",
  "dll",
  "so",
  "dylib",
  "bin",
  "wasm",
  "pdf",
  "class",
  "jar",
  "pyc",
  "o",
  "a",
]);

const IGNORED_FILES = new Set([
  "package-lock.json",
  "pnpm-lock.yaml",
  "bun.lockb",
  ".gitignore",
  ".env",
]);

function isIgnoredFileName(name: string): boolean {
  if (IGNORED_FILES.has(name)) return true;
  if (name.startsWith(".env.")) return true;
  const dot = name.lastIndexOf(".");
  if (dot > 0) {
    const ext = name.slice(dot + 1).toLowerCase();
    if (IGNORED_EXTENSIONS.has(ext)) return true;
  }
  return false;
}

function toPosix(path: string): string {
  return SEPARATOR === "/" ? path : path.replaceAll(SEPARATOR, "/");
}

/** A compiled exclusion from a negated source. */
interface Exclusion {
  /** Every path at or under this directory is excluded. */
  directory: string | null;
  /** Or: files matching this pattern (absolute, posix). */
  pattern: RegExp | null;
}

async function isDirectory(path: string): Promise<boolean> {
  try {
    return (await Deno.stat(path)).isDirectory;
  } catch {
    return false;
  }
}

async function isFile(path: string): Promise<boolean> {
  try {
    return (await Deno.stat(path)).isFile;
  } catch {
    return false;
  }
}

/**
 * Enumerates source files from globs and auto-detection rules, extracts
 * candidates, and tracks modification times for incremental scans.
 */
export class Scanner {
  readonly #sources: SourceEntry[];
  readonly #candidates = new Set<string>();
  readonly #mtimes = new Map<string, number>();
  #exclusions: Exclusion[] | null = null;

  /** The files read by the most recent `scan()`. */
  scannedFiles: string[] = [];

  constructor(sources: SourceEntry[]) {
    this.#sources = sources;
  }

  /** The sources this scanner was created with. */
  get sources(): SourceEntry[] {
    return this.#sources;
  }

  /** The distinct absolute base directories of the positive sources. */
  get bases(): string[] {
    const bases = new Set<string>();
    for (const source of this.#sources) {
      if (source.negated) continue;
      bases.add(resolve(source.base));
    }
    return [...bases];
  }

  async #exclusionsFor(): Promise<Exclusion[]> {
    if (this.#exclusions !== null) return this.#exclusions;
    const exclusions: Exclusion[] = [];
    for (const source of this.#sources) {
      if (!source.negated) continue;
      const target = resolve(source.base, source.pattern);
      if (!isGlob(source.pattern)) {
        if (await isDirectory(target)) {
          exclusions.push({ directory: target, pattern: null });
          continue;
        }
        exclusions.push({
          directory: null,
          pattern: new RegExp(`^${escapeRegExp(toPosix(target))}$`),
        });
        continue;
      }
      exclusions.push({
        directory: null,
        pattern: globToRegExp(toPosix(normalize(target)), {
          extended: true,
          globstar: true,
        }),
      });
    }
    this.#exclusions = exclusions;
    return exclusions;
  }

  #isExcluded(path: string, exclusions: Exclusion[]): boolean {
    const posix = toPosix(path);
    for (const exclusion of exclusions) {
      if (exclusion.directory !== null) {
        if (
          path === exclusion.directory ||
          path.startsWith(exclusion.directory + SEPARATOR)
        ) {
          return true;
        }
      } else if (exclusion.pattern !== null && exclusion.pattern.test(posix)) {
        return true;
      }
    }
    return false;
  }

  /** Enumerates every file covered by the sources. */
  async files(): Promise<string[]> {
    const exclusions = await this.#exclusionsFor();
    const found = new Set<string>();
    for (const source of this.#sources) {
      if (source.negated) continue;
      const base = resolve(source.base);
      if (source.pattern === "**/*") {
        await this.#walk(base, base, [], exclusions, found);
        continue;
      }
      const target = resolve(base, source.pattern);
      if (!isGlob(source.pattern)) {
        if (await isDirectory(target)) {
          await this.#walk(target, target, [], exclusions, found);
        } else if (await isFile(target)) {
          if (!this.#isExcluded(target, exclusions)) found.add(target);
        }
        continue;
      }
      await this.#glob(base, source.pattern, exclusions, found);
    }
    return [...found];
  }

  async #walk(
    root: string,
    directory: string,
    ignores: { directory: string; matcher: Gitignore }[],
    exclusions: Exclusion[],
    found: Set<string>,
  ): Promise<void> {
    let scoped = ignores;
    try {
      const content = await Deno.readTextFile(join(directory, ".gitignore"));
      const matcher = new Gitignore(content);
      if (matcher.size > 0) scoped = [...ignores, { directory, matcher }];
    } catch {
      // No .gitignore here.
    }

    let entries: Deno.DirEntry[];
    try {
      entries = [];
      for await (const entry of Deno.readDir(directory)) entries.push(entry);
    } catch {
      return;
    }
    entries.sort((a, b) => (a.name < b.name ? -1 : a.name > b.name ? 1 : 0));

    for (const entry of entries) {
      const path = join(directory, entry.name);
      let isDir = entry.isDirectory;
      if (entry.isSymlink) {
        isDir = await isDirectory(path);
      }
      if (isDir) {
        if (IGNORED_DIRECTORIES.has(entry.name)) continue;
        if (this.#isGitignored(path, true, scoped)) continue;
        if (this.#isExcluded(path, exclusions)) continue;
        await this.#walk(root, path, scoped, exclusions, found);
        continue;
      }
      if (isIgnoredFileName(entry.name)) continue;
      if (this.#isGitignored(path, false, scoped)) continue;
      if (this.#isExcluded(path, exclusions)) continue;
      found.add(path);
    }
  }

  #isGitignored(
    path: string,
    isDir: boolean,
    ignores: { directory: string; matcher: Gitignore }[],
  ): boolean {
    for (const { directory, matcher } of ignores) {
      const rel = toPosix(relative(directory, path));
      if (matcher.ignores(rel, isDir)) return true;
    }
    return false;
  }

  async #glob(
    base: string,
    pattern: string,
    exclusions: Exclusion[],
    found: Set<string>,
  ): Promise<void> {
    const absolute = toPosix(normalize(resolve(base, pattern)));
    const regexp = globToRegExp(absolute, { extended: true, globstar: true });
    // Walk from the first non-glob segment of the absolute pattern.
    const segments = absolute.split("/");
    const fixed: string[] = [];
    for (const segment of segments) {
      if (isGlob(segment)) break;
      fixed.push(segment);
    }
    let start = fixed.join("/");
    if (start === "") start = "/";
    if (await isFile(start)) {
      if (!this.#isExcluded(start, exclusions)) found.add(start);
      return;
    }
    await this.#walkAll(start, regexp, exclusions, found);
  }

  async #walkAll(
    directory: string,
    regexp: RegExp,
    exclusions: Exclusion[],
    found: Set<string>,
  ): Promise<void> {
    let entries: Deno.DirEntry[];
    try {
      entries = [];
      for await (const entry of Deno.readDir(directory)) entries.push(entry);
    } catch {
      return;
    }
    entries.sort((a, b) => (a.name < b.name ? -1 : a.name > b.name ? 1 : 0));
    for (const entry of entries) {
      const path = join(directory, entry.name);
      const isDir = entry.isSymlink
        ? await isDirectory(path)
        : entry.isDirectory;
      if (isDir) {
        if (entry.name === ".git") continue;
        if (this.#isExcluded(path, exclusions)) continue;
        await this.#walkAll(path, regexp, exclusions, found);
        continue;
      }
      if (!regexp.test(toPosix(path))) continue;
      if (this.#isExcluded(path, exclusions)) continue;
      found.add(path);
    }
  }

  async #read(path: string): Promise<boolean> {
    let content: string;
    try {
      content = await Deno.readTextFile(path);
    } catch {
      return false;
    }
    extractCandidates(content, this.#candidates);
    return true;
  }

  /**
   * Walks every source, reads files whose modification time changed since
   * the last scan (all files on the first scan), and returns the full
   * deduplicated candidate set seen so far.
   */
  async scan(): Promise<string[]> {
    const files = await this.files();
    const read: string[] = [];
    for (const path of files) {
      let mtime: number;
      try {
        const stat = await Deno.stat(path);
        mtime = stat.mtime?.getTime() ?? 0;
      } catch {
        continue;
      }
      if (this.#mtimes.get(path) === mtime) continue;
      this.#mtimes.set(path, mtime);
      if (await this.#read(path)) read.push(path);
    }
    this.scannedFiles = read;
    return [...this.#candidates];
  }

  /**
   * Reads only the given files and returns the candidates not seen before.
   */
  async scanFiles(changed: string[]): Promise<string[]> {
    const before = new Set(this.#candidates);
    for (const path of changed) {
      try {
        const stat = await Deno.stat(path);
        if (!stat.isFile) continue;
        this.#mtimes.set(path, stat.mtime?.getTime() ?? 0);
      } catch {
        continue;
      }
      await this.#read(path);
    }
    const fresh: string[] = [];
    for (const candidate of this.#candidates) {
      if (!before.has(candidate)) fresh.push(candidate);
    }
    return fresh;
  }

  /** Every candidate seen so far. */
  get candidates(): string[] {
    return [...this.#candidates];
  }
}

function escapeRegExp(text: string): string {
  return text.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}

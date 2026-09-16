/**
 * The Twill command-line interface (SPEC §14).
 *
 * ```sh
 * deno run -A jsr:@kt3k/twill/cli -i input.css -o output.css --watch
 * ```
 *
 * @module
 */

import { parseArgs } from "@std/cli/parse-args";
import { basename, dirname, fromFileUrl, resolve } from "@std/path";
import { ensureDir } from "@std/fs/ensure-dir";
import {
  isBuiltinPath,
  resolveBuiltin,
  type StylesheetLoader,
} from "./src/builtin.ts";
import { compile, type Compiler } from "./src/compile.ts";
import { parse } from "./src/parser.ts";
import { Scanner, type SourceEntry } from "./src/scanner.ts";
import { serializeCompact } from "./src/serializer.ts";

const USAGE = `Usage: twill [build] [options]

Options:
  -i, --input <path>     Entry stylesheet ("-" reads stdin; default: @import 'twill';)
  -o, --output <path>    Output file ("-" writes stdout; default: -)
  -w, --watch [always]   Rebuild on changes ("always" keeps watching after stdin closes)
      --poll [ms]        Poll for changes instead of using filesystem events (default: 250)
  -m, --minify           Optimize and minify the output
      --optimize         Optimize without minifying
      --cwd <dir>        Base directory (default: .)
      --silent           Suppress everything except errors
  -h, --help             Show this help
`;

const DEFAULT_INPUT = "@import 'twill';\n";
const DEFAULT_POLL_INTERVAL = 250;

export interface CliOptions {
  input: string | null;
  output: string;
  watch: false | true | "always";
  poll: number | null;
  minify: boolean;
  optimize: boolean;
  cwd: string;
  silent: boolean;
  help: boolean;
}

/** Parses command-line arguments. Throws on invalid input. */
export function parseCliArgs(argv: string[]): CliOptions {
  // `--watch [always]` and `--poll [ms]` take an optional value.
  const rest: string[] = [];
  let watch: false | true | "always" = false;
  let poll: number | null = null;
  for (let i = 0; i < argv.length; i++) {
    const arg = argv[i];
    if (arg === "-w" || arg === "--watch") {
      watch = true;
      if (argv[i + 1] === "always") {
        watch = "always";
        i++;
      }
      continue;
    }
    if (arg === "--watch=always" || arg === "-w=always") {
      watch = "always";
      continue;
    }
    if (arg === "--poll") {
      poll = DEFAULT_POLL_INTERVAL;
      if (argv[i + 1] !== undefined && /^\d+$/.test(argv[i + 1])) {
        poll = Number(argv[i + 1]);
        i++;
      }
      continue;
    }
    if (arg.startsWith("--poll=")) {
      const value = arg.slice("--poll=".length);
      if (!/^\d+$/.test(value)) {
        throw new Error(`Invalid --poll interval: ${value}`);
      }
      poll = Number(value);
      continue;
    }
    rest.push(arg);
  }
  if (poll !== null && poll <= 0) {
    throw new Error(
      "The --poll interval must be a positive number of milliseconds.",
    );
  }

  const parsed = parseArgs(rest, {
    string: ["input", "output", "cwd"],
    boolean: ["minify", "optimize", "silent", "help"],
    alias: { i: "input", o: "output", m: "minify", h: "help" },
    default: { output: "-", cwd: "." },
    unknown(arg) {
      if (arg === "build") return false;
      if (arg.startsWith("-")) throw new Error(`Unknown option: ${arg}`);
      return false;
    },
  });

  return {
    input: parsed.input ?? null,
    output: parsed.output,
    watch,
    poll,
    minify: parsed.minify,
    optimize: parsed.optimize,
    cwd: parsed.cwd,
    silent: parsed.silent,
    help: parsed.help,
  };
}

interface Io {
  stdout(text: string): Promise<void>;
  stderr(text: string): void;
}

const defaultIo: Io = {
  async stdout(text) {
    // `Deno.stdout.write` may write only part of the buffer.
    const bytes = new TextEncoder().encode(text);
    let written = 0;
    while (written < bytes.length) {
      written += await Deno.stdout.write(bytes.subarray(written));
    }
  },
  stderr(text) {
    Deno.stderr.writeSync(new TextEncoder().encode(text));
  },
};

/** Creates a loader resolving relative ids against the filesystem. */
export function createLoader(fullRebuildPaths: Set<string>): StylesheetLoader {
  return async (id, base) => {
    const builtin = resolveBuiltin(id, base);
    if (builtin !== null) return builtin;
    if (isBuiltinPath(base)) {
      throw new Error(`Cannot import \`${id}\` from a built-in stylesheet.`);
    }
    const path = resolve(base, id);
    const content = await Deno.readTextFile(path);
    fullRebuildPaths.add(path);
    return { path, base: dirname(path), content };
  };
}

/** Assembles the source set (SPEC §13.1). */
export function assembleSources(
  compiler: Compiler,
  options: { cwd: string; inputPath: string | null; executable: string | null },
): SourceEntry[] {
  const sources: SourceEntry[] = [];
  if (compiler.root === "none") {
    // Auto-detection disabled.
  } else if (compiler.root === null) {
    sources.push({ base: options.cwd, pattern: "**/*", negated: false });
  } else {
    sources.push({ ...compiler.root, negated: false });
  }
  sources.push(...compiler.sources);
  if (options.executable !== null) {
    sources.push({
      base: dirname(options.executable),
      pattern: basename(options.executable),
      negated: true,
    });
  }
  if (options.inputPath !== null) {
    sources.push({
      base: dirname(options.inputPath),
      pattern: basename(options.inputPath),
      negated: false,
    });
  }
  return sources;
}

function executablePath(): string | null {
  try {
    const main = Deno.mainModule;
    if (main.startsWith("file:")) return fromFileUrl(main);
  } catch {
    // Not available.
  }
  return null;
}

/** Minifies CSS by re-serializing it compactly. */
export function minifyCss(css: string): string {
  return serializeCompact(parse(css));
}

function formatDuration(ms: number): string {
  if (ms < 1000) return `${Math.round(ms)}ms`;
  return `${(ms / 1000).toFixed(2)}s`;
}

interface State {
  compiler: Compiler;
  scanner: Scanner;
  fullRebuildPaths: Set<string>;
}

/** Runs the CLI. Returns the exit code. */
export async function main(
  argv: string[],
  io: Io = defaultIo,
): Promise<number> {
  let options: CliOptions;
  try {
    options = parseCliArgs(argv);
  } catch (error) {
    io.stderr(`${(error as Error).message}\n\n${USAGE}`);
    return 1;
  }

  if (options.help) {
    io.stderr(USAGE);
    return 0;
  }
  if (argv.length === 0 && Deno.stdin.isTerminal()) {
    io.stderr(USAGE);
    return 0;
  }

  const cwd = resolve(options.cwd);
  const inputPath = options.input !== null && options.input !== "-"
    ? resolve(cwd, options.input)
    : null;
  const outputPath = options.output !== "-"
    ? resolve(cwd, options.output)
    : null;

  if (inputPath !== null) {
    try {
      const stat = await Deno.stat(inputPath);
      if (!stat.isFile) throw new Error();
    } catch {
      io.stderr(`Specified input file \`${options.input}\` does not exist.\n`);
      return 1;
    }
  }
  if (inputPath !== null && outputPath !== null && inputPath === outputPath) {
    io.stderr("Specified input file and output file are identical.\n");
    return 1;
  }

  if (!options.silent) io.stderr("twill\n\n");

  const readInput = async (): Promise<string> => {
    if (inputPath !== null) return await Deno.readTextFile(inputPath);
    if (options.input === "-") {
      return await new Response(Deno.stdin.readable).text();
    }
    return DEFAULT_INPUT;
  };

  const executable = executablePath();
  let previousCss: string | null = null;
  let previousWritten: string | null = null;

  const write = async (css: string): Promise<void> => {
    let output = css;
    if (options.minify || options.optimize) {
      output = css === previousCss && previousWritten !== null
        ? previousWritten
        : minifyCss(css);
    }
    previousCss = css;
    previousWritten = output;
    if (outputPath !== null) {
      await ensureDir(dirname(outputPath));
      await Deno.writeTextFile(outputPath, output);
    } else if (output !== lastStdout) {
      await io.stdout(output);
    }
    lastStdout = output;
  };
  let lastStdout: string | null = null;

  const createState = async (): Promise<State> => {
    const css = await readInput();
    const fullRebuildPaths = new Set<string>();
    if (inputPath !== null) fullRebuildPaths.add(inputPath);
    const compiler = await compile(css, {
      base: inputPath !== null ? dirname(inputPath) : cwd,
      loadStylesheet: createLoader(fullRebuildPaths),
    });
    const sources = assembleSources(compiler, { cwd, inputPath, executable });
    const scanner = new Scanner(sources);
    return { compiler, scanner, fullRebuildPaths };
  };

  const timed = async (fn: () => Promise<void>): Promise<void> => {
    const start = performance.now();
    await fn();
    if (!options.silent) {
      io.stderr(`Done in ${formatDuration(performance.now() - start)}\n`);
    }
  };

  let state: State;
  try {
    state = await createState();
    await timed(async () => {
      const candidates = await state.scanner.scan();
      await write(state.compiler.build(candidates));
    });
  } catch (error) {
    io.stderr(`${(error as Error).message}\n`);
    return 1;
  }

  if (!options.watch && options.poll === null) return 0;

  const fullRebuild = async (): Promise<void> => {
    const previous = state.fullRebuildPaths;
    try {
      const next = await createState();
      const candidates = await next.scanner.scan();
      state = next;
      await write(state.compiler.build(candidates));
    } catch (error) {
      // Keep the previous dependency list so a later change to a deleted
      // dependency still triggers a rebuild.
      state.fullRebuildPaths = previous;
      throw error;
    }
  };

  const report = (error: unknown) => {
    io.stderr(`${(error as Error).message}\n`);
  };

  if (options.poll !== null) {
    const interval = options.poll;
    await new Promise<void>(() => {
      const tick = async () => {
        try {
          const candidates = await state.scanner.scan();
          const files = state.scanner.scannedFiles.filter((f) =>
            f !== outputPath
          );
          if (files.length === 0) return;
          if (files.some((f) => state.fullRebuildPaths.has(f))) {
            await timed(fullRebuild);
          } else if (candidates.length > 0) {
            await timed(async () => {
              await write(state.compiler.build(candidates));
            });
          }
        } catch (error) {
          report(error);
        }
      };
      let running = false;
      setInterval(async () => {
        if (running) return;
        running = true;
        try {
          await tick();
        } finally {
          running = false;
        }
      }, interval);
    });
    return 0;
  }

  // Event-driven watch mode.
  const watcherRef: { current: Deno.FsWatcher | null } = { current: null };
  let pending = new Set<string>();
  let timer: ReturnType<typeof setTimeout> | null = null;
  let processing: Promise<void> = Promise.resolve();

  const handle = (paths: string[]) => {
    processing = processing.then(async () => {
      if (paths.length === 0) return;
      if (paths.every((p) => p === outputPath)) return;
      try {
        if (paths.some((p) => state.fullRebuildPaths.has(p))) {
          await timed(fullRebuild);
          startWatcher();
          return;
        }
        const fresh = await state.scanner.scanFiles(
          paths.filter((p) => p !== outputPath),
        );
        if (fresh.length === 0) return;
        await timed(async () => {
          await write(state.compiler.build(fresh));
        });
      } catch (error) {
        report(error);
      }
    });
  };

  const startWatcher = () => {
    watcherRef.current?.close();
    const bases = new Set<string>(state.scanner.bases);
    for (const path of state.fullRebuildPaths) {
      if (!isBuiltinPath(path)) bases.add(dirname(path));
    }
    const roots = [...bases].filter((b) => {
      try {
        return Deno.statSync(b).isDirectory;
      } catch {
        return false;
      }
    });
    if (roots.length === 0) return;
    const watcher = Deno.watchFs(roots, { recursive: true });
    watcherRef.current = watcher;
    (async () => {
      try {
        for await (const event of watcher) {
          for (const path of event.paths) pending.add(path);
          if (timer !== null) clearTimeout(timer);
          timer = setTimeout(() => {
            timer = null;
            const paths = [...pending];
            pending = new Set();
            handle(paths);
          }, 50);
        }
      } catch {
        // The watcher was closed.
      }
    })();
  };

  startWatcher();

  if (options.watch !== "always") {
    // Exit when stdin reaches end of file.
    try {
      for await (const _ of Deno.stdin.readable) {
        // Discard input.
      }
    } catch {
      // stdin unavailable.
    }
    await processing;
    watcherRef.current?.close();
    return 0;
  }

  await new Promise<void>(() => {});
  return 0;
}

if (import.meta.main) {
  Deno.exit(await main(Deno.args));
}

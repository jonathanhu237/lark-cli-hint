import { analyzeBaseline } from "./analyzer.js";
import { createTranslator, resolveLocale } from "./i18n.js";
import { createJsonEnvelope, renderHintCard } from "./renderer.js";
import { runCommand } from "./runner.js";
import type { JsonEnvelope, LocaleCode, OutputMode, SourceStream } from "./types.js";

export class MissingCommandError extends Error {
  constructor() {
    super("Missing wrapped command.");
    this.name = "MissingCommandError";
  }
}

export interface RunAppOptions {
  command: string[];
  mode: OutputMode;
  locale?: string;
  cwd?: string;
  env?: NodeJS.ProcessEnv;
  stdout?: SourceStream;
  stderr?: SourceStream;
  captureLimitBytes?: number;
}

export interface RunAppResult {
  exitCode: number;
  locale: LocaleCode;
  envelope: JsonEnvelope;
}

export async function runApp(options: RunAppOptions): Promise<RunAppResult> {
  if (options.command.length === 0) {
    throw new MissingCommandError();
  }

  const locale = resolveLocale(options.locale, options.env);
  const t = createTranslator(locale);
  const humanMode = options.mode === "human";

  const result = await runCommand({
    command: options.command,
    cwd: options.cwd,
    env: options.env,
    inheritStdin: humanMode,
    streamOutput: humanMode,
    stdout: options.stdout,
    stderr: options.stderr,
    captureLimitBytes: options.captureLimitBytes
  });

  const hint = analyzeBaseline(result, t);
  const envelope = createJsonEnvelope(result, hint);

  if (options.mode === "json") {
    options.stdout?.write(`${JSON.stringify(envelope, null, 2)}\n`);
  } else {
    options.stdout?.write(`\n${renderHintCard(hint, t)}\n`);
  }

  return {
    exitCode: result.exitCode,
    locale,
    envelope
  };
}

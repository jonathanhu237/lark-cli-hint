import { spawn } from "node:child_process";
import type { RunCommandResult, SourceStream } from "./types.js";

const defaultCaptureLimitBytes = 256 * 1024;

export interface CommandRunnerOptions {
  command: string[];
  cwd?: string;
  env?: NodeJS.ProcessEnv;
  inheritStdin?: boolean;
  streamOutput?: boolean;
  stdout?: SourceStream;
  stderr?: SourceStream;
  captureLimitBytes?: number;
}

export function runCommand(options: CommandRunnerOptions): Promise<RunCommandResult> {
  const [executable, ...args] = options.command;
  const limit = options.captureLimitBytes ?? defaultCaptureLimitBytes;

  return new Promise((resolve) => {
    let stdout = "";
    let stderr = "";
    let settled = false;

    const child = spawn(executable, args, {
      cwd: options.cwd,
      env: options.env,
      stdio: [options.inheritStdin ? "inherit" : "ignore", "pipe", "pipe"]
    });

    child.stdout?.on("data", (chunk: Buffer) => {
      stdout = appendBounded(stdout, chunk, limit);
      if (options.streamOutput) {
        options.stdout?.write(chunk);
      }
    });

    child.stderr?.on("data", (chunk: Buffer) => {
      stderr = appendBounded(stderr, chunk, limit);
      if (options.streamOutput) {
        options.stderr?.write(chunk);
      }
    });

    child.on("error", (error) => {
      if (settled) {
        return;
      }

      settled = true;
      stderr = appendBounded(stderr, Buffer.from(error.message), limit);
      if (options.streamOutput) {
        options.stderr?.write(`${error.message}\n`);
      }

      resolve({
        command: options.command,
        exitCode: 127,
        signal: null,
        stdout,
        stderr
      });
    });

    child.on("close", (code, signal) => {
      if (settled) {
        return;
      }

      settled = true;
      resolve({
        command: options.command,
        exitCode: code ?? 1,
        signal,
        stdout,
        stderr
      });
    });
  });
}

function appendBounded(current: string, chunk: Buffer, limit: number): string {
  const combined = Buffer.concat([Buffer.from(current), chunk]);
  if (combined.byteLength <= limit) {
    return combined.toString("utf8");
  }

  return combined.subarray(combined.byteLength - limit).toString("utf8");
}

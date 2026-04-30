import { execFile } from "node:child_process";
import { mkdtemp, mkdir, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";

const repoRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const tsxCli = resolve(repoRoot, "node_modules/tsx/dist/cli.mjs");
const cliPath = resolve(repoRoot, "src/cli.ts");

interface ExecResult {
  stdout: string;
  stderr: string;
  exitCode: number;
}

function runCli(args: string[], options: { cwd?: string; env?: NodeJS.ProcessEnv } = {}): Promise<ExecResult> {
  return new Promise((resolveResult) => {
    execFile(
      process.execPath,
      [tsxCli, cliPath, ...args],
      {
        cwd: options.cwd ?? repoRoot,
        env: {
          ...process.env,
          ...options.env
        },
        maxBuffer: 1024 * 1024
      },
      (error, stdout, stderr) => {
        resolveResult({
          stdout,
          stderr,
          exitCode: typeof error?.code === "number" ? error.code : 0
        });
      }
    );
  });
}

describe("lark-cli-hint CLI", () => {
  it("passes the command after -- through in JSON mode", async () => {
    const result = await runCli([
      "run",
      "--json",
      "--",
      process.execPath,
      "-e",
      "process.stdout.write('cli-json')"
    ]);

    const envelope = JSON.parse(result.stdout);

    expect(result.exitCode).toBe(0);
    expect(result.stderr).toBe("");
    expect(envelope.stdout).toBe("cli-json");
    expect(envelope.command.raw).toEqual([
      process.execPath,
      "-e",
      "process.stdout.write('cli-json')"
    ]);
  });

  it("streams human output before the Hint Card through the CLI entry", async () => {
    const result = await runCli([
      "run",
      "--",
      process.execPath,
      "-e",
      "process.stdout.write('cli-human')"
    ]);

    expect(result.exitCode).toBe(0);
    expect(result.stdout.indexOf("cli-human")).toBeGreaterThanOrEqual(0);
    expect(result.stdout.indexOf("Status")).toBeGreaterThan(result.stdout.indexOf("cli-human"));
  });

  it("does not load locale files from the caller cwd", async () => {
    const cwd = await mkdtemp(resolve(tmpdir(), "lark-cli-hint-"));
    await mkdir(resolve(cwd, "locales"));
    await writeFile(
      resolve(cwd, "locales", "en-US.json"),
      JSON.stringify({ labels: { status: "POLLUTED" } })
    );

    const result = await runCli(
      [
        "run",
        "--",
        process.execPath,
        "-e",
        ""
      ],
      {
        cwd,
        env: { LANG: "en_US.UTF-8" }
      }
    );

    expect(result.exitCode).toBe(0);
    expect(result.stdout).toContain("Status");
    expect(result.stdout).not.toContain("POLLUTED");
  });

  it("allows wrapped commands to read from stdin in human mode", async () => {
    const result = await new Promise<ExecResult>((resolveResult) => {
      const child = execFile(
        process.execPath,
        [
          tsxCli,
          cliPath,
          "run",
          "--",
          process.execPath,
          "-e",
          "process.stdin.pipe(process.stdout)"
        ],
        {
          cwd: repoRoot,
          env: {
            ...process.env,
            LANG: "en_US.UTF-8"
          },
          maxBuffer: 1024 * 1024
        },
        (error, stdout, stderr) => {
          resolveResult({
            stdout,
            stderr,
            exitCode: typeof error?.code === "number" ? error.code : 0
          });
        }
      );

      child.stdin?.end("from-stdin");
    });

    expect(result.exitCode).toBe(0);
    expect(result.stdout).toContain("from-stdin");
    expect(result.stdout.indexOf("Status")).toBeGreaterThan(result.stdout.indexOf("from-stdin"));
  });
});

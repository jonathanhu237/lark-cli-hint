import { execFile } from "node:child_process";
import { chmod, mkdtemp, mkdir, writeFile } from "node:fs/promises";
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

async function createFakeLarkCli(script: string): Promise<string> {
  const directory = await mkdtemp(resolve(tmpdir(), "fake-lark-cli-"));
  const executable = resolve(directory, "lark-cli");
  await writeFile(executable, `#!/usr/bin/env node\n${script}\n`);
  await chmod(executable, 0o755);
  return directory;
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

  it("emits docs workflow JSON hints through the CLI entry", async () => {
    const fakeBin = await createFakeLarkCli(`
const [, , domain, operation] = process.argv;
if (domain === "docs" && operation === "+search") {
  process.stdout.write(JSON.stringify({
    ok: true,
    data: {
      items: [{ title: "CLI Fixture Doc", doc_token: "doccn_cli_fixture" }]
    }
  }));
  process.exit(0);
}
process.stderr.write("unexpected command");
process.exit(1);
`);

    const result = await runCli(
      ["run", "--json", "--", "lark-cli", "docs", "+search", "--query", "demo"],
      {
        env: {
          PATH: `${fakeBin}:${process.env.PATH ?? ""}`,
          LANG: "en_US.UTF-8"
        }
      }
    );

    const envelope = JSON.parse(result.stdout);

    expect(result.exitCode).toBe(0);
    expect(envelope.hint.next.command).toBe("lark-cli docs +fetch --doc doccn_cli_fixture");
  });

  it("renders docs workflow human Hint Card through the CLI entry", async () => {
    const fakeBin = await createFakeLarkCli(`
const [, , domain, operation] = process.argv;
if (domain === "docs" && operation === "+fetch") {
  process.stdout.write(JSON.stringify({
    ok: true,
    data: { title: "CLI Fetch Doc", content: "body" }
  }));
  process.exit(0);
}
process.stderr.write("unexpected command");
process.exit(1);
`);

    const result = await runCli(
      ["run", "--", "lark-cli", "docs", "+fetch", "--doc", "doccn_cli_fetch"],
      {
        env: {
          PATH: `${fakeBin}:${process.env.PATH ?? ""}`,
          LANG: "en_US.UTF-8"
        }
      }
    );

    expect(result.exitCode).toBe(0);
    expect(result.stdout).toContain("CLI Fetch Doc");
    expect(result.stdout).toContain("Next");
    expect(result.stdout).toContain("lark-cli im +messages-send --chat-id <chat_id> --markdown");
  });
});

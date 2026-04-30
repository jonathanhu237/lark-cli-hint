import { describe, expect, it } from "vitest";
import { MissingCommandError, runApp } from "../src/app.js";

class MemoryStream {
  private chunks: string[] = [];

  write(chunk: string | Uint8Array): void {
    this.chunks.push(typeof chunk === "string" ? chunk : Buffer.from(chunk).toString("utf8"));
  }

  toString(): string {
    return this.chunks.join("");
  }
}

describe("runApp", () => {
  it("reports an error when no wrapped command is provided", async () => {
    await expect(
      runApp({
        command: [],
        mode: "json",
        env: { LANG: "en_US.UTF-8" }
      })
    ).rejects.toBeInstanceOf(MissingCommandError);
  });

  it("emits one JSON envelope for a successful wrapped command", async () => {
    const stdout = new MemoryStream();

    const result = await runApp({
      command: [process.execPath, "-e", "process.stdout.write('ok')"],
      mode: "json",
      env: { LANG: "en_US.UTF-8" },
      stdout
    });

    const envelope = JSON.parse(stdout.toString());

    expect(result.exitCode).toBe(0);
    expect(envelope.exitCode).toBe(0);
    expect(envelope.stdout).toBe("ok");
    expect(envelope.stderr).toBe("");
    expect(envelope.hint.kind).toBe("success");
    expect(envelope.hint.next.command).toBeNull();
    expect(envelope.command.raw).toEqual([process.execPath, "-e", "process.stdout.write('ok')"]);
  });

  it("emits one JSON envelope for a failed wrapped command", async () => {
    const stdout = new MemoryStream();

    const result = await runApp({
      command: [process.execPath, "-e", "process.stderr.write('bad'); process.exit(7)"],
      mode: "json",
      env: { LANG: "en_US.UTF-8" },
      stdout
    });

    const envelope = JSON.parse(stdout.toString());

    expect(result.exitCode).toBe(7);
    expect(envelope.exitCode).toBe(7);
    expect(envelope.stdout).toBe("");
    expect(envelope.stderr).toBe("bad");
    expect(envelope.hint.kind).toBe("failure");
    expect(envelope.hint.status).toContain("7");
  });

  it("streams wrapped stdout before appending the human Hint Card", async () => {
    const stdout = new MemoryStream();

    await runApp({
      command: [process.execPath, "-e", "process.stdout.write('wrapped-output')"],
      mode: "human",
      env: { LANG: "en_US.UTF-8" },
      stdout
    });

    const output = stdout.toString();
    const wrappedIndex = output.indexOf("wrapped-output");
    const hintIndex = output.indexOf("Status");

    expect(wrappedIndex).toBeGreaterThanOrEqual(0);
    expect(hintIndex).toBeGreaterThan(wrappedIndex);
  });

  it("streams wrapped stderr before appending a failure Hint Card", async () => {
    const stdout = new MemoryStream();
    const stderr = new MemoryStream();

    const result = await runApp({
      command: [process.execPath, "-e", "process.stderr.write('wrapped-error'); process.exit(5)"],
      mode: "human",
      env: { LANG: "en_US.UTF-8" },
      stdout,
      stderr
    });

    expect(result.exitCode).toBe(5);
    expect(stderr.toString()).toBe("wrapped-error");
    expect(stdout.toString()).toContain("Command failed with exit code 5.");
  });

  it("keeps captured output bounded", async () => {
    const stdout = new MemoryStream();

    await runApp({
      command: [process.execPath, "-e", "process.stdout.write('abcdef')"],
      mode: "json",
      env: { LANG: "en_US.UTF-8" },
      stdout,
      captureLimitBytes: 4
    });

    const envelope = JSON.parse(stdout.toString());

    expect(envelope.stdout).toBe("cdef");
  });

  it("renders baseline Hint Card in Simplified Chinese for Chinese environments", async () => {
    const stdout = new MemoryStream();

    await runApp({
      command: [process.execPath, "-e", ""],
      mode: "human",
      env: { LANG: "zh_CN.UTF-8" },
      stdout
    });

    const output = stdout.toString();

    expect(output).toContain("状态");
    expect(output).toContain("提示");
    expect(output).toContain("下一步");
    expect(output).toContain("暂无可信的下一条命令。");
  });
});

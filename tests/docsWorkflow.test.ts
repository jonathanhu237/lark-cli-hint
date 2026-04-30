import { readFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";
import { analyze } from "../src/analyzer.js";
import { createTranslator } from "../src/i18n.js";
import { createJsonEnvelope, renderHintCard } from "../src/renderer.js";
import type { RunCommandResult } from "../src/types.js";

const fixtureDir = resolve(dirname(fileURLToPath(import.meta.url)), "..", "fixtures");

function fixture(name: string): string {
  return readFileSync(resolve(fixtureDir, name), "utf8");
}

function result(overrides: Partial<RunCommandResult>): RunCommandResult {
  return {
    command: ["lark-cli", "--version"],
    exitCode: 0,
    signal: null,
    stdout: "",
    stderr: "",
    ...overrides
  };
}

describe("docs workflow analyzer", () => {
  it("suggests docs +fetch --doc for successful docs search results", () => {
    const t = createTranslator("en-US");
    const hint = analyze(result({
      command: ["lark-cli", "docs", "+search", "--query", "AI challenge"],
      stdout: fixture("docs-search.success.json")
    }), t);

    expect(hint.kind).toBe("success");
    expect(hint.next.command).toBe("lark-cli docs +fetch --doc https://example.feishu.cn/docx/demo_project_brief");
    expect(hint.sources.map((source) => source.type)).toContain("stdout");
  });

  it("falls back to baseline when docs search has no usable document", () => {
    const t = createTranslator("en-US");
    const hint = analyze(result({
      command: ["lark-cli", "docs", "+search", "--query", "AI challenge"],
      stdout: fixture("docs-search.no-document.json")
    }), t);

    expect(hint.status).toBe("Command completed successfully.");
    expect(hint.next.command).toBeNull();
    expect(hint.next.text).toBe("No confident next command.");
  });

  it("suggests im +messages-send after successful docs fetch", () => {
    const t = createTranslator("en-US");
    const hint = analyze(result({
      command: ["lark-cli", "docs", "+fetch", "--doc", "doccn_demo_project_brief"],
      stdout: fixture("docs-fetch.success.json")
    }), t);

    expect(hint.kind).toBe("success");
    expect(hint.next.command).toContain("lark-cli im +messages-send --chat-id <chat_id> --markdown");
    expect(hint.next.command).toContain("AI Challenge Project Brief");
  });

  it("recovers outdated docs +fetch --doc-token usage", () => {
    const t = createTranslator("en-US");
    const hint = analyze(result({
      command: ["lark-cli", "docs", "+fetch", "--doc-token", "doccn_demo_project_brief"],
      exitCode: 2,
      stdout: fixture("docs-fetch.doc-token-error.json")
    }), t);

    expect(hint.kind).toBe("failure");
    expect(hint.status).toContain("outdated flag");
    expect(hint.next.command).toBe("lark-cli docs +fetch --doc doccn_demo_project_brief");
  });

  it("recovers missing docs +fetch --doc usage", () => {
    const t = createTranslator("en-US");
    const hint = analyze(result({
      command: ["lark-cli", "docs", "+fetch"],
      exitCode: 2,
      stdout: fixture("docs-fetch.missing-doc-error.json")
    }), t);

    expect(hint.kind).toBe("failure");
    expect(hint.status).toContain("missing a document");
    expect(hint.next.command).toBe("lark-cli docs +search --query <project_keyword>");
  });

  it("recovers wiki-looking docs +fetch --doc values", () => {
    const t = createTranslator("en-US");
    const hint = analyze(result({
      command: ["lark-cli", "docs", "+fetch", "--doc", "wiki_demo_node"],
      exitCode: 2,
      stdout: fixture("docs-fetch.wiki-token-error.json")
    }), t);

    expect(hint.kind).toBe("failure");
    expect(hint.status).toContain("wiki node token");
    expect(hint.next.command).toBe("lark-cli docs +search --query <project_keyword>");
  });

  it("recovers not configured docs fetch failures", () => {
    const t = createTranslator("en-US");
    const hint = analyze(result({
      command: ["lark-cli", "docs", "+fetch", "--doc", "doccn_demo_project_brief"],
      exitCode: 2,
      stdout: fixture("docs-fetch.not-configured-error.json")
    }), t);

    expect(hint.kind).toBe("failure");
    expect(hint.status).toContain("not configured");
    expect(hint.next.command).toBe("lark-cli config init --new");
  });

  it("recovers unsupported identity docs fetch failures", () => {
    const t = createTranslator("en-US");
    const hint = analyze(result({
      command: ["lark-cli", "docs", "+fetch", "--as", "bot", "--doc", "doccn_demo_project_brief"],
      exitCode: 2,
      stdout: fixture("docs-fetch.identity-error.json")
    }), t);

    expect(hint.kind).toBe("failure");
    expect(hint.status).toContain("identity is unsupported");
    expect(hint.next.command).toBe("lark-cli docs +fetch --as user --doc doccn_demo_project_brief");
  });

  it("uses generic docs fetch failure when no known recovery matches", () => {
    const t = createTranslator("en-US");
    const hint = analyze(result({
      command: ["lark-cli", "docs", "+fetch", "--doc", "doccn_demo_project_brief"],
      exitCode: 1,
      stderr: fixture("docs-fetch.generic-error.txt")
    }), t);

    expect(hint.kind).toBe("failure");
    expect(hint.status).toContain("docs +fetch failed");
    expect(hint.next.command).toBeNull();
  });

  it("keeps JSON field names stable while localizing docs workflow values", () => {
    const t = createTranslator("zh-CN");
    const runResult = result({
      command: ["lark-cli", "docs", "+search", "--query", "AI challenge"],
      stdout: fixture("docs-search.success.json")
    });
    const hint = analyze(runResult, t);
    const envelope = createJsonEnvelope(runResult, hint);
    const card = renderHintCard(hint, t);

    expect(envelope.exitCode).toBe(0);
    expect(envelope.hint.status).toContain("找到候选文档");
    expect(JSON.stringify(envelope)).toContain("\"exitCode\"");
    expect(JSON.stringify(envelope)).not.toContain("\"退出码\"");
    expect(card).toContain("状态");
    expect(card).toContain("下一步");
    expect(card).toContain("lark-cli docs +fetch --doc");
  });
});

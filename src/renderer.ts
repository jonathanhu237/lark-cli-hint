import type { Hint, JsonEnvelope, RunCommandResult } from "./types.js";

export function renderHintCard(
  hint: Hint,
  t: (key: string, values?: Record<string, string | number>) => string
): string {
  const sources = hint.sources.length
    ? hint.sources.map((source) => `- ${source.label}`).join("\n")
    : `- ${t("sources.none")}`;

  return [
    t("labels.status"),
    hint.status,
    "",
    t("labels.hint"),
    hint.hint,
    "",
    t("labels.next"),
    hint.next.command ?? hint.next.text,
    "",
    t("labels.why"),
    hint.why,
    "",
    t("labels.sources"),
    sources
  ].join("\n");
}

export function createJsonEnvelope(result: RunCommandResult, hint: Hint): JsonEnvelope {
  const [executable = "", ...args] = result.command;

  return {
    command: {
      executable,
      args,
      raw: result.command
    },
    exitCode: result.exitCode,
    signal: result.signal,
    stdout: result.stdout,
    stderr: result.stderr,
    hint
  };
}

import type { Hint, HintSource, RunCommandResult } from "./types.js";

export function analyzeBaseline(
  result: RunCommandResult,
  t: (key: string, values?: Record<string, string | number>) => string
): Hint {
  const success = result.exitCode === 0;
  const sources: HintSource[] = [
    {
      type: "exit-code" as const,
      label: t("sources.exitCode", { exitCode: result.exitCode })
    }
  ];

  if (result.stderr.trim()) {
    sources.push({
      type: "stderr",
      label: t("sources.stderr")
    });
  } else if (result.stdout.trim()) {
    sources.push({
      type: "stdout",
      label: t("sources.stdout")
    });
  }

  return {
    kind: success ? "success" : "failure",
    confidence: success ? 0.3 : 0.4,
    status: success
      ? t("baseline.success.status")
      : t("baseline.failure.status", { exitCode: result.exitCode }),
    hint: success ? t("baseline.success.hint") : t("baseline.failure.hint"),
    next: {
      command: null,
      text: t("baseline.noNext"),
      confidence: 0
    },
    why: success ? t("baseline.success.why") : t("baseline.failure.why"),
    sources
  };
}

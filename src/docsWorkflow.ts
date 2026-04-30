import { classifyLarkCommand, getOption, hasOption, shellQuote } from "./command.js";
import { parseJsonObject } from "./json.js";
import type { Hint, HintSource, RunCommandResult } from "./types.js";

interface DocumentCandidate {
  document: string;
  title: string;
}

type Translator = (key: string, values?: Record<string, string | number>) => string;

export function analyzeDocsWorkflow(result: RunCommandResult, t: Translator): Hint | null {
  const kind = classifyLarkCommand(result.command);

  if (kind === "docs-search" && result.exitCode === 0) {
    return analyzeDocsSearchSuccess(result, t);
  }

  if (kind === "docs-fetch" && result.exitCode === 0) {
    return analyzeDocsFetchSuccess(result, t);
  }

  if (kind === "docs-fetch" && result.exitCode !== 0) {
    return analyzeDocsFetchFailure(result, t);
  }

  return null;
}

function analyzeDocsSearchSuccess(result: RunCommandResult, t: Translator): Hint | null {
  const parsed = parseJsonObject(result.stdout);
  const candidate = parsed ? extractDocumentCandidate(parsed) : null;

  if (!candidate) {
    return null;
  }

  const nextCommand = `lark-cli docs +fetch --doc ${shellQuote(candidate.document)}`;

  return {
    kind: "success",
    confidence: 0.78,
    status: t("docs.searchSuccess.status", { title: candidate.title }),
    hint: t("docs.searchSuccess.hint"),
    next: {
      command: nextCommand,
      text: nextCommand,
      confidence: 0.82
    },
    why: t("docs.searchSuccess.why"),
    sources: [
      source("stdout", t("docs.sources.searchCandidate", { document: candidate.document })),
      source("command", t("docs.sources.commandArgs"))
    ]
  };
}

function analyzeDocsFetchSuccess(result: RunCommandResult, t: Translator): Hint {
  const title = extractFetchedTitle(result.stdout) ?? getOption(result.command, "doc") ?? t("docs.unknownDocument");
  const markdown = t("docs.fetchSuccess.markdown", { title });
  const nextCommand = `lark-cli im +messages-send --chat-id <chat_id> --markdown ${shellQuote(markdown)}`;

  return {
    kind: "success",
    confidence: 0.74,
    status: t("docs.fetchSuccess.status", { title }),
    hint: t("docs.fetchSuccess.hint"),
    next: {
      command: nextCommand,
      text: nextCommand,
      confidence: 0.68
    },
    why: t("docs.fetchSuccess.why"),
    sources: [
      source("stdout", t("docs.sources.fetchOutput")),
      source("command", t("docs.sources.commandArgs"))
    ]
  };
}

function analyzeDocsFetchFailure(result: RunCommandResult, t: Translator): Hint {
  const docToken = getOption(result.command, "doc-token");
  const doc = getOption(result.command, "doc");
  const evidence = `${result.stderr}\n${result.stdout}`;

  if (hasOption(result.command, "doc-token")) {
    const document = docToken || "<document>";
    const nextCommand = `lark-cli docs +fetch --doc ${shellQuote(document)}`;
    return failureHint({
      t,
      confidence: 0.86,
      statusKey: "docs.fetchFailure.docToken.status",
      hintKey: "docs.fetchFailure.docToken.hint",
      whyKey: "docs.fetchFailure.docToken.why",
      nextCommand,
      sources: failureSources(result, t)
    });
  }

  if (!hasOption(result.command, "doc")) {
    const nextCommand = "lark-cli docs +search --query <project_keyword>";
    return failureHint({
      t,
      confidence: 0.8,
      statusKey: "docs.fetchFailure.missingDoc.status",
      hintKey: "docs.fetchFailure.missingDoc.hint",
      whyKey: "docs.fetchFailure.missingDoc.why",
      nextCommand,
      sources: failureSources(result, t)
    });
  }

  if (doc && isWikiLookingToken(doc)) {
    const nextCommand = "lark-cli docs +search --query <project_keyword>";
    return failureHint({
      t,
      confidence: 0.72,
      statusKey: "docs.fetchFailure.wikiToken.status",
      hintKey: "docs.fetchFailure.wikiToken.hint",
      whyKey: "docs.fetchFailure.wikiToken.why",
      nextCommand,
      sources: failureSources(result, t)
    });
  }

  if (/not configured/i.test(evidence)) {
    const nextCommand = "lark-cli config init --new";
    return failureHint({
      t,
      confidence: 0.84,
      statusKey: "docs.fetchFailure.notConfigured.status",
      hintKey: "docs.fetchFailure.notConfigured.hint",
      whyKey: "docs.fetchFailure.notConfigured.why",
      nextCommand,
      sources: failureSources(result, t)
    });
  }

  if (/identity/i.test(evidence) && /not supported|only supports/i.test(evidence)) {
    const document = doc || docToken || "<document>";
    const nextCommand = `lark-cli docs +fetch --as user --doc ${shellQuote(document)}`;
    return failureHint({
      t,
      confidence: 0.76,
      statusKey: "docs.fetchFailure.identity.status",
      hintKey: "docs.fetchFailure.identity.hint",
      whyKey: "docs.fetchFailure.identity.why",
      nextCommand,
      sources: failureSources(result, t)
    });
  }

  return {
    kind: "failure",
    confidence: 0.46,
    status: t("docs.fetchFailure.generic.status", { exitCode: result.exitCode }),
    hint: t("docs.fetchFailure.generic.hint"),
    next: {
      command: null,
      text: t("baseline.noNext"),
      confidence: 0
    },
    why: t("docs.fetchFailure.generic.why"),
    sources: failureSources(result, t)
  };
}

function failureHint(options: {
  t: Translator;
  confidence: number;
  statusKey: string;
  hintKey: string;
  whyKey: string;
  nextCommand: string;
  sources: HintSource[];
}): Hint {
  return {
    kind: "failure",
    confidence: options.confidence,
    status: options.t(options.statusKey),
    hint: options.t(options.hintKey),
    next: {
      command: options.nextCommand,
      text: options.nextCommand,
      confidence: options.confidence
    },
    why: options.t(options.whyKey),
    sources: options.sources
  };
}

function failureSources(result: RunCommandResult, t: Translator): HintSource[] {
  const sources: HintSource[] = [
    source("exit-code", t("sources.exitCode", { exitCode: result.exitCode })),
    source("command", t("docs.sources.commandArgs"))
  ];

  if (result.stderr.trim()) {
    sources.push(source("stderr", t("sources.stderr")));
  } else if (result.stdout.trim()) {
    sources.push(source("stdout", t("sources.stdout")));
  }

  return sources;
}

function source(type: HintSource["type"], label: string): HintSource {
  return { type, label };
}

function extractFetchedTitle(stdout: string): string | null {
  const parsed = parseJsonObject(stdout);
  if (!parsed) {
    return null;
  }

  const title = firstStringAtPaths(parsed, [
    ["data", "title"],
    ["title"],
    ["data", "document", "title"],
    ["document", "title"]
  ]);

  return title;
}

function extractDocumentCandidate(root: Record<string, unknown>): DocumentCandidate | null {
  for (const item of findCandidateContainers(root)) {
    const document = firstStringAtPaths(item, [
      ["url"],
      ["document_url"],
      ["doc_url"],
      ["docs_url"],
      ["doc"],
      ["doc_token"],
      ["document_token"],
      ["docs_token"],
      ["token"],
      ["obj_token"]
    ]);

    if (!document) {
      continue;
    }

    const title = firstStringAtPaths(item, [
      ["title"],
      ["name"],
      ["doc_name"],
      ["document", "title"]
    ]) ?? document;

    return {
      document,
      title
    };
  }

  return null;
}

function findCandidateContainers(root: Record<string, unknown>): Record<string, unknown>[] {
  const containers = [
    valueAtPath(root, ["data", "items"]),
    valueAtPath(root, ["items"]),
    valueAtPath(root, ["data", "results"]),
    valueAtPath(root, ["results"])
  ];

  return containers.flatMap((container) => {
    if (!Array.isArray(container)) {
      return [];
    }

    return container.filter((item): item is Record<string, unknown> => (
      Boolean(item) && typeof item === "object" && !Array.isArray(item)
    ));
  });
}

function firstStringAtPaths(root: Record<string, unknown>, paths: string[][]): string | null {
  for (const path of paths) {
    const value = valueAtPath(root, path);
    if (typeof value === "string" && value.trim()) {
      return value;
    }
  }

  return null;
}

function valueAtPath(root: Record<string, unknown>, path: string[]): unknown {
  return path.reduce<unknown>((current, part) => {
    if (current && typeof current === "object" && !Array.isArray(current) && part in current) {
      return (current as Record<string, unknown>)[part];
    }

    return undefined;
  }, root);
}

function isWikiLookingToken(value: string): boolean {
  return /^(wiki_|wikcn|wiki-)/i.test(value);
}

export type LocaleCode = "en-US" | "zh-CN";

export type OutputMode = "human" | "json";

export interface SourceStream {
  write(chunk: string | Uint8Array): void;
}

export interface RunCommandResult {
  command: string[];
  exitCode: number;
  signal: NodeJS.Signals | null;
  stdout: string;
  stderr: string;
}

export interface HintSource {
  type: "command" | "exit-code" | "stdout" | "stderr";
  label: string;
}

export interface Hint {
  kind: "success" | "failure";
  confidence: number;
  status: string;
  hint: string;
  next: {
    command: string | null;
    text: string;
    confidence: number;
  };
  why: string;
  sources: HintSource[];
}

export interface JsonEnvelope {
  command: {
    executable: string;
    args: string[];
    raw: string[];
  };
  exitCode: number;
  signal: NodeJS.Signals | null;
  stdout: string;
  stderr: string;
  hint: Hint;
}

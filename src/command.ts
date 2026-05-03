export type LarkCommandKind = "docs-search" | "docs-fetch" | null;

export function classifyLarkCommand(command: string[]): LarkCommandKind {
  const [, domain, operation] = command;

  if (domain === "docs" && operation === "+search") {
    return "docs-search";
  }

  if (domain === "docs" && operation === "+fetch") {
    return "docs-fetch";
  }

  return null;
}

export function getOption(command: string[], name: string): string | null {
  const flag = `--${name}`;

  for (let index = 0; index < command.length; index += 1) {
    const value = command[index];

    if (value === flag) {
      const next = command[index + 1];
      return next && !next.startsWith("--") ? next : "";
    }

    if (value.startsWith(`${flag}=`)) {
      return value.slice(flag.length + 1);
    }
  }

  return null;
}

export function hasOption(command: string[], name: string): boolean {
  return getOption(command, name) !== null;
}

export function renderCommand(command: string[]): string {
  return command.map(shellQuote).join(" ");
}

export function renderWithUserIdentity(command: string[]): string {
  const rewritten = command.slice(0, 3);

  rewritten.push("--as", "user");

  for (let index = 3; index < command.length; index += 1) {
    const value = command[index];

    if (value === "--as") {
      index += 1;
      continue;
    }

    if (value.startsWith("--as=")) {
      continue;
    }

    rewritten.push(value);
  }

  return renderCommand(rewritten);
}

export function shellQuote(value: string): string {
  if (/^[A-Za-z0-9_./:@%+=,-]+$/.test(value)) {
    return value;
  }

  return `"${value.replace(/(["\\$`])/g, "\\$1")}"`;
}

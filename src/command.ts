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

export function shellQuote(value: string): string {
  if (/^[A-Za-z0-9_./:@%+=,-]+$/.test(value)) {
    return value;
  }

  return `"${value.replace(/(["\\$`])/g, "\\$1")}"`;
}

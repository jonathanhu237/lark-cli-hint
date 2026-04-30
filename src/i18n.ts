import { readFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import type { LocaleCode } from "./types.js";

type Messages = Record<string, unknown>;

const supportedLocales = new Set<LocaleCode>(["en-US", "zh-CN"]);
const cache = new Map<LocaleCode, Messages>();

export function resolveLocale(
  explicitLocale?: string,
  env: NodeJS.ProcessEnv = process.env
): LocaleCode {
  if (explicitLocale) {
    return normalizeLocale(explicitLocale);
  }

  const systemLocale = [env.LC_ALL, env.LC_MESSAGES, env.LANG]
    .filter(Boolean)
    .join(" ");

  return /zh|chinese/i.test(systemLocale) ? "zh-CN" : "en-US";
}

export function createTranslator(locale: LocaleCode): (key: string, values?: Record<string, string | number>) => string {
  const messages = loadMessages(locale);

  return (key, values = {}) => {
    const value = lookup(messages, key);
    const template = typeof value === "string" ? value : key;
    return template.replace(/\{\{(\w+)\}\}/g, (_, name: string) => String(values[name] ?? ""));
  };
}

function normalizeLocale(locale: string): LocaleCode {
  if (/^zh/i.test(locale)) {
    return "zh-CN";
  }

  if (supportedLocales.has(locale as LocaleCode)) {
    return locale as LocaleCode;
  }

  return "en-US";
}

function loadMessages(locale: LocaleCode): Messages {
  const cached = cache.get(locale);
  if (cached) {
    return cached;
  }

  const file = resolveLocaleFile(locale);
  const messages = JSON.parse(readFileSync(file, "utf8")) as Messages;
  cache.set(locale, messages);
  return messages;
}

function resolveLocaleFile(locale: LocaleCode): string {
  const moduleDir = dirname(fileURLToPath(import.meta.url));
  return resolve(moduleDir, "..", "locales", `${locale}.json`);
}

function lookup(messages: Messages, key: string): unknown {
  return key.split(".").reduce<unknown>((current, part) => {
    if (current && typeof current === "object" && part in current) {
      return (current as Record<string, unknown>)[part];
    }

    return undefined;
  }, messages);
}

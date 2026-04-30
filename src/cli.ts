import { Command } from "commander";
import { MissingCommandError, runApp } from "./app.js";
import { createTranslator, resolveLocale } from "./i18n.js";

const program = new Command();

program
  .name("lark-cli-hint")
  .description("A lark-cli command copilot for Feishu knowledge-base workflows.")
  .version("0.0.0");

program
  .command("run")
  .description("Run a lark-cli command and append an evidence-backed hint.")
  .option("--json", "emit one JSON envelope instead of human-readable output")
  .option("--locale <locale>", "override output locale")
  .allowUnknownOption(true)
  .argument("[command...]", "wrapped command after --")
  .action(async (command: string[] | undefined, options: { json?: boolean; locale?: string }) => {
    try {
      const result = await runApp({
        command: command ?? [],
        mode: options.json ? "json" : "human",
        locale: options.locale,
        env: process.env,
        cwd: process.cwd(),
        stdout: process.stdout,
        stderr: process.stderr
      });

      process.exitCode = result.exitCode;
    } catch (error) {
      const locale = resolveLocale(options.locale, process.env);
      const t = createTranslator(locale);
      const message = error instanceof MissingCommandError
        ? t("errors.missingCommand")
        : error instanceof Error
          ? error.message
          : String(error);

      process.stderr.write(`${message}\n`);
      process.exitCode = 2;
    }
  });

await program.parseAsync(process.argv);

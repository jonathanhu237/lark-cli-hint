## 1. Project Scaffold

- [x] 1.1 Add `package.json`, `pnpm-lock.yaml`, `tsconfig.json`, and `tsup.config.ts` for a TypeScript/Node.js CLI package named `lark-cli-hint`.
- [x] 1.2 Configure the package `bin` entry so `lark-cli-hint` points to the built CLI file.
- [x] 1.3 Add npm scripts for development, build, typecheck, and test using tsx, tsup, TypeScript, and vitest.

## 2. CLI and Core Boundaries

- [x] 2.1 Create the commander-based CLI shell for `lark-cli-hint run`.
- [x] 2.2 Create a core app `run` function that can be called without commander or direct process exits.
- [x] 2.3 Validate that `run` reports an error when no wrapped command is provided after `--`.

## 3. Wrapped Command Runner

- [x] 3.1 Implement child process execution for wrapped commands with captured `stdout`, `stderr`, and exit status.
- [x] 3.2 Stream wrapped command output in human mode while retaining bounded captured output for analysis.
- [x] 3.3 Suppress streamed prose in `--json` mode so stdout can contain one JSON document.

## 4. Hint Analysis and Rendering

- [x] 4.1 Implement a baseline analyzer that generates conservative success and failure hints without domain-specific Next commands.
- [x] 4.2 Implement terminal Hint Card rendering with `Status`, `Hint`, `Next`, `Why`, and `Sources` sections.
- [x] 4.3 Implement JSON envelope rendering with stable English field names and localized user-facing values.

## 5. i18n

- [x] 5.1 Add locale files for `en-US` and `zh-CN`.
- [x] 5.2 Detect Chinese user environments from `LANG`, `LC_ALL`, or `LC_MESSAGES`; default to `en-US` otherwise.
- [x] 5.3 Ensure terminal labels and baseline messages are localized while JSON field names remain stable.

## 6. Verification

- [x] 6.1 Add tests for missing wrapped command validation.
- [x] 6.2 Add tests for successful wrapped command JSON envelope output.
- [x] 6.3 Add tests for failed wrapped command JSON envelope output.
- [x] 6.4 Add tests for locale selection and localized baseline Hint Card rendering.
- [x] 6.5 Run build, typecheck, and test scripts.

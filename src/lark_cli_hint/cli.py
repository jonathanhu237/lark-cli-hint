"""CLI entry point.

Dispatch model:
- Internal subcommands (`explain`, `share`) handled by lch.
- Everything else passes through to `lark-cli` unchanged, preserving
  arguments, stdin/stdout/stderr, and the exit code.
"""

import sys

from lark_cli_hint import runner

INTERNAL_SUBCOMMANDS = {"explain", "share"}


def main() -> None:
    args = sys.argv[1:]

    if args and args[0] in INTERNAL_SUBCOMMANDS:
        print(f"[lch] '{args[0]}' is not implemented yet", file=sys.stderr)
        sys.exit(2)

    sys.exit(runner.run_lark_cli(args))

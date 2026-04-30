"""Spawn lark-cli as a subprocess and proxy its IO."""

import os
import subprocess
import sys


def run_lark_cli(args: list[str]) -> int:
    """Run `lark-cli` with the given args. The child inherits stdin/stdout/stderr
    from the current process by default, so output streams to the user's terminal
    in real time.

    Returns the subprocess exit code, or 127 if lark-cli is not on PATH.
    """
    cmd = ["lark-cli", *args]
    try:
        proc = subprocess.run(cmd, env=os.environ.copy(), check=False)
    except FileNotFoundError:
        print(
            "lark-cli not found on PATH. Install from https://github.com/larksuite/cli",
            file=sys.stderr,
        )
        return 127
    return proc.returncode

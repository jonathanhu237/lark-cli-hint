"""Tests for the lark-cli passthrough runner."""

from lark_cli_hint import runner


def test_run_lark_cli_proxies_help(capfd):
    """Running with --help should proxy lark-cli's help and exit 0."""
    exit_code = runner.run_lark_cli(["--help"])
    out, _ = capfd.readouterr()
    assert exit_code == 0
    assert "lark-cli" in out.lower()


def test_run_lark_cli_propagates_nonzero_exit():
    """An unknown command should return non-zero."""
    exit_code = runner.run_lark_cli(["this-command-does-not-exist"])
    assert exit_code != 0

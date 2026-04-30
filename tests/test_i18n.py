"""Tests for locale resolution and translation loading."""

from lark_cli_hint import i18n


def test_resolve_lang_explicit_overrides_env(monkeypatch):
    monkeypatch.setenv("LANG", "zh_CN.UTF-8")
    assert i18n.resolve_lang("en") == "en"


def test_resolve_lang_invalid_explicit_falls_back(monkeypatch):
    monkeypatch.delenv("LC_ALL", raising=False)
    monkeypatch.delenv("LANG", raising=False)
    assert i18n.resolve_lang("xx") == "en"


def test_resolve_lang_zh_env(monkeypatch):
    monkeypatch.delenv("LC_ALL", raising=False)
    monkeypatch.setenv("LANG", "zh_CN.UTF-8")
    assert i18n.resolve_lang() == "zh"


def test_resolve_lang_en_env(monkeypatch):
    monkeypatch.delenv("LC_ALL", raising=False)
    monkeypatch.setenv("LANG", "en_US.UTF-8")
    assert i18n.resolve_lang() == "en"


def test_resolve_lang_default_when_no_env(monkeypatch):
    monkeypatch.delenv("LC_ALL", raising=False)
    monkeypatch.delenv("LANG", raising=False)
    assert i18n.resolve_lang() == "en"


def test_load_strings_zh():
    assert i18n.load_strings("zh")["ui"]["sources"] == "来源"


def test_load_strings_en():
    assert i18n.load_strings("en")["ui"]["sources"] == "Sources"

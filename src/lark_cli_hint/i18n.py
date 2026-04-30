"""Locale resolution and translation lookup.

Resolution order:
1. Explicit argument (e.g., from CLI --lang).
2. LC_ALL / LANG environment variables (`zh*` -> Chinese, otherwise English).
3. Default: English.
"""

import os
import tomllib
from pathlib import Path
from typing import Any

DEFAULT_LANG = "en"
SUPPORTED_LANGS = {"en", "zh"}

_LOCALES_DIR = Path(__file__).resolve().parents[2] / "locales"


def resolve_lang(explicit: str | None = None) -> str:
    if explicit in SUPPORTED_LANGS:
        return explicit  # type: ignore[return-value]
    env = os.environ.get("LC_ALL") or os.environ.get("LANG") or ""
    if env.startswith("zh"):
        return "zh"
    return DEFAULT_LANG


def load_strings(lang: str) -> dict[str, Any]:
    if lang not in SUPPORTED_LANGS:
        raise ValueError(f"Unsupported language: {lang}")
    path = _LOCALES_DIR / f"{lang}.toml"
    with path.open("rb") as f:
        return tomllib.load(f)

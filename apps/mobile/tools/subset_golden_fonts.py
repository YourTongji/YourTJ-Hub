#!/usr/bin/env python3
"""Regenerate the deterministic CJK font subsets used by forum_app golden tests.

Why this exists: golden tests must render identically on macOS and the Linux CI
runner. The bundled `NotoSansCJKsc` subsets pin the CJK font; without them the
two hosts fall back to different system fonts and every golden pixel-diffs
(331cfa90). The subsets are cut to the charset actually used by mobile sources
— when new zh strings land (new l10n keys, new fixtures, new test copy), new
glyphs can fall outside the subset and render as tofu (□) in goldens. Re-run
this script after adding zh strings, then re-render the affected goldens via
the `mobile-golden-refresh` workflow.

Scope: defaults to forum_app. Use --package ui_kit or --package all when
component samples also add glyphs. Re-render the affected package goldens
after changing its font bytes.

Usage:
    python3 apps/mobile/tools/subset_golden_fonts.py [--source-dir DIR] [--package forum_app|ui_kit|all]

`--source-dir` defaults to ~/Library/Fonts and must contain the full static
`NotoSansSC-Regular.ttf` / `NotoSansSC-Bold.ttf` (the non-variable releases).
"""

from __future__ import annotations

import argparse
import sys
from pathlib import Path

from fontTools import subset
from fontTools.ttLib import TTFont

MOBILE_ROOT = Path(__file__).resolve().parents[1]

# Always keep ASCII printable + a safety margin of CJK punctuation/quotes that
# UI copy commonly uses even if a given string is temporarily removed.
_BASE_CHARS = (
    "".join(chr(c) for c in range(0x20, 0x7F))
    + "·—…“”‘’「」『』（）《》、，。：；！？％℃"
)

_SKIP_DIR_PARTS = {".dart_tool", "build", ".synergy"}


def collect_charset() -> set[int]:
    """Union of every non-ASCII codepoint in mobile Dart/ARB sources + base."""
    chars: set[int] = {ord(c) for c in _BASE_CHARS}
    patterns = ["packages/**/*.dart", "packages/**/*.arb"]
    seen = 0
    for pattern in patterns:
        for path in MOBILE_ROOT.glob(pattern):
            # 跳过检查必须用相对 MOBILE_ROOT 的 parts：worktree 的绝对路径
            # 本身就含 .synergy，按绝对 parts 判断会把所有文件过滤光。
            try:
                relative_parts = set(path.relative_to(MOBILE_ROOT).parts)
            except ValueError:
                continue
            if _SKIP_DIR_PARTS & relative_parts:
                continue
            seen += 1
            try:
                text = path.read_text(encoding="utf-8")
            except UnicodeDecodeError:
                continue
            chars.update(ord(ch) for ch in text if ord(ch) >= 0x80)
    print(f"scanned {seen} source files -> {len(chars)} codepoints")
    return chars


def subset_font(source: Path, output: Path, unicodes: list[int]) -> None:
    font = TTFont(str(source))
    options = subset.Options()
    options.layout_features = ["*"]  # keep shaping features (vert, kern, ...)
    options.glyph_names = False
    options.notdef_outline = True  # keep a visible .notdef for true gaps
    options.name_IDs = ["*"]
    options.drop_tables += ["FFTM"]
    subsetter = subset.Subsetter(options=options)
    subsetter.populate(unicodes=unicodes)
    subsetter.subset(font)
    output.parent.mkdir(parents=True, exist_ok=True)
    font.save(str(output))
    kept = len(TTFont(str(output)).getBestCmap())
    print(f"{output.name}: {kept} glyphs, {output.stat().st_size // 1024} KB")


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--source-dir",
        type=Path,
        default=Path.home() / "Library" / "Fonts",
        help="directory containing full NotoSansSC-{Regular,Bold}.ttf",
    )
    parser.add_argument("--package", choices=("forum_app", "ui_kit", "all"), default="forum_app")
    args = parser.parse_args()

    unicodes = sorted(collect_charset())
    for weight in ("Regular", "Bold"):
        source = args.source_dir / f"NotoSansSC-{weight}.ttf"
        if not source.exists():
            print(f"missing full font: {source}", file=sys.stderr)
            return 1
        packages = ("forum_app", "ui_kit") if args.package == "all" else (args.package,)
        for package in packages:
            output = MOBILE_ROOT / "packages" / package / "test/assets/fonts" / f"NotoSansCJKsc-{weight}.otf"
            subset_font(source, output, unicodes)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

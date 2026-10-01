"""Publishers require an explicit reviewed platform file; source metadata is not a fallback."""
import os
from pathlib import Path
import re


def read_notes(variable, apple=False):
    name = os.environ.get(variable)
    if not name:
        raise ValueError(f"Missing reviewed release notes: {variable}")
    path = Path(name)
    if not path.is_file() or path.is_symlink():
        raise ValueError(f"Missing reviewed release notes file: {variable}")
    text = path.read_text().strip()
    if not text or '[DRAFT:' in text or len(text) > (4000 if apple else 16000):
        raise ValueError(f"Invalid reviewed release notes: {variable}")
    if apple and re.search(r'(?m)^\s{0,3}(?:#{1,6}\s|```)|\[[^\]]+\]\(|<[^>]+>', text):
        raise ValueError('Apple release notes must be plain text')
    return text

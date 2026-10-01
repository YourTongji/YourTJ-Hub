"""Publishers require an explicit reviewed platform file; source metadata is not a fallback."""
import hashlib
import json
import os
from pathlib import Path
import sys
sys.path.insert(0, str(Path(__file__).resolve().parents[1] / 'release'))
from model import FILES, validate_note_text

CHANNELS = {'WEB_NOTES_PATH': 'web', 'ANDROID_NOTES_PATH': 'android',
            'IOS_STORE_NOTES_PATH': 'ios-app-store', 'IOS_TESTFLIGHT_NOTES_PATH': 'ios-testflight'}


def read_notes(variable):
    name = os.environ.get(variable)
    if not name:
        raise ValueError(f"Missing reviewed release notes: {variable}")
    path = Path(name)
    if not path.is_file() or path.is_symlink():
        raise ValueError(f"Missing reviewed release notes file: {variable}")
    channel = CHANNELS[variable]
    # The trusted authorize step's output is passed separately from mutable workspace files.
    # Hash and decode the same read, so the content checked is the content returned.
    raw = path.read_bytes()
    digests = json.loads(os.environ.get('RELEASE_NOTES_DIGESTS', '{}'))
    if not isinstance(digests, dict) or digests.get(FILES[channel]) != hashlib.sha256(raw).hexdigest():
        raise ValueError(f'Reviewed release notes digest missing or changed: {variable}')
    return validate_note_text(raw.decode('utf-8').strip(), FILES[channel], channel)

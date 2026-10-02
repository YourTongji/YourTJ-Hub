#!/usr/bin/env python3
"""Never overwrite a release image after an uncertain push/receipt outcome."""
import re
import subprocess
import sys
from model import require


def require_absent(reference):
    require(re.fullmatch(r'ghcr\.io/yourtongji/yourtj-hub:main-v[0-9]+\.[0-9]+\.[0-9]+', reference),
            'Expected a versioned production image reference')
    result = subprocess.run(['docker', 'manifest', 'inspect', reference], capture_output=True, text=True)
    require(result.returncode != 0,
            'Image already exists without its original digest receipt; reconcile that receipt or prepare a new release. Refusing to overwrite.')
    # Docker's explicit missing-manifest result is the only absence proof. Permission,
    # rate-limit and connectivity failures must not authorize a replacement push.
    require(result.stderr.strip() in {f'no such manifest: {reference}', 'manifest unknown'},
            'Cannot establish image absence; inspect registry access before retrying')


if __name__ == '__main__':
    require_absent(sys.argv[1])

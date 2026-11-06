"""Download the pinned official Linux toolchain without exposing its signed URL."""
import hashlib
import os
from pathlib import Path
import urllib.error
import urllib.parse
import urllib.request

CLI_SHA256 = 'e5fda7d37eef11263588203d724d1b72380273d3c07535bb7d361a1d0e95cf34'


def download(url, destination):
    parsed = urllib.parse.urlsplit(url)
    if (parsed.scheme != 'https' or parsed.hostname != 'contentcenter-vali-drcn.dbankcdn.cn'
            or parsed.username or parsed.password or parsed.port not in (None, 443)
            or not parsed.path.endswith('/commandline-tools-linux-x64-26.0.0.865.zip')):
        raise ValueError('Expected the official Linux x64 26.0.0.865 toolchain URL')
    digest = hashlib.sha256()
    class NoRedirect(urllib.request.HTTPRedirectHandler):
        def redirect_request(self, req, fp, code, msg, headers, newurl):
            return None
    opener = urllib.request.build_opener(NoRedirect)
    created = False
    try:
        with opener.open(url, timeout=120) as response, destination.open('xb') as output:
            created = True
            while block := response.read(1024 * 1024):
                digest.update(block)
                output.write(block)
        if digest.hexdigest() != CLI_SHA256:
            raise ValueError('Harmony toolchain checksum mismatch')
    except (urllib.error.URLError, TimeoutError, ValueError):
        if created:
            destination.unlink(missing_ok=True)
        raise ValueError('Official Harmony toolchain download failed; signed URL withheld') from None


if __name__ == '__main__':
    try:
        download(os.environ['HARMONY_CLI_DOWNLOAD_URL'], Path(os.environ['RUNNER_TEMP']) / 'harmony-cli.zip')
        print('Official Harmony Linux toolchain SHA-256 verified')
    except (ValueError, OSError, KeyError) as error:
        raise SystemExit(str(error))

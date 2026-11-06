"""YourTJ-only HarmonyOS APP Test client. Never logs credentials or signed URLs."""
import base64
import hashlib
import json
import os
from pathlib import Path
import re
import subprocess
import tempfile
import time
import urllib.error
import urllib.parse
import urllib.request

APP_ID = '6917621118093079103'
BUNDLE = 'tj.yourtj.forumapp'
API = 'https://connect-api.cloud.huawei.com'
AUDIENCE = 'https://oauth-login.cloud.huawei.com/oauth2/v3/token'


def encode(value):
    return base64.urlsafe_b64encode(value).rstrip(b'=').decode()


def authorization(credentials, now=None):
    now = int(time.time()) if now is None else now
    header = {'alg': 'PS256', 'typ': 'JWT', 'kid': credentials['key_id']}
    claims = {'aud': AUDIENCE, 'iss': credentials['sub_account'], 'iat': now, 'exp': now + 3600}
    unsigned = '.'.join(encode(json.dumps(v, separators=(',', ':')).encode()) for v in (header, claims))
    with tempfile.TemporaryDirectory(prefix='yourtj-harmony-auth-') as directory:
        key = Path(directory) / 'private.pem'
        key.touch(mode=0o600)
        key.write_text(credentials['private_key'])
        result = subprocess.run(['openssl', 'dgst', '-sha256', '-sign', str(key),
                                 '-sigopt', 'rsa_padding_mode:pss', '-sigopt', 'rsa_pss_saltlen:digest'],
                                input=unsigned.encode(), capture_output=True, timeout=15)
        if result.returncode:
            raise ValueError('Harmony service-account signing failed')
    return unsigned + '.' + encode(result.stdout)


class Client:
    def __init__(self, credentials):
        self.token = authorization(credentials)
        # Redirects must never forward the developer account credential.
        class NoRedirect(urllib.request.HTTPRedirectHandler):
            def redirect_request(self, req, fp, code, msg, headers, newurl):
                return None
        self.opener = urllib.request.build_opener(NoRedirect)

    def request(self, method, path, body=None, query=None):
        if not path.startswith('/api/') or '?' in path or '#' in path:
            raise ValueError('Invalid Harmony API path')
        url = API + path
        if query is not None:
            url += '?' + urllib.parse.urlencode({**query, 'appId': APP_ID})
        request = urllib.request.Request(url, method=method,
            data=json.dumps(body).encode() if body is not None else None,
            headers={'Authorization': 'Bearer ' + self.token, 'Content-Type': 'application/json',
                     'Accept': 'application/json', 'User-Agent': 'YourTJ-Harmony-Release/1.0', 'appId': APP_ID})
        try:
            with self.opener.open(request, timeout=120) as response:
                result = json.load(response)
        except urllib.error.HTTPError as error:
            raise ValueError(f'Harmony API HTTP {error.code}; response details withheld') from None
        except (urllib.error.URLError, TimeoutError, json.JSONDecodeError):
            raise ValueError('Harmony API request failed; response details withheld') from None
        code = result.get('ret', {}).get('code')
        if code != 0:
            raise ValueError(f'Harmony API rejected request (code {code}); response details withheld')
        return result

    def versions(self):
        return self.request('POST', '/api/publish/v3/version/brief-info/list', {'packageName': BUNDLE})['versionList']

    def invitation(self, version_id):
        if not re.fullmatch(r'[0-9]{1,32}', version_id):
            raise ValueError('A numeric test version ID is required')
        for version in self.versions():
            if str(version.get('versionId')) == version_id and version.get('testType') == 3 and version.get('releaseType') == 6:
                return version
        raise ValueError('Only an existing YourTJ invitation-test version may be submitted')

    def upload(self, app):
        if app.suffix != '.app' or not app.is_file():
            raise ValueError('A verified HarmonyOS .app package is required')
        with app.open('rb') as file:
            digest = hashlib.file_digest(file, 'sha256').hexdigest()
        info = self.request('GET', '/api/publish/v2/upload-url/for-obs', query={
            'fileName': app.name, 'sha256': digest, 'contentLength': app.stat().st_size,
            'chineseMainlandFlag': 1})['urlInfo']
        if info.get('method') != 'PUT':
            raise ValueError('Unexpected Harmony upload method')
        upload_url = info['url']
        host = urllib.parse.urlsplit(upload_url)
        if host.scheme != 'https' or host.username or host.password or host.port not in (None, 443) or not any(
                host.hostname and host.hostname.endswith('.' + domain)
                for domain in ('myhuaweicloud.com', 'myhuaweicloud.cn', 'dbankcloud.cn', 'huawei.com')):
            raise ValueError('Unexpected Harmony upload destination')
        with app.open('rb') as file:
            request = urllib.request.Request(upload_url, data=file, method='PUT',
                headers={**info.get('headers', {}), 'Content-Type': 'application/octet-stream',
                         'Content-Length': str(app.stat().st_size)})
            try:
                with self.opener.open(request, timeout=300) as response:
                    if response.status not in (200, 201, 204):
                        raise ValueError('Harmony file upload rejected')
            except (urllib.error.URLError, TimeoutError):
                raise ValueError('Harmony upload failed; signed URL withheld') from None
        result = self.request('POST', '/api/publish/v2/test/version/pkg', {
            'distributeMode': 1, 'file': {'fileName': app.name, 'objectId': info['objectId']}}, query={})
        packages = result.get('pkgVersion', [])
        if len(packages) != 1 or not re.fullmatch(r'[0-9]+', packages[0]):
            raise ValueError('Harmony did not return one package identity')
        return packages[0], digest


def main():
    client = Client(json.loads(Path(os.environ['HARMONY_SERVICE_ACCOUNT_PATH']).read_text()))
    print(json.dumps({'versions': [{key: version.get(key) for key in
          ('versionId', 'state', 'testType', 'releaseType')} for version in client.versions()]}))


if __name__ == '__main__':
    try:
        main()
    except (ValueError, KeyError, OSError) as error:
        raise SystemExit(str(error))

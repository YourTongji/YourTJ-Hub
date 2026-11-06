"""Build, sign, verify and pack the native HarmonyOS invitation-test artifact."""
import base64
import hashlib
import json
import os
from pathlib import Path
import re
import subprocess
import tempfile
from zipfile import ZipFile

from build_environment import dart_defines
from harmony_api import BUNDLE
from sign_harmony import sign

ROOT = Path(__file__).resolve().parents[2]
APP = ROOT / 'apps/mobile/packages/forum_app'


def validate_profile(profile):
    result = subprocess.run(['openssl', 'cms', '-verify', '-inform', 'DER', '-noverify'],
                            input=profile, capture_output=True, timeout=15)
    if result.returncode:
        raise ValueError('Unable to decode the signed Harmony profile')
    data = json.loads(result.stdout)
    if data.get('type') != 'release' or data.get('bundle-info', {}).get('bundle-name') != BUNDLE:
        raise ValueError('Expected the YourTJ release profile')
    if 'ohos.permission.READ_PASTEBOARD' not in data.get('acls', {}).get('allowed-acls', []):
        raise ValueError('Release profile lacks approved READ_PASTEBOARD ACL; update it after Huawei approval')
    # Structural preflight only; the vendor signer subsequently verifies the profile signature.


def run(command, cwd=ROOT):
    result = subprocess.run([str(x) for x in command], cwd=cwd, capture_output=True, text=True)
    if result.returncode:
        raise ValueError(f'Harmony {Path(str(command[0])).name} failed; private diagnostics withheld')


def package_app(unsigned, signed_hap, directory, tools, output):
    with ZipFile(unsigned) as archive:
        info = json.loads(archive.read('pack.info'))
        if info['summary']['app']['bundleName'] != BUNDLE:
            raise ValueError('Unexpected Harmony application identity')
        packages = info['packages']
        if len(packages) != 1 or packages[0]['name'] != 'entry-default':
            raise ValueError('Expected exactly one Harmony entry module')
        (directory / 'pack.info').write_bytes(archive.read('pack.info'))
        (directory / 'pac.json').write_bytes(archive.read('pac.json'))
    # The default packer rewrites pack.info inside HAPs and invalidates their signatures.
    packed = directory / 'packed-unsigned.app'
    run(['java', '-jar', tools / 'app_packing_tool.jar', '--mode', 'app', '--hap-path', signed_hap,
         '--pack-info-path', directory / 'pack.info', '--pac-json-path', directory / 'pac.json',
         '--replace-pack-info', 'false', '--out-path', packed])
    # APP Test verifies the APP envelope as well as its contained HAP.
    sign(tools / 'hap-sign-tool.jar', directory, packed, output)
    run(['java', '-jar', tools / 'hap-sign-tool.jar', 'verify-app', '-inFile', output,
         '-outCertChain', directory / 'app-verified.cer', '-outProfile', directory / 'app-verified.p7b'])
    with ZipFile(output) as archive:
        if archive.read('entry-default.hap') != signed_hap.read_bytes():
            raise ValueError('APP packaging changed the verified signed HAP')


def main():
    version = os.environ['MOBILE_VERSION']
    number = os.environ['MOBILE_BUILD_NUMBER']
    if not re.fullmatch(r'(0|[1-9][0-9]{0,2})(\.(0|[1-9][0-9]{0,2})){2}', version):
        raise ValueError('Expected a numeric three-component mobile version')
    if not number.isascii() or not number.isdecimal() or not 1 <= int(number) <= 2147483647:
        raise ValueError('Invalid Harmony version code')
    validate_profile(base64.b64decode(os.environ['HARMONY_PROFILE_BASE64'], validate=True))
    defines = dart_defines('production', os.environ['SOURCE_SHA'], True)
    tools = Path(os.environ['DEVECO_SDK_HOME']) / 'default/openharmony/toolchains/lib'
    output_dir = ROOT / 'apps/mobile/build/release/harmonyos'
    output_dir.mkdir(parents=True, exist_ok=True)
    output = output_dir / f'YourTJ-{version}-{number}.app'
    if output.exists():
        raise ValueError('Refusing to overwrite an existing signed APP; recover the original artifact')
    run(['flutter', 'build', 'app', '--release', '--no-pub', '--no-codesign', '--target-platform=ohos-arm64',
         '--build-name=' + version, '--build-number=' + number, *defines], APP)
    unsigned_hap = APP / 'ohos/entry/build/default/outputs/default/entry-default-unsigned.hap'
    run(['python3', APP / 'ohos/test/check_hap_abi.py', unsigned_hap, '--abi', 'arm64-v8a', '--release'])
    with tempfile.TemporaryDirectory(prefix='yourtj-harmony-sign-') as temp:
        directory = Path(temp)
        for name, file in {'HARMONY_KEYSTORE_BASE64': 'yourtj-release.p12',
                           'HARMONY_CERTIFICATE_BASE64': 'yourtj-release.cer',
                           'HARMONY_PROFILE_BASE64': 'yourtj-release.p7b'}.items():
            target = directory / file
            target.touch(mode=0o600)
            target.write_bytes(base64.b64decode(os.environ[name], validate=True))
        password = directory / 'keystore-password'
        password.touch(mode=0o600)
        password.write_text(os.environ['HARMONY_KEYSTORE_PASSWORD'])
        signed_hap = directory / 'entry-default.hap'
        sign(tools / 'hap-sign-tool.jar', directory, unsigned_hap, signed_hap)
        run(['java', '-jar', tools / 'hap-sign-tool.jar', 'verify-app', '-inFile', signed_hap,
             '-outCertChain', directory / 'verified.cer', '-outProfile', directory / 'verified.p7b'])
        package_app(APP / 'build/ohos/app/ohos-default-unsigned.app', signed_hap, directory, tools, output)
    with output.open('rb') as file:
        digest = hashlib.file_digest(file, 'sha256').hexdigest()
    output.with_suffix('.sha256').write_text(f'{digest}  {output.name}\n')
    print(f'Verified signed APP: {output.name}; SHA-256 {digest}')


if __name__ == '__main__':
    try:
        main()
    except (ValueError, KeyError, OSError) as error:
        raise SystemExit(str(error))

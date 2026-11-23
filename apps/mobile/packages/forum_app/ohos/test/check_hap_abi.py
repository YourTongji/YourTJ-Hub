"""Check a built HAP's packaged ABI and ELF headers without extracting libraries."""
import argparse
import struct
import zipfile

parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument('hap')
parser.add_argument('--abi', required=True, choices=['arm64-v8a', 'x86_64'])
parser.add_argument('--release', action='store_true')
args = parser.parse_args()
expected_machine = {'arm64-v8a': 183, 'x86_64': 62}[args.abi]
with zipfile.ZipFile(args.hap) as archive:
    libraries = [name for name in archive.namelist()
                 if name.startswith('libs/') and name.endswith('.so')]
    packaged = {name.split('/')[1] for name in libraries}
    if packaged != {args.abi}:
        raise SystemExit(f'Incorrect packaged ABIs: {sorted(packaged)}')
    required = {'libflutter.so', 'libsqlite3mc.so'}
    if args.release:
        required.add('libapp.so')
    missing = required - {name.split('/')[-1] for name in libraries}
    if missing:
        raise SystemExit(f'Missing native libraries: {sorted(missing)}')
    for name in libraries:
        data = archive.read(name)
        if len(data) < 20 or data[:6] != b'\x7fELF\x02\x01':
            raise SystemExit(f'Not a little-endian ELF64 library: {name}')
        if struct.unpack_from('<H', data, 18)[0] != expected_machine:
            raise SystemExit(f'ELF architecture does not match package path: {name}')
        if b'GLIBC_' in data or b'libc.so.6\x00' in data:
            raise SystemExit(f'Linux glibc dependency in HarmonyOS library: {name}')
print(f'PASS: {args.abi}, {len(libraries)} native libraries')

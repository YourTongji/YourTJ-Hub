"""Sign a verified HAP using the vendor CLI without passwords in argv or logs."""
import argparse
import os
from pathlib import Path
import pty
import re
import select
import signal
import termios
import time


def sign(tool, directory, source, output):
    password = (directory / 'keystore-password').read_bytes().rstrip(b'\r\n')
    if len(password) < 8 or b'\n' in password or b'\r' in password:
        raise ValueError('Invalid Harmony keystore password')
    args = ['java', '-jar', str(tool), 'sign-app', '-mode', 'localSign',
            '-keyAlias', 'yourtj-release', '-appCertFile', str(directory / 'yourtj-release.cer'),
            '-profileFile', str(directory / 'yourtj-release.p7b'), '-inFile', str(source),
            '-outFile', str(output), '-signAlg', 'SHA256withECDSA',
            '-keystoreFile', str(directory / 'yourtj-release.p12'), '-compatibleVersion', '18', '-pwdInputMode', '1']
    pid, fd = pty.fork()
    if pid == 0:
        os.execvp(args[0], args)
    prompts = 0
    pending = b''
    status = None
    try:
        attrs = termios.tcgetattr(fd)
        attrs[3] &= ~termios.ECHO
        termios.tcsetattr(fd, termios.TCSANOW, attrs)
        deadline = time.monotonic() + 120
        while time.monotonic() < deadline:
            if not select.select([fd], [], [], 1)[0]:
                continue
            try:
                data = os.read(fd, 65536)
            except OSError:
                break
            if not data:
                break
            pending = (pending + data)[-4096:]
            if prompts < 2 and re.search(rb'please input (?:KeystorePwd|KeyPwd)[^\r\n]*:', pending):
                os.write(fd, password + b'\n')
                prompts += 1
                pending = b''
        else:
            raise ValueError('Harmony signing timed out')
        _, status = os.waitpid(pid, 0)
        if status or prompts != 2 or not output.is_file():
            raise ValueError('Harmony signing failed; private signer diagnostics withheld')
    finally:
        os.close(fd)
        if status is None:
            os.kill(pid, signal.SIGTERM)
            os.waitpid(pid, 0)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--tool', type=Path, required=True)
    parser.add_argument('--signing-dir', type=Path, required=True)
    parser.add_argument('--input', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    sign(args.tool.resolve(), args.signing_dir.resolve(), args.input.resolve(), args.output.resolve())
    print('Harmony HAP signed; signature verification is still required')


if __name__ == '__main__':
    try:
        main()
    except (ValueError, OSError) as error:
        raise SystemExit(str(error))

#!/usr/bin/env python3
"""Archive Runner with an ephemeral keychain and target-scoped signing settings.

Requires IOS_P12_PATH, IOS_P12_PASSWORD, IOS_PROFILE_PATH, IOS_TEAM_ID,
MOBILE_VERSION and MOBILE_BUILD_NUMBER. Credentials are never printed.
"""
from contextlib import ExitStack
import datetime
import hashlib
import os
from pathlib import Path
import plistlib
import re
import secrets
import shlex
import shutil
import subprocess
import tempfile
import zipfile

ROOT = Path(__file__).resolve().parents[2]
APP = ROOT / "apps/mobile/packages/forum_app"
BUNDLE_ID = "tj.yourtj.forumApp"


def validate_profile(profile, team, now=None):
    now = now or datetime.datetime.now(datetime.timezone.utc)
    expires = profile["ExpirationDate"].replace(tzinfo=datetime.timezone.utc)
    entitlements = profile["Entitlements"]
    if expires <= now:
        raise ValueError("Distribution profile has expired")
    if profile["TeamIdentifier"] != [team]:
        raise ValueError("Distribution profile belongs to another team")
    if entitlements.get("application-identifier") != f"{team}.{BUNDLE_ID}":
        raise ValueError("Distribution profile has the wrong app identifier")
    if entitlements.get("get-task-allow") or profile.get("ProvisionedDevices") or profile.get("ProvisionsAllDevices"):
        raise ValueError("An App Store distribution profile is required")
    if entitlements.get("aps-environment") != "production":
        raise ValueError("App Store profile must enable production Push Notifications; regenerate the profile")
    if not re.fullmatch(r"[A-Fa-f0-9-]{36}", profile["UUID"]):
        raise ValueError("Invalid provisioning profile UUID")


def validate_app_entitlements(entitlements, team):
    if entitlements.get("application-identifier") != f"{team}.{BUNDLE_ID}" or entitlements.get("aps-environment") != "production":
        raise ValueError("Exported app must carry the correct application identifier and production APNs entitlement")


def validate_exported_ipa(path, team):
    # Inspect the signed artifact, not just its provisioning input.
    with tempfile.TemporaryDirectory(prefix="yourtj-ipa-check-") as temporary:
        with zipfile.ZipFile(path) as archive:
            for name in archive.namelist():
                parts = Path(name)
                if parts.is_absolute() or ".." in parts.parts:
                    raise ValueError("Invalid IPA archive path")
            archive.extractall(temporary)
        apps = list((Path(temporary) / "Payload").glob("*.app"))
        if len(apps) != 1:
            raise ValueError("Expected exactly one exported iOS application")
        raw = subprocess.check_output(["codesign", "-d", "--entitlements", ":-", str(apps[0])], stderr=subprocess.DEVNULL)
        validate_app_entitlements(plistlib.loads(raw), team)


def security(*args):
    result = subprocess.run(["security", *args], capture_output=True, text=True)
    if result.returncode:
        # Do not include argv: it can contain passwords.
        raise RuntimeError(f"security {args[0]} failed: {result.stderr.strip()}")
    return result.stdout


def main():
    os.umask(0o077)
    team = os.environ["IOS_TEAM_ID"]
    version = os.environ["MOBILE_VERSION"]
    number = os.environ["MOBILE_BUILD_NUMBER"]
    if not re.fullmatch(r"[A-Z0-9]{10}", team):
        raise ValueError("Invalid Apple team ID")
    if not re.fullmatch(r"\d+\.\d+\.\d+", version) or not re.fullmatch(r"[1-9]\d*", number):
        raise ValueError("Invalid mobile version/build number")
    profile_path = Path(os.environ["IOS_PROFILE_PATH"]).resolve()
    profile = plistlib.loads(subprocess.check_output(
        ["security", "cms", "-D", "-i", str(profile_path)], stderr=subprocess.DEVNULL))
    validate_profile(profile, team)
    output = Path(os.environ.get("MOBILE_OUTPUT_DIR", str(ROOT / "apps/mobile/build/release"))).resolve()
    output.mkdir(parents=True, exist_ok=True)
    flutter = os.environ.get("FLUTTER_BIN", "flutter")
    subprocess.run([
        flutter, "build", "ios", "--config-only", "--release", "--no-codesign",
        f"--build-name={version}", f"--build-number={number}",
        "--dart-define=YOURTJ_API_BASE_URL=https://f.yourtj.de",
        "--dart-define=YOURTJ_OIDC_ISSUER=https://f.yourtj.de/api/oauth",
        "--dart-define=YOURTJ_OIDC_CLIENT_ID=yourtj-mobile",
    ], cwd=APP, check=True)
    signing = APP / "ios/Flutter/ReleaseSigning.xcconfig"
    previous_signing = signing.read_bytes() if signing.exists() else None
    previous_keychains = shlex.split(security("list-keychains", "-d", "user"))
    installed = []
    with tempfile.TemporaryDirectory(prefix="yourtj-signing-") as temporary:
        keychain = str(Path(temporary) / "release.keychain-db")
        password = secrets.token_urlsafe(32)
        try:
            security("create-keychain", "-p", password, keychain)
            security("set-keychain-settings", "-lut", "21600", keychain)
            security("unlock-keychain", "-p", password, keychain)
            security("import", os.environ["IOS_P12_PATH"], "-k", keychain,
                     "-P", os.environ["IOS_P12_PASSWORD"], "-T", "/usr/bin/codesign", "-T", "/usr/bin/security")
            security("set-key-partition-list", "-S", "apple-tool:,apple:,codesign:", "-s", "-k", password, keychain)
            security("list-keychains", "-d", "user", "-s", *previous_keychains, keychain)
            identities = security("find-identity", "-v", "-p", "codesigning", keychain)
            fingerprints = [hashlib.sha1(cert).hexdigest().upper() for cert in profile["DeveloperCertificates"]]
            identity = next((fingerprint for fingerprint in fingerprints if fingerprint in identities), None)
            if not identity:
                raise ValueError("No valid signing identity matches the provisioning profile")
            for folder in ["Library/MobileDevice/Provisioning Profiles", "Library/Developer/Xcode/UserData/Provisioning Profiles"]:
                target = Path.home() / folder / f"{profile['UUID']}.mobileprovision"
                target.parent.mkdir(parents=True, exist_ok=True)
                if target.exists():
                    if target.read_bytes() != profile_path.read_bytes():
                        raise ValueError("An installed profile has the same UUID but different contents")
                else:
                    shutil.copyfile(profile_path, target)
                    installed.append(target)
            signing.write_text(f"DEVELOPMENT_TEAM = {team}\nCODE_SIGN_STYLE = Manual\n"
                               f"CODE_SIGN_IDENTITY = {identity}\nPROVISIONING_PROFILE_SPECIFIER = {profile['UUID']}\n")
            archive = output / "YourTJ.xcarchive"
            options = output / "ExportOptions.plist"
            options.write_bytes(plistlib.dumps({
                "method": "app-store-connect", "teamID": team, "signingStyle": "manual",
                "signingCertificate": identity, "provisioningProfiles": {BUNDLE_ID: profile["UUID"]},
                "uploadSymbols": True, "manageAppVersionAndBuildNumber": False,
            }))
            # Never pass PROVISIONING_PROFILE_SPECIFIER globally: SwiftPM and Pods
            # targets cannot use the app's profile. Release.xcconfig scopes it to Runner.
            commands = [
                ("archive", ["-workspace", "ios/Runner.xcworkspace", "-scheme", "Runner", "-configuration", "Release",
                             "-destination", "generic/platform=iOS", "-archivePath", str(archive), "archive"]),
                ("export", ["-exportArchive", "-archivePath", str(archive), "-exportOptionsPlist", str(options),
                            "-exportPath", str(output / "ios")]),
            ]
            for name, args in commands:
                with (output / f"{name}.log").open("w") as log:
                    result = subprocess.run(["xcodebuild", *args], cwd=APP, stdout=log, stderr=subprocess.STDOUT)
                if result.returncode:
                    raise RuntimeError(f"iOS {name} failed; inspect {output / (name + '.log')}")
            artifacts = list((output / "ios").glob("*.ipa"))
            if len(artifacts) != 1:
                raise ValueError("Expected one IPA after export")
            validate_exported_ipa(artifacts[0], team)
        finally:
            # Attempt every cleanup even when a keychain operation fails.
            with ExitStack() as cleanup:
                if previous_signing is None:
                    cleanup.callback(signing.unlink, missing_ok=True)
                else:
                    cleanup.callback(signing.write_bytes, previous_signing)
                for path in installed:
                    cleanup.callback(path.unlink, missing_ok=True)
                if Path(keychain).exists():
                    cleanup.callback(security, "delete-keychain", keychain)
                cleanup.callback(security, "list-keychains", "-d", "user", "-s", *previous_keychains)
    print(f"Signed IPA exported to {output / 'ios'}")


if __name__ == "__main__":
    main()

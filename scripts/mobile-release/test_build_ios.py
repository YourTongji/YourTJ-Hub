import datetime
import os
import plistlib
import tempfile
import unittest
from unittest.mock import patch

import build_ios
from build_ios import validate_profile, validate_app_entitlements


class DistributionProfileTest(unittest.TestCase):
    def setUp(self):
        self.now = datetime.datetime(2026, 9, 7, tzinfo=datetime.timezone.utc)
        self.profile = {
            "UUID": "ad5aac45-d8c0-4efa-bad7-cd8e4023afd5",
            "ExpirationDate": datetime.datetime(2027, 1, 1),
            "TeamIdentifier": ["4HJTS3G3T2"],
            "Entitlements": {"application-identifier": "4HJTS3G3T2.tj.yourtj.forumApp", "get-task-allow": False, "aps-environment": "production"},
        }

    def test_app_store_profile(self):
        validate_profile(self.profile, "4HJTS3G3T2", self.now)

    def test_release_entry_decodes_profile_before_preparing_build(self):
        self.profile["ExpirationDate"] = datetime.datetime.now() + datetime.timedelta(days=1)
        with tempfile.TemporaryDirectory() as output, patch.dict(os.environ, {
            "IOS_TEAM_ID": "4HJTS3G3T2", "MOBILE_VERSION": "1.2.3",
            "MOBILE_BUILD_NUMBER": "12", "IOS_PROFILE_PATH": output + "/profile",
            "MOBILE_OUTPUT_DIR": output,
        }), patch.object(build_ios.os, "umask"), patch.object(
            build_ios.subprocess, "check_output", return_value=plistlib.dumps(self.profile),
        ) as decode, patch.object(
            build_ios.subprocess, "run", side_effect=RuntimeError("build reached"),
        ) as build:
            with self.assertRaisesRegex(RuntimeError, "build reached"):
                build_ios.main()
            self.assertEqual(decode.call_args.args[0][:3], ["security", "cms", "-D"])
            self.assertEqual(build.call_args.args[0][1:3], ["build", "ios"])
            self.profile["Entitlements"].pop("aps-environment")
            decode.return_value = plistlib.dumps(self.profile)
            build.reset_mock()
            with self.assertRaisesRegex(ValueError, "production Push Notifications"):
                build_ios.main()
            build.assert_not_called()

    def test_exported_app_requires_push_entitlement(self):
        valid = self.profile["Entitlements"].copy()
        validate_app_entitlements(valid, "4HJTS3G3T2")
        for field, value in [("aps-environment", None), ("aps-environment", "development"), ("application-identifier", "other.app")]:
            with self.subTest(field=field, value=value), self.assertRaises(ValueError):
                validate_app_entitlements({**valid, field: value}, "4HJTS3G3T2")

    def test_requires_production_push_entitlement(self):
        for environment in (None, "development"):
            with self.subTest(environment=environment):
                self.profile["Entitlements"].pop("aps-environment", None)
                if environment:
                    self.profile["Entitlements"]["aps-environment"] = environment
                with self.assertRaises(ValueError):
                    validate_profile(self.profile, "4HJTS3G3T2", self.now)

    def test_rejects_wrong_team_bundle_expiry_or_profile_type(self):
        from copy import deepcopy
        variants = []
        for key, value in [
            ("TeamIdentifier", ["OTHERTEAM1"]),
            ("ExpirationDate", datetime.datetime(2026, 9, 6)),
            ("ProvisionedDevices", ["some-device"]),
            ("ProvisionsAllDevices", True),
            ("UUID", "../../profile"),
        ]:
            profile = deepcopy(self.profile)
            profile[key] = value
            variants.append(profile)
        for key, value in [("get-task-allow", True), ("application-identifier", "4HJTS3G3T2.other.app")]:
            profile = deepcopy(self.profile)
            profile["Entitlements"][key] = value
            variants.append(profile)
        for profile in variants:
            with self.subTest(profile=profile), self.assertRaises(ValueError):
                validate_profile(profile, "4HJTS3G3T2", self.now)


if __name__ == "__main__":
    unittest.main()

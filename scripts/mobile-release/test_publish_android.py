import unittest
from publish_android import check_package, android_version_code


class ApkIdentityTest(unittest.TestCase):
    package = "package: name='tj.yourtj.forum_app' versionCode='12' versionName='1.2.0'"
    certificate = 'ab' * 32

    def test_flutter_split_version_codes(self):
        self.assertEqual(android_version_code('2', 'arm64-v8a'), '2002')
        self.assertEqual(android_version_code('2', 'armeabi-v7a'), '1002')
        self.assertEqual(android_version_code('2', 'x86_64'), '4002')
        with self.assertRaises(ValueError): android_version_code('2100000000', 'arm64-v8a')

    def test_accepts_verified_release_identity(self):
        check_package(self.package, 'Signer #1 certificate SHA-256 digest: ' + self.certificate,
                      '1.2.0', '12', self.certificate)

    def test_build_tools_37_scheme_labels(self):
        for label in ('V1 Signer', 'V2 Signer', 'V3 Signer', 'V3.1 Signer', 'V3.2 Signer'):
            with self.subTest(label=label):
                check_package(self.package, label + ': certificate SHA-256 digest: ' + self.certificate,
                              '1.2.0', '12', self.certificate)

    def test_same_certificate_across_schemes_is_one_identity(self):
        signatures = '\n'.join(label + ': certificate SHA-256 digest: ' + self.certificate
                               for label in ('V2 Signer', 'V3 Signer'))
        check_package(self.package, signatures, '1.2.0', '12', self.certificate)

    def test_rejects_unknown_or_conflicting_scheme_certificates(self):
        valid = 'V2 Signer: certificate SHA-256 digest: ' + self.certificate
        for signatures in (valid + '\nV3 Signer: certificate SHA-256 digest: ' + 'cd' * 32,
                           'Source Stamp Signer certificate SHA-256 digest: ' + self.certificate,
                           'V2 Signer: public key SHA-256 digest: ' + self.certificate,
                           valid + 'ab', valid[:-2]):
            with self.subTest(signatures=signatures), self.assertRaises(ValueError):
                check_package(self.package, signatures, '1.2.0', '12', self.certificate)

    def test_rejects_wrong_identity_or_debug_certificate(self):
        signature = 'Signer #1 certificate SHA-256 digest: ' + self.certificate
        for package, signed in [(self.package.replace('forum_app', 'other'), signature),
                                (self.package.replace("'12'", "'11'"), signature),
                                (self.package, signature.replace(self.certificate, 'cd' * 32)),
                                (self.package, ''),
                                (self.package, signature + '\nSigner #2 certificate SHA-256 digest: ' + 'cd' * 32)]:
            with self.subTest(package=package), self.assertRaises(ValueError):
                check_package(package, signed, '1.2.0', '12', self.certificate)


class DraftReleaseTest(unittest.TestCase):
    def test_reads_existing_draft_by_id_instead_of_tag_endpoint(self):
        import json
        from unittest.mock import patch
        import publish_android as publisher
        draft = {'id': 123, 'tag_name': 'mobile-v1.0.1', 'draft': True, 'assets': []}
        with patch.object(publisher, 'gh', side_effect=[json.dumps({'databaseId': 123}), json.dumps(draft)]) as gh:
            self.assertEqual(publisher.find_release('mobile-v1.0.1'), draft)
        self.assertEqual(gh.call_args_list[1].args, ('api', 'repos/YourTongji/YourTJ-Hub/releases/123'))

    def test_missing_release_does_not_query_an_id(self):
        from unittest.mock import patch
        import publish_android as publisher
        with patch.object(publisher, 'gh', return_value=None) as gh:
            self.assertIsNone(publisher.find_release('mobile-v1.0.1'))
        gh.assert_called_once()

    def test_only_explicit_missing_release_is_treated_as_absent(self):
        from unittest.mock import patch
        from subprocess import CompletedProcess
        import publish_android as publisher
        for message in ('release not found', 'HTTP 403: Forbidden'):
            with patch.object(publisher.subprocess, 'run', return_value=CompletedProcess([], 1, '', message)):
                if message == 'release not found':
                    self.assertIsNone(publisher.gh('release', 'view', 'mobile-v1.0.1', allow_missing=True))
                else:
                    with self.assertRaises(RuntimeError):
                        publisher.gh('release', 'view', 'mobile-v1.0.1', allow_missing=True)


if __name__ == '__main__':
    unittest.main()

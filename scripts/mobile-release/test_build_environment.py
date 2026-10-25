import unittest
from pathlib import Path
from build_environment import android_package, dart_defines
from publish_preview_ios import check_groups


class BuildEnvironmentTests(unittest.TestCase):
    def test_hosts_identity_and_distribution_are_bound_together(self):
        dev = dart_defines('dev', 'a'*40, True)
        production = dart_defines('production', 'a'*40, False)
        self.assertIn('--dart-define=YOURTJ_API_BASE_URL=https://dev.yourtj.de', dev)
        self.assertIn('--dart-define=YOURTJ_OIDC_ISSUER=https://dev.yourtj.de/api/oauth', dev)
        self.assertIn('--dart-define=YOURTJ_PREVIEW=true', dev)
        self.assertIn('--dart-define=YOURTJ_API_BASE_URL=https://f.yourtj.de', production)
        for environment, source, preview in [('dev', 'a'*40, False), ('other', 'a'*40, True), ('dev', 'main', True)]:
            with self.assertRaises(ValueError): dart_defines(environment, source, preview)

    def test_dev_android_identity_is_separate_from_production(self):
        self.assertEqual(android_package('production'), 'tj.yourtj.forum_app')
        self.assertEqual(android_package('dev'), 'tj.yourtj.forum_app.dev')
        with self.assertRaises(ValueError): android_package('other')
        # Gradle derives the same identity from the YOURTJ_ENVIRONMENT define.
        gradle = (Path(__file__).resolve().parents[2] /
                  'apps/mobile/packages/forum_app/android/app/build.gradle.kts').read_text()
        self.assertIn('dartDefines["YOURTJ_ENVIRONMENT"] == "dev"', gradle)
        for environment in ('dev', 'production'):
            self.assertIn(f'"{android_package(environment)}"', gradle)

    def test_external_group_without_internal_auto_access_flag_does_not_block(self):
        # Actual ASC group response: external groups omit the internal-only flag.
        groups = [
            {'id': 'qa', 'attributes': {'name': 'Internal Test', 'isInternalGroup': True,
                                      'hasAccessToAllBuilds': True}},
            {'id': 'public', 'attributes': {'name': 'YourTJ TestFlight',
                                          'isInternalGroup': False, 'publicLinkEnabled': True}},
        ]
        check_groups(groups, 'qa')
        for internal, access in [(True, None), (True, True), (None, None), (False, True)]:
            groups[1]['attributes'].update(isInternalGroup=internal, hasAccessToAllBuilds=access)
            with self.subTest(internal=internal, access=access), self.assertRaises(ValueError):
                check_groups(groups, 'qa')

    def test_preview_group_cannot_leak_to_automatic_or_external_groups(self):
        groups = [{'id': 'qa', 'attributes': {'isInternalGroup': True, 'hasAccessToAllBuilds': False}},
                  {'id': 'public', 'attributes': {'isInternalGroup': False, 'hasAccessToAllBuilds': False}}]
        check_groups(groups, 'qa')
        with self.assertRaises(ValueError): check_groups(groups, 'public')
        groups[1]['attributes']['isInternalGroup'] = True
        for value in [True, None]:
            groups[1]['attributes']['hasAccessToAllBuilds'] = value
            with self.assertRaises(ValueError): check_groups(groups, 'qa')


if __name__ == '__main__': unittest.main()

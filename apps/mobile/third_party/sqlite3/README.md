# Local sqlite3 adaptation

The Dart implementation and ordinary platform hook are sqlite3 3.5.2 from
pub.dev, under the included MIT license. The HarmonyOS hook cross-compiles the
same SQLite3 Multiple Ciphers 2.5.0 / SQLite 3.53.4 used by that release.

The Flutter OH tool currently presents its native-asset target to hosted hooks
as Linux. A compiler from the OpenHarmony SDK therefore selects the additional
hook instead of downloading a desktop glibc binary. Other builds retain the
upstream hook and the workspace's existing encryption configuration.

Source: https://github.com/utelle/SQLite3MultipleCiphers/releases/tag/v2.5.0

Amalgamation archive SHA-256 (verified against the upstream release digest):
`cd3a598b667dea206b6c5319d4ecb9d687ee40565f9fd2ba280d0c2f93790f58`.
The amalgamation retains its upstream license notices.

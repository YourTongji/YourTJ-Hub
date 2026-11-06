import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/l10n/app_localizations.dart';
import 'package:forum_app/src/updates/preview_status.dart';

void main() {
  final now = DateTime.fromMillisecondsSinceEpoch(1000000);
  const sha = 'aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa';
  Map<String, dynamic> status() => {
    'schema': 1,
    'updatedAt': 999,
    'active': {
      'source': sha,
      'phase': 'testing',
      'until': 2000,
      'artifacts': {
        'dev': ['web', 'android'],
      },
    },
  };
  PreviewValidity classify(Object? data) => previewValidity(
    data,
    source: sha,
    purpose: 'dev',
    artifact: 'android',
    now: now,
  );

  test(
    'preview status uses the current artifact rather than Android readiness',
    () {
      addTearDown(() => debugDefaultTargetPlatformOverride = null);
      for (final platform in TargetPlatform.values) {
        debugDefaultTargetPlatformOverride = platform;
        final artifact = previewArtifact;
        expect(
          artifact,
          platform.name == 'iOS'
              ? 'ios'
              : platform.name == 'ohos'
              ? 'harmonyos'
              : platform.name,
        );
        final ready = previewValidity(
          {
            'schema': 2,
            'updatedAt': 999,
            'source': sha,
            'platforms': {
              'android': {'source': sha},
            },
          },
          source: sha,
          purpose: 'dev',
          artifact: artifact,
          now: now,
        );
        expect(
          ready,
          platform == TargetPlatform.android
              ? PreviewValidity.current
              : PreviewValidity.unknown,
        );
      }
    },
  );

  test(
    'continuous dev builds do not expire while their source stays current',
    () {
      expect(
        classify({
          'schema': 2,
          'updatedAt': 1,
          'source': sha,
          'platforms': {
            'android': {'source': sha},
          },
        }),
        PreviewValidity.current,
      );
    },
  );

  test(
    'fixed SHA remains valid across rebuilds; superseded/ended previews are stale',
    () {
      expect(classify(status()), PreviewValidity.current);
      final changed = status();
      (changed['active'] as Map)['source'] = 'b' * 40;
      expect(classify(changed), PreviewValidity.stale);
      changed['active'] = null;
      expect(classify(changed), PreviewValidity.stale);
      for (final phase in [
        'paused',
        'waiting',
        'aborted',
        'closed',
        'cancelled',
      ]) {
        final ended = status();
        (ended['active'] as Map)['phase'] = phase;
        expect(classify(ended), PreviewValidity.stale);
      }
      final expired = status();
      (expired['active'] as Map)['until'] = 900;
      expect(classify(expired), PreviewValidity.stale);
    },
  );

  test('missing, malformed, old and future status never claims current', () {
    for (final data in [
      null,
      {},
      {'schema': 1},
      status()..['updatedAt'] = -100000,
      status()..['updatedAt'] = 999999,
    ]) {
      expect(classify(data), PreviewValidity.unknown);
    }
  });

  testWidgets(
    'offline previews show an actionable notice without blocking the app',
    (tester) async {
      await tester.pumpWidget(
        MaterialApp(
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: PreviewStatusHost(
            fetchStatus: () async => throw Exception('offline'),
            child: const Text('Forum remains available'),
          ),
        ),
      );
      await tester.pump();
      expect(find.text('Forum remains available'), findsOneWidget);
      expect(find.text('Dev builds'), findsOneWidget);
      expect(find.textContaining('Cannot read'), findsOneWidget);
      await tester.pumpWidget(const SizedBox());
    },
  );
  test(
    'a newer source building is different from an available newer package',
    () {
      final data = {
        'schema': 2,
        'updatedAt': 999,
        'source': 'b' * 40,
        'platforms': {
          'android': {'source': sha},
        },
      };
      expect(classify(data), PreviewValidity.building);
      data['platforms'] = {
        'android': {'source': 'b' * 40},
      };
      expect(classify(data), PreviewValidity.stale);
      data['platforms'] = {};
      expect(classify(data), PreviewValidity.unknown);
    },
  );
}

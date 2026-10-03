import 'dart:async';
import 'dart:convert';
import 'dart:typed_data';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/src/storage/media_host.dart';
import 'package:forum_app/src/storage/media_repository.dart';
import 'package:ui_kit/ui_kit.dart';

final _gif = base64Decode(
  'R0lGODlhAQABAIAAAAAAAP///yH5BAEAAAAALAAAAAABAAEAAAIBRAA7',
);

class _Media extends MediaRepository {
  final scopes = <String>[];
  final originPolicies = <Set<String>?>[];
  final tokenReaders = <Future<String?> Function()?>[];
  final delayed = Completer<Uint8List>();
  @override
  Future<Uint8List> load(
    String url, {
    required String scopeKey,
    required String apiOrigin,
    Set<String>? allowedOrigins,
    Future<String?> Function()? readAccessToken,
  }) {
    scopes.add(scopeKey);
    originPolicies.add(allowedOrigins);
    tokenReaders.add(readAccessToken);
    return scopeKey == 'account-a:zh' ? delayed.future : Future.value(_gif);
  }

  @override
  Future<void> clear() async {
    suspend();
    await Future<void>.value();
    resume();
  }

  @override
  Object? decodedIdentity(
    String url, {
    required String scopeKey,
    required String apiOrigin,
  }) => null;
}

void main() {
  testWidgets(
    'image origin policy reaches the media transport and partitions providers',
    (tester) async {
      final repo = _Media();
      addTearDown(repo.dispose);
      Future<String?> readToken() async => null;
      Widget app(Set<String> origins) => MaterialApp(
        home: MediaHost(
          repository: repo,
          scopeKey: 'guest:zh',
          apiOrigin: 'https://example.test',
          readAccessToken: readToken,
          child: GfMediaOriginPolicy(
            origins: origins,
            child: const GfNetworkImage(
              'https://example.test/image.gif',
              errorBuilder: _error,
            ),
          ),
        ),
      );
      await tester.pumpWidget(app({'https://example.test'}));
      final first = tester.widget<Image>(find.byType(Image)).image;
      expect(repo.originPolicies.last, {'https://example.test'});
      expect(repo.tokenReaders.last, same(readToken));
      await tester.pumpWidget(
        app({'https://example.test', 'https://cdn.example.test'}),
      );
      expect(repo.originPolicies.last, {
        'https://example.test',
        'https://cdn.example.test',
      });
      expect(tester.widget<Image>(find.byType(Image)).image, isNot(first));
      await tester.runAsync(() async {
        await Future<void>.delayed(const Duration(milliseconds: 20));
      });
      await tester.pump();
      expect(tester.takeException(), isNull);
      await tester.pumpWidget(const SizedBox());
    },
  );
  testWidgets(
    'identity and clear changes fence old frames and clamp decoded memory',
    (tester) async {
      final repo = _Media();
      addTearDown(repo.dispose);
      Widget app(String scope) => MaterialApp(
        home: MediaHost(
          repository: repo,
          scopeKey: scope,
          apiOrigin: 'https://example.test',
          child: const GfNetworkImage(
            'https://example.test/image.gif',
            errorBuilder: _error,
          ),
        ),
      );
      await tester.pumpWidget(app('account-a:zh'));
      final oldProvider = tester.widget<Image>(find.byType(Image)).image;
      await tester.pumpWidget(app('account-b:de'));
      final nextProvider = tester.widget<Image>(find.byType(Image)).image;
      expect(oldProvider, isNot(nextProvider));
      repo.delayed.complete(_gif);
      await tester.runAsync(() async {
        await Future<void>.delayed(const Duration(milliseconds: 20));
      });
      await tester.pump();
      expect(repo.scopes, containsAll(['account-a:zh', 'account-b:de']));
      expect(tester.takeException(), isNull);
      expect(
        PaintingBinding.instance.imageCache.maximumSizeBytes,
        48 * 1024 * 1024,
      );
      repo.invalidate();
      await tester.pump();
      final clearedProvider = tester.widget<Image>(find.byType(Image)).image;
      expect(clearedProvider, isNot(nextProvider));
      await tester.pumpWidget(const SizedBox());
    },
  );
  testWidgets('clear never auto-refills after its asynchronous work finishes', (
    tester,
  ) async {
    final repo = _Media();
    addTearDown(repo.dispose);
    Widget app(int revision) => MaterialApp(
      home: MediaHost(
        repository: repo,
        scopeKey: 'guest:zh',
        apiOrigin: 'https://example.test',
        child: GfNetworkImage(
          'https://example.test/image.gif',
          key: ValueKey(revision),
          errorBuilder: _error,
        ),
      ),
    );
    await tester.pumpWidget(app(0));
    await tester.runAsync(() async {
      await Future<void>.delayed(const Duration(milliseconds: 20));
    });
    await tester.pump();
    final before = repo.scopes.length;
    await repo.clear();
    await tester.pump();
    await tester.pump();
    expect(repo.scopes.length, before);
    expect(find.byType(RawImage), findsNothing);
    expect(tester.takeException(), isNull);
    await tester.pumpWidget(app(1)); // Explicit retry/new view.
    await tester.runAsync(() async {
      await Future<void>.delayed(const Duration(milliseconds: 20));
    });
    await tester.pump();
    expect(repo.scopes.length, greaterThan(before));
    expect(tester.takeException(), isNull);
    await tester.pumpWidget(const SizedBox());
  });
  testWidgets(
    'invalidating while parent builds defers its notification safely',
    (tester) async {
      final repo = _Media();
      addTearDown(repo.dispose);
      var invalidate = false;
      Widget app() => MaterialApp(
        home: Builder(
          builder: (context) {
            if (invalidate) {
              invalidate = false;
              repo.invalidate();
            }
            return MediaHost(
              repository: repo,
              scopeKey: 'guest:zh',
              apiOrigin: 'https://example.test',
              child: const SizedBox(),
            );
          },
        ),
      );
      await tester.pumpWidget(app());
      invalidate = true;
      await tester.pumpWidget(app());
      await tester.pump();
      expect(tester.takeException(), isNull);
      await tester.pumpWidget(const SizedBox());
    },
  );
}

Widget _error(BuildContext context, Object error, StackTrace? stack) =>
    const SizedBox();

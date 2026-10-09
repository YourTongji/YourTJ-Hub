import 'dart:convert';
import 'dart:io';
import 'package:core/core.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  Map<String, dynamic> fixture(String name) =>
      (jsonDecode(
                File(
                  '../../../../packages/api-contract/fixtures/$name.json',
                ).readAsStringSync(),
              )
              as Map<String, dynamic>)['result']
          as Map<String, dynamic>;
  test('restricted admin mirror matches the controlled fixture', () {
    final result = AdminAnonymousList.fromJson(
      fixture('admin-anonymous-list-success'),
    );
    expect(result.items.single.owner.userId, 1);
    expect(result.items.single.persona.name, '躲进云里的猫');
    expect(result.items.single.owner.closed, false);
    expect(result.pageSize, 10);
    expect(
      const AdminAnonymousListRequest(reason: 'audit').toJson()['status'],
      'all',
    );
    expect(
      const AdminAnonymousGovernRequest(
        publicUid: 'uid',
        disabled: false,
        reason: 'restore',
      ).toJson()['disabled'],
      false,
    );
  });
  test(
    'anonymous settings and candidate mirrors preserve the controlled contract',
    () {
      final state = AnonymousIdentityState.fromJson(
        fixture('anonymous-state-success'),
      );
      expect(state.lexiconVersion, 'phrase6-v1');
      expect(state.showContent, true);
      expect(
        AnonymousIdentityState.fromJson({
          ...fixture('anonymous-state-success'),
          'showContent': false,
        }).showContent,
        false,
      );
      // Existing names/batches remain valid across generator versions.
      expect(state.persona!.name, 'C++');
      expect(state.batches.first.words.first, 'C++');
      expect(state.remaining, inInclusiveRange(0, 10));
      final batch = AnonymousNameBatch.fromJson(
        fixture('anonymous-batches-success'),
      );
      expect(batch.words, hasLength(10));
      expect(batch.words.toSet(), hasLength(10));
      for (final name in batch.words) {
        expect(name.runes, hasLength(6));
        expect(name.runes.elementAt(4), '的'.runes.single);
      }
      expect(batch.day, isNotEmpty);
      final persona = AnonymousPersona.fromJson(
        fixture('anonymous-confirm-success'),
      );
      expect(persona.kind, 'persona');
      expect(persona.publicUid, hasLength(32));
      expect(persona.profileUrl, '/a/${persona.publicUid}');
      final author = UserBriefPayload.fromJson({
        'id': 0,
        'username': persona.name,
        'avatarUrl': persona.avatarUrl,
        'kind': 'persona',
        'publicUid': persona.publicUid,
        'profileUrl': persona.profileUrl,
      });
      expect(author.id, 0);
      expect(author.publicUid, persona.publicUid);
      expect(author.toJson()['publicUid'], persona.publicUid);
    },
  );
}

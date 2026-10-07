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
  test(
    'anonymous settings and candidate mirrors preserve the controlled contract',
    () {
      final state = AnonymousIdentityState.fromJson(
        fixture('anonymous-state-success'),
      );
      expect(state.lexiconVersion, startsWith('THUOCL-'));
      expect(state.remaining, inInclusiveRange(0, 10));
      final batch = AnonymousNameBatch.fromJson(
        fixture('anonymous-batches-success'),
      );
      expect(batch.words, hasLength(10));
      expect(batch.words.toSet(), hasLength(10));
      expect(batch.day, isNotEmpty);
      final persona = AnonymousPersona.fromJson(
        fixture('anonymous-confirm-success'),
      );
      expect(persona.kind, 'persona');
      final reveal = AnonymousReveal.fromJson(fixture('anonymous-reveal-success'));
      expect(reveal.publicUid, persona.publicUid);
      expect(reveal.userId, 123);
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

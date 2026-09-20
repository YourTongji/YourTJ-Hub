import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/src/link_navigation.dart';

void main() {
  tearDown(LinkNavigation.clearSessionTrust);

  test('session trust uses the Public Suffix List registrable domain', () {
    expect(
      LinkNavigation.trustDomainFor('docs.team.example.co.uk'),
      'example.co.uk',
    );

    LinkNavigation.trustForSession('docs.team.example.co.uk');
    expect(LinkNavigation.isTrusted('status.example.co.uk'), isTrue);
    expect(LinkNavigation.isTrusted('example.com'), isFalse);
  });

  test('private suffix tenants do not share trust', () {
    LinkNavigation.trustForSession('alice.github.io');
    expect(LinkNavigation.isTrusted('alice.github.io'), isTrue);
    expect(LinkNavigation.isTrusted('bob.github.io'), isFalse);
  });
}

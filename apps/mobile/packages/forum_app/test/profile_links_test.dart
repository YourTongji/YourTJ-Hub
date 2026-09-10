import 'package:core/core.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/src/profile_links.dart';
import 'fixtures/page_fixtures.dart';

void main() {
  test(
    'public profiles retain valid unknown social providers and reject unsafe URLs',
    () {
      final user =
          UserCardPayload.fromJson(
            (userProfilePayloadJson()['props'] as Map<String, dynamic>)['user']
                as Map<String, dynamic>,
          ).copyWith(
            websiteName: 'Personal site',
            website: 'https://example.test/about',
            externalInformation: const {
              'github': ExternalLinkPayload(link: 'https://github.com/alice'),
              'mastodon': ExternalLinkPayload(
                link: 'https://social.example/@alice',
              ),
              'bad': ExternalLinkPayload(link: 'javascript:alert(1)'),
              'credentials': ExternalLinkPayload(
                link: 'https://user:pass@example.test',
              ),
              'relative': ExternalLinkPayload(link: '//example.test'),
            },
          );
      expect(
        publicProfileLinks(user).map((link) => (link.$1, link.$2.toString())),
        [
          ('Personal site', 'https://example.test/about'),
          ('GitHub', 'https://github.com/alice'),
          ('mastodon', 'https://social.example/@alice'),
        ],
      );
      expect(
        normalizeProfileLink('alice/name', prefix: 'https://github.com/'),
        'https://github.com/alice%2Fname',
      );
      expect(
        normalizeProfileLink(
          'javascript:alert(1)',
          prefix: 'https://github.com/',
        ),
        isNull,
      );
    },
  );
}

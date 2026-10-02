import 'package:core/core.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:forum_app/src/widgets/user_badge.dart';
import 'package:ui_kit/ui_kit.dart';

UserBadgePayload badge(String iconUrl) => UserBadgePayload(
  code: 'b',
  type: 'custom',
  grantMode: 'manual',
  name: 'Badge',
  description: '',
  iconType: 'asset',
  iconKey: '',
  iconUrl: iconUrl,
  color: 'blue',
  level: 'bronze',
  isEnabled: true,
  isWearable: true,
  sortOrder: 0,
  source: 'manual',
  reason: '',
  grantedAt: '',
);

void main() {
  // Built-in SVGs carry their own optical margin and fill the medallion's
  // centre box; custom artwork (often an edge-to-edge logo) keeps the
  // original 60% box so it does not outgrow the built-in badges. The rule
  // matches web isSystemBadgeArtwork: only relative /static/badges/ URLs
  // count, and an empty URL falls back to the built-in contributor artwork.
  test('built-in badge artwork fills the medallion centre box', () {
    for (final url in [
      '/static/badges/robot.svg?v=2',
      '/static/badges/robot.svg',
      '',
    ]) {
      expect(
        UserBadgeArtwork.medallion(badge(url), 80).size,
        GfBadgeMedallion.artworkSize(80),
      );
    }
  });

  test('custom badge artwork keeps the original 60% box', () {
    for (final url in [
      // absolute URL on any host is custom artwork, unlike a relative one
      'https://f.yourtj.de/static/badges/robot.svg',
      'https://thesvg.org/icons/deepseek/default.svg',
      '/file/img/2026/10/badge.png',
    ]) {
      expect(UserBadgeArtwork.medallion(badge(url), 80).size, 80 * .60);
    }
  });
}

import 'dart:math' as math;

import 'package:flutter/material.dart';
import 'package:flutter/rendering.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:ui_kit/ui_kit.dart';

import '../helpers.dart';

/// Effective clip path of [node] in its own coordinates, or null when the
/// render object does not clip its child.
Path? effectiveClipPath(RenderObject node) {
  if (node is RenderClipPath) {
    return node.clipper?.getClip(node.size) ??
        (Path()..addRect(Offset.zero & node.size));
  }
  if (node is RenderPhysicalShape) {
    return node.clipper?.getClip(node.size) ??
        (Path()..addRect(Offset.zero & node.size));
  }
  if (node is RenderClipOval) {
    final Rect oval =
        node.clipper?.getClip(node.size) ?? (Offset.zero & node.size);
    return Path()..addOval(oval);
  }
  if (node is RenderClipRRect) {
    return Path()..addRRect(
      node.clipper?.getClip(node.size) ??
          node.borderRadius
              .resolve(node.textDirection)
              .toRRect(Offset.zero & node.size),
    );
  }
  if (node is RenderClipRect) {
    return Path()..addRect(Offset.zero & node.size);
  }
  return null;
}

/// The outer avatar and inset image each keep a circular clip at their own
/// bounds. Ancestor clips must accept the whole outer circle. Sampling 24
/// directions catches the historical hexagon without mistaking a correctly
/// inset image circle for a clipped outer ring.
void expectCircularClipGeometry(WidgetTester tester, Finder avatarFinder) {
  final RenderObject avatar = tester.renderObject(avatarFinder);
  final Rect avatarRect = tester.getRect(avatarFinder);
  final List<(Path, Rect)> clips = <(Path, Rect)>[];

  void collect(RenderObject node) {
    final Path? path = effectiveClipPath(node);
    if (path != null) {
      final globalPath = path.transform(node.getTransformTo(null).storage);
      clips.add((globalPath, globalPath.getBounds()));
    }
    node.visitChildren(collect);
  }

  collect(avatar);
  for (RenderObject? node = avatar.parent; node != null; node = node.parent) {
    final Path? path = effectiveClipPath(node);
    if (path != null) {
      clips.add((
        path.transform(node.getTransformTo(null).storage),
        avatarRect,
      ));
    }
  }

  expect(clips, isNotEmpty, reason: 'the avatar must be clipped');
  for (final (clip, bounds) in clips) {
    final double radius = bounds.size.shortestSide / 2 * 0.95;
    for (int step = 0; step < 24; step++) {
      final double angle = step * math.pi / 12;
      final Offset point =
          bounds.center + Offset(math.cos(angle), math.sin(angle)) * radius;
      expect(
        clip.contains(point),
        isTrue,
        reason:
            'every clip over the avatar must accept the inscribed circle; a '
            'clip rejects $point (${step * 15}°)',
      );
    }
  }
}

/// Asserts the shared circular-avatar contract: the decoration is a circle,
/// the child is clipped to it, neither size nor ring drifts, and the painted
/// region really is circular.
///
/// Regression guard for issue #877 (DM avatars reported as hexagons): the
/// clip shape must survive every avatar refactor.
void expectCircularAvatar(
  WidgetTester tester,
  Finder avatarFinder, {
  required double size,
  required bool ring,
}) {
  final GfAvatar avatar = tester.widget<GfAvatar>(avatarFinder);
  expect(avatar.size, size);
  expect(avatar.ring, ring);

  final Finder containerFinder = find
      .descendant(of: avatarFinder, matching: find.byType(Container))
      .first;
  final Container container = tester.widget<Container>(containerFinder);
  final BoxDecoration decoration = container.decoration! as BoxDecoration;
  expect(
    decoration.shape,
    BoxShape.circle,
    reason: 'DM avatars must stay circular (issue #877)',
  );
  expect(
    container.clipBehavior,
    Clip.antiAlias,
    reason: 'the image must be clipped to the circular decoration',
  );
  expect(
    decoration.border,
    isNull,
    reason: 'the ring and inner image own their bounds independently',
  );
  final foreground = container.foregroundDecoration as BoxDecoration?;
  expect(foreground?.border, ring ? isNotNull : isNull);
  expect(tester.getSize(avatarFinder), Size(size, size));
  expect(
    find.descendant(of: avatarFinder, matching: find.byType(CustomPaint)),
    findsNothing,
    reason:
        'a custom painter inside the avatar could draw a non-circular shape',
  );
  if (ring) {
    final inner = find.descendant(
      of: avatarFinder,
      matching: find.byType(ClipOval),
    );
    expect(inner, findsOneWidget);
    expect(
      tester.getRect(inner),
      tester.getRect(avatarFinder).deflate(2),
      reason: 'the inset portrait must have its own circular clip',
    );
  }
  expectCircularClipGeometry(tester, avatarFinder);
}

void main() {
  testWidgets(
    'ring reserves space around a separately clipped circular image',
    (tester) async {
      await tester.pumpWidget(
        gfApp(
          const GfAvatar(
            src: 'https://example.test/avatar.png',
            size: 40,
            ring: true,
          ),
        ),
      );
      // Keep the source framing inside the ring, with a separate inner circle
      // so the inset square never introduces the old flat-sided shape.
      expect(tester.getSize(find.byType(Image)), const Size(36, 36));
      expect(
        tester.getRect(find.byType(ClipOval)),
        tester.getRect(find.byType(Image)),
      );
    },
  );

  testWidgets('GfAvatar keeps every DM size circular and clipped', (
    tester,
  ) async {
    await tester.pumpWidget(
      gfApp(
        const Column(
          children: <Widget>[
            GfAvatar(src: '', size: 32),
            GfAvatar(src: '', size: 36, ring: true),
            GfAvatar(src: '', size: 40, ring: true),
          ],
        ),
      ),
    );

    expectCircularAvatar(
      tester,
      find.byType(GfAvatar).at(0),
      size: 32,
      ring: false,
    );
    expectCircularAvatar(
      tester,
      find.byType(GfAvatar).at(1),
      size: 36,
      ring: true,
    );
    expectCircularAvatar(
      tester,
      find.byType(GfAvatar).at(2),
      size: 40,
      ring: true,
    );
  });

  testWidgets('GfAvatar clips a loaded image to the circular decoration', (
    tester,
  ) async {
    await tester.pumpWidget(
      gfApp(const GfAvatar(src: 'https://example.test/avatar.png', size: 32)),
    );

    final Finder avatarFinder = find.byType(GfAvatar);
    expectCircularAvatar(tester, avatarFinder, size: 32, ring: false);
    expect(
      find.descendant(of: avatarFinder, matching: find.byType(Image)),
      findsOneWidget,
    );
  });

  testWidgets('small ring avatars fit the image inside the circular border', (
    tester,
  ) async {
    await tester.pumpWidget(
      gfApp(
        const Column(
          children: <Widget>[
            GfAvatar(
              src: 'https://example.test/avatar-small.png',
              size: 24,
              ring: true,
            ),
            GfAvatar(
              src: 'https://example.test/avatar-conversation.png',
              size: 40,
              ring: true,
            ),
          ],
        ),
      ),
    );

    for (int index = 0; index < 2; index++) {
      final Finder avatarFinder = find.byType(GfAvatar).at(index);
      final double size = index == 0 ? 24 : 40;
      expectCircularAvatar(tester, avatarFinder, size: size, ring: true);

      final Finder imageFinder = find.descendant(
        of: avatarFinder,
        matching: find.byType(Image),
      );
      expect(imageFinder, findsOneWidget);
      expect(
        tester.getRect(imageFinder),
        tester.getRect(avatarFinder).deflate(2),
        reason: 'the ring must not cover the $size px avatar image',
      );
    }
  });

  testWidgets('conversation list row keeps the circular 40px ring avatar', (
    tester,
  ) async {
    await tester.pumpWidget(
      gfApp(
        const SizedBox(
          width: 320,
          child: GfConversationRow(
            avatarUrl: '',
            name: 'Bob',
            lastMessage: '你好',
            time: '10:30',
            unreadCount: 0,
          ),
        ),
      ),
    );

    final Finder avatarFinder = find.descendant(
      of: find.byType(GfConversationRow),
      matching: find.byType(GfAvatar),
    );
    expect(avatarFinder, findsOneWidget);
    expectCircularAvatar(tester, avatarFinder, size: 40, ring: true);
  });

  testWidgets('topic feed list keeps circular participant avatars', (
    tester,
  ) async {
    await tester.pumpWidget(
      gfApp(
        const SizedBox(
          width: 360,
          child: GfTopicRow(
            pinnedLabel: 'pinned',
            title: 'Topic',
            description: '',
            categories: <GfTopicCategory>[],
            participantAvatarUrls: <String>['', ''],
            activityText: 'updated',
            replyCount: 2,
            showDivider: false,
          ),
        ),
      ),
    );

    final Finder avatars = find.descendant(
      of: find.byType(GfTopicRow),
      matching: find.byType(GfAvatar),
    );
    expect(avatars, findsNWidgets(2));
    for (int index = 0; index < 2; index++) {
      expectCircularAvatar(tester, avatars.at(index), size: 24, ring: true);
    }
  });
}

import 'dart:math' as math;

import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:flutter/rendering.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:ui_kit/ui_kit.dart';

import 'chat_visible_read_test.dart' show pumpChat;
import 'pages_behavior_test.dart' show makeChatMessage;

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

/// Asserts the circular-avatar contract on every DM surface: the decoration is
/// a circle, the child is clipped to it, neither size nor ring drifts, and the
/// painted region really is circular.
///
/// Regression guard for issue #877 (DM avatars reported as hexagons): the
/// conversation list and both message directions must keep a circular
/// decoration clipped with [Clip.antiAlias] at the documented size.
void expectCircularAvatar(
  WidgetTester tester,
  Finder avatarFinder, {
  required double size,
  required bool ring,
}) {
  final GfAvatar avatar = tester.widget<GfAvatar>(avatarFinder);
  expect(avatar.size, size);
  expect(avatar.ring, ring);

  final Container container = tester.widget<Container>(
    find.descendant(of: avatarFinder, matching: find.byType(Container)).first,
  );
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
  expect(
    (container.foregroundDecoration as BoxDecoration?)?.border,
    ring ? isNotNull : isNull,
  );
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
  setUp(() {
    SharedPreferences.setMockInitialValues({});
    FlutterSecureStorage.setMockInitialValues({});
  });

  testWidgets('conversation list rows keep circular 40px avatars', (
    tester,
  ) async {
    await pumpChat(tester, targetUserId: null);

    final Finder rows = find.byType(GfConversationRow);
    expect(rows, findsNWidgets(2));
    for (int index = 0; index < 2; index++) {
      final Finder avatarFinder = find.descendant(
        of: rows.at(index),
        matching: find.byType(GfAvatar),
      );
      expect(avatarFinder, findsOneWidget);
      expectCircularAvatar(tester, avatarFinder, size: 40, ring: true);
    }
  });

  testWidgets('both DM message directions keep circular 32px avatars', (
    tester,
  ) async {
    await pumpChat(
      tester,
      messages: <ChatMessagePayload>[
        makeChatMessage(1),
        makeChatMessage(2).copyWith(isSelf: true),
      ],
    );

    final Finder avatars = find.byType(GfAvatar);
    expect(avatars, findsNWidgets(3), reason: 'app bar + peer row + self row');

    Finder? header;
    final List<Finder> messageAvatars = <Finder>[];
    for (int index = 0; index < avatars.evaluate().length; index++) {
      final Finder finder = avatars.at(index);
      if (tester.widget<GfAvatar>(finder).size == 32) {
        messageAvatars.add(finder);
      } else {
        header = finder;
      }
    }

    expect(header, isNotNull);
    expectCircularAvatar(tester, header!, size: 36, ring: true);

    // Both directions use the full 32px image. An inset ring only on the
    // outgoing avatar would shrink its portrait to 28px.
    expect(messageAvatars, hasLength(2));
    final double screenWidth = tester
        .getSize(find.byType(Scaffold).first)
        .width;
    final Finder peerAvatar = messageAvatars.singleWhere(
      (Finder finder) => tester.getCenter(finder).dx < screenWidth / 2,
    );
    final Finder selfAvatar = messageAvatars.singleWhere(
      (Finder finder) => tester.getCenter(finder).dx > screenWidth / 2,
    );
    expectCircularAvatar(tester, peerAvatar, size: 32, ring: false);
    expectCircularAvatar(tester, selfAvatar, size: 32, ring: false);
  });
}

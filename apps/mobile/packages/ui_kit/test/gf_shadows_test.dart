import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:ui_kit/ui_kit.dart';

/// Exact-value assertions for every [GfShadows] preset.
///
/// The authoritative source is `apps/gooseforum/resource/src/styles/
/// components.css` — every shadow is `rgb(15 23 42)` (slate-900) with an
/// alpha scaled by `--gf-depth` (1):
/// - card:      `0 2px 12px rgb(0 0 0 / 0.04)`
/// - menu:      `0 18px 40px -24px rgb(15 23 42 / 0.45)`
/// - floating:  `0 18px 48px -24px rgb(15 23 42 / 0.50)` +
///              `0 4px 16px -12px rgb(15 23 42 / 0.35)`
/// - drawer:    `18px 0 42px -30px rgb(15 23 42 / 0.55)`
/// - alert:     `0 14px 42px -32px rgb(15 23 42 / 0.65)`
/// - tooltip:   `0 10px 24px -18px rgb(15 23 42 / 0.65)`
void main() {
  void expectShadow(
    List<BoxShadow> shadows,
    int index, {
    required Color color,
    required Offset offset,
    required double blurRadius,
    required double spreadRadius,
  }) {
    expect(shadows.length, greaterThan(index), reason: 'shadow layer exists');
    final BoxShadow s = shadows[index];
    expect(s.color, color, reason: 'layer $index color');
    expect(s.offset, offset, reason: 'layer $index offset');
    expect(s.blurRadius, blurRadius, reason: 'layer $index blur');
    expect(s.spreadRadius, spreadRadius, reason: 'layer $index spread');
  }

  group('GfShadows exact web values', () {
    test('card: 0 2px 12px rgb(0 0 0 / 0.04)', () {
      final List<BoxShadow> s = GfShadows.standard.card;
      expect(s, hasLength(1));
      expectShadow(
        s,
        0,
        color: const Color(0x0A000000),
        offset: const Offset(0, 2),
        blurRadius: 12,
        spreadRadius: 0,
      );
    });

    test('menu: 0 18px 40px -24px rgb(15 23 42 / 0.45)', () {
      final List<BoxShadow> s = GfShadows.standard.menu;
      expect(s, hasLength(1));
      expectShadow(
        s,
        0,
        color: const Color(0x730F172A),
        offset: const Offset(0, 18),
        blurRadius: 40,
        spreadRadius: -24,
      );
    });

    test('floating: two-layer 0.50 + 0.35 shadows', () {
      final List<BoxShadow> s = GfShadows.standard.floating;
      expect(s, hasLength(2));
      expectShadow(
        s,
        0,
        color: const Color(0x800F172A),
        offset: const Offset(0, 18),
        blurRadius: 48,
        spreadRadius: -24,
      );
      expectShadow(
        s,
        1,
        color: const Color(0x590F172A),
        offset: const Offset(0, 4),
        blurRadius: 16,
        spreadRadius: -12,
      );
    });

    test('drawer: 18px 0 42px -30px rgb(15 23 42 / 0.55)', () {
      final List<BoxShadow> s = GfShadows.standard.drawer;
      expect(s, hasLength(1));
      expectShadow(
        s,
        0,
        color: const Color(0x8C0F172A),
        offset: const Offset(18, 0),
        blurRadius: 42,
        spreadRadius: -30,
      );
    });

    test('alert: 0 14px 42px -32px rgb(15 23 42 / 0.65)', () {
      final List<BoxShadow> s = GfShadows.standard.alert;
      expect(s, hasLength(1));
      expectShadow(
        s,
        0,
        color: const Color(0xA60F172A),
        offset: const Offset(0, 14),
        blurRadius: 42,
        spreadRadius: -32,
      );
    });

    test('tooltip: 0 10px 24px -18px rgb(15 23 42 / 0.65)', () {
      final List<BoxShadow> s = GfShadows.standard.tooltip;
      expect(s, hasLength(1));
      expectShadow(
        s,
        0,
        color: const Color(0xA60F172A),
        offset: const Offset(0, 10),
        blurRadius: 24,
        spreadRadius: -18,
      );
    });

    test('shadowColor is slate-900', () {
      expect(GfShadows.shadowColor, const Color(0xFF0F172A));
    });
  });
}

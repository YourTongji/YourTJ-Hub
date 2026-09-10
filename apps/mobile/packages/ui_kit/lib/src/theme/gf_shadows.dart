import 'package:flutter/material.dart';

/// Shadow tokens mirroring the web floating-surface hierarchy
/// (`apps/gooseforum/resource/src/styles/components.css`).
///
/// Every web shadow is built from `rgb(15 23 42)` (slate-900) with an alpha
/// that scales with `--gf-depth` (1). The six presets map 1:1:
/// - card:      `0 2px 12px rgb(0 0 0 / 0.04)`
/// - menu:      `0 18px 40px -24px rgb(15 23 42 / 0.45)`
/// - floating:  `0 18px 48px -24px rgb(15 23 42 / 0.50)` +
///              `0 4px 16px -12px rgb(15 23 42 / 0.35)`
/// - drawer:    `18px 0 42px -30px rgb(15 23 42 / 0.55)`
/// - alert:     `0 14px 42px -32px rgb(15 23 42 / 0.65)`
/// - tooltip:   `0 10px 24px -18px rgb(15 23 42 / 0.65)`
@immutable
class GfShadows extends ThemeExtension<GfShadows> {
  const GfShadows({
    required this.card,
    required this.menu,
    required this.floating,
    required this.drawer,
    required this.alert,
    required this.tooltip,
  });

  /// `0 2px 12px rgb(0 0 0 / 0.04)` — raised cards.
  final List<BoxShadow> card;

  /// `0 18px 40px -24px rgb(15 23 42 / 0.45)` — dropdown menus.
  final List<BoxShadow> menu;

  /// Two-layer `gf-floating-surface` shadow (`0 18px 48px -24px / 0.50` +
  /// `0 4px 16px -12px / 0.35`) — floating bars, composers, sheets.
  final List<BoxShadow> floating;

  /// `18px 0 42px -30px rgb(15 23 42 / 0.55)` — drawer surfaces.
  final List<BoxShadow> drawer;

  /// `0 14px 42px -32px rgb(15 23 42 / 0.65)` — alerts.
  final List<BoxShadow> alert;

  /// `0 10px 24px -18px rgb(15 23 42 / 0.65)` — tooltips.
  final List<BoxShadow> tooltip;

  /// The slate-900 shadow color used across all presets.
  static const Color shadowColor = Color(0xFF0F172A);

  /// Values for `--gf-depth: 1` (the only depth currently defined).
  static const GfShadows standard = GfShadows(
    card: <BoxShadow>[
      BoxShadow(
        color: Color(0x0A000000), // rgb(0 0 0 / 0.04)
        offset: Offset(0, 2),
        blurRadius: 12,
      ),
    ],
    menu: <BoxShadow>[
      BoxShadow(
        color: Color(0x730F172A), // rgb(15 23 42 / 0.45)
        offset: Offset(0, 18),
        blurRadius: 40,
        spreadRadius: -24,
      ),
    ],
    floating: <BoxShadow>[
      BoxShadow(
        color: Color(0x800F172A), // rgb(15 23 42 / 0.50)
        offset: Offset(0, 18),
        blurRadius: 48,
        spreadRadius: -24,
      ),
      BoxShadow(
        color: Color(0x590F172A), // rgb(15 23 42 / 0.35)
        offset: Offset(0, 4),
        blurRadius: 16,
        spreadRadius: -12,
      ),
    ],
    drawer: <BoxShadow>[
      BoxShadow(
        color: Color(0x8C0F172A), // rgb(15 23 42 / 0.55)
        offset: Offset(18, 0),
        blurRadius: 42,
        spreadRadius: -30,
      ),
    ],
    alert: <BoxShadow>[
      BoxShadow(
        color: Color(0xA60F172A), // rgb(15 23 42 / 0.65)
        offset: Offset(0, 14),
        blurRadius: 42,
        spreadRadius: -32,
      ),
    ],
    tooltip: <BoxShadow>[
      BoxShadow(
        color: Color(0xA60F172A), // rgb(15 23 42 / 0.65)
        offset: Offset(0, 10),
        blurRadius: 24,
        spreadRadius: -18,
      ),
    ],
  );

  @override
  GfShadows copyWith({
    List<BoxShadow>? card,
    List<BoxShadow>? menu,
    List<BoxShadow>? floating,
    List<BoxShadow>? drawer,
    List<BoxShadow>? alert,
    List<BoxShadow>? tooltip,
  }) {
    return GfShadows(
      card: card ?? this.card,
      menu: menu ?? this.menu,
      floating: floating ?? this.floating,
      drawer: drawer ?? this.drawer,
      alert: alert ?? this.alert,
      tooltip: tooltip ?? this.tooltip,
    );
  }

  @override
  GfShadows lerp(GfShadows? other, double t) {
    if (other == null) return this;
    List<BoxShadow> lerpList(List<BoxShadow> a, List<BoxShadow> b) {
      if (a.length != b.length) return t < 0.5 ? a : b;
      return List<BoxShadow>.generate(
        a.length,
        (int i) => BoxShadow.lerp(a[i], b[i], t)!,
      );
    }

    return GfShadows(
      card: lerpList(card, other.card),
      menu: lerpList(menu, other.menu),
      floating: lerpList(floating, other.floating),
      drawer: lerpList(drawer, other.drawer),
      alert: lerpList(alert, other.alert),
      tooltip: lerpList(tooltip, other.tooltip),
    );
  }

  @override
  bool operator ==(Object other) {
    if (other is! GfShadows) return false;
    bool listEq(List<BoxShadow> a, List<BoxShadow> b) {
      if (a.length != b.length) return false;
      for (int i = 0; i < a.length; i++) {
        if (a[i] != b[i]) return false;
      }
      return true;
    }

    return listEq(other.card, card) &&
        listEq(other.menu, menu) &&
        listEq(other.floating, floating) &&
        listEq(other.drawer, drawer) &&
        listEq(other.alert, alert) &&
        listEq(other.tooltip, tooltip);
  }

  @override
  int get hashCode => Object.hash(
    Object.hashAll(card),
    Object.hashAll(menu),
    Object.hashAll(floating),
    Object.hashAll(drawer),
    Object.hashAll(alert),
    Object.hashAll(tooltip),
  );
}

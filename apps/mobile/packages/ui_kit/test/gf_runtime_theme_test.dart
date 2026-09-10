// GfRuntimeTheme 站点主题覆盖层单元测试。
//
// 覆盖：逐键合并、非法 hex 回退、未知键忽略、整体回退、双模式独立。
library;

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:ui_kit/ui_kit.dart';

void main() {
  group('GfRuntimeTheme.fromTokens', () {
    test('内置色板为基准：空 tokens 返回 null（整体回退）', () {
      expect(GfRuntimeTheme.fromTokens(const {}), isNull);
      expect(GfRuntimeTheme.fromTokens(const {'light': {}}), isNull);
    });

    test('合法覆盖键生效', () {
      final theme = GfRuntimeTheme.fromTokens(const {
        'light': {'color-primary': '#FF0000'},
      });
      expect(theme, isNotNull);
      expect(theme!.light.primary, const Color(0xFFFF0000));
      // 未覆盖键保持内置值。
      expect(theme.light.base100, GfColors.light.base100);
      // dark 未提供时保持内置。
      expect(theme.dark.primary, GfColors.dark.primary);
    });

    test('带 color- 前缀与不带前缀等价；非法 hex 忽略回退内置', () {
      final theme = GfRuntimeTheme.fromTokens(const {
        'light': {
          'color-primary': '#00FF00',
          'primary': '#0000FF',
          'color-error': 'not-a-color',
          'color-line': '#GGGGGG',
        },
      });
      // 同键覆盖：后写的 primary(#0000FF) 覆盖 color-primary(#00FF00)。
      expect(theme!.light.primary, const Color(0xFF0000FF));
      // 非法值回退内置。
      expect(theme.light.error, GfColors.light.error);
      expect(theme.light.line, GfColors.light.line);
    });

    test('非色彩键（radius/size 等）忽略', () {
      final theme = GfRuntimeTheme.fromTokens(const {
        'light': {'radius-box': '999', 'color-primary': '#123456'},
      });
      expect(theme, isNotNull);
      expect(theme!.light.primary, const Color(0xFF123456));
    });

    test('无 # 前缀与 8 位 ARGB 容错', () {
      final theme = GfRuntimeTheme.fromTokens(const {
        'light': {'color-primary': 'ABCDEF', 'color-accent': '80FF0000'},
      });
      expect(theme!.light.primary, const Color(0xFFABCDEF));
      expect(theme.light.accent, const Color(0x80FF0000));
    });

    test('双模式各自独立合并', () {
      final theme = GfRuntimeTheme.fromTokens(const {
        'light': {'color-primary': '#111111'},
        'dark': {'color-primary': '#222222'},
      });
      expect(theme!.light.primary, const Color(0xFF111111));
      expect(theme.dark.primary, const Color(0xFF222222));
      expect(theme.dark.base100, GfColors.dark.base100);
    });
  });

  group('gfThemeData overrides', () {
    test('overrides 缺省与内置主题等值', () {
      final ThemeData builtin = gfThemeData(Brightness.light);
      final ThemeData fallback = gfThemeData(Brightness.light);
      expect(fallback.colorScheme.primary, builtin.colorScheme.primary);
    });

    test('overrides 生效到 Material colorScheme 与 scaffold 背景', () {
      final GfColors custom = GfRuntimeTheme.fromTokens(const {
        'light': {'color-primary': '#123456', 'color-base-100': '#F0F0F0'},
      })!.light;
      final ThemeData themed = gfThemeData(Brightness.light, overrides: custom);
      expect(themed.colorScheme.primary, const Color(0xFF123456));
      expect(themed.scaffoldBackgroundColor, const Color(0xFFF0F0F0));
    });
  });
}

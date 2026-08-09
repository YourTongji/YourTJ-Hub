import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:ui_kit/ui_kit.dart';

import '../l10n/app_localizations.dart';
import 'router.dart';
import 'theme_mode.dart';

/// yourtj 移动端根应用。
///
/// 主题严格来自 ui_kit 设计 token(web tokens.css 的 1:1 镜像),
/// light/dark 双主题,默认跟随系统,设置页可手动切换。
class GfApp extends ConsumerWidget {
  const GfApp({super.key, this.locale});

  /// 强制语言(测试用);null 时跟随系统(zh/en 一期)。
  final Locale? locale;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final ThemeMode mode = ref.watch(themeModeProvider);

    return MaterialApp.router(
      title: 'yourtj',
      debugShowCheckedModeBanner: false,
      theme: gfThemeData(Brightness.light),
      darkTheme: gfThemeData(Brightness.dark),
      themeMode: mode,
      routerConfig: appRouter,
      // i18n:zh/en 一期(web locale 对齐),跟随系统语言。
      localizationsDelegates: const [
        AppLocalizations.delegate,
        GlobalMaterialLocalizations.delegate,
        GlobalWidgetsLocalizations.delegate,
        GlobalCupertinoLocalizations.delegate,
      ],
      supportedLocales: AppLocalizations.supportedLocales,
      locale: locale,
    );
  }
}

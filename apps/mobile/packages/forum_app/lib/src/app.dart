import 'package:flutter/material.dart';
import 'package:flutter_quill/flutter_quill.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:ui_kit/ui_kit.dart';

import '../l10n/app_localizations.dart';
import 'router.dart';
import 'app_locale.dart';
import 'site_theme.dart';
import 'theme_mode.dart';
import 'push/push_service.dart';
import 'updates/update_host.dart';

/// yourtj 移动端根应用。
///
/// 主题严格来自 ui_kit 设计 token(web tokens.css 的 1:1 镜像),
/// light/dark 双主题,默认跟随系统,设置页可手动切换。
class GfApp extends ConsumerWidget {
  const GfApp({super.key, this.locale});

  /// 强制语言(测试用);null 时使用设备内语言偏好或跟随系统。
  final Locale? locale;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final ThemeMode mode = ref.watch(themeModeProvider);
    final GfRuntimeTheme? runtime = ref.watch(
      siteThemeProvider.select((s) => s.following ? s.runtime : null),
    );

    // Restore opted-in native delivery and notification navigation.
    ref.watch(pushBootstrapProvider);

    return MaterialApp.router(
      title: 'YourTJ',
      debugShowCheckedModeBanner: false,
      // 站点主题同步：runtime 覆盖为 null 时回退内置 tokens.json 镜像主题。
      theme: gfThemeData(Brightness.light, overrides: runtime?.light),
      darkTheme: gfThemeData(Brightness.dark, overrides: runtime?.dark),
      themeMode: mode,
      routerConfig: appRouter,
      builder: (context, child) => MobileUpdateHost(
        key: appUpdateHostKey,
        navigatorKey: appNavigatorKey,
        child: child ?? const SizedBox.shrink(),
      ),
      // The same four languages as Web, resolved without a locale flash on switching.
      localizationsDelegates: const [
        AppLocalizations.delegate,
        FlutterQuillLocalizations.delegate,
        GlobalMaterialLocalizations.delegate,
        GlobalWidgetsLocalizations.delegate,
        GlobalCupertinoLocalizations.delegate,
      ],
      supportedLocales: AppLocalizations.supportedLocales,
      locale: locale ?? ref.watch(appLocaleProvider),
      localeListResolutionCallback: (locales, supported) =>
          resolveAppLocale(locale ?? ref.read(appLocaleProvider), locales),
    );
  }
}

import 'dart:convert';
import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:flutter_quill/flutter_quill.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:ui_kit/ui_kit.dart';

import '../l10n/app_localizations.dart';
import 'navigation/session_overlays.dart';
import 'analytics/analytics_host.dart';
import 'router.dart';
import 'private_notes.dart';
import 'app_locale.dart';
import 'site_theme.dart';
import 'theme_mode.dart';
import 'push/push_service.dart';
import 'startup_experience.dart';
import 'updates/update_host.dart';
import 'providers.dart';
import 'apple/apple_sign_in.dart';
import 'widgets/app_system_ui_overlay.dart';
import 'app_config.dart';
import 'current_user.dart';
import 'storage/media_host.dart';
import 'storage/storage_providers.dart';
import 'storage/storage_gate.dart';

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
    final disableAnimations = MediaQuery.disableAnimationsOf(context);
    final ThemeMode mode = ref.watch(themeModeProvider);
    final GfRuntimeTheme? runtime = ref.watch(storageReadyProvider)
        ? ref.watch(
            siteThemeProvider.select((s) => s.following ? s.runtime : null),
          )
        : null;

    return MaterialApp.router(
      title: 'YourTJ',
      debugShowCheckedModeBanner: false,
      // 站点主题同步：runtime 覆盖为 null 时回退内置 tokens.json 镜像主题。
      theme: gfThemeData(
        Brightness.light,
        overrides: runtime?.light,
        disableAnimations: disableAnimations,
      ),
      darkTheme: gfThemeData(
        Brightness.dark,
        overrides: runtime?.dark,
        disableAnimations: disableAnimations,
      ),
      themeMode: mode,
      themeAnimationDuration: GfMotion.duration(context, GfMotion.layout),
      themeAnimationCurve: GfMotion.layoutCurve,
      routerConfig: appRouter,
      builder: (context, child) => AppSystemUiOverlay(
        child: StorageGate(
          child: _AppBusinessHosts(
            locale: locale,
            child: child ?? const SizedBox.shrink(),
          ),
        ),
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

/// No business bootstrap, navigation listeners or writable feature hosts run
/// before StorageGate has completed recovery.
class _AppBusinessHosts extends ConsumerWidget {
  const _AppBusinessHosts({required this.locale, required this.child});
  final Locale? locale;
  final Widget child;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final mediaRepository = ref.watch(mediaRepositoryProvider);
    final user = ref.watch(currentUserProvider);
    final origin = Uri.parse(
      AppConfig.apiBaseUrl.isNotEmpty
          ? AppConfig.apiBaseUrl
          : GfApiClient.defaultBaseUrl,
    ).origin;
    final language = resolveAppLocale(
      locale ?? ref.watch(appLocaleProvider),
    ).languageCode;
    final epoch = ref.watch(offlineCacheEpochProvider);
    final mediaScope = user.isLoading || user.hasError
        ? 'pending:$epoch'
        : jsonEncode([origin, user.valueOrNull?.id ?? 0, language]);

    // Restore opted-in native delivery and notification navigation.
    ref.watch(pushBootstrapProvider);
    ref.watch(appleAuthBootstrapProvider);
    ref.listen(scheduleWidgetLinkProvider, (_, next) {
      final uri = next.valueOrNull;
      if (uri?.scheme == 'yourtj' && uri?.host == 'campus') {
        appRouter.go('/campus');
      }
    });

    return MediaHost(
      repository: mediaRepository,
      scopeKey: mediaScope,
      apiOrigin: origin,
      child: StartupExperience(
        child: MobileUpdateHost(
          key: appUpdateHostKey,
          navigatorKey: appNavigatorKey,
          child: SessionOverlayHost(
            registry: appSessionOverlays,
            child: AnalyticsHost(
              router: appRouter,
              child: PrivateNotesHost(child: child),
            ),
          ),
        ),
      ),
    );
  }
}

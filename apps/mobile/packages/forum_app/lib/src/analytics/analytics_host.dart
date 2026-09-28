import 'package:flutter/foundation.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../app_config.dart';
import 'analytics_consent.dart';
import 'visitor_analytics.dart';

final visitorAnalyticsProvider = Provider<VisitorAnalytics?>((ref) {
  if (!visitorAnalyticsAvailable(
    apiBaseUrl: AppConfig.apiBaseUrl,
    release: kReleaseMode,
    web: kIsWeb,
    platform: defaultTargetPlatform,
  )) {
    return null;
  }
  final analytics = VisitorAnalytics(
    os: defaultTargetPlatform == TargetPlatform.iOS ? 'iOS' : 'Android OS',
  );
  ref.onDispose(analytics.dispose);
  return analytics;
});

class AnalyticsHost extends ConsumerStatefulWidget {
  const AnalyticsHost({super.key, required this.router, required this.child});
  final GoRouter router;
  final Widget child;

  @override
  ConsumerState<AnalyticsHost> createState() => _AnalyticsHostState();
}

class _AnalyticsHostState extends ConsumerState<AnalyticsHost>
    with WidgetsBindingObserver {
  bool _scheduled = false;
  VisitorAnalytics? _analytics;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
    widget.router.routerDelegate.addListener(_schedule);
  }

  @override
  void didUpdateWidget(AnalyticsHost oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.router != widget.router) {
      oldWidget.router.routerDelegate.removeListener(_schedule);
      widget.router.routerDelegate.addListener(_schedule);
      _schedule();
    }
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    ref
        .read(visitorAnalyticsProvider)
        ?.setActive(state == AppLifecycleState.resumed);
    if (state == AppLifecycleState.resumed) _schedule();
  }

  void _schedule() {
    if (_scheduled) return;
    _scheduled = true;
    WidgetsBinding.instance.addPostFrameCallback((_) {
      _scheduled = false;
      if (!mounted) return;
      final analytics = ref.read(visitorAnalyticsProvider);
      final lifecycle = WidgetsBinding.instance.lifecycleState;
      analytics?.setActive(
        lifecycle == null || lifecycle == AppLifecycleState.resumed,
      );
      analytics?.setEnabled(ref.read(analyticsConsentProvider));
      final display = View.of(context).display;
      analytics?.visit(
        widget.router.routerDelegate.currentConfiguration.uri,
        tablet: display.size.shortestSide / display.devicePixelRatio >= 600,
      );
    });
    WidgetsBinding.instance.ensureVisualUpdate();
  }

  @override
  Widget build(BuildContext context) {
    _analytics = ref.watch(visitorAnalyticsProvider);
    ref.listen(analyticsConsentProvider, (_, enabled) {
      ref.read(visitorAnalyticsProvider)?.setEnabled(enabled);
      if (enabled) _schedule();
    });
    _schedule();
    return widget.child;
  }

  @override
  void dispose() {
    _analytics?.setActive(false);
    widget.router.routerDelegate.removeListener(_schedule);
    WidgetsBinding.instance.removeObserver(this);
    super.dispose();
  }
}

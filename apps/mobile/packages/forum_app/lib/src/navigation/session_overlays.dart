import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../providers.dart';

/// Root and nested navigators can register observers with the same registry.
/// Removing the exact old route completes its pending dialog result with null;
/// a newly opened route in the next session is never popped accidentally.
class SessionOverlayRegistry {
  int _epoch = 0;
  final Map<PopupRoute<dynamic>, int> _routes = {};

  NavigatorObserver observer() => _SessionOverlayObserver(this);

  void changeSession(int epoch) {
    _epoch = epoch;
    final expired = _routes.entries
        .where((entry) => entry.value != epoch)
        .toList();
    if (expired.isEmpty) return;
    scheduleMicrotask(() {
      for (final entry in expired.reversed) {
        final route = entry.key;
        if (_routes[route] == entry.value &&
            entry.value != _epoch &&
            route.isActive) {
          route.navigator?.removeRoute(route);
        }
      }
    });
  }
}

class _SessionOverlayObserver extends NavigatorObserver {
  _SessionOverlayObserver(this.registry);
  final SessionOverlayRegistry registry;

  @override
  void didPush(Route<dynamic> route, Route<dynamic>? previousRoute) {
    if (route is PopupRoute) registry._routes[route] = registry._epoch;
  }

  @override
  void didPop(Route<dynamic> route, Route<dynamic>? previousRoute) {
    registry._routes.remove(route);
  }

  @override
  void didRemove(Route<dynamic> route, Route<dynamic>? previousRoute) {
    registry._routes.remove(route);
  }

  @override
  void didReplace({Route<dynamic>? newRoute, Route<dynamic>? oldRoute}) {
    registry._routes.remove(oldRoute);
    if (newRoute is PopupRoute) registry._routes[newRoute] = registry._epoch;
  }
}

class SessionOverlayHost extends ConsumerWidget {
  const SessionOverlayHost({
    super.key,
    required this.registry,
    required this.child,
  });
  final SessionOverlayRegistry registry;
  final Widget child;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    registry.changeSession(ref.read(offlineCacheEpochProvider));
    ref.listen(
      offlineCacheEpochProvider,
      (_, epoch) => registry.changeSession(epoch),
    );
    return child;
  }
}

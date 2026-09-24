import 'package:flutter/widgets.dart';

/// One observer per navigator shares a notification source. A root popup also
/// changes the effective visibility of a current route in a nested navigator.
final routeVisibilityChanges = _RouteVisibilityChanges();

class _RouteVisibilityChanges extends ChangeNotifier {
  void changed() => notifyListeners();
}

class VisibilityRouteObserver extends NavigatorObserver {
  void _changed() => routeVisibilityChanges.changed();
  @override
  void didPush(Route<dynamic> route, Route<dynamic>? previousRoute) =>
      _changed();
  @override
  void didPop(Route<dynamic> route, Route<dynamic>? previousRoute) =>
      _changed();
  @override
  void didRemove(Route<dynamic> route, Route<dynamic>? previousRoute) =>
      _changed();
  @override
  void didReplace({Route<dynamic>? newRoute, Route<dynamic>? oldRoute}) =>
      _changed();
}

bool routeIsUncovered(BuildContext context) {
  BuildContext? current = context;
  while (current != null) {
    final route = ModalRoute.of(current);
    if (route != null && !route.isCurrent) return false;
    current = current.findAncestorStateOfType<NavigatorState>()?.context;
  }
  return true;
}

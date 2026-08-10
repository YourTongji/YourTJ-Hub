import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:ui_kit/ui_kit.dart';

/// Stable destinations owned by the persistent mobile shell.
enum GfShellDestination { home, search, messages, profile }

/// Connects persistent shell destinations with their page-owned scroll views.
///
/// Pages keep ownership of their [GfScrollToTopController]. The shell only
/// requests a user-initiated return to top when the active destination is
/// tapped again.
class GfTabScrollRegistry {
  final Map<GfShellDestination, GfScrollToTopController> _controllers =
      <GfShellDestination, GfScrollToTopController>{};

  void register(
    GfShellDestination destination,
    GfScrollToTopController controller,
  ) {
    _controllers[destination] = controller;
  }

  void unregister(
    GfShellDestination destination,
    GfScrollToTopController controller,
  ) {
    if (identical(_controllers[destination], controller)) {
      _controllers.remove(destination);
    }
  }

  Future<void> scrollToTop(GfShellDestination destination) async {
    await _controllers[destination]?.scrollToTop();
  }
}

final tabScrollRegistryProvider = Provider<GfTabScrollRegistry>(
  (Ref ref) => GfTabScrollRegistry(),
);

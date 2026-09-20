import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../providers.dart';

/// Drop private child state on backgrounding, tab exit and session changes.
/// A separate foreground cache may retain selected datasets briefly across tabs;
/// no academic data enters the app's offline cache or restoration system.
class CampusPrivateSurface extends ConsumerStatefulWidget {
  const CampusPrivateSurface({super.key, required this.builder});
  final WidgetBuilder builder;
  @override
  ConsumerState<CampusPrivateSurface> createState() =>
      _CampusPrivateSurfaceState();
}

class _CampusPrivateSurfaceState extends ConsumerState<CampusPrivateSurface>
    with WidgetsBindingObserver {
  bool _foreground = true;
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    setState(() => _foreground = state == AppLifecycleState.resumed);
  }

  @override
  void dispose() {
    WidgetsBinding.instance.removeObserver(this);
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final epoch = ref.watch(offlineCacheEpochProvider);
    if (!_foreground || !TickerMode.valuesOf(context).enabled) {
      return const SizedBox.expand();
    }
    return KeyedSubtree(key: ValueKey(epoch), child: widget.builder(context));
  }
}

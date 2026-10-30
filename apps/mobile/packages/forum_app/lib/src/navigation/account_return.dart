import 'package:flutter/services.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

final accountReturnLinksProvider = StreamProvider<Uri>(
  (ref) => const EventChannel('yourtj/account_return')
      .receiveBroadcastStream()
      .where((event) => event == 'yourtj://account/email-verified')
      .map((_) => Uri.parse('yourtj://account/email-verified')),
);

// A hint invalidates the owner view; it never carries verification authority.
final accountReturnRefreshProvider = StateProvider<int>((ref) => 0);

bool isAccountReturnLink(Uri uri) =>
    uri.toString() == 'yourtj://account/email-verified';

/// An external return is navigation only, never proof of verification or identity.
class AccountReturnHost extends ConsumerStatefulWidget {
  const AccountReturnHost({
    required this.router,
    required this.child,
    super.key,
  });
  final GoRouter router;
  final Widget child;
  @override
  ConsumerState<AccountReturnHost> createState() => _AccountReturnHostState();
}

class _AccountReturnHostState extends ConsumerState<AccountReturnHost> {
  bool _opening = false;
  @override
  void initState() {
    super.initState();
    ref.listenManual(accountReturnLinksProvider, (_, next) {
      final uri = next.valueOrNull;
      if (uri == null || !isAccountReturnLink(uri)) return;
      WidgetsBinding.instance.addPostFrameCallback((_) async {
        if (!mounted) return;
        ref.read(accountReturnRefreshProvider.notifier).state++;
        // Deduplicate navigation only. A new hint still refreshes an open page.
        if (_opening) return;
        _opening = true;
        try {
          await widget.router.push('/settings/account');
        } finally {
          _opening = false;
        }
      });
      WidgetsBinding.instance.ensureVisualUpdate();
    }, fireImmediately: true);
  }

  @override
  Widget build(BuildContext context) => widget.child;
}

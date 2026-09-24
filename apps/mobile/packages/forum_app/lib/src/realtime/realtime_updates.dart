import 'package:flutter_riverpod/flutter_riverpod.dart';

class RealtimeInvalidations {
  const RealtimeInvalidations({
    required this.chatRevision,
    required this.chatConvId,
    required this.notificationsRevision,
  });

  final int chatRevision;

  /// Zero means reconcile every open conversation (for example after hello).
  final int chatConvId;
  final int notificationsRevision;
}

class RealtimeInvalidationNotifier extends Notifier<RealtimeInvalidations> {
  @override
  RealtimeInvalidations build() => const RealtimeInvalidations(
    chatRevision: 0,
    chatConvId: 0,
    notificationsRevision: 0,
  );

  void chat(int convId) => state = RealtimeInvalidations(
    chatRevision: state.chatRevision + 1,
    chatConvId: convId,
    notificationsRevision: state.notificationsRevision,
  );

  void notifications() => state = RealtimeInvalidations(
    chatRevision: state.chatRevision,
    chatConvId: state.chatConvId,
    notificationsRevision: state.notificationsRevision + 1,
  );

  void resync() => state = RealtimeInvalidations(
    chatRevision: state.chatRevision + 1,
    chatConvId: 0,
    notificationsRevision: state.notificationsRevision + 1,
  );
}

final realtimeInvalidationsProvider =
    NotifierProvider<RealtimeInvalidationNotifier, RealtimeInvalidations>(
      RealtimeInvalidationNotifier.new,
    );

class RealtimeHealthNotifier extends Notifier<bool> {
  @override
  bool build() => false;

  void setHealthy(bool value) => state = value;
}

final realtimeHealthyProvider = NotifierProvider<RealtimeHealthNotifier, bool>(
  RealtimeHealthNotifier.new,
);

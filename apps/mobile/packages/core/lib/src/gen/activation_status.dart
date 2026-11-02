/// Owner-only GET /api/resend-activation-email mirror. No public identity data.
class ActivationStatus {
  const ActivationStatus({
    required this.currentEmail,
    required this.pendingEmail,
    required this.targetEmail,
    required this.targetKind,
    required this.state,
    required this.remainingToday,
    required this.dailyLimit,
    required this.retryAfterSeconds,
  });
  final String currentEmail, pendingEmail, targetEmail, targetKind, state;
  final int remainingToday, dailyLimit, retryAfterSeconds;
  factory ActivationStatus.fromJson(Map<String, dynamic> json) =>
      ActivationStatus(
        currentEmail: json['currentEmail'] as String,
        pendingEmail: json['pendingEmail'] as String,
        targetEmail: json['targetEmail'] as String,
        targetKind: json['targetKind'] as String,
        state: json['state'] as String,
        remainingToday: json['remainingToday'] as int,
        dailyLimit: json['dailyLimit'] as int,
        retryAfterSeconds: json['retryAfterSeconds'] as int,
      );
}

// GENERATED CODE - DO NOT MODIFY BY HAND.
// Source: packages/api-contract/components/schemas.yaml + dart-models.json.
// Regenerate: cd packages/api-contract && pnpm generate:dart
// Wire models only; enum/range validation remains on the server.

/// OpenAPI schema: ActivationStatus.
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

  final String currentEmail;
  final String pendingEmail;
  final String targetEmail;
  final String targetKind;
  final String state;
  final int remainingToday;
  final int dailyLimit;
  final int retryAfterSeconds;

  factory ActivationStatus.fromJson(Map<String, dynamic> json) {
    return ActivationStatus(
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
}

/// OpenAPI schema: PushDeviceRegisterRequest.
class RegisterPushDeviceInput {
  const RegisterPushDeviceInput({
    required this.platform,
    this.provider,
    required this.token,
  });

  final String platform;
  final String? provider;
  final String token;

  Map<String, dynamic> toJson() {
    return {
      'platform': platform,
      if (provider != null) 'provider': provider,
      'token': token,
    };
  }
}

/// OpenAPI schema: PushDeviceUnregisterRequest.
class UnregisterPushDeviceInput {
  const UnregisterPushDeviceInput({required this.token});

  final String token;

  Map<String, dynamic> toJson() {
    return {'token': token};
  }
}

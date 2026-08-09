/// 后端统一响应包装 component.ResultStruct:
/// `{code, messageCode, params, result}`,code == 0 表示成功。
class GfResponse<T> {
  const GfResponse({
    required this.code,
    this.messageCode,
    this.params,
    this.result,
  });

  final int code;
  final String? messageCode;
  final Map<String, dynamic>? params;
  final T? result;

  bool get isSuccess => code == 0;

  factory GfResponse.fromJson(
    Map<String, dynamic> json,
    T Function(Object? json) fromJsonT,
  ) {
    return GfResponse<T>(
      code: (json['code'] as num?)?.toInt() ?? -1,
      messageCode: json['messageCode'] as String?,
      params: (json['params'] as Map<String, dynamic>?)?.cast<String, dynamic>(),
      result: json.containsKey('result') ? fromJsonT(json['result']) : null,
    );
  }
}

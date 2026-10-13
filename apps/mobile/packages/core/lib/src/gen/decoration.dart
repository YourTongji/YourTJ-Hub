/// Server-owned decoration styles; clients accept only safe local frame paths.
class DecorationStyle {
  static final _assetPattern = RegExp(
    r'^/static/decorations/[A-Za-z0-9][A-Za-z0-9._-]{0,120}\.(png|webp)$',
  );
  const DecorationStyle({
    required this.code,
    required this.name,
    required this.kind,
    required this.lightColors,
    required this.darkColors,
    this.isEnabled = true,
    this.sortOrder = 0,
    this.assetUrl,
    this.assetScale = 1.25,
  });
  final String code, name, kind;
  final String? assetUrl;
  final double assetScale;
  final List<String> lightColors, darkColors;
  final bool isEnabled;
  final int sortOrder;
  static List<String> _colors(dynamic value) {
    if (value is! List ||
        value.isEmpty ||
        value.length > 2 ||
        value.any(
          (c) => c is! String || !RegExp(r'^#[0-9a-fA-F]{6}$').hasMatch(c),
        )) {
      return const [];
    }
    return value.cast<String>();
  }

  factory DecorationStyle.fromJson(Map<String, dynamic> json) =>
      DecorationStyle(
        code: json['code'] is String ? json['code'] : '',
        name: json['name'] is String ? json['name'] : '',
        kind: json['kind'] is String ? json['kind'] : '',
        lightColors: _colors(json['lightColors']),
        darkColors: _colors(json['darkColors']),
        isEnabled: json['isEnabled'] != false,
        assetUrl:
            json['kind'] == 'avatarFrame' &&
                json['assetUrl'] is String &&
                _assetPattern.hasMatch(json['assetUrl'] as String)
            ? json['assetUrl'] as String
            : null,
        assetScale:
            json['assetScale'] is num &&
                (json['assetScale'] as num).isFinite &&
                (json['assetScale'] as num) >= 0.5 &&
                (json['assetScale'] as num) <= 2
            ? (json['assetScale'] as num).toDouble()
            : 1.25,
        sortOrder: json['sortOrder'] is num
            ? (json['sortOrder'] as num).toInt()
            : 0,
      );
  static DecorationStyle? parse(dynamic json, String kind) {
    if (json is! Map<String, dynamic>) return null;
    final style = DecorationStyle.fromJson(json);
    return style.kind == kind &&
            style.code.isNotEmpty &&
            style.isEnabled &&
            style.lightColors.isNotEmpty &&
            style.darkColors.isNotEmpty
        ? style
        : null;
  }

  Map<String, dynamic> toJson() => {
    'code': code,
    'name': name,
    'kind': kind,
    'lightColors': lightColors,
    'darkColors': darkColors,
    'isEnabled': isEnabled,
    if (assetUrl != null) 'assetUrl': assetUrl,
    'assetScale': assetScale,
    'sortOrder': sortOrder,
  };
}

class UserDecoration {
  const UserDecoration({this.avatarFrame, this.nicknameStyle});
  final DecorationStyle? avatarFrame, nicknameStyle;
  static UserDecoration? parse(dynamic json) =>
      json is Map<String, dynamic> ? UserDecoration.fromJson(json) : null;
  factory UserDecoration.fromJson(Map<String, dynamic> json) => UserDecoration(
    avatarFrame: DecorationStyle.parse(json['avatarFrame'], 'avatarFrame'),
    nicknameStyle: DecorationStyle.parse(
      json['nicknameStyle'],
      'nicknameStyle',
    ),
  );
  Map<String, dynamic> toJson() => {
    'avatarFrame': avatarFrame?.toJson(),
    'nicknameStyle': nicknameStyle?.toJson(),
  };
}

class DecorationState {
  const DecorationState({
    required this.catalog,
    required this.ownedCodes,
    required this.avatarFrame,
    required this.nicknameStyle,
  });
  final List<DecorationStyle> catalog;
  final List<String> ownedCodes;
  final String avatarFrame, nicknameStyle;
  factory DecorationState.fromJson(Map<String, dynamic> json) {
    final equipped = json['equipped'] as Map<String, dynamic>? ?? {};
    return DecorationState(
      catalog: (json['catalog'] as List? ?? [])
          .whereType<Map<String, dynamic>>()
          .map(DecorationStyle.fromJson)
          .toList(),
      ownedCodes: (json['ownedCodes'] as List? ?? [])
          .whereType<String>()
          .toList(),
      avatarFrame: equipped['avatarFrame'] as String? ?? '',
      nicknameStyle: equipped['nicknameStyle'] as String? ?? '',
    );
  }
}

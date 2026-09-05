// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'layout.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

T _$identity<T>(T value) => value;

final _privateConstructorUsedError = UnsupportedError(
  'It seems like you constructed your class using `MyClass._()`. This constructor is only meant to be used by freezed and you are not supposed to need it nor use it.\nPlease check the documentation here for more information: https://github.com/rrousselGit/freezed#adding-getters-and-methods-to-our-models',
);

LayoutPayload _$LayoutPayloadFromJson(Map<String, dynamic> json) {
  return _LayoutPayload.fromJson(json);
}

/// @nodoc
mixin _$LayoutPayload {
  SitePayload get site => throw _privateConstructorUsedError;
  ViewerPayload get viewer => throw _privateConstructorUsedError;
  List<NavItemPayload>? get header => throw _privateConstructorUsedError;
  SidebarPayload get sidebar => throw _privateConstructorUsedError;
  FooterPayload get footer => throw _privateConstructorUsedError;
  UnreadStatusPayload get unread => throw _privateConstructorUsedError;
  ThemePayload get theme => throw _privateConstructorUsedError;

  /// Serializes this LayoutPayload to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of LayoutPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $LayoutPayloadCopyWith<LayoutPayload> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $LayoutPayloadCopyWith<$Res> {
  factory $LayoutPayloadCopyWith(
    LayoutPayload value,
    $Res Function(LayoutPayload) then,
  ) = _$LayoutPayloadCopyWithImpl<$Res, LayoutPayload>;
  @useResult
  $Res call({
    SitePayload site,
    ViewerPayload viewer,
    List<NavItemPayload>? header,
    SidebarPayload sidebar,
    FooterPayload footer,
    UnreadStatusPayload unread,
    ThemePayload theme,
  });

  $SitePayloadCopyWith<$Res> get site;
  $ViewerPayloadCopyWith<$Res> get viewer;
  $SidebarPayloadCopyWith<$Res> get sidebar;
  $FooterPayloadCopyWith<$Res> get footer;
  $UnreadStatusPayloadCopyWith<$Res> get unread;
  $ThemePayloadCopyWith<$Res> get theme;
}

/// @nodoc
class _$LayoutPayloadCopyWithImpl<$Res, $Val extends LayoutPayload>
    implements $LayoutPayloadCopyWith<$Res> {
  _$LayoutPayloadCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of LayoutPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? site = null,
    Object? viewer = null,
    Object? header = freezed,
    Object? sidebar = null,
    Object? footer = null,
    Object? unread = null,
    Object? theme = null,
  }) {
    return _then(
      _value.copyWith(
            site: null == site
                ? _value.site
                : site // ignore: cast_nullable_to_non_nullable
                      as SitePayload,
            viewer: null == viewer
                ? _value.viewer
                : viewer // ignore: cast_nullable_to_non_nullable
                      as ViewerPayload,
            header: freezed == header
                ? _value.header
                : header // ignore: cast_nullable_to_non_nullable
                      as List<NavItemPayload>?,
            sidebar: null == sidebar
                ? _value.sidebar
                : sidebar // ignore: cast_nullable_to_non_nullable
                      as SidebarPayload,
            footer: null == footer
                ? _value.footer
                : footer // ignore: cast_nullable_to_non_nullable
                      as FooterPayload,
            unread: null == unread
                ? _value.unread
                : unread // ignore: cast_nullable_to_non_nullable
                      as UnreadStatusPayload,
            theme: null == theme
                ? _value.theme
                : theme // ignore: cast_nullable_to_non_nullable
                      as ThemePayload,
          )
          as $Val,
    );
  }

  /// Create a copy of LayoutPayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @pragma('vm:prefer-inline')
  $SitePayloadCopyWith<$Res> get site {
    return $SitePayloadCopyWith<$Res>(_value.site, (value) {
      return _then(_value.copyWith(site: value) as $Val);
    });
  }

  /// Create a copy of LayoutPayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @pragma('vm:prefer-inline')
  $ViewerPayloadCopyWith<$Res> get viewer {
    return $ViewerPayloadCopyWith<$Res>(_value.viewer, (value) {
      return _then(_value.copyWith(viewer: value) as $Val);
    });
  }

  /// Create a copy of LayoutPayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @pragma('vm:prefer-inline')
  $SidebarPayloadCopyWith<$Res> get sidebar {
    return $SidebarPayloadCopyWith<$Res>(_value.sidebar, (value) {
      return _then(_value.copyWith(sidebar: value) as $Val);
    });
  }

  /// Create a copy of LayoutPayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @pragma('vm:prefer-inline')
  $FooterPayloadCopyWith<$Res> get footer {
    return $FooterPayloadCopyWith<$Res>(_value.footer, (value) {
      return _then(_value.copyWith(footer: value) as $Val);
    });
  }

  /// Create a copy of LayoutPayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @pragma('vm:prefer-inline')
  $UnreadStatusPayloadCopyWith<$Res> get unread {
    return $UnreadStatusPayloadCopyWith<$Res>(_value.unread, (value) {
      return _then(_value.copyWith(unread: value) as $Val);
    });
  }

  /// Create a copy of LayoutPayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @pragma('vm:prefer-inline')
  $ThemePayloadCopyWith<$Res> get theme {
    return $ThemePayloadCopyWith<$Res>(_value.theme, (value) {
      return _then(_value.copyWith(theme: value) as $Val);
    });
  }
}

/// @nodoc
abstract class _$$LayoutPayloadImplCopyWith<$Res>
    implements $LayoutPayloadCopyWith<$Res> {
  factory _$$LayoutPayloadImplCopyWith(
    _$LayoutPayloadImpl value,
    $Res Function(_$LayoutPayloadImpl) then,
  ) = __$$LayoutPayloadImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    SitePayload site,
    ViewerPayload viewer,
    List<NavItemPayload>? header,
    SidebarPayload sidebar,
    FooterPayload footer,
    UnreadStatusPayload unread,
    ThemePayload theme,
  });

  @override
  $SitePayloadCopyWith<$Res> get site;
  @override
  $ViewerPayloadCopyWith<$Res> get viewer;
  @override
  $SidebarPayloadCopyWith<$Res> get sidebar;
  @override
  $FooterPayloadCopyWith<$Res> get footer;
  @override
  $UnreadStatusPayloadCopyWith<$Res> get unread;
  @override
  $ThemePayloadCopyWith<$Res> get theme;
}

/// @nodoc
class __$$LayoutPayloadImplCopyWithImpl<$Res>
    extends _$LayoutPayloadCopyWithImpl<$Res, _$LayoutPayloadImpl>
    implements _$$LayoutPayloadImplCopyWith<$Res> {
  __$$LayoutPayloadImplCopyWithImpl(
    _$LayoutPayloadImpl _value,
    $Res Function(_$LayoutPayloadImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of LayoutPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? site = null,
    Object? viewer = null,
    Object? header = freezed,
    Object? sidebar = null,
    Object? footer = null,
    Object? unread = null,
    Object? theme = null,
  }) {
    return _then(
      _$LayoutPayloadImpl(
        site: null == site
            ? _value.site
            : site // ignore: cast_nullable_to_non_nullable
                  as SitePayload,
        viewer: null == viewer
            ? _value.viewer
            : viewer // ignore: cast_nullable_to_non_nullable
                  as ViewerPayload,
        header: freezed == header
            ? _value._header
            : header // ignore: cast_nullable_to_non_nullable
                  as List<NavItemPayload>?,
        sidebar: null == sidebar
            ? _value.sidebar
            : sidebar // ignore: cast_nullable_to_non_nullable
                  as SidebarPayload,
        footer: null == footer
            ? _value.footer
            : footer // ignore: cast_nullable_to_non_nullable
                  as FooterPayload,
        unread: null == unread
            ? _value.unread
            : unread // ignore: cast_nullable_to_non_nullable
                  as UnreadStatusPayload,
        theme: null == theme
            ? _value.theme
            : theme // ignore: cast_nullable_to_non_nullable
                  as ThemePayload,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$LayoutPayloadImpl implements _LayoutPayload {
  const _$LayoutPayloadImpl({
    required this.site,
    required this.viewer,
    final List<NavItemPayload>? header,
    required this.sidebar,
    required this.footer,
    required this.unread,
    required this.theme,
  }) : _header = header;

  factory _$LayoutPayloadImpl.fromJson(Map<String, dynamic> json) =>
      _$$LayoutPayloadImplFromJson(json);

  @override
  final SitePayload site;
  @override
  final ViewerPayload viewer;
  final List<NavItemPayload>? _header;
  @override
  List<NavItemPayload>? get header {
    final value = _header;
    if (value == null) return null;
    if (_header is EqualUnmodifiableListView) return _header;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(value);
  }

  @override
  final SidebarPayload sidebar;
  @override
  final FooterPayload footer;
  @override
  final UnreadStatusPayload unread;
  @override
  final ThemePayload theme;

  @override
  String toString() {
    return 'LayoutPayload(site: $site, viewer: $viewer, header: $header, sidebar: $sidebar, footer: $footer, unread: $unread, theme: $theme)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$LayoutPayloadImpl &&
            (identical(other.site, site) || other.site == site) &&
            (identical(other.viewer, viewer) || other.viewer == viewer) &&
            const DeepCollectionEquality().equals(other._header, _header) &&
            (identical(other.sidebar, sidebar) || other.sidebar == sidebar) &&
            (identical(other.footer, footer) || other.footer == footer) &&
            (identical(other.unread, unread) || other.unread == unread) &&
            (identical(other.theme, theme) || other.theme == theme));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
    runtimeType,
    site,
    viewer,
    const DeepCollectionEquality().hash(_header),
    sidebar,
    footer,
    unread,
    theme,
  );

  /// Create a copy of LayoutPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$LayoutPayloadImplCopyWith<_$LayoutPayloadImpl> get copyWith =>
      __$$LayoutPayloadImplCopyWithImpl<_$LayoutPayloadImpl>(this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$$LayoutPayloadImplToJson(this);
  }
}

abstract class _LayoutPayload implements LayoutPayload {
  const factory _LayoutPayload({
    required final SitePayload site,
    required final ViewerPayload viewer,
    final List<NavItemPayload>? header,
    required final SidebarPayload sidebar,
    required final FooterPayload footer,
    required final UnreadStatusPayload unread,
    required final ThemePayload theme,
  }) = _$LayoutPayloadImpl;

  factory _LayoutPayload.fromJson(Map<String, dynamic> json) =
      _$LayoutPayloadImpl.fromJson;

  @override
  SitePayload get site;
  @override
  ViewerPayload get viewer;
  @override
  List<NavItemPayload>? get header;
  @override
  SidebarPayload get sidebar;
  @override
  FooterPayload get footer;
  @override
  UnreadStatusPayload get unread;
  @override
  ThemePayload get theme;

  /// Create a copy of LayoutPayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$LayoutPayloadImplCopyWith<_$LayoutPayloadImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

SitePayload _$SitePayloadFromJson(Map<String, dynamic> json) {
  return _SitePayload.fromJson(json);
}

/// @nodoc
mixin _$SitePayload {
  String get name => throw _privateConstructorUsedError;
  String get description => throw _privateConstructorUsedError;
  String get logo => throw _privateConstructorUsedError;
  String get favicon => throw _privateConstructorUsedError;
  String? get externalLinks => throw _privateConstructorUsedError;
  String get brandType => throw _privateConstructorUsedError;
  String get brandText => throw _privateConstructorUsedError;
  String get brandImage => throw _privateConstructorUsedError;

  /// Serializes this SitePayload to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of SitePayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $SitePayloadCopyWith<SitePayload> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $SitePayloadCopyWith<$Res> {
  factory $SitePayloadCopyWith(
    SitePayload value,
    $Res Function(SitePayload) then,
  ) = _$SitePayloadCopyWithImpl<$Res, SitePayload>;
  @useResult
  $Res call({
    String name,
    String description,
    String logo,
    String favicon,
    String? externalLinks,
    String brandType,
    String brandText,
    String brandImage,
  });
}

/// @nodoc
class _$SitePayloadCopyWithImpl<$Res, $Val extends SitePayload>
    implements $SitePayloadCopyWith<$Res> {
  _$SitePayloadCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of SitePayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? name = null,
    Object? description = null,
    Object? logo = null,
    Object? favicon = null,
    Object? externalLinks = freezed,
    Object? brandType = null,
    Object? brandText = null,
    Object? brandImage = null,
  }) {
    return _then(
      _value.copyWith(
            name: null == name
                ? _value.name
                : name // ignore: cast_nullable_to_non_nullable
                      as String,
            description: null == description
                ? _value.description
                : description // ignore: cast_nullable_to_non_nullable
                      as String,
            logo: null == logo
                ? _value.logo
                : logo // ignore: cast_nullable_to_non_nullable
                      as String,
            favicon: null == favicon
                ? _value.favicon
                : favicon // ignore: cast_nullable_to_non_nullable
                      as String,
            externalLinks: freezed == externalLinks
                ? _value.externalLinks
                : externalLinks // ignore: cast_nullable_to_non_nullable
                      as String?,
            brandType: null == brandType
                ? _value.brandType
                : brandType // ignore: cast_nullable_to_non_nullable
                      as String,
            brandText: null == brandText
                ? _value.brandText
                : brandText // ignore: cast_nullable_to_non_nullable
                      as String,
            brandImage: null == brandImage
                ? _value.brandImage
                : brandImage // ignore: cast_nullable_to_non_nullable
                      as String,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$SitePayloadImplCopyWith<$Res>
    implements $SitePayloadCopyWith<$Res> {
  factory _$$SitePayloadImplCopyWith(
    _$SitePayloadImpl value,
    $Res Function(_$SitePayloadImpl) then,
  ) = __$$SitePayloadImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    String name,
    String description,
    String logo,
    String favicon,
    String? externalLinks,
    String brandType,
    String brandText,
    String brandImage,
  });
}

/// @nodoc
class __$$SitePayloadImplCopyWithImpl<$Res>
    extends _$SitePayloadCopyWithImpl<$Res, _$SitePayloadImpl>
    implements _$$SitePayloadImplCopyWith<$Res> {
  __$$SitePayloadImplCopyWithImpl(
    _$SitePayloadImpl _value,
    $Res Function(_$SitePayloadImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of SitePayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? name = null,
    Object? description = null,
    Object? logo = null,
    Object? favicon = null,
    Object? externalLinks = freezed,
    Object? brandType = null,
    Object? brandText = null,
    Object? brandImage = null,
  }) {
    return _then(
      _$SitePayloadImpl(
        name: null == name
            ? _value.name
            : name // ignore: cast_nullable_to_non_nullable
                  as String,
        description: null == description
            ? _value.description
            : description // ignore: cast_nullable_to_non_nullable
                  as String,
        logo: null == logo
            ? _value.logo
            : logo // ignore: cast_nullable_to_non_nullable
                  as String,
        favicon: null == favicon
            ? _value.favicon
            : favicon // ignore: cast_nullable_to_non_nullable
                  as String,
        externalLinks: freezed == externalLinks
            ? _value.externalLinks
            : externalLinks // ignore: cast_nullable_to_non_nullable
                  as String?,
        brandType: null == brandType
            ? _value.brandType
            : brandType // ignore: cast_nullable_to_non_nullable
                  as String,
        brandText: null == brandText
            ? _value.brandText
            : brandText // ignore: cast_nullable_to_non_nullable
                  as String,
        brandImage: null == brandImage
            ? _value.brandImage
            : brandImage // ignore: cast_nullable_to_non_nullable
                  as String,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$SitePayloadImpl implements _SitePayload {
  const _$SitePayloadImpl({
    required this.name,
    required this.description,
    required this.logo,
    required this.favicon,
    this.externalLinks,
    required this.brandType,
    required this.brandText,
    required this.brandImage,
  });

  factory _$SitePayloadImpl.fromJson(Map<String, dynamic> json) =>
      _$$SitePayloadImplFromJson(json);

  @override
  final String name;
  @override
  final String description;
  @override
  final String logo;
  @override
  final String favicon;
  @override
  final String? externalLinks;
  @override
  final String brandType;
  @override
  final String brandText;
  @override
  final String brandImage;

  @override
  String toString() {
    return 'SitePayload(name: $name, description: $description, logo: $logo, favicon: $favicon, externalLinks: $externalLinks, brandType: $brandType, brandText: $brandText, brandImage: $brandImage)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$SitePayloadImpl &&
            (identical(other.name, name) || other.name == name) &&
            (identical(other.description, description) ||
                other.description == description) &&
            (identical(other.logo, logo) || other.logo == logo) &&
            (identical(other.favicon, favicon) || other.favicon == favicon) &&
            (identical(other.externalLinks, externalLinks) ||
                other.externalLinks == externalLinks) &&
            (identical(other.brandType, brandType) ||
                other.brandType == brandType) &&
            (identical(other.brandText, brandText) ||
                other.brandText == brandText) &&
            (identical(other.brandImage, brandImage) ||
                other.brandImage == brandImage));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
    runtimeType,
    name,
    description,
    logo,
    favicon,
    externalLinks,
    brandType,
    brandText,
    brandImage,
  );

  /// Create a copy of SitePayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$SitePayloadImplCopyWith<_$SitePayloadImpl> get copyWith =>
      __$$SitePayloadImplCopyWithImpl<_$SitePayloadImpl>(this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$$SitePayloadImplToJson(this);
  }
}

abstract class _SitePayload implements SitePayload {
  const factory _SitePayload({
    required final String name,
    required final String description,
    required final String logo,
    required final String favicon,
    final String? externalLinks,
    required final String brandType,
    required final String brandText,
    required final String brandImage,
  }) = _$SitePayloadImpl;

  factory _SitePayload.fromJson(Map<String, dynamic> json) =
      _$SitePayloadImpl.fromJson;

  @override
  String get name;
  @override
  String get description;
  @override
  String get logo;
  @override
  String get favicon;
  @override
  String? get externalLinks;
  @override
  String get brandType;
  @override
  String get brandText;
  @override
  String get brandImage;

  /// Create a copy of SitePayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$SitePayloadImplCopyWith<_$SitePayloadImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

ViewerPayload _$ViewerPayloadFromJson(Map<String, dynamic> json) {
  return _ViewerPayload.fromJson(json);
}

/// @nodoc
mixin _$ViewerPayload {
  int get id => throw _privateConstructorUsedError;
  String get username => throw _privateConstructorUsedError;
  String get email => throw _privateConstructorUsedError;
  String get avatarUrl => throw _privateConstructorUsedError;
  bool get isAuthenticated => throw _privateConstructorUsedError;
  bool get canAccessAdmin => throw _privateConstructorUsedError;
  bool get isModerator => throw _privateConstructorUsedError;
  bool get requiresEmailVerification => throw _privateConstructorUsedError;
  List<int>? get adminPermissions => throw _privateConstructorUsedError;

  /// Serializes this ViewerPayload to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of ViewerPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $ViewerPayloadCopyWith<ViewerPayload> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $ViewerPayloadCopyWith<$Res> {
  factory $ViewerPayloadCopyWith(
    ViewerPayload value,
    $Res Function(ViewerPayload) then,
  ) = _$ViewerPayloadCopyWithImpl<$Res, ViewerPayload>;
  @useResult
  $Res call({
    int id,
    String username,
    String email,
    String avatarUrl,
    bool isAuthenticated,
    bool canAccessAdmin,
    bool isModerator,
    bool requiresEmailVerification,
    List<int>? adminPermissions,
  });
}

/// @nodoc
class _$ViewerPayloadCopyWithImpl<$Res, $Val extends ViewerPayload>
    implements $ViewerPayloadCopyWith<$Res> {
  _$ViewerPayloadCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of ViewerPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = null,
    Object? username = null,
    Object? email = null,
    Object? avatarUrl = null,
    Object? isAuthenticated = null,
    Object? canAccessAdmin = null,
    Object? isModerator = null,
    Object? requiresEmailVerification = null,
    Object? adminPermissions = freezed,
  }) {
    return _then(
      _value.copyWith(
            id: null == id
                ? _value.id
                : id // ignore: cast_nullable_to_non_nullable
                      as int,
            username: null == username
                ? _value.username
                : username // ignore: cast_nullable_to_non_nullable
                      as String,
            email: null == email
                ? _value.email
                : email // ignore: cast_nullable_to_non_nullable
                      as String,
            avatarUrl: null == avatarUrl
                ? _value.avatarUrl
                : avatarUrl // ignore: cast_nullable_to_non_nullable
                      as String,
            isAuthenticated: null == isAuthenticated
                ? _value.isAuthenticated
                : isAuthenticated // ignore: cast_nullable_to_non_nullable
                      as bool,
            canAccessAdmin: null == canAccessAdmin
                ? _value.canAccessAdmin
                : canAccessAdmin // ignore: cast_nullable_to_non_nullable
                      as bool,
            isModerator: null == isModerator
                ? _value.isModerator
                : isModerator // ignore: cast_nullable_to_non_nullable
                      as bool,
            requiresEmailVerification: null == requiresEmailVerification
                ? _value.requiresEmailVerification
                : requiresEmailVerification // ignore: cast_nullable_to_non_nullable
                      as bool,
            adminPermissions: freezed == adminPermissions
                ? _value.adminPermissions
                : adminPermissions // ignore: cast_nullable_to_non_nullable
                      as List<int>?,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$ViewerPayloadImplCopyWith<$Res>
    implements $ViewerPayloadCopyWith<$Res> {
  factory _$$ViewerPayloadImplCopyWith(
    _$ViewerPayloadImpl value,
    $Res Function(_$ViewerPayloadImpl) then,
  ) = __$$ViewerPayloadImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    int id,
    String username,
    String email,
    String avatarUrl,
    bool isAuthenticated,
    bool canAccessAdmin,
    bool isModerator,
    bool requiresEmailVerification,
    List<int>? adminPermissions,
  });
}

/// @nodoc
class __$$ViewerPayloadImplCopyWithImpl<$Res>
    extends _$ViewerPayloadCopyWithImpl<$Res, _$ViewerPayloadImpl>
    implements _$$ViewerPayloadImplCopyWith<$Res> {
  __$$ViewerPayloadImplCopyWithImpl(
    _$ViewerPayloadImpl _value,
    $Res Function(_$ViewerPayloadImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of ViewerPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = null,
    Object? username = null,
    Object? email = null,
    Object? avatarUrl = null,
    Object? isAuthenticated = null,
    Object? canAccessAdmin = null,
    Object? isModerator = null,
    Object? requiresEmailVerification = null,
    Object? adminPermissions = freezed,
  }) {
    return _then(
      _$ViewerPayloadImpl(
        id: null == id
            ? _value.id
            : id // ignore: cast_nullable_to_non_nullable
                  as int,
        username: null == username
            ? _value.username
            : username // ignore: cast_nullable_to_non_nullable
                  as String,
        email: null == email
            ? _value.email
            : email // ignore: cast_nullable_to_non_nullable
                  as String,
        avatarUrl: null == avatarUrl
            ? _value.avatarUrl
            : avatarUrl // ignore: cast_nullable_to_non_nullable
                  as String,
        isAuthenticated: null == isAuthenticated
            ? _value.isAuthenticated
            : isAuthenticated // ignore: cast_nullable_to_non_nullable
                  as bool,
        canAccessAdmin: null == canAccessAdmin
            ? _value.canAccessAdmin
            : canAccessAdmin // ignore: cast_nullable_to_non_nullable
                  as bool,
        isModerator: null == isModerator
            ? _value.isModerator
            : isModerator // ignore: cast_nullable_to_non_nullable
                  as bool,
        requiresEmailVerification: null == requiresEmailVerification
            ? _value.requiresEmailVerification
            : requiresEmailVerification // ignore: cast_nullable_to_non_nullable
                  as bool,
        adminPermissions: freezed == adminPermissions
            ? _value._adminPermissions
            : adminPermissions // ignore: cast_nullable_to_non_nullable
                  as List<int>?,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$ViewerPayloadImpl implements _ViewerPayload {
  const _$ViewerPayloadImpl({
    required this.id,
    required this.username,
    required this.email,
    required this.avatarUrl,
    required this.isAuthenticated,
    required this.canAccessAdmin,
    required this.isModerator,
    required this.requiresEmailVerification,
    final List<int>? adminPermissions,
  }) : _adminPermissions = adminPermissions;

  factory _$ViewerPayloadImpl.fromJson(Map<String, dynamic> json) =>
      _$$ViewerPayloadImplFromJson(json);

  @override
  final int id;
  @override
  final String username;
  @override
  final String email;
  @override
  final String avatarUrl;
  @override
  final bool isAuthenticated;
  @override
  final bool canAccessAdmin;
  @override
  final bool isModerator;
  @override
  final bool requiresEmailVerification;
  final List<int>? _adminPermissions;
  @override
  List<int>? get adminPermissions {
    final value = _adminPermissions;
    if (value == null) return null;
    if (_adminPermissions is EqualUnmodifiableListView)
      return _adminPermissions;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(value);
  }

  @override
  String toString() {
    return 'ViewerPayload(id: $id, username: $username, email: $email, avatarUrl: $avatarUrl, isAuthenticated: $isAuthenticated, canAccessAdmin: $canAccessAdmin, isModerator: $isModerator, requiresEmailVerification: $requiresEmailVerification, adminPermissions: $adminPermissions)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$ViewerPayloadImpl &&
            (identical(other.id, id) || other.id == id) &&
            (identical(other.username, username) ||
                other.username == username) &&
            (identical(other.email, email) || other.email == email) &&
            (identical(other.avatarUrl, avatarUrl) ||
                other.avatarUrl == avatarUrl) &&
            (identical(other.isAuthenticated, isAuthenticated) ||
                other.isAuthenticated == isAuthenticated) &&
            (identical(other.canAccessAdmin, canAccessAdmin) ||
                other.canAccessAdmin == canAccessAdmin) &&
            (identical(other.isModerator, isModerator) ||
                other.isModerator == isModerator) &&
            (identical(
                  other.requiresEmailVerification,
                  requiresEmailVerification,
                ) ||
                other.requiresEmailVerification == requiresEmailVerification) &&
            const DeepCollectionEquality().equals(
              other._adminPermissions,
              _adminPermissions,
            ));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
    runtimeType,
    id,
    username,
    email,
    avatarUrl,
    isAuthenticated,
    canAccessAdmin,
    isModerator,
    requiresEmailVerification,
    const DeepCollectionEquality().hash(_adminPermissions),
  );

  /// Create a copy of ViewerPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$ViewerPayloadImplCopyWith<_$ViewerPayloadImpl> get copyWith =>
      __$$ViewerPayloadImplCopyWithImpl<_$ViewerPayloadImpl>(this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$$ViewerPayloadImplToJson(this);
  }
}

abstract class _ViewerPayload implements ViewerPayload {
  const factory _ViewerPayload({
    required final int id,
    required final String username,
    required final String email,
    required final String avatarUrl,
    required final bool isAuthenticated,
    required final bool canAccessAdmin,
    required final bool isModerator,
    required final bool requiresEmailVerification,
    final List<int>? adminPermissions,
  }) = _$ViewerPayloadImpl;

  factory _ViewerPayload.fromJson(Map<String, dynamic> json) =
      _$ViewerPayloadImpl.fromJson;

  @override
  int get id;
  @override
  String get username;
  @override
  String get email;
  @override
  String get avatarUrl;
  @override
  bool get isAuthenticated;
  @override
  bool get canAccessAdmin;
  @override
  bool get isModerator;
  @override
  bool get requiresEmailVerification;
  @override
  List<int>? get adminPermissions;

  /// Create a copy of ViewerPayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$ViewerPayloadImplCopyWith<_$ViewerPayloadImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

NavItemPayload _$NavItemPayloadFromJson(Map<String, dynamic> json) {
  return _NavItemPayload.fromJson(json);
}

/// @nodoc
mixin _$NavItemPayload {
  String get key => throw _privateConstructorUsedError;
  String get label => throw _privateConstructorUsedError;
  String? get i18nLabel => throw _privateConstructorUsedError;
  String get url => throw _privateConstructorUsedError;

  /// Serializes this NavItemPayload to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of NavItemPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $NavItemPayloadCopyWith<NavItemPayload> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $NavItemPayloadCopyWith<$Res> {
  factory $NavItemPayloadCopyWith(
    NavItemPayload value,
    $Res Function(NavItemPayload) then,
  ) = _$NavItemPayloadCopyWithImpl<$Res, NavItemPayload>;
  @useResult
  $Res call({String key, String label, String? i18nLabel, String url});
}

/// @nodoc
class _$NavItemPayloadCopyWithImpl<$Res, $Val extends NavItemPayload>
    implements $NavItemPayloadCopyWith<$Res> {
  _$NavItemPayloadCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of NavItemPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? key = null,
    Object? label = null,
    Object? i18nLabel = freezed,
    Object? url = null,
  }) {
    return _then(
      _value.copyWith(
            key: null == key
                ? _value.key
                : key // ignore: cast_nullable_to_non_nullable
                      as String,
            label: null == label
                ? _value.label
                : label // ignore: cast_nullable_to_non_nullable
                      as String,
            i18nLabel: freezed == i18nLabel
                ? _value.i18nLabel
                : i18nLabel // ignore: cast_nullable_to_non_nullable
                      as String?,
            url: null == url
                ? _value.url
                : url // ignore: cast_nullable_to_non_nullable
                      as String,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$NavItemPayloadImplCopyWith<$Res>
    implements $NavItemPayloadCopyWith<$Res> {
  factory _$$NavItemPayloadImplCopyWith(
    _$NavItemPayloadImpl value,
    $Res Function(_$NavItemPayloadImpl) then,
  ) = __$$NavItemPayloadImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({String key, String label, String? i18nLabel, String url});
}

/// @nodoc
class __$$NavItemPayloadImplCopyWithImpl<$Res>
    extends _$NavItemPayloadCopyWithImpl<$Res, _$NavItemPayloadImpl>
    implements _$$NavItemPayloadImplCopyWith<$Res> {
  __$$NavItemPayloadImplCopyWithImpl(
    _$NavItemPayloadImpl _value,
    $Res Function(_$NavItemPayloadImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of NavItemPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? key = null,
    Object? label = null,
    Object? i18nLabel = freezed,
    Object? url = null,
  }) {
    return _then(
      _$NavItemPayloadImpl(
        key: null == key
            ? _value.key
            : key // ignore: cast_nullable_to_non_nullable
                  as String,
        label: null == label
            ? _value.label
            : label // ignore: cast_nullable_to_non_nullable
                  as String,
        i18nLabel: freezed == i18nLabel
            ? _value.i18nLabel
            : i18nLabel // ignore: cast_nullable_to_non_nullable
                  as String?,
        url: null == url
            ? _value.url
            : url // ignore: cast_nullable_to_non_nullable
                  as String,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$NavItemPayloadImpl implements _NavItemPayload {
  const _$NavItemPayloadImpl({
    required this.key,
    required this.label,
    this.i18nLabel,
    required this.url,
  });

  factory _$NavItemPayloadImpl.fromJson(Map<String, dynamic> json) =>
      _$$NavItemPayloadImplFromJson(json);

  @override
  final String key;
  @override
  final String label;
  @override
  final String? i18nLabel;
  @override
  final String url;

  @override
  String toString() {
    return 'NavItemPayload(key: $key, label: $label, i18nLabel: $i18nLabel, url: $url)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$NavItemPayloadImpl &&
            (identical(other.key, key) || other.key == key) &&
            (identical(other.label, label) || other.label == label) &&
            (identical(other.i18nLabel, i18nLabel) ||
                other.i18nLabel == i18nLabel) &&
            (identical(other.url, url) || other.url == url));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(runtimeType, key, label, i18nLabel, url);

  /// Create a copy of NavItemPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$NavItemPayloadImplCopyWith<_$NavItemPayloadImpl> get copyWith =>
      __$$NavItemPayloadImplCopyWithImpl<_$NavItemPayloadImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$NavItemPayloadImplToJson(this);
  }
}

abstract class _NavItemPayload implements NavItemPayload {
  const factory _NavItemPayload({
    required final String key,
    required final String label,
    final String? i18nLabel,
    required final String url,
  }) = _$NavItemPayloadImpl;

  factory _NavItemPayload.fromJson(Map<String, dynamic> json) =
      _$NavItemPayloadImpl.fromJson;

  @override
  String get key;
  @override
  String get label;
  @override
  String? get i18nLabel;
  @override
  String get url;

  /// Create a copy of NavItemPayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$NavItemPayloadImplCopyWith<_$NavItemPayloadImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

CategoryNavPayload _$CategoryNavPayloadFromJson(Map<String, dynamic> json) {
  return _CategoryNavPayload.fromJson(json);
}

/// @nodoc
mixin _$CategoryNavPayload {
  int get id => throw _privateConstructorUsedError;
  String get label => throw _privateConstructorUsedError;
  String get url => throw _privateConstructorUsedError;
  String get color => throw _privateConstructorUsedError;

  /// Serializes this CategoryNavPayload to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of CategoryNavPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $CategoryNavPayloadCopyWith<CategoryNavPayload> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $CategoryNavPayloadCopyWith<$Res> {
  factory $CategoryNavPayloadCopyWith(
    CategoryNavPayload value,
    $Res Function(CategoryNavPayload) then,
  ) = _$CategoryNavPayloadCopyWithImpl<$Res, CategoryNavPayload>;
  @useResult
  $Res call({int id, String label, String url, String color});
}

/// @nodoc
class _$CategoryNavPayloadCopyWithImpl<$Res, $Val extends CategoryNavPayload>
    implements $CategoryNavPayloadCopyWith<$Res> {
  _$CategoryNavPayloadCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of CategoryNavPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = null,
    Object? label = null,
    Object? url = null,
    Object? color = null,
  }) {
    return _then(
      _value.copyWith(
            id: null == id
                ? _value.id
                : id // ignore: cast_nullable_to_non_nullable
                      as int,
            label: null == label
                ? _value.label
                : label // ignore: cast_nullable_to_non_nullable
                      as String,
            url: null == url
                ? _value.url
                : url // ignore: cast_nullable_to_non_nullable
                      as String,
            color: null == color
                ? _value.color
                : color // ignore: cast_nullable_to_non_nullable
                      as String,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$CategoryNavPayloadImplCopyWith<$Res>
    implements $CategoryNavPayloadCopyWith<$Res> {
  factory _$$CategoryNavPayloadImplCopyWith(
    _$CategoryNavPayloadImpl value,
    $Res Function(_$CategoryNavPayloadImpl) then,
  ) = __$$CategoryNavPayloadImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({int id, String label, String url, String color});
}

/// @nodoc
class __$$CategoryNavPayloadImplCopyWithImpl<$Res>
    extends _$CategoryNavPayloadCopyWithImpl<$Res, _$CategoryNavPayloadImpl>
    implements _$$CategoryNavPayloadImplCopyWith<$Res> {
  __$$CategoryNavPayloadImplCopyWithImpl(
    _$CategoryNavPayloadImpl _value,
    $Res Function(_$CategoryNavPayloadImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of CategoryNavPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = null,
    Object? label = null,
    Object? url = null,
    Object? color = null,
  }) {
    return _then(
      _$CategoryNavPayloadImpl(
        id: null == id
            ? _value.id
            : id // ignore: cast_nullable_to_non_nullable
                  as int,
        label: null == label
            ? _value.label
            : label // ignore: cast_nullable_to_non_nullable
                  as String,
        url: null == url
            ? _value.url
            : url // ignore: cast_nullable_to_non_nullable
                  as String,
        color: null == color
            ? _value.color
            : color // ignore: cast_nullable_to_non_nullable
                  as String,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$CategoryNavPayloadImpl implements _CategoryNavPayload {
  const _$CategoryNavPayloadImpl({
    required this.id,
    required this.label,
    required this.url,
    required this.color,
  });

  factory _$CategoryNavPayloadImpl.fromJson(Map<String, dynamic> json) =>
      _$$CategoryNavPayloadImplFromJson(json);

  @override
  final int id;
  @override
  final String label;
  @override
  final String url;
  @override
  final String color;

  @override
  String toString() {
    return 'CategoryNavPayload(id: $id, label: $label, url: $url, color: $color)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$CategoryNavPayloadImpl &&
            (identical(other.id, id) || other.id == id) &&
            (identical(other.label, label) || other.label == label) &&
            (identical(other.url, url) || other.url == url) &&
            (identical(other.color, color) || other.color == color));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(runtimeType, id, label, url, color);

  /// Create a copy of CategoryNavPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$CategoryNavPayloadImplCopyWith<_$CategoryNavPayloadImpl> get copyWith =>
      __$$CategoryNavPayloadImplCopyWithImpl<_$CategoryNavPayloadImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$CategoryNavPayloadImplToJson(this);
  }
}

abstract class _CategoryNavPayload implements CategoryNavPayload {
  const factory _CategoryNavPayload({
    required final int id,
    required final String label,
    required final String url,
    required final String color,
  }) = _$CategoryNavPayloadImpl;

  factory _CategoryNavPayload.fromJson(Map<String, dynamic> json) =
      _$CategoryNavPayloadImpl.fromJson;

  @override
  int get id;
  @override
  String get label;
  @override
  String get url;
  @override
  String get color;

  /// Create a copy of CategoryNavPayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$CategoryNavPayloadImplCopyWith<_$CategoryNavPayloadImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

SidebarGroupPayload _$SidebarGroupPayloadFromJson(Map<String, dynamic> json) {
  return _SidebarGroupPayload.fromJson(json);
}

/// @nodoc
mixin _$SidebarGroupPayload {
  String get key => throw _privateConstructorUsedError;
  String get title => throw _privateConstructorUsedError;
  String? get i18nLabel => throw _privateConstructorUsedError;
  List<NavItemPayload> get items => throw _privateConstructorUsedError;

  /// Serializes this SidebarGroupPayload to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of SidebarGroupPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $SidebarGroupPayloadCopyWith<SidebarGroupPayload> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $SidebarGroupPayloadCopyWith<$Res> {
  factory $SidebarGroupPayloadCopyWith(
    SidebarGroupPayload value,
    $Res Function(SidebarGroupPayload) then,
  ) = _$SidebarGroupPayloadCopyWithImpl<$Res, SidebarGroupPayload>;
  @useResult
  $Res call({
    String key,
    String title,
    String? i18nLabel,
    List<NavItemPayload> items,
  });
}

/// @nodoc
class _$SidebarGroupPayloadCopyWithImpl<$Res, $Val extends SidebarGroupPayload>
    implements $SidebarGroupPayloadCopyWith<$Res> {
  _$SidebarGroupPayloadCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of SidebarGroupPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? key = null,
    Object? title = null,
    Object? i18nLabel = freezed,
    Object? items = null,
  }) {
    return _then(
      _value.copyWith(
            key: null == key
                ? _value.key
                : key // ignore: cast_nullable_to_non_nullable
                      as String,
            title: null == title
                ? _value.title
                : title // ignore: cast_nullable_to_non_nullable
                      as String,
            i18nLabel: freezed == i18nLabel
                ? _value.i18nLabel
                : i18nLabel // ignore: cast_nullable_to_non_nullable
                      as String?,
            items: null == items
                ? _value.items
                : items // ignore: cast_nullable_to_non_nullable
                      as List<NavItemPayload>,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$SidebarGroupPayloadImplCopyWith<$Res>
    implements $SidebarGroupPayloadCopyWith<$Res> {
  factory _$$SidebarGroupPayloadImplCopyWith(
    _$SidebarGroupPayloadImpl value,
    $Res Function(_$SidebarGroupPayloadImpl) then,
  ) = __$$SidebarGroupPayloadImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    String key,
    String title,
    String? i18nLabel,
    List<NavItemPayload> items,
  });
}

/// @nodoc
class __$$SidebarGroupPayloadImplCopyWithImpl<$Res>
    extends _$SidebarGroupPayloadCopyWithImpl<$Res, _$SidebarGroupPayloadImpl>
    implements _$$SidebarGroupPayloadImplCopyWith<$Res> {
  __$$SidebarGroupPayloadImplCopyWithImpl(
    _$SidebarGroupPayloadImpl _value,
    $Res Function(_$SidebarGroupPayloadImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of SidebarGroupPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? key = null,
    Object? title = null,
    Object? i18nLabel = freezed,
    Object? items = null,
  }) {
    return _then(
      _$SidebarGroupPayloadImpl(
        key: null == key
            ? _value.key
            : key // ignore: cast_nullable_to_non_nullable
                  as String,
        title: null == title
            ? _value.title
            : title // ignore: cast_nullable_to_non_nullable
                  as String,
        i18nLabel: freezed == i18nLabel
            ? _value.i18nLabel
            : i18nLabel // ignore: cast_nullable_to_non_nullable
                  as String?,
        items: null == items
            ? _value._items
            : items // ignore: cast_nullable_to_non_nullable
                  as List<NavItemPayload>,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$SidebarGroupPayloadImpl implements _SidebarGroupPayload {
  const _$SidebarGroupPayloadImpl({
    required this.key,
    required this.title,
    this.i18nLabel,
    required final List<NavItemPayload> items,
  }) : _items = items;

  factory _$SidebarGroupPayloadImpl.fromJson(Map<String, dynamic> json) =>
      _$$SidebarGroupPayloadImplFromJson(json);

  @override
  final String key;
  @override
  final String title;
  @override
  final String? i18nLabel;
  final List<NavItemPayload> _items;
  @override
  List<NavItemPayload> get items {
    if (_items is EqualUnmodifiableListView) return _items;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_items);
  }

  @override
  String toString() {
    return 'SidebarGroupPayload(key: $key, title: $title, i18nLabel: $i18nLabel, items: $items)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$SidebarGroupPayloadImpl &&
            (identical(other.key, key) || other.key == key) &&
            (identical(other.title, title) || other.title == title) &&
            (identical(other.i18nLabel, i18nLabel) ||
                other.i18nLabel == i18nLabel) &&
            const DeepCollectionEquality().equals(other._items, _items));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
    runtimeType,
    key,
    title,
    i18nLabel,
    const DeepCollectionEquality().hash(_items),
  );

  /// Create a copy of SidebarGroupPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$SidebarGroupPayloadImplCopyWith<_$SidebarGroupPayloadImpl> get copyWith =>
      __$$SidebarGroupPayloadImplCopyWithImpl<_$SidebarGroupPayloadImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$SidebarGroupPayloadImplToJson(this);
  }
}

abstract class _SidebarGroupPayload implements SidebarGroupPayload {
  const factory _SidebarGroupPayload({
    required final String key,
    required final String title,
    final String? i18nLabel,
    required final List<NavItemPayload> items,
  }) = _$SidebarGroupPayloadImpl;

  factory _SidebarGroupPayload.fromJson(Map<String, dynamic> json) =
      _$SidebarGroupPayloadImpl.fromJson;

  @override
  String get key;
  @override
  String get title;
  @override
  String? get i18nLabel;
  @override
  List<NavItemPayload> get items;

  /// Create a copy of SidebarGroupPayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$SidebarGroupPayloadImplCopyWith<_$SidebarGroupPayloadImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

SidebarPayload _$SidebarPayloadFromJson(Map<String, dynamic> json) {
  return _SidebarPayload.fromJson(json);
}

/// @nodoc
mixin _$SidebarPayload {
  List<NavItemPayload>? get main => throw _privateConstructorUsedError;
  List<NavItemPayload>? get resources => throw _privateConstructorUsedError;
  List<SidebarGroupPayload>? get groups => throw _privateConstructorUsedError;
  List<CategoryNavPayload> get categories => throw _privateConstructorUsedError;
  String get activeKey => throw _privateConstructorUsedError;

  /// Serializes this SidebarPayload to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of SidebarPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $SidebarPayloadCopyWith<SidebarPayload> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $SidebarPayloadCopyWith<$Res> {
  factory $SidebarPayloadCopyWith(
    SidebarPayload value,
    $Res Function(SidebarPayload) then,
  ) = _$SidebarPayloadCopyWithImpl<$Res, SidebarPayload>;
  @useResult
  $Res call({
    List<NavItemPayload>? main,
    List<NavItemPayload>? resources,
    List<SidebarGroupPayload>? groups,
    List<CategoryNavPayload> categories,
    String activeKey,
  });
}

/// @nodoc
class _$SidebarPayloadCopyWithImpl<$Res, $Val extends SidebarPayload>
    implements $SidebarPayloadCopyWith<$Res> {
  _$SidebarPayloadCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of SidebarPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? main = freezed,
    Object? resources = freezed,
    Object? groups = freezed,
    Object? categories = null,
    Object? activeKey = null,
  }) {
    return _then(
      _value.copyWith(
            main: freezed == main
                ? _value.main
                : main // ignore: cast_nullable_to_non_nullable
                      as List<NavItemPayload>?,
            resources: freezed == resources
                ? _value.resources
                : resources // ignore: cast_nullable_to_non_nullable
                      as List<NavItemPayload>?,
            groups: freezed == groups
                ? _value.groups
                : groups // ignore: cast_nullable_to_non_nullable
                      as List<SidebarGroupPayload>?,
            categories: null == categories
                ? _value.categories
                : categories // ignore: cast_nullable_to_non_nullable
                      as List<CategoryNavPayload>,
            activeKey: null == activeKey
                ? _value.activeKey
                : activeKey // ignore: cast_nullable_to_non_nullable
                      as String,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$SidebarPayloadImplCopyWith<$Res>
    implements $SidebarPayloadCopyWith<$Res> {
  factory _$$SidebarPayloadImplCopyWith(
    _$SidebarPayloadImpl value,
    $Res Function(_$SidebarPayloadImpl) then,
  ) = __$$SidebarPayloadImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    List<NavItemPayload>? main,
    List<NavItemPayload>? resources,
    List<SidebarGroupPayload>? groups,
    List<CategoryNavPayload> categories,
    String activeKey,
  });
}

/// @nodoc
class __$$SidebarPayloadImplCopyWithImpl<$Res>
    extends _$SidebarPayloadCopyWithImpl<$Res, _$SidebarPayloadImpl>
    implements _$$SidebarPayloadImplCopyWith<$Res> {
  __$$SidebarPayloadImplCopyWithImpl(
    _$SidebarPayloadImpl _value,
    $Res Function(_$SidebarPayloadImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of SidebarPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? main = freezed,
    Object? resources = freezed,
    Object? groups = freezed,
    Object? categories = null,
    Object? activeKey = null,
  }) {
    return _then(
      _$SidebarPayloadImpl(
        main: freezed == main
            ? _value._main
            : main // ignore: cast_nullable_to_non_nullable
                  as List<NavItemPayload>?,
        resources: freezed == resources
            ? _value._resources
            : resources // ignore: cast_nullable_to_non_nullable
                  as List<NavItemPayload>?,
        groups: freezed == groups
            ? _value._groups
            : groups // ignore: cast_nullable_to_non_nullable
                  as List<SidebarGroupPayload>?,
        categories: null == categories
            ? _value._categories
            : categories // ignore: cast_nullable_to_non_nullable
                  as List<CategoryNavPayload>,
        activeKey: null == activeKey
            ? _value.activeKey
            : activeKey // ignore: cast_nullable_to_non_nullable
                  as String,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$SidebarPayloadImpl implements _SidebarPayload {
  const _$SidebarPayloadImpl({
    final List<NavItemPayload>? main,
    final List<NavItemPayload>? resources,
    final List<SidebarGroupPayload>? groups,
    required final List<CategoryNavPayload> categories,
    required this.activeKey,
  }) : _main = main,
       _resources = resources,
       _groups = groups,
       _categories = categories;

  factory _$SidebarPayloadImpl.fromJson(Map<String, dynamic> json) =>
      _$$SidebarPayloadImplFromJson(json);

  final List<NavItemPayload>? _main;
  @override
  List<NavItemPayload>? get main {
    final value = _main;
    if (value == null) return null;
    if (_main is EqualUnmodifiableListView) return _main;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(value);
  }

  final List<NavItemPayload>? _resources;
  @override
  List<NavItemPayload>? get resources {
    final value = _resources;
    if (value == null) return null;
    if (_resources is EqualUnmodifiableListView) return _resources;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(value);
  }

  final List<SidebarGroupPayload>? _groups;
  @override
  List<SidebarGroupPayload>? get groups {
    final value = _groups;
    if (value == null) return null;
    if (_groups is EqualUnmodifiableListView) return _groups;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(value);
  }

  final List<CategoryNavPayload> _categories;
  @override
  List<CategoryNavPayload> get categories {
    if (_categories is EqualUnmodifiableListView) return _categories;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_categories);
  }

  @override
  final String activeKey;

  @override
  String toString() {
    return 'SidebarPayload(main: $main, resources: $resources, groups: $groups, categories: $categories, activeKey: $activeKey)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$SidebarPayloadImpl &&
            const DeepCollectionEquality().equals(other._main, _main) &&
            const DeepCollectionEquality().equals(
              other._resources,
              _resources,
            ) &&
            const DeepCollectionEquality().equals(other._groups, _groups) &&
            const DeepCollectionEquality().equals(
              other._categories,
              _categories,
            ) &&
            (identical(other.activeKey, activeKey) ||
                other.activeKey == activeKey));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
    runtimeType,
    const DeepCollectionEquality().hash(_main),
    const DeepCollectionEquality().hash(_resources),
    const DeepCollectionEquality().hash(_groups),
    const DeepCollectionEquality().hash(_categories),
    activeKey,
  );

  /// Create a copy of SidebarPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$SidebarPayloadImplCopyWith<_$SidebarPayloadImpl> get copyWith =>
      __$$SidebarPayloadImplCopyWithImpl<_$SidebarPayloadImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$SidebarPayloadImplToJson(this);
  }
}

abstract class _SidebarPayload implements SidebarPayload {
  const factory _SidebarPayload({
    final List<NavItemPayload>? main,
    final List<NavItemPayload>? resources,
    final List<SidebarGroupPayload>? groups,
    required final List<CategoryNavPayload> categories,
    required final String activeKey,
  }) = _$SidebarPayloadImpl;

  factory _SidebarPayload.fromJson(Map<String, dynamic> json) =
      _$SidebarPayloadImpl.fromJson;

  @override
  List<NavItemPayload>? get main;
  @override
  List<NavItemPayload>? get resources;
  @override
  List<SidebarGroupPayload>? get groups;
  @override
  List<CategoryNavPayload> get categories;
  @override
  String get activeKey;

  /// Create a copy of SidebarPayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$SidebarPayloadImplCopyWith<_$SidebarPayloadImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

FooterPayload _$FooterPayloadFromJson(Map<String, dynamic> json) {
  return _FooterPayload.fromJson(json);
}

/// @nodoc
mixin _$FooterPayload {
  List<FooterLinkPayload> get links => throw _privateConstructorUsedError;
  List<String> get primary => throw _privateConstructorUsedError;

  /// Serializes this FooterPayload to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of FooterPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $FooterPayloadCopyWith<FooterPayload> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $FooterPayloadCopyWith<$Res> {
  factory $FooterPayloadCopyWith(
    FooterPayload value,
    $Res Function(FooterPayload) then,
  ) = _$FooterPayloadCopyWithImpl<$Res, FooterPayload>;
  @useResult
  $Res call({List<FooterLinkPayload> links, List<String> primary});
}

/// @nodoc
class _$FooterPayloadCopyWithImpl<$Res, $Val extends FooterPayload>
    implements $FooterPayloadCopyWith<$Res> {
  _$FooterPayloadCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of FooterPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({Object? links = null, Object? primary = null}) {
    return _then(
      _value.copyWith(
            links: null == links
                ? _value.links
                : links // ignore: cast_nullable_to_non_nullable
                      as List<FooterLinkPayload>,
            primary: null == primary
                ? _value.primary
                : primary // ignore: cast_nullable_to_non_nullable
                      as List<String>,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$FooterPayloadImplCopyWith<$Res>
    implements $FooterPayloadCopyWith<$Res> {
  factory _$$FooterPayloadImplCopyWith(
    _$FooterPayloadImpl value,
    $Res Function(_$FooterPayloadImpl) then,
  ) = __$$FooterPayloadImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({List<FooterLinkPayload> links, List<String> primary});
}

/// @nodoc
class __$$FooterPayloadImplCopyWithImpl<$Res>
    extends _$FooterPayloadCopyWithImpl<$Res, _$FooterPayloadImpl>
    implements _$$FooterPayloadImplCopyWith<$Res> {
  __$$FooterPayloadImplCopyWithImpl(
    _$FooterPayloadImpl _value,
    $Res Function(_$FooterPayloadImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of FooterPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({Object? links = null, Object? primary = null}) {
    return _then(
      _$FooterPayloadImpl(
        links: null == links
            ? _value._links
            : links // ignore: cast_nullable_to_non_nullable
                  as List<FooterLinkPayload>,
        primary: null == primary
            ? _value._primary
            : primary // ignore: cast_nullable_to_non_nullable
                  as List<String>,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$FooterPayloadImpl implements _FooterPayload {
  const _$FooterPayloadImpl({
    required final List<FooterLinkPayload> links,
    required final List<String> primary,
  }) : _links = links,
       _primary = primary;

  factory _$FooterPayloadImpl.fromJson(Map<String, dynamic> json) =>
      _$$FooterPayloadImplFromJson(json);

  final List<FooterLinkPayload> _links;
  @override
  List<FooterLinkPayload> get links {
    if (_links is EqualUnmodifiableListView) return _links;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_links);
  }

  final List<String> _primary;
  @override
  List<String> get primary {
    if (_primary is EqualUnmodifiableListView) return _primary;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_primary);
  }

  @override
  String toString() {
    return 'FooterPayload(links: $links, primary: $primary)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$FooterPayloadImpl &&
            const DeepCollectionEquality().equals(other._links, _links) &&
            const DeepCollectionEquality().equals(other._primary, _primary));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
    runtimeType,
    const DeepCollectionEquality().hash(_links),
    const DeepCollectionEquality().hash(_primary),
  );

  /// Create a copy of FooterPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$FooterPayloadImplCopyWith<_$FooterPayloadImpl> get copyWith =>
      __$$FooterPayloadImplCopyWithImpl<_$FooterPayloadImpl>(this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$$FooterPayloadImplToJson(this);
  }
}

abstract class _FooterPayload implements FooterPayload {
  const factory _FooterPayload({
    required final List<FooterLinkPayload> links,
    required final List<String> primary,
  }) = _$FooterPayloadImpl;

  factory _FooterPayload.fromJson(Map<String, dynamic> json) =
      _$FooterPayloadImpl.fromJson;

  @override
  List<FooterLinkPayload> get links;
  @override
  List<String> get primary;

  /// Create a copy of FooterPayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$FooterPayloadImplCopyWith<_$FooterPayloadImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

FooterLinkPayload _$FooterLinkPayloadFromJson(Map<String, dynamic> json) {
  return _FooterLinkPayload.fromJson(json);
}

/// @nodoc
mixin _$FooterLinkPayload {
  String get name => throw _privateConstructorUsedError;
  String get url => throw _privateConstructorUsedError;

  /// Serializes this FooterLinkPayload to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of FooterLinkPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $FooterLinkPayloadCopyWith<FooterLinkPayload> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $FooterLinkPayloadCopyWith<$Res> {
  factory $FooterLinkPayloadCopyWith(
    FooterLinkPayload value,
    $Res Function(FooterLinkPayload) then,
  ) = _$FooterLinkPayloadCopyWithImpl<$Res, FooterLinkPayload>;
  @useResult
  $Res call({String name, String url});
}

/// @nodoc
class _$FooterLinkPayloadCopyWithImpl<$Res, $Val extends FooterLinkPayload>
    implements $FooterLinkPayloadCopyWith<$Res> {
  _$FooterLinkPayloadCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of FooterLinkPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({Object? name = null, Object? url = null}) {
    return _then(
      _value.copyWith(
            name: null == name
                ? _value.name
                : name // ignore: cast_nullable_to_non_nullable
                      as String,
            url: null == url
                ? _value.url
                : url // ignore: cast_nullable_to_non_nullable
                      as String,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$FooterLinkPayloadImplCopyWith<$Res>
    implements $FooterLinkPayloadCopyWith<$Res> {
  factory _$$FooterLinkPayloadImplCopyWith(
    _$FooterLinkPayloadImpl value,
    $Res Function(_$FooterLinkPayloadImpl) then,
  ) = __$$FooterLinkPayloadImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({String name, String url});
}

/// @nodoc
class __$$FooterLinkPayloadImplCopyWithImpl<$Res>
    extends _$FooterLinkPayloadCopyWithImpl<$Res, _$FooterLinkPayloadImpl>
    implements _$$FooterLinkPayloadImplCopyWith<$Res> {
  __$$FooterLinkPayloadImplCopyWithImpl(
    _$FooterLinkPayloadImpl _value,
    $Res Function(_$FooterLinkPayloadImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of FooterLinkPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({Object? name = null, Object? url = null}) {
    return _then(
      _$FooterLinkPayloadImpl(
        name: null == name
            ? _value.name
            : name // ignore: cast_nullable_to_non_nullable
                  as String,
        url: null == url
            ? _value.url
            : url // ignore: cast_nullable_to_non_nullable
                  as String,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$FooterLinkPayloadImpl implements _FooterLinkPayload {
  const _$FooterLinkPayloadImpl({required this.name, required this.url});

  factory _$FooterLinkPayloadImpl.fromJson(Map<String, dynamic> json) =>
      _$$FooterLinkPayloadImplFromJson(json);

  @override
  final String name;
  @override
  final String url;

  @override
  String toString() {
    return 'FooterLinkPayload(name: $name, url: $url)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$FooterLinkPayloadImpl &&
            (identical(other.name, name) || other.name == name) &&
            (identical(other.url, url) || other.url == url));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(runtimeType, name, url);

  /// Create a copy of FooterLinkPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$FooterLinkPayloadImplCopyWith<_$FooterLinkPayloadImpl> get copyWith =>
      __$$FooterLinkPayloadImplCopyWithImpl<_$FooterLinkPayloadImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$FooterLinkPayloadImplToJson(this);
  }
}

abstract class _FooterLinkPayload implements FooterLinkPayload {
  const factory _FooterLinkPayload({
    required final String name,
    required final String url,
  }) = _$FooterLinkPayloadImpl;

  factory _FooterLinkPayload.fromJson(Map<String, dynamic> json) =
      _$FooterLinkPayloadImpl.fromJson;

  @override
  String get name;
  @override
  String get url;

  /// Create a copy of FooterLinkPayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$FooterLinkPayloadImplCopyWith<_$FooterLinkPayloadImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

ThemePayload _$ThemePayloadFromJson(Map<String, dynamic> json) {
  return _ThemePayload.fromJson(json);
}

/// @nodoc
mixin _$ThemePayload {
  bool get enabled => throw _privateConstructorUsedError;
  String? get href => throw _privateConstructorUsedError;
  Map<String, String>? get colors => throw _privateConstructorUsedError;
  String get current => throw _privateConstructorUsedError;
  String get themeColor => throw _privateConstructorUsedError;

  /// Serializes this ThemePayload to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of ThemePayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $ThemePayloadCopyWith<ThemePayload> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $ThemePayloadCopyWith<$Res> {
  factory $ThemePayloadCopyWith(
    ThemePayload value,
    $Res Function(ThemePayload) then,
  ) = _$ThemePayloadCopyWithImpl<$Res, ThemePayload>;
  @useResult
  $Res call({
    bool enabled,
    String? href,
    Map<String, String>? colors,
    String current,
    String themeColor,
  });
}

/// @nodoc
class _$ThemePayloadCopyWithImpl<$Res, $Val extends ThemePayload>
    implements $ThemePayloadCopyWith<$Res> {
  _$ThemePayloadCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of ThemePayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? enabled = null,
    Object? href = freezed,
    Object? colors = freezed,
    Object? current = null,
    Object? themeColor = null,
  }) {
    return _then(
      _value.copyWith(
            enabled: null == enabled
                ? _value.enabled
                : enabled // ignore: cast_nullable_to_non_nullable
                      as bool,
            href: freezed == href
                ? _value.href
                : href // ignore: cast_nullable_to_non_nullable
                      as String?,
            colors: freezed == colors
                ? _value.colors
                : colors // ignore: cast_nullable_to_non_nullable
                      as Map<String, String>?,
            current: null == current
                ? _value.current
                : current // ignore: cast_nullable_to_non_nullable
                      as String,
            themeColor: null == themeColor
                ? _value.themeColor
                : themeColor // ignore: cast_nullable_to_non_nullable
                      as String,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$ThemePayloadImplCopyWith<$Res>
    implements $ThemePayloadCopyWith<$Res> {
  factory _$$ThemePayloadImplCopyWith(
    _$ThemePayloadImpl value,
    $Res Function(_$ThemePayloadImpl) then,
  ) = __$$ThemePayloadImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    bool enabled,
    String? href,
    Map<String, String>? colors,
    String current,
    String themeColor,
  });
}

/// @nodoc
class __$$ThemePayloadImplCopyWithImpl<$Res>
    extends _$ThemePayloadCopyWithImpl<$Res, _$ThemePayloadImpl>
    implements _$$ThemePayloadImplCopyWith<$Res> {
  __$$ThemePayloadImplCopyWithImpl(
    _$ThemePayloadImpl _value,
    $Res Function(_$ThemePayloadImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of ThemePayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? enabled = null,
    Object? href = freezed,
    Object? colors = freezed,
    Object? current = null,
    Object? themeColor = null,
  }) {
    return _then(
      _$ThemePayloadImpl(
        enabled: null == enabled
            ? _value.enabled
            : enabled // ignore: cast_nullable_to_non_nullable
                  as bool,
        href: freezed == href
            ? _value.href
            : href // ignore: cast_nullable_to_non_nullable
                  as String?,
        colors: freezed == colors
            ? _value._colors
            : colors // ignore: cast_nullable_to_non_nullable
                  as Map<String, String>?,
        current: null == current
            ? _value.current
            : current // ignore: cast_nullable_to_non_nullable
                  as String,
        themeColor: null == themeColor
            ? _value.themeColor
            : themeColor // ignore: cast_nullable_to_non_nullable
                  as String,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$ThemePayloadImpl implements _ThemePayload {
  const _$ThemePayloadImpl({
    required this.enabled,
    this.href,
    final Map<String, String>? colors,
    required this.current,
    required this.themeColor,
  }) : _colors = colors;

  factory _$ThemePayloadImpl.fromJson(Map<String, dynamic> json) =>
      _$$ThemePayloadImplFromJson(json);

  @override
  final bool enabled;
  @override
  final String? href;
  final Map<String, String>? _colors;
  @override
  Map<String, String>? get colors {
    final value = _colors;
    if (value == null) return null;
    if (_colors is EqualUnmodifiableMapView) return _colors;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableMapView(value);
  }

  @override
  final String current;
  @override
  final String themeColor;

  @override
  String toString() {
    return 'ThemePayload(enabled: $enabled, href: $href, colors: $colors, current: $current, themeColor: $themeColor)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$ThemePayloadImpl &&
            (identical(other.enabled, enabled) || other.enabled == enabled) &&
            (identical(other.href, href) || other.href == href) &&
            const DeepCollectionEquality().equals(other._colors, _colors) &&
            (identical(other.current, current) || other.current == current) &&
            (identical(other.themeColor, themeColor) ||
                other.themeColor == themeColor));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
    runtimeType,
    enabled,
    href,
    const DeepCollectionEquality().hash(_colors),
    current,
    themeColor,
  );

  /// Create a copy of ThemePayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$ThemePayloadImplCopyWith<_$ThemePayloadImpl> get copyWith =>
      __$$ThemePayloadImplCopyWithImpl<_$ThemePayloadImpl>(this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$$ThemePayloadImplToJson(this);
  }
}

abstract class _ThemePayload implements ThemePayload {
  const factory _ThemePayload({
    required final bool enabled,
    final String? href,
    final Map<String, String>? colors,
    required final String current,
    required final String themeColor,
  }) = _$ThemePayloadImpl;

  factory _ThemePayload.fromJson(Map<String, dynamic> json) =
      _$ThemePayloadImpl.fromJson;

  @override
  bool get enabled;
  @override
  String? get href;
  @override
  Map<String, String>? get colors;
  @override
  String get current;
  @override
  String get themeColor;

  /// Create a copy of ThemePayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$ThemePayloadImplCopyWith<_$ThemePayloadImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

UnreadStatusPayload _$UnreadStatusPayloadFromJson(Map<String, dynamic> json) {
  return _UnreadStatusPayload.fromJson(json);
}

/// @nodoc
mixin _$UnreadStatusPayload {
  bool get notifications => throw _privateConstructorUsedError;
  bool get messages => throw _privateConstructorUsedError;
  bool? get moderationReports => throw _privateConstructorUsedError;
  String? get latestNotificationType => throw _privateConstructorUsedError;
  int? get latestUnreadId => throw _privateConstructorUsedError;

  /// Serializes this UnreadStatusPayload to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of UnreadStatusPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $UnreadStatusPayloadCopyWith<UnreadStatusPayload> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $UnreadStatusPayloadCopyWith<$Res> {
  factory $UnreadStatusPayloadCopyWith(
    UnreadStatusPayload value,
    $Res Function(UnreadStatusPayload) then,
  ) = _$UnreadStatusPayloadCopyWithImpl<$Res, UnreadStatusPayload>;
  @useResult
  $Res call({
    bool notifications,
    bool messages,
    bool? moderationReports,
    String? latestNotificationType,
    int? latestUnreadId,
  });
}

/// @nodoc
class _$UnreadStatusPayloadCopyWithImpl<$Res, $Val extends UnreadStatusPayload>
    implements $UnreadStatusPayloadCopyWith<$Res> {
  _$UnreadStatusPayloadCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of UnreadStatusPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? notifications = null,
    Object? messages = null,
    Object? moderationReports = freezed,
    Object? latestNotificationType = freezed,
    Object? latestUnreadId = freezed,
  }) {
    return _then(
      _value.copyWith(
            notifications: null == notifications
                ? _value.notifications
                : notifications // ignore: cast_nullable_to_non_nullable
                      as bool,
            messages: null == messages
                ? _value.messages
                : messages // ignore: cast_nullable_to_non_nullable
                      as bool,
            moderationReports: freezed == moderationReports
                ? _value.moderationReports
                : moderationReports // ignore: cast_nullable_to_non_nullable
                      as bool?,
            latestNotificationType: freezed == latestNotificationType
                ? _value.latestNotificationType
                : latestNotificationType // ignore: cast_nullable_to_non_nullable
                      as String?,
            latestUnreadId: freezed == latestUnreadId
                ? _value.latestUnreadId
                : latestUnreadId // ignore: cast_nullable_to_non_nullable
                      as int?,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$UnreadStatusPayloadImplCopyWith<$Res>
    implements $UnreadStatusPayloadCopyWith<$Res> {
  factory _$$UnreadStatusPayloadImplCopyWith(
    _$UnreadStatusPayloadImpl value,
    $Res Function(_$UnreadStatusPayloadImpl) then,
  ) = __$$UnreadStatusPayloadImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    bool notifications,
    bool messages,
    bool? moderationReports,
    String? latestNotificationType,
    int? latestUnreadId,
  });
}

/// @nodoc
class __$$UnreadStatusPayloadImplCopyWithImpl<$Res>
    extends _$UnreadStatusPayloadCopyWithImpl<$Res, _$UnreadStatusPayloadImpl>
    implements _$$UnreadStatusPayloadImplCopyWith<$Res> {
  __$$UnreadStatusPayloadImplCopyWithImpl(
    _$UnreadStatusPayloadImpl _value,
    $Res Function(_$UnreadStatusPayloadImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of UnreadStatusPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? notifications = null,
    Object? messages = null,
    Object? moderationReports = freezed,
    Object? latestNotificationType = freezed,
    Object? latestUnreadId = freezed,
  }) {
    return _then(
      _$UnreadStatusPayloadImpl(
        notifications: null == notifications
            ? _value.notifications
            : notifications // ignore: cast_nullable_to_non_nullable
                  as bool,
        messages: null == messages
            ? _value.messages
            : messages // ignore: cast_nullable_to_non_nullable
                  as bool,
        moderationReports: freezed == moderationReports
            ? _value.moderationReports
            : moderationReports // ignore: cast_nullable_to_non_nullable
                  as bool?,
        latestNotificationType: freezed == latestNotificationType
            ? _value.latestNotificationType
            : latestNotificationType // ignore: cast_nullable_to_non_nullable
                  as String?,
        latestUnreadId: freezed == latestUnreadId
            ? _value.latestUnreadId
            : latestUnreadId // ignore: cast_nullable_to_non_nullable
                  as int?,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$UnreadStatusPayloadImpl implements _UnreadStatusPayload {
  const _$UnreadStatusPayloadImpl({
    required this.notifications,
    required this.messages,
    this.moderationReports,
    this.latestNotificationType,
    this.latestUnreadId,
  });

  factory _$UnreadStatusPayloadImpl.fromJson(Map<String, dynamic> json) =>
      _$$UnreadStatusPayloadImplFromJson(json);

  @override
  final bool notifications;
  @override
  final bool messages;
  @override
  final bool? moderationReports;
  @override
  final String? latestNotificationType;
  @override
  final int? latestUnreadId;

  @override
  String toString() {
    return 'UnreadStatusPayload(notifications: $notifications, messages: $messages, moderationReports: $moderationReports, latestNotificationType: $latestNotificationType, latestUnreadId: $latestUnreadId)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$UnreadStatusPayloadImpl &&
            (identical(other.notifications, notifications) ||
                other.notifications == notifications) &&
            (identical(other.messages, messages) ||
                other.messages == messages) &&
            (identical(other.moderationReports, moderationReports) ||
                other.moderationReports == moderationReports) &&
            (identical(other.latestNotificationType, latestNotificationType) ||
                other.latestNotificationType == latestNotificationType) &&
            (identical(other.latestUnreadId, latestUnreadId) ||
                other.latestUnreadId == latestUnreadId));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
    runtimeType,
    notifications,
    messages,
    moderationReports,
    latestNotificationType,
    latestUnreadId,
  );

  /// Create a copy of UnreadStatusPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$UnreadStatusPayloadImplCopyWith<_$UnreadStatusPayloadImpl> get copyWith =>
      __$$UnreadStatusPayloadImplCopyWithImpl<_$UnreadStatusPayloadImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$UnreadStatusPayloadImplToJson(this);
  }
}

abstract class _UnreadStatusPayload implements UnreadStatusPayload {
  const factory _UnreadStatusPayload({
    required final bool notifications,
    required final bool messages,
    final bool? moderationReports,
    final String? latestNotificationType,
    final int? latestUnreadId,
  }) = _$UnreadStatusPayloadImpl;

  factory _UnreadStatusPayload.fromJson(Map<String, dynamic> json) =
      _$UnreadStatusPayloadImpl.fromJson;

  @override
  bool get notifications;
  @override
  bool get messages;
  @override
  bool? get moderationReports;
  @override
  String? get latestNotificationType;
  @override
  int? get latestUnreadId;

  /// Create a copy of UnreadStatusPayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$UnreadStatusPayloadImplCopyWith<_$UnreadStatusPayloadImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

ThemePreviewProps _$ThemePreviewPropsFromJson(Map<String, dynamic> json) {
  return _ThemePreviewProps.fromJson(json);
}

/// @nodoc
mixin _$ThemePreviewProps {
  SiteThemeConfig get theme => throw _privateConstructorUsedError;
  SiteThemeConfig get defaults => throw _privateConstructorUsedError;

  /// Serializes this ThemePreviewProps to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of ThemePreviewProps
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $ThemePreviewPropsCopyWith<ThemePreviewProps> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $ThemePreviewPropsCopyWith<$Res> {
  factory $ThemePreviewPropsCopyWith(
    ThemePreviewProps value,
    $Res Function(ThemePreviewProps) then,
  ) = _$ThemePreviewPropsCopyWithImpl<$Res, ThemePreviewProps>;
  @useResult
  $Res call({SiteThemeConfig theme, SiteThemeConfig defaults});

  $SiteThemeConfigCopyWith<$Res> get theme;
  $SiteThemeConfigCopyWith<$Res> get defaults;
}

/// @nodoc
class _$ThemePreviewPropsCopyWithImpl<$Res, $Val extends ThemePreviewProps>
    implements $ThemePreviewPropsCopyWith<$Res> {
  _$ThemePreviewPropsCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of ThemePreviewProps
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({Object? theme = null, Object? defaults = null}) {
    return _then(
      _value.copyWith(
            theme: null == theme
                ? _value.theme
                : theme // ignore: cast_nullable_to_non_nullable
                      as SiteThemeConfig,
            defaults: null == defaults
                ? _value.defaults
                : defaults // ignore: cast_nullable_to_non_nullable
                      as SiteThemeConfig,
          )
          as $Val,
    );
  }

  /// Create a copy of ThemePreviewProps
  /// with the given fields replaced by the non-null parameter values.
  @override
  @pragma('vm:prefer-inline')
  $SiteThemeConfigCopyWith<$Res> get theme {
    return $SiteThemeConfigCopyWith<$Res>(_value.theme, (value) {
      return _then(_value.copyWith(theme: value) as $Val);
    });
  }

  /// Create a copy of ThemePreviewProps
  /// with the given fields replaced by the non-null parameter values.
  @override
  @pragma('vm:prefer-inline')
  $SiteThemeConfigCopyWith<$Res> get defaults {
    return $SiteThemeConfigCopyWith<$Res>(_value.defaults, (value) {
      return _then(_value.copyWith(defaults: value) as $Val);
    });
  }
}

/// @nodoc
abstract class _$$ThemePreviewPropsImplCopyWith<$Res>
    implements $ThemePreviewPropsCopyWith<$Res> {
  factory _$$ThemePreviewPropsImplCopyWith(
    _$ThemePreviewPropsImpl value,
    $Res Function(_$ThemePreviewPropsImpl) then,
  ) = __$$ThemePreviewPropsImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({SiteThemeConfig theme, SiteThemeConfig defaults});

  @override
  $SiteThemeConfigCopyWith<$Res> get theme;
  @override
  $SiteThemeConfigCopyWith<$Res> get defaults;
}

/// @nodoc
class __$$ThemePreviewPropsImplCopyWithImpl<$Res>
    extends _$ThemePreviewPropsCopyWithImpl<$Res, _$ThemePreviewPropsImpl>
    implements _$$ThemePreviewPropsImplCopyWith<$Res> {
  __$$ThemePreviewPropsImplCopyWithImpl(
    _$ThemePreviewPropsImpl _value,
    $Res Function(_$ThemePreviewPropsImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of ThemePreviewProps
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({Object? theme = null, Object? defaults = null}) {
    return _then(
      _$ThemePreviewPropsImpl(
        theme: null == theme
            ? _value.theme
            : theme // ignore: cast_nullable_to_non_nullable
                  as SiteThemeConfig,
        defaults: null == defaults
            ? _value.defaults
            : defaults // ignore: cast_nullable_to_non_nullable
                  as SiteThemeConfig,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$ThemePreviewPropsImpl implements _ThemePreviewProps {
  const _$ThemePreviewPropsImpl({required this.theme, required this.defaults});

  factory _$ThemePreviewPropsImpl.fromJson(Map<String, dynamic> json) =>
      _$$ThemePreviewPropsImplFromJson(json);

  @override
  final SiteThemeConfig theme;
  @override
  final SiteThemeConfig defaults;

  @override
  String toString() {
    return 'ThemePreviewProps(theme: $theme, defaults: $defaults)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$ThemePreviewPropsImpl &&
            (identical(other.theme, theme) || other.theme == theme) &&
            (identical(other.defaults, defaults) ||
                other.defaults == defaults));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(runtimeType, theme, defaults);

  /// Create a copy of ThemePreviewProps
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$ThemePreviewPropsImplCopyWith<_$ThemePreviewPropsImpl> get copyWith =>
      __$$ThemePreviewPropsImplCopyWithImpl<_$ThemePreviewPropsImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$ThemePreviewPropsImplToJson(this);
  }
}

abstract class _ThemePreviewProps implements ThemePreviewProps {
  const factory _ThemePreviewProps({
    required final SiteThemeConfig theme,
    required final SiteThemeConfig defaults,
  }) = _$ThemePreviewPropsImpl;

  factory _ThemePreviewProps.fromJson(Map<String, dynamic> json) =
      _$ThemePreviewPropsImpl.fromJson;

  @override
  SiteThemeConfig get theme;
  @override
  SiteThemeConfig get defaults;

  /// Create a copy of ThemePreviewProps
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$ThemePreviewPropsImplCopyWith<_$ThemePreviewPropsImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

SiteThemeConfig _$SiteThemeConfigFromJson(Map<String, dynamic> json) {
  return _SiteThemeConfig.fromJson(json);
}

/// @nodoc
mixin _$SiteThemeConfig {
  int get version => throw _privateConstructorUsedError;
  bool get enabled => throw _privateConstructorUsedError;
  List<SiteThemeDefinition> get themes => throw _privateConstructorUsedError;
  SiteThemePrepublish? get prepublish => throw _privateConstructorUsedError;
  String? get publishedAt => throw _privateConstructorUsedError;

  /// Serializes this SiteThemeConfig to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of SiteThemeConfig
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $SiteThemeConfigCopyWith<SiteThemeConfig> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $SiteThemeConfigCopyWith<$Res> {
  factory $SiteThemeConfigCopyWith(
    SiteThemeConfig value,
    $Res Function(SiteThemeConfig) then,
  ) = _$SiteThemeConfigCopyWithImpl<$Res, SiteThemeConfig>;
  @useResult
  $Res call({
    int version,
    bool enabled,
    List<SiteThemeDefinition> themes,
    SiteThemePrepublish? prepublish,
    String? publishedAt,
  });

  $SiteThemePrepublishCopyWith<$Res>? get prepublish;
}

/// @nodoc
class _$SiteThemeConfigCopyWithImpl<$Res, $Val extends SiteThemeConfig>
    implements $SiteThemeConfigCopyWith<$Res> {
  _$SiteThemeConfigCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of SiteThemeConfig
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? version = null,
    Object? enabled = null,
    Object? themes = null,
    Object? prepublish = freezed,
    Object? publishedAt = freezed,
  }) {
    return _then(
      _value.copyWith(
            version: null == version
                ? _value.version
                : version // ignore: cast_nullable_to_non_nullable
                      as int,
            enabled: null == enabled
                ? _value.enabled
                : enabled // ignore: cast_nullable_to_non_nullable
                      as bool,
            themes: null == themes
                ? _value.themes
                : themes // ignore: cast_nullable_to_non_nullable
                      as List<SiteThemeDefinition>,
            prepublish: freezed == prepublish
                ? _value.prepublish
                : prepublish // ignore: cast_nullable_to_non_nullable
                      as SiteThemePrepublish?,
            publishedAt: freezed == publishedAt
                ? _value.publishedAt
                : publishedAt // ignore: cast_nullable_to_non_nullable
                      as String?,
          )
          as $Val,
    );
  }

  /// Create a copy of SiteThemeConfig
  /// with the given fields replaced by the non-null parameter values.
  @override
  @pragma('vm:prefer-inline')
  $SiteThemePrepublishCopyWith<$Res>? get prepublish {
    if (_value.prepublish == null) {
      return null;
    }

    return $SiteThemePrepublishCopyWith<$Res>(_value.prepublish!, (value) {
      return _then(_value.copyWith(prepublish: value) as $Val);
    });
  }
}

/// @nodoc
abstract class _$$SiteThemeConfigImplCopyWith<$Res>
    implements $SiteThemeConfigCopyWith<$Res> {
  factory _$$SiteThemeConfigImplCopyWith(
    _$SiteThemeConfigImpl value,
    $Res Function(_$SiteThemeConfigImpl) then,
  ) = __$$SiteThemeConfigImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    int version,
    bool enabled,
    List<SiteThemeDefinition> themes,
    SiteThemePrepublish? prepublish,
    String? publishedAt,
  });

  @override
  $SiteThemePrepublishCopyWith<$Res>? get prepublish;
}

/// @nodoc
class __$$SiteThemeConfigImplCopyWithImpl<$Res>
    extends _$SiteThemeConfigCopyWithImpl<$Res, _$SiteThemeConfigImpl>
    implements _$$SiteThemeConfigImplCopyWith<$Res> {
  __$$SiteThemeConfigImplCopyWithImpl(
    _$SiteThemeConfigImpl _value,
    $Res Function(_$SiteThemeConfigImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of SiteThemeConfig
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? version = null,
    Object? enabled = null,
    Object? themes = null,
    Object? prepublish = freezed,
    Object? publishedAt = freezed,
  }) {
    return _then(
      _$SiteThemeConfigImpl(
        version: null == version
            ? _value.version
            : version // ignore: cast_nullable_to_non_nullable
                  as int,
        enabled: null == enabled
            ? _value.enabled
            : enabled // ignore: cast_nullable_to_non_nullable
                  as bool,
        themes: null == themes
            ? _value._themes
            : themes // ignore: cast_nullable_to_non_nullable
                  as List<SiteThemeDefinition>,
        prepublish: freezed == prepublish
            ? _value.prepublish
            : prepublish // ignore: cast_nullable_to_non_nullable
                  as SiteThemePrepublish?,
        publishedAt: freezed == publishedAt
            ? _value.publishedAt
            : publishedAt // ignore: cast_nullable_to_non_nullable
                  as String?,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$SiteThemeConfigImpl implements _SiteThemeConfig {
  const _$SiteThemeConfigImpl({
    required this.version,
    required this.enabled,
    required final List<SiteThemeDefinition> themes,
    this.prepublish,
    this.publishedAt,
  }) : _themes = themes;

  factory _$SiteThemeConfigImpl.fromJson(Map<String, dynamic> json) =>
      _$$SiteThemeConfigImplFromJson(json);

  @override
  final int version;
  @override
  final bool enabled;
  final List<SiteThemeDefinition> _themes;
  @override
  List<SiteThemeDefinition> get themes {
    if (_themes is EqualUnmodifiableListView) return _themes;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_themes);
  }

  @override
  final SiteThemePrepublish? prepublish;
  @override
  final String? publishedAt;

  @override
  String toString() {
    return 'SiteThemeConfig(version: $version, enabled: $enabled, themes: $themes, prepublish: $prepublish, publishedAt: $publishedAt)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$SiteThemeConfigImpl &&
            (identical(other.version, version) || other.version == version) &&
            (identical(other.enabled, enabled) || other.enabled == enabled) &&
            const DeepCollectionEquality().equals(other._themes, _themes) &&
            (identical(other.prepublish, prepublish) ||
                other.prepublish == prepublish) &&
            (identical(other.publishedAt, publishedAt) ||
                other.publishedAt == publishedAt));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
    runtimeType,
    version,
    enabled,
    const DeepCollectionEquality().hash(_themes),
    prepublish,
    publishedAt,
  );

  /// Create a copy of SiteThemeConfig
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$SiteThemeConfigImplCopyWith<_$SiteThemeConfigImpl> get copyWith =>
      __$$SiteThemeConfigImplCopyWithImpl<_$SiteThemeConfigImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$SiteThemeConfigImplToJson(this);
  }
}

abstract class _SiteThemeConfig implements SiteThemeConfig {
  const factory _SiteThemeConfig({
    required final int version,
    required final bool enabled,
    required final List<SiteThemeDefinition> themes,
    final SiteThemePrepublish? prepublish,
    final String? publishedAt,
  }) = _$SiteThemeConfigImpl;

  factory _SiteThemeConfig.fromJson(Map<String, dynamic> json) =
      _$SiteThemeConfigImpl.fromJson;

  @override
  int get version;
  @override
  bool get enabled;
  @override
  List<SiteThemeDefinition> get themes;
  @override
  SiteThemePrepublish? get prepublish;
  @override
  String? get publishedAt;

  /// Create a copy of SiteThemeConfig
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$SiteThemeConfigImplCopyWith<_$SiteThemeConfigImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

SiteThemeDefinition _$SiteThemeDefinitionFromJson(Map<String, dynamic> json) {
  return _SiteThemeDefinition.fromJson(json);
}

/// @nodoc
mixin _$SiteThemeDefinition {
  String get name => throw _privateConstructorUsedError;
  String get label => throw _privateConstructorUsedError;
  String get colorScheme => throw _privateConstructorUsedError;
  Map<String, String> get tokens => throw _privateConstructorUsedError;

  /// Serializes this SiteThemeDefinition to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of SiteThemeDefinition
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $SiteThemeDefinitionCopyWith<SiteThemeDefinition> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $SiteThemeDefinitionCopyWith<$Res> {
  factory $SiteThemeDefinitionCopyWith(
    SiteThemeDefinition value,
    $Res Function(SiteThemeDefinition) then,
  ) = _$SiteThemeDefinitionCopyWithImpl<$Res, SiteThemeDefinition>;
  @useResult
  $Res call({
    String name,
    String label,
    String colorScheme,
    Map<String, String> tokens,
  });
}

/// @nodoc
class _$SiteThemeDefinitionCopyWithImpl<$Res, $Val extends SiteThemeDefinition>
    implements $SiteThemeDefinitionCopyWith<$Res> {
  _$SiteThemeDefinitionCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of SiteThemeDefinition
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? name = null,
    Object? label = null,
    Object? colorScheme = null,
    Object? tokens = null,
  }) {
    return _then(
      _value.copyWith(
            name: null == name
                ? _value.name
                : name // ignore: cast_nullable_to_non_nullable
                      as String,
            label: null == label
                ? _value.label
                : label // ignore: cast_nullable_to_non_nullable
                      as String,
            colorScheme: null == colorScheme
                ? _value.colorScheme
                : colorScheme // ignore: cast_nullable_to_non_nullable
                      as String,
            tokens: null == tokens
                ? _value.tokens
                : tokens // ignore: cast_nullable_to_non_nullable
                      as Map<String, String>,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$SiteThemeDefinitionImplCopyWith<$Res>
    implements $SiteThemeDefinitionCopyWith<$Res> {
  factory _$$SiteThemeDefinitionImplCopyWith(
    _$SiteThemeDefinitionImpl value,
    $Res Function(_$SiteThemeDefinitionImpl) then,
  ) = __$$SiteThemeDefinitionImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    String name,
    String label,
    String colorScheme,
    Map<String, String> tokens,
  });
}

/// @nodoc
class __$$SiteThemeDefinitionImplCopyWithImpl<$Res>
    extends _$SiteThemeDefinitionCopyWithImpl<$Res, _$SiteThemeDefinitionImpl>
    implements _$$SiteThemeDefinitionImplCopyWith<$Res> {
  __$$SiteThemeDefinitionImplCopyWithImpl(
    _$SiteThemeDefinitionImpl _value,
    $Res Function(_$SiteThemeDefinitionImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of SiteThemeDefinition
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? name = null,
    Object? label = null,
    Object? colorScheme = null,
    Object? tokens = null,
  }) {
    return _then(
      _$SiteThemeDefinitionImpl(
        name: null == name
            ? _value.name
            : name // ignore: cast_nullable_to_non_nullable
                  as String,
        label: null == label
            ? _value.label
            : label // ignore: cast_nullable_to_non_nullable
                  as String,
        colorScheme: null == colorScheme
            ? _value.colorScheme
            : colorScheme // ignore: cast_nullable_to_non_nullable
                  as String,
        tokens: null == tokens
            ? _value._tokens
            : tokens // ignore: cast_nullable_to_non_nullable
                  as Map<String, String>,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$SiteThemeDefinitionImpl implements _SiteThemeDefinition {
  const _$SiteThemeDefinitionImpl({
    required this.name,
    required this.label,
    required this.colorScheme,
    required final Map<String, String> tokens,
  }) : _tokens = tokens;

  factory _$SiteThemeDefinitionImpl.fromJson(Map<String, dynamic> json) =>
      _$$SiteThemeDefinitionImplFromJson(json);

  @override
  final String name;
  @override
  final String label;
  @override
  final String colorScheme;
  final Map<String, String> _tokens;
  @override
  Map<String, String> get tokens {
    if (_tokens is EqualUnmodifiableMapView) return _tokens;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableMapView(_tokens);
  }

  @override
  String toString() {
    return 'SiteThemeDefinition(name: $name, label: $label, colorScheme: $colorScheme, tokens: $tokens)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$SiteThemeDefinitionImpl &&
            (identical(other.name, name) || other.name == name) &&
            (identical(other.label, label) || other.label == label) &&
            (identical(other.colorScheme, colorScheme) ||
                other.colorScheme == colorScheme) &&
            const DeepCollectionEquality().equals(other._tokens, _tokens));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
    runtimeType,
    name,
    label,
    colorScheme,
    const DeepCollectionEquality().hash(_tokens),
  );

  /// Create a copy of SiteThemeDefinition
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$SiteThemeDefinitionImplCopyWith<_$SiteThemeDefinitionImpl> get copyWith =>
      __$$SiteThemeDefinitionImplCopyWithImpl<_$SiteThemeDefinitionImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$SiteThemeDefinitionImplToJson(this);
  }
}

abstract class _SiteThemeDefinition implements SiteThemeDefinition {
  const factory _SiteThemeDefinition({
    required final String name,
    required final String label,
    required final String colorScheme,
    required final Map<String, String> tokens,
  }) = _$SiteThemeDefinitionImpl;

  factory _SiteThemeDefinition.fromJson(Map<String, dynamic> json) =
      _$SiteThemeDefinitionImpl.fromJson;

  @override
  String get name;
  @override
  String get label;
  @override
  String get colorScheme;
  @override
  Map<String, String> get tokens;

  /// Create a copy of SiteThemeDefinition
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$SiteThemeDefinitionImplCopyWith<_$SiteThemeDefinitionImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

SiteThemePrepublish _$SiteThemePrepublishFromJson(Map<String, dynamic> json) {
  return _SiteThemePrepublish.fromJson(json);
}

/// @nodoc
mixin _$SiteThemePrepublish {
  bool get enabled => throw _privateConstructorUsedError;
  List<SiteThemeDefinition> get themes => throw _privateConstructorUsedError;
  String? get updatedAt => throw _privateConstructorUsedError;

  /// Serializes this SiteThemePrepublish to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of SiteThemePrepublish
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $SiteThemePrepublishCopyWith<SiteThemePrepublish> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $SiteThemePrepublishCopyWith<$Res> {
  factory $SiteThemePrepublishCopyWith(
    SiteThemePrepublish value,
    $Res Function(SiteThemePrepublish) then,
  ) = _$SiteThemePrepublishCopyWithImpl<$Res, SiteThemePrepublish>;
  @useResult
  $Res call({
    bool enabled,
    List<SiteThemeDefinition> themes,
    String? updatedAt,
  });
}

/// @nodoc
class _$SiteThemePrepublishCopyWithImpl<$Res, $Val extends SiteThemePrepublish>
    implements $SiteThemePrepublishCopyWith<$Res> {
  _$SiteThemePrepublishCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of SiteThemePrepublish
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? enabled = null,
    Object? themes = null,
    Object? updatedAt = freezed,
  }) {
    return _then(
      _value.copyWith(
            enabled: null == enabled
                ? _value.enabled
                : enabled // ignore: cast_nullable_to_non_nullable
                      as bool,
            themes: null == themes
                ? _value.themes
                : themes // ignore: cast_nullable_to_non_nullable
                      as List<SiteThemeDefinition>,
            updatedAt: freezed == updatedAt
                ? _value.updatedAt
                : updatedAt // ignore: cast_nullable_to_non_nullable
                      as String?,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$SiteThemePrepublishImplCopyWith<$Res>
    implements $SiteThemePrepublishCopyWith<$Res> {
  factory _$$SiteThemePrepublishImplCopyWith(
    _$SiteThemePrepublishImpl value,
    $Res Function(_$SiteThemePrepublishImpl) then,
  ) = __$$SiteThemePrepublishImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    bool enabled,
    List<SiteThemeDefinition> themes,
    String? updatedAt,
  });
}

/// @nodoc
class __$$SiteThemePrepublishImplCopyWithImpl<$Res>
    extends _$SiteThemePrepublishCopyWithImpl<$Res, _$SiteThemePrepublishImpl>
    implements _$$SiteThemePrepublishImplCopyWith<$Res> {
  __$$SiteThemePrepublishImplCopyWithImpl(
    _$SiteThemePrepublishImpl _value,
    $Res Function(_$SiteThemePrepublishImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of SiteThemePrepublish
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? enabled = null,
    Object? themes = null,
    Object? updatedAt = freezed,
  }) {
    return _then(
      _$SiteThemePrepublishImpl(
        enabled: null == enabled
            ? _value.enabled
            : enabled // ignore: cast_nullable_to_non_nullable
                  as bool,
        themes: null == themes
            ? _value._themes
            : themes // ignore: cast_nullable_to_non_nullable
                  as List<SiteThemeDefinition>,
        updatedAt: freezed == updatedAt
            ? _value.updatedAt
            : updatedAt // ignore: cast_nullable_to_non_nullable
                  as String?,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$SiteThemePrepublishImpl implements _SiteThemePrepublish {
  const _$SiteThemePrepublishImpl({
    required this.enabled,
    required final List<SiteThemeDefinition> themes,
    this.updatedAt,
  }) : _themes = themes;

  factory _$SiteThemePrepublishImpl.fromJson(Map<String, dynamic> json) =>
      _$$SiteThemePrepublishImplFromJson(json);

  @override
  final bool enabled;
  final List<SiteThemeDefinition> _themes;
  @override
  List<SiteThemeDefinition> get themes {
    if (_themes is EqualUnmodifiableListView) return _themes;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_themes);
  }

  @override
  final String? updatedAt;

  @override
  String toString() {
    return 'SiteThemePrepublish(enabled: $enabled, themes: $themes, updatedAt: $updatedAt)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$SiteThemePrepublishImpl &&
            (identical(other.enabled, enabled) || other.enabled == enabled) &&
            const DeepCollectionEquality().equals(other._themes, _themes) &&
            (identical(other.updatedAt, updatedAt) ||
                other.updatedAt == updatedAt));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
    runtimeType,
    enabled,
    const DeepCollectionEquality().hash(_themes),
    updatedAt,
  );

  /// Create a copy of SiteThemePrepublish
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$SiteThemePrepublishImplCopyWith<_$SiteThemePrepublishImpl> get copyWith =>
      __$$SiteThemePrepublishImplCopyWithImpl<_$SiteThemePrepublishImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$SiteThemePrepublishImplToJson(this);
  }
}

abstract class _SiteThemePrepublish implements SiteThemePrepublish {
  const factory _SiteThemePrepublish({
    required final bool enabled,
    required final List<SiteThemeDefinition> themes,
    final String? updatedAt,
  }) = _$SiteThemePrepublishImpl;

  factory _SiteThemePrepublish.fromJson(Map<String, dynamic> json) =
      _$SiteThemePrepublishImpl.fromJson;

  @override
  bool get enabled;
  @override
  List<SiteThemeDefinition> get themes;
  @override
  String? get updatedAt;

  /// Create a copy of SiteThemePrepublish
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$SiteThemePrepublishImplCopyWith<_$SiteThemePrepublishImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

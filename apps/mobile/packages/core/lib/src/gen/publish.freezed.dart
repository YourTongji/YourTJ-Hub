// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'publish.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

T _$identity<T>(T value) => value;

final _privateConstructorUsedError = UnsupportedError(
  'It seems like you constructed your class using `MyClass._()`. This constructor is only meant to be used by freezed and you are not supposed to need it nor use it.\nPlease check the documentation here for more information: https://github.com/rrousselGit/freezed#adding-getters-and-methods-to-our-models',
);

PublishCategoryPayload _$PublishCategoryPayloadFromJson(
  Map<String, dynamic> json,
) {
  return _PublishCategoryPayload.fromJson(json);
}

/// @nodoc
mixin _$PublishCategoryPayload {
  int get id => throw _privateConstructorUsedError;
  String get name => throw _privateConstructorUsedError;
  String get color => throw _privateConstructorUsedError;

  /// Serializes this PublishCategoryPayload to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of PublishCategoryPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $PublishCategoryPayloadCopyWith<PublishCategoryPayload> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $PublishCategoryPayloadCopyWith<$Res> {
  factory $PublishCategoryPayloadCopyWith(
    PublishCategoryPayload value,
    $Res Function(PublishCategoryPayload) then,
  ) = _$PublishCategoryPayloadCopyWithImpl<$Res, PublishCategoryPayload>;
  @useResult
  $Res call({int id, String name, String color});
}

/// @nodoc
class _$PublishCategoryPayloadCopyWithImpl<
  $Res,
  $Val extends PublishCategoryPayload
>
    implements $PublishCategoryPayloadCopyWith<$Res> {
  _$PublishCategoryPayloadCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of PublishCategoryPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({Object? id = null, Object? name = null, Object? color = null}) {
    return _then(
      _value.copyWith(
            id: null == id
                ? _value.id
                : id // ignore: cast_nullable_to_non_nullable
                      as int,
            name: null == name
                ? _value.name
                : name // ignore: cast_nullable_to_non_nullable
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
abstract class _$$PublishCategoryPayloadImplCopyWith<$Res>
    implements $PublishCategoryPayloadCopyWith<$Res> {
  factory _$$PublishCategoryPayloadImplCopyWith(
    _$PublishCategoryPayloadImpl value,
    $Res Function(_$PublishCategoryPayloadImpl) then,
  ) = __$$PublishCategoryPayloadImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({int id, String name, String color});
}

/// @nodoc
class __$$PublishCategoryPayloadImplCopyWithImpl<$Res>
    extends
        _$PublishCategoryPayloadCopyWithImpl<$Res, _$PublishCategoryPayloadImpl>
    implements _$$PublishCategoryPayloadImplCopyWith<$Res> {
  __$$PublishCategoryPayloadImplCopyWithImpl(
    _$PublishCategoryPayloadImpl _value,
    $Res Function(_$PublishCategoryPayloadImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of PublishCategoryPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({Object? id = null, Object? name = null, Object? color = null}) {
    return _then(
      _$PublishCategoryPayloadImpl(
        id: null == id
            ? _value.id
            : id // ignore: cast_nullable_to_non_nullable
                  as int,
        name: null == name
            ? _value.name
            : name // ignore: cast_nullable_to_non_nullable
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
class _$PublishCategoryPayloadImpl implements _PublishCategoryPayload {
  const _$PublishCategoryPayloadImpl({
    required this.id,
    required this.name,
    required this.color,
  });

  factory _$PublishCategoryPayloadImpl.fromJson(Map<String, dynamic> json) =>
      _$$PublishCategoryPayloadImplFromJson(json);

  @override
  final int id;
  @override
  final String name;
  @override
  final String color;

  @override
  String toString() {
    return 'PublishCategoryPayload(id: $id, name: $name, color: $color)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$PublishCategoryPayloadImpl &&
            (identical(other.id, id) || other.id == id) &&
            (identical(other.name, name) || other.name == name) &&
            (identical(other.color, color) || other.color == color));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(runtimeType, id, name, color);

  /// Create a copy of PublishCategoryPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$PublishCategoryPayloadImplCopyWith<_$PublishCategoryPayloadImpl>
  get copyWith =>
      __$$PublishCategoryPayloadImplCopyWithImpl<_$PublishCategoryPayloadImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$PublishCategoryPayloadImplToJson(this);
  }
}

abstract class _PublishCategoryPayload implements PublishCategoryPayload {
  const factory _PublishCategoryPayload({
    required final int id,
    required final String name,
    required final String color,
  }) = _$PublishCategoryPayloadImpl;

  factory _PublishCategoryPayload.fromJson(Map<String, dynamic> json) =
      _$PublishCategoryPayloadImpl.fromJson;

  @override
  int get id;
  @override
  String get name;
  @override
  String get color;

  /// Create a copy of PublishCategoryPayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$PublishCategoryPayloadImplCopyWith<_$PublishCategoryPayloadImpl>
  get copyWith => throw _privateConstructorUsedError;
}

PublishTopicPayload _$PublishTopicPayloadFromJson(Map<String, dynamic> json) {
  return _PublishTopicPayload.fromJson(json);
}

/// @nodoc
mixin _$PublishTopicPayload {
  String get title => throw _privateConstructorUsedError;
  String get content => throw _privateConstructorUsedError;
  List<int> get categoryIds => throw _privateConstructorUsedError;
  int get topicStatus => throw _privateConstructorUsedError;

  /// Serializes this PublishTopicPayload to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of PublishTopicPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $PublishTopicPayloadCopyWith<PublishTopicPayload> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $PublishTopicPayloadCopyWith<$Res> {
  factory $PublishTopicPayloadCopyWith(
    PublishTopicPayload value,
    $Res Function(PublishTopicPayload) then,
  ) = _$PublishTopicPayloadCopyWithImpl<$Res, PublishTopicPayload>;
  @useResult
  $Res call({
    String title,
    String content,
    List<int> categoryIds,
    int topicStatus,
  });
}

/// @nodoc
class _$PublishTopicPayloadCopyWithImpl<$Res, $Val extends PublishTopicPayload>
    implements $PublishTopicPayloadCopyWith<$Res> {
  _$PublishTopicPayloadCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of PublishTopicPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? title = null,
    Object? content = null,
    Object? categoryIds = null,
    Object? topicStatus = null,
  }) {
    return _then(
      _value.copyWith(
            title: null == title
                ? _value.title
                : title // ignore: cast_nullable_to_non_nullable
                      as String,
            content: null == content
                ? _value.content
                : content // ignore: cast_nullable_to_non_nullable
                      as String,
            categoryIds: null == categoryIds
                ? _value.categoryIds
                : categoryIds // ignore: cast_nullable_to_non_nullable
                      as List<int>,
            topicStatus: null == topicStatus
                ? _value.topicStatus
                : topicStatus // ignore: cast_nullable_to_non_nullable
                      as int,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$PublishTopicPayloadImplCopyWith<$Res>
    implements $PublishTopicPayloadCopyWith<$Res> {
  factory _$$PublishTopicPayloadImplCopyWith(
    _$PublishTopicPayloadImpl value,
    $Res Function(_$PublishTopicPayloadImpl) then,
  ) = __$$PublishTopicPayloadImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    String title,
    String content,
    List<int> categoryIds,
    int topicStatus,
  });
}

/// @nodoc
class __$$PublishTopicPayloadImplCopyWithImpl<$Res>
    extends _$PublishTopicPayloadCopyWithImpl<$Res, _$PublishTopicPayloadImpl>
    implements _$$PublishTopicPayloadImplCopyWith<$Res> {
  __$$PublishTopicPayloadImplCopyWithImpl(
    _$PublishTopicPayloadImpl _value,
    $Res Function(_$PublishTopicPayloadImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of PublishTopicPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? title = null,
    Object? content = null,
    Object? categoryIds = null,
    Object? topicStatus = null,
  }) {
    return _then(
      _$PublishTopicPayloadImpl(
        title: null == title
            ? _value.title
            : title // ignore: cast_nullable_to_non_nullable
                  as String,
        content: null == content
            ? _value.content
            : content // ignore: cast_nullable_to_non_nullable
                  as String,
        categoryIds: null == categoryIds
            ? _value._categoryIds
            : categoryIds // ignore: cast_nullable_to_non_nullable
                  as List<int>,
        topicStatus: null == topicStatus
            ? _value.topicStatus
            : topicStatus // ignore: cast_nullable_to_non_nullable
                  as int,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$PublishTopicPayloadImpl implements _PublishTopicPayload {
  const _$PublishTopicPayloadImpl({
    required this.title,
    required this.content,
    required final List<int> categoryIds,
    required this.topicStatus,
  }) : _categoryIds = categoryIds;

  factory _$PublishTopicPayloadImpl.fromJson(Map<String, dynamic> json) =>
      _$$PublishTopicPayloadImplFromJson(json);

  @override
  final String title;
  @override
  final String content;
  final List<int> _categoryIds;
  @override
  List<int> get categoryIds {
    if (_categoryIds is EqualUnmodifiableListView) return _categoryIds;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_categoryIds);
  }

  @override
  final int topicStatus;

  @override
  String toString() {
    return 'PublishTopicPayload(title: $title, content: $content, categoryIds: $categoryIds, topicStatus: $topicStatus)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$PublishTopicPayloadImpl &&
            (identical(other.title, title) || other.title == title) &&
            (identical(other.content, content) || other.content == content) &&
            const DeepCollectionEquality().equals(
              other._categoryIds,
              _categoryIds,
            ) &&
            (identical(other.topicStatus, topicStatus) ||
                other.topicStatus == topicStatus));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
    runtimeType,
    title,
    content,
    const DeepCollectionEquality().hash(_categoryIds),
    topicStatus,
  );

  /// Create a copy of PublishTopicPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$PublishTopicPayloadImplCopyWith<_$PublishTopicPayloadImpl> get copyWith =>
      __$$PublishTopicPayloadImplCopyWithImpl<_$PublishTopicPayloadImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$PublishTopicPayloadImplToJson(this);
  }
}

abstract class _PublishTopicPayload implements PublishTopicPayload {
  const factory _PublishTopicPayload({
    required final String title,
    required final String content,
    required final List<int> categoryIds,
    required final int topicStatus,
  }) = _$PublishTopicPayloadImpl;

  factory _PublishTopicPayload.fromJson(Map<String, dynamic> json) =
      _$PublishTopicPayloadImpl.fromJson;

  @override
  String get title;
  @override
  String get content;
  @override
  List<int> get categoryIds;
  @override
  int get topicStatus;

  /// Create a copy of PublishTopicPayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$PublishTopicPayloadImplCopyWith<_$PublishTopicPayloadImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

PublishPageProps _$PublishPagePropsFromJson(Map<String, dynamic> json) {
  return _PublishPageProps.fromJson(json);
}

/// @nodoc
mixin _$PublishPageProps {
  int get topicId => throw _privateConstructorUsedError;
  bool get isEditing => throw _privateConstructorUsedError;
  List<PublishCategoryPayload> get categories =>
      throw _privateConstructorUsedError;
  PublishTopicPayload get topic => throw _privateConstructorUsedError;

  /// Serializes this PublishPageProps to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of PublishPageProps
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $PublishPagePropsCopyWith<PublishPageProps> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $PublishPagePropsCopyWith<$Res> {
  factory $PublishPagePropsCopyWith(
    PublishPageProps value,
    $Res Function(PublishPageProps) then,
  ) = _$PublishPagePropsCopyWithImpl<$Res, PublishPageProps>;
  @useResult
  $Res call({
    int topicId,
    bool isEditing,
    List<PublishCategoryPayload> categories,
    PublishTopicPayload topic,
  });

  $PublishTopicPayloadCopyWith<$Res> get topic;
}

/// @nodoc
class _$PublishPagePropsCopyWithImpl<$Res, $Val extends PublishPageProps>
    implements $PublishPagePropsCopyWith<$Res> {
  _$PublishPagePropsCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of PublishPageProps
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? topicId = null,
    Object? isEditing = null,
    Object? categories = null,
    Object? topic = null,
  }) {
    return _then(
      _value.copyWith(
            topicId: null == topicId
                ? _value.topicId
                : topicId // ignore: cast_nullable_to_non_nullable
                      as int,
            isEditing: null == isEditing
                ? _value.isEditing
                : isEditing // ignore: cast_nullable_to_non_nullable
                      as bool,
            categories: null == categories
                ? _value.categories
                : categories // ignore: cast_nullable_to_non_nullable
                      as List<PublishCategoryPayload>,
            topic: null == topic
                ? _value.topic
                : topic // ignore: cast_nullable_to_non_nullable
                      as PublishTopicPayload,
          )
          as $Val,
    );
  }

  /// Create a copy of PublishPageProps
  /// with the given fields replaced by the non-null parameter values.
  @override
  @pragma('vm:prefer-inline')
  $PublishTopicPayloadCopyWith<$Res> get topic {
    return $PublishTopicPayloadCopyWith<$Res>(_value.topic, (value) {
      return _then(_value.copyWith(topic: value) as $Val);
    });
  }
}

/// @nodoc
abstract class _$$PublishPagePropsImplCopyWith<$Res>
    implements $PublishPagePropsCopyWith<$Res> {
  factory _$$PublishPagePropsImplCopyWith(
    _$PublishPagePropsImpl value,
    $Res Function(_$PublishPagePropsImpl) then,
  ) = __$$PublishPagePropsImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    int topicId,
    bool isEditing,
    List<PublishCategoryPayload> categories,
    PublishTopicPayload topic,
  });

  @override
  $PublishTopicPayloadCopyWith<$Res> get topic;
}

/// @nodoc
class __$$PublishPagePropsImplCopyWithImpl<$Res>
    extends _$PublishPagePropsCopyWithImpl<$Res, _$PublishPagePropsImpl>
    implements _$$PublishPagePropsImplCopyWith<$Res> {
  __$$PublishPagePropsImplCopyWithImpl(
    _$PublishPagePropsImpl _value,
    $Res Function(_$PublishPagePropsImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of PublishPageProps
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? topicId = null,
    Object? isEditing = null,
    Object? categories = null,
    Object? topic = null,
  }) {
    return _then(
      _$PublishPagePropsImpl(
        topicId: null == topicId
            ? _value.topicId
            : topicId // ignore: cast_nullable_to_non_nullable
                  as int,
        isEditing: null == isEditing
            ? _value.isEditing
            : isEditing // ignore: cast_nullable_to_non_nullable
                  as bool,
        categories: null == categories
            ? _value._categories
            : categories // ignore: cast_nullable_to_non_nullable
                  as List<PublishCategoryPayload>,
        topic: null == topic
            ? _value.topic
            : topic // ignore: cast_nullable_to_non_nullable
                  as PublishTopicPayload,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$PublishPagePropsImpl implements _PublishPageProps {
  const _$PublishPagePropsImpl({
    required this.topicId,
    required this.isEditing,
    required final List<PublishCategoryPayload> categories,
    required this.topic,
  }) : _categories = categories;

  factory _$PublishPagePropsImpl.fromJson(Map<String, dynamic> json) =>
      _$$PublishPagePropsImplFromJson(json);

  @override
  final int topicId;
  @override
  final bool isEditing;
  final List<PublishCategoryPayload> _categories;
  @override
  List<PublishCategoryPayload> get categories {
    if (_categories is EqualUnmodifiableListView) return _categories;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_categories);
  }

  @override
  final PublishTopicPayload topic;

  @override
  String toString() {
    return 'PublishPageProps(topicId: $topicId, isEditing: $isEditing, categories: $categories, topic: $topic)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$PublishPagePropsImpl &&
            (identical(other.topicId, topicId) || other.topicId == topicId) &&
            (identical(other.isEditing, isEditing) ||
                other.isEditing == isEditing) &&
            const DeepCollectionEquality().equals(
              other._categories,
              _categories,
            ) &&
            (identical(other.topic, topic) || other.topic == topic));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
    runtimeType,
    topicId,
    isEditing,
    const DeepCollectionEquality().hash(_categories),
    topic,
  );

  /// Create a copy of PublishPageProps
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$PublishPagePropsImplCopyWith<_$PublishPagePropsImpl> get copyWith =>
      __$$PublishPagePropsImplCopyWithImpl<_$PublishPagePropsImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$PublishPagePropsImplToJson(this);
  }
}

abstract class _PublishPageProps implements PublishPageProps {
  const factory _PublishPageProps({
    required final int topicId,
    required final bool isEditing,
    required final List<PublishCategoryPayload> categories,
    required final PublishTopicPayload topic,
  }) = _$PublishPagePropsImpl;

  factory _PublishPageProps.fromJson(Map<String, dynamic> json) =
      _$PublishPagePropsImpl.fromJson;

  @override
  int get topicId;
  @override
  bool get isEditing;
  @override
  List<PublishCategoryPayload> get categories;
  @override
  PublishTopicPayload get topic;

  /// Create a copy of PublishPageProps
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$PublishPagePropsImplCopyWith<_$PublishPagePropsImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

CategoryHeaderPayload _$CategoryHeaderPayloadFromJson(
  Map<String, dynamic> json,
) {
  return _CategoryHeaderPayload.fromJson(json);
}

/// @nodoc
mixin _$CategoryHeaderPayload {
  int get id => throw _privateConstructorUsedError;
  String get name => throw _privateConstructorUsedError;
  String get description => throw _privateConstructorUsedError;
  String get icon => throw _privateConstructorUsedError;
  String get color => throw _privateConstructorUsedError;
  String get url => throw _privateConstructorUsedError;

  /// Serializes this CategoryHeaderPayload to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of CategoryHeaderPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $CategoryHeaderPayloadCopyWith<CategoryHeaderPayload> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $CategoryHeaderPayloadCopyWith<$Res> {
  factory $CategoryHeaderPayloadCopyWith(
    CategoryHeaderPayload value,
    $Res Function(CategoryHeaderPayload) then,
  ) = _$CategoryHeaderPayloadCopyWithImpl<$Res, CategoryHeaderPayload>;
  @useResult
  $Res call({
    int id,
    String name,
    String description,
    String icon,
    String color,
    String url,
  });
}

/// @nodoc
class _$CategoryHeaderPayloadCopyWithImpl<
  $Res,
  $Val extends CategoryHeaderPayload
>
    implements $CategoryHeaderPayloadCopyWith<$Res> {
  _$CategoryHeaderPayloadCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of CategoryHeaderPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = null,
    Object? name = null,
    Object? description = null,
    Object? icon = null,
    Object? color = null,
    Object? url = null,
  }) {
    return _then(
      _value.copyWith(
            id: null == id
                ? _value.id
                : id // ignore: cast_nullable_to_non_nullable
                      as int,
            name: null == name
                ? _value.name
                : name // ignore: cast_nullable_to_non_nullable
                      as String,
            description: null == description
                ? _value.description
                : description // ignore: cast_nullable_to_non_nullable
                      as String,
            icon: null == icon
                ? _value.icon
                : icon // ignore: cast_nullable_to_non_nullable
                      as String,
            color: null == color
                ? _value.color
                : color // ignore: cast_nullable_to_non_nullable
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
abstract class _$$CategoryHeaderPayloadImplCopyWith<$Res>
    implements $CategoryHeaderPayloadCopyWith<$Res> {
  factory _$$CategoryHeaderPayloadImplCopyWith(
    _$CategoryHeaderPayloadImpl value,
    $Res Function(_$CategoryHeaderPayloadImpl) then,
  ) = __$$CategoryHeaderPayloadImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    int id,
    String name,
    String description,
    String icon,
    String color,
    String url,
  });
}

/// @nodoc
class __$$CategoryHeaderPayloadImplCopyWithImpl<$Res>
    extends
        _$CategoryHeaderPayloadCopyWithImpl<$Res, _$CategoryHeaderPayloadImpl>
    implements _$$CategoryHeaderPayloadImplCopyWith<$Res> {
  __$$CategoryHeaderPayloadImplCopyWithImpl(
    _$CategoryHeaderPayloadImpl _value,
    $Res Function(_$CategoryHeaderPayloadImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of CategoryHeaderPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = null,
    Object? name = null,
    Object? description = null,
    Object? icon = null,
    Object? color = null,
    Object? url = null,
  }) {
    return _then(
      _$CategoryHeaderPayloadImpl(
        id: null == id
            ? _value.id
            : id // ignore: cast_nullable_to_non_nullable
                  as int,
        name: null == name
            ? _value.name
            : name // ignore: cast_nullable_to_non_nullable
                  as String,
        description: null == description
            ? _value.description
            : description // ignore: cast_nullable_to_non_nullable
                  as String,
        icon: null == icon
            ? _value.icon
            : icon // ignore: cast_nullable_to_non_nullable
                  as String,
        color: null == color
            ? _value.color
            : color // ignore: cast_nullable_to_non_nullable
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
class _$CategoryHeaderPayloadImpl implements _CategoryHeaderPayload {
  const _$CategoryHeaderPayloadImpl({
    required this.id,
    required this.name,
    required this.description,
    required this.icon,
    required this.color,
    required this.url,
  });

  factory _$CategoryHeaderPayloadImpl.fromJson(Map<String, dynamic> json) =>
      _$$CategoryHeaderPayloadImplFromJson(json);

  @override
  final int id;
  @override
  final String name;
  @override
  final String description;
  @override
  final String icon;
  @override
  final String color;
  @override
  final String url;

  @override
  String toString() {
    return 'CategoryHeaderPayload(id: $id, name: $name, description: $description, icon: $icon, color: $color, url: $url)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$CategoryHeaderPayloadImpl &&
            (identical(other.id, id) || other.id == id) &&
            (identical(other.name, name) || other.name == name) &&
            (identical(other.description, description) ||
                other.description == description) &&
            (identical(other.icon, icon) || other.icon == icon) &&
            (identical(other.color, color) || other.color == color) &&
            (identical(other.url, url) || other.url == url));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode =>
      Object.hash(runtimeType, id, name, description, icon, color, url);

  /// Create a copy of CategoryHeaderPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$CategoryHeaderPayloadImplCopyWith<_$CategoryHeaderPayloadImpl>
  get copyWith =>
      __$$CategoryHeaderPayloadImplCopyWithImpl<_$CategoryHeaderPayloadImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$CategoryHeaderPayloadImplToJson(this);
  }
}

abstract class _CategoryHeaderPayload implements CategoryHeaderPayload {
  const factory _CategoryHeaderPayload({
    required final int id,
    required final String name,
    required final String description,
    required final String icon,
    required final String color,
    required final String url,
  }) = _$CategoryHeaderPayloadImpl;

  factory _CategoryHeaderPayload.fromJson(Map<String, dynamic> json) =
      _$CategoryHeaderPayloadImpl.fromJson;

  @override
  int get id;
  @override
  String get name;
  @override
  String get description;
  @override
  String get icon;
  @override
  String get color;
  @override
  String get url;

  /// Create a copy of CategoryHeaderPayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$CategoryHeaderPayloadImplCopyWith<_$CategoryHeaderPayloadImpl>
  get copyWith => throw _privateConstructorUsedError;
}

CategoryPageProps _$CategoryPagePropsFromJson(Map<String, dynamic> json) {
  return _CategoryPageProps.fromJson(json);
}

/// @nodoc
mixin _$CategoryPageProps {
  CategoryHeaderPayload get category => throw _privateConstructorUsedError;
  String get sort => throw _privateConstructorUsedError;
  List<TabItemPayload> get tabs => throw _privateConstructorUsedError;
  List<TopicPayload> get topics => throw _privateConstructorUsedError;
  PaginationPayload get pagination => throw _privateConstructorUsedError;

  /// Serializes this CategoryPageProps to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of CategoryPageProps
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $CategoryPagePropsCopyWith<CategoryPageProps> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $CategoryPagePropsCopyWith<$Res> {
  factory $CategoryPagePropsCopyWith(
    CategoryPageProps value,
    $Res Function(CategoryPageProps) then,
  ) = _$CategoryPagePropsCopyWithImpl<$Res, CategoryPageProps>;
  @useResult
  $Res call({
    CategoryHeaderPayload category,
    String sort,
    List<TabItemPayload> tabs,
    List<TopicPayload> topics,
    PaginationPayload pagination,
  });

  $CategoryHeaderPayloadCopyWith<$Res> get category;
  $PaginationPayloadCopyWith<$Res> get pagination;
}

/// @nodoc
class _$CategoryPagePropsCopyWithImpl<$Res, $Val extends CategoryPageProps>
    implements $CategoryPagePropsCopyWith<$Res> {
  _$CategoryPagePropsCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of CategoryPageProps
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? category = null,
    Object? sort = null,
    Object? tabs = null,
    Object? topics = null,
    Object? pagination = null,
  }) {
    return _then(
      _value.copyWith(
            category: null == category
                ? _value.category
                : category // ignore: cast_nullable_to_non_nullable
                      as CategoryHeaderPayload,
            sort: null == sort
                ? _value.sort
                : sort // ignore: cast_nullable_to_non_nullable
                      as String,
            tabs: null == tabs
                ? _value.tabs
                : tabs // ignore: cast_nullable_to_non_nullable
                      as List<TabItemPayload>,
            topics: null == topics
                ? _value.topics
                : topics // ignore: cast_nullable_to_non_nullable
                      as List<TopicPayload>,
            pagination: null == pagination
                ? _value.pagination
                : pagination // ignore: cast_nullable_to_non_nullable
                      as PaginationPayload,
          )
          as $Val,
    );
  }

  /// Create a copy of CategoryPageProps
  /// with the given fields replaced by the non-null parameter values.
  @override
  @pragma('vm:prefer-inline')
  $CategoryHeaderPayloadCopyWith<$Res> get category {
    return $CategoryHeaderPayloadCopyWith<$Res>(_value.category, (value) {
      return _then(_value.copyWith(category: value) as $Val);
    });
  }

  /// Create a copy of CategoryPageProps
  /// with the given fields replaced by the non-null parameter values.
  @override
  @pragma('vm:prefer-inline')
  $PaginationPayloadCopyWith<$Res> get pagination {
    return $PaginationPayloadCopyWith<$Res>(_value.pagination, (value) {
      return _then(_value.copyWith(pagination: value) as $Val);
    });
  }
}

/// @nodoc
abstract class _$$CategoryPagePropsImplCopyWith<$Res>
    implements $CategoryPagePropsCopyWith<$Res> {
  factory _$$CategoryPagePropsImplCopyWith(
    _$CategoryPagePropsImpl value,
    $Res Function(_$CategoryPagePropsImpl) then,
  ) = __$$CategoryPagePropsImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    CategoryHeaderPayload category,
    String sort,
    List<TabItemPayload> tabs,
    List<TopicPayload> topics,
    PaginationPayload pagination,
  });

  @override
  $CategoryHeaderPayloadCopyWith<$Res> get category;
  @override
  $PaginationPayloadCopyWith<$Res> get pagination;
}

/// @nodoc
class __$$CategoryPagePropsImplCopyWithImpl<$Res>
    extends _$CategoryPagePropsCopyWithImpl<$Res, _$CategoryPagePropsImpl>
    implements _$$CategoryPagePropsImplCopyWith<$Res> {
  __$$CategoryPagePropsImplCopyWithImpl(
    _$CategoryPagePropsImpl _value,
    $Res Function(_$CategoryPagePropsImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of CategoryPageProps
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? category = null,
    Object? sort = null,
    Object? tabs = null,
    Object? topics = null,
    Object? pagination = null,
  }) {
    return _then(
      _$CategoryPagePropsImpl(
        category: null == category
            ? _value.category
            : category // ignore: cast_nullable_to_non_nullable
                  as CategoryHeaderPayload,
        sort: null == sort
            ? _value.sort
            : sort // ignore: cast_nullable_to_non_nullable
                  as String,
        tabs: null == tabs
            ? _value._tabs
            : tabs // ignore: cast_nullable_to_non_nullable
                  as List<TabItemPayload>,
        topics: null == topics
            ? _value._topics
            : topics // ignore: cast_nullable_to_non_nullable
                  as List<TopicPayload>,
        pagination: null == pagination
            ? _value.pagination
            : pagination // ignore: cast_nullable_to_non_nullable
                  as PaginationPayload,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$CategoryPagePropsImpl implements _CategoryPageProps {
  const _$CategoryPagePropsImpl({
    required this.category,
    required this.sort,
    required final List<TabItemPayload> tabs,
    required final List<TopicPayload> topics,
    required this.pagination,
  }) : _tabs = tabs,
       _topics = topics;

  factory _$CategoryPagePropsImpl.fromJson(Map<String, dynamic> json) =>
      _$$CategoryPagePropsImplFromJson(json);

  @override
  final CategoryHeaderPayload category;
  @override
  final String sort;
  final List<TabItemPayload> _tabs;
  @override
  List<TabItemPayload> get tabs {
    if (_tabs is EqualUnmodifiableListView) return _tabs;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_tabs);
  }

  final List<TopicPayload> _topics;
  @override
  List<TopicPayload> get topics {
    if (_topics is EqualUnmodifiableListView) return _topics;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_topics);
  }

  @override
  final PaginationPayload pagination;

  @override
  String toString() {
    return 'CategoryPageProps(category: $category, sort: $sort, tabs: $tabs, topics: $topics, pagination: $pagination)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$CategoryPagePropsImpl &&
            (identical(other.category, category) ||
                other.category == category) &&
            (identical(other.sort, sort) || other.sort == sort) &&
            const DeepCollectionEquality().equals(other._tabs, _tabs) &&
            const DeepCollectionEquality().equals(other._topics, _topics) &&
            (identical(other.pagination, pagination) ||
                other.pagination == pagination));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
    runtimeType,
    category,
    sort,
    const DeepCollectionEquality().hash(_tabs),
    const DeepCollectionEquality().hash(_topics),
    pagination,
  );

  /// Create a copy of CategoryPageProps
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$CategoryPagePropsImplCopyWith<_$CategoryPagePropsImpl> get copyWith =>
      __$$CategoryPagePropsImplCopyWithImpl<_$CategoryPagePropsImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$CategoryPagePropsImplToJson(this);
  }
}

abstract class _CategoryPageProps implements CategoryPageProps {
  const factory _CategoryPageProps({
    required final CategoryHeaderPayload category,
    required final String sort,
    required final List<TabItemPayload> tabs,
    required final List<TopicPayload> topics,
    required final PaginationPayload pagination,
  }) = _$CategoryPagePropsImpl;

  factory _CategoryPageProps.fromJson(Map<String, dynamic> json) =
      _$CategoryPagePropsImpl.fromJson;

  @override
  CategoryHeaderPayload get category;
  @override
  String get sort;
  @override
  List<TabItemPayload> get tabs;
  @override
  List<TopicPayload> get topics;
  @override
  PaginationPayload get pagination;

  /// Create a copy of CategoryPageProps
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$CategoryPagePropsImplCopyWith<_$CategoryPagePropsImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

AnnouncementItemPayload _$AnnouncementItemPayloadFromJson(
  Map<String, dynamic> json,
) {
  return _AnnouncementItemPayload.fromJson(json);
}

/// @nodoc
mixin _$AnnouncementItemPayload {
  String get id => throw _privateConstructorUsedError;
  String get title => throw _privateConstructorUsedError;
  String get html => throw _privateConstructorUsedError;

  /// Serializes this AnnouncementItemPayload to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of AnnouncementItemPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $AnnouncementItemPayloadCopyWith<AnnouncementItemPayload> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $AnnouncementItemPayloadCopyWith<$Res> {
  factory $AnnouncementItemPayloadCopyWith(
    AnnouncementItemPayload value,
    $Res Function(AnnouncementItemPayload) then,
  ) = _$AnnouncementItemPayloadCopyWithImpl<$Res, AnnouncementItemPayload>;
  @useResult
  $Res call({String id, String title, String html});
}

/// @nodoc
class _$AnnouncementItemPayloadCopyWithImpl<
  $Res,
  $Val extends AnnouncementItemPayload
>
    implements $AnnouncementItemPayloadCopyWith<$Res> {
  _$AnnouncementItemPayloadCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of AnnouncementItemPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({Object? id = null, Object? title = null, Object? html = null}) {
    return _then(
      _value.copyWith(
            id: null == id
                ? _value.id
                : id // ignore: cast_nullable_to_non_nullable
                      as String,
            title: null == title
                ? _value.title
                : title // ignore: cast_nullable_to_non_nullable
                      as String,
            html: null == html
                ? _value.html
                : html // ignore: cast_nullable_to_non_nullable
                      as String,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$AnnouncementItemPayloadImplCopyWith<$Res>
    implements $AnnouncementItemPayloadCopyWith<$Res> {
  factory _$$AnnouncementItemPayloadImplCopyWith(
    _$AnnouncementItemPayloadImpl value,
    $Res Function(_$AnnouncementItemPayloadImpl) then,
  ) = __$$AnnouncementItemPayloadImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({String id, String title, String html});
}

/// @nodoc
class __$$AnnouncementItemPayloadImplCopyWithImpl<$Res>
    extends
        _$AnnouncementItemPayloadCopyWithImpl<
          $Res,
          _$AnnouncementItemPayloadImpl
        >
    implements _$$AnnouncementItemPayloadImplCopyWith<$Res> {
  __$$AnnouncementItemPayloadImplCopyWithImpl(
    _$AnnouncementItemPayloadImpl _value,
    $Res Function(_$AnnouncementItemPayloadImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of AnnouncementItemPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({Object? id = null, Object? title = null, Object? html = null}) {
    return _then(
      _$AnnouncementItemPayloadImpl(
        id: null == id
            ? _value.id
            : id // ignore: cast_nullable_to_non_nullable
                  as String,
        title: null == title
            ? _value.title
            : title // ignore: cast_nullable_to_non_nullable
                  as String,
        html: null == html
            ? _value.html
            : html // ignore: cast_nullable_to_non_nullable
                  as String,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$AnnouncementItemPayloadImpl implements _AnnouncementItemPayload {
  const _$AnnouncementItemPayloadImpl({
    required this.id,
    required this.title,
    required this.html,
  });

  factory _$AnnouncementItemPayloadImpl.fromJson(Map<String, dynamic> json) =>
      _$$AnnouncementItemPayloadImplFromJson(json);

  @override
  final String id;
  @override
  final String title;
  @override
  final String html;

  @override
  String toString() {
    return 'AnnouncementItemPayload(id: $id, title: $title, html: $html)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$AnnouncementItemPayloadImpl &&
            (identical(other.id, id) || other.id == id) &&
            (identical(other.title, title) || other.title == title) &&
            (identical(other.html, html) || other.html == html));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(runtimeType, id, title, html);

  /// Create a copy of AnnouncementItemPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$AnnouncementItemPayloadImplCopyWith<_$AnnouncementItemPayloadImpl>
  get copyWith =>
      __$$AnnouncementItemPayloadImplCopyWithImpl<
        _$AnnouncementItemPayloadImpl
      >(this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$$AnnouncementItemPayloadImplToJson(this);
  }
}

abstract class _AnnouncementItemPayload implements AnnouncementItemPayload {
  const factory _AnnouncementItemPayload({
    required final String id,
    required final String title,
    required final String html,
  }) = _$AnnouncementItemPayloadImpl;

  factory _AnnouncementItemPayload.fromJson(Map<String, dynamic> json) =
      _$AnnouncementItemPayloadImpl.fromJson;

  @override
  String get id;
  @override
  String get title;
  @override
  String get html;

  /// Create a copy of AnnouncementItemPayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$AnnouncementItemPayloadImplCopyWith<_$AnnouncementItemPayloadImpl>
  get copyWith => throw _privateConstructorUsedError;
}

AnnouncementPayload _$AnnouncementPayloadFromJson(Map<String, dynamic> json) {
  return _AnnouncementPayload.fromJson(json);
}

/// @nodoc
mixin _$AnnouncementPayload {
  bool get enabled => throw _privateConstructorUsedError;
  String get html => throw _privateConstructorUsedError;
  String? get publishedAt => throw _privateConstructorUsedError;
  List<AnnouncementItemPayload>? get items =>
      throw _privateConstructorUsedError;

  /// Serializes this AnnouncementPayload to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of AnnouncementPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $AnnouncementPayloadCopyWith<AnnouncementPayload> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $AnnouncementPayloadCopyWith<$Res> {
  factory $AnnouncementPayloadCopyWith(
    AnnouncementPayload value,
    $Res Function(AnnouncementPayload) then,
  ) = _$AnnouncementPayloadCopyWithImpl<$Res, AnnouncementPayload>;
  @useResult
  $Res call({
    bool enabled,
    String html,
    String? publishedAt,
    List<AnnouncementItemPayload>? items,
  });
}

/// @nodoc
class _$AnnouncementPayloadCopyWithImpl<$Res, $Val extends AnnouncementPayload>
    implements $AnnouncementPayloadCopyWith<$Res> {
  _$AnnouncementPayloadCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of AnnouncementPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? enabled = null,
    Object? html = null,
    Object? publishedAt = freezed,
    Object? items = freezed,
  }) {
    return _then(
      _value.copyWith(
            enabled: null == enabled
                ? _value.enabled
                : enabled // ignore: cast_nullable_to_non_nullable
                      as bool,
            html: null == html
                ? _value.html
                : html // ignore: cast_nullable_to_non_nullable
                      as String,
            publishedAt: freezed == publishedAt
                ? _value.publishedAt
                : publishedAt // ignore: cast_nullable_to_non_nullable
                      as String?,
            items: freezed == items
                ? _value.items
                : items // ignore: cast_nullable_to_non_nullable
                      as List<AnnouncementItemPayload>?,
          )
          as $Val,
    );
  }
}

/// @nodoc
abstract class _$$AnnouncementPayloadImplCopyWith<$Res>
    implements $AnnouncementPayloadCopyWith<$Res> {
  factory _$$AnnouncementPayloadImplCopyWith(
    _$AnnouncementPayloadImpl value,
    $Res Function(_$AnnouncementPayloadImpl) then,
  ) = __$$AnnouncementPayloadImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    bool enabled,
    String html,
    String? publishedAt,
    List<AnnouncementItemPayload>? items,
  });
}

/// @nodoc
class __$$AnnouncementPayloadImplCopyWithImpl<$Res>
    extends _$AnnouncementPayloadCopyWithImpl<$Res, _$AnnouncementPayloadImpl>
    implements _$$AnnouncementPayloadImplCopyWith<$Res> {
  __$$AnnouncementPayloadImplCopyWithImpl(
    _$AnnouncementPayloadImpl _value,
    $Res Function(_$AnnouncementPayloadImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of AnnouncementPayload
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? enabled = null,
    Object? html = null,
    Object? publishedAt = freezed,
    Object? items = freezed,
  }) {
    return _then(
      _$AnnouncementPayloadImpl(
        enabled: null == enabled
            ? _value.enabled
            : enabled // ignore: cast_nullable_to_non_nullable
                  as bool,
        html: null == html
            ? _value.html
            : html // ignore: cast_nullable_to_non_nullable
                  as String,
        publishedAt: freezed == publishedAt
            ? _value.publishedAt
            : publishedAt // ignore: cast_nullable_to_non_nullable
                  as String?,
        items: freezed == items
            ? _value._items
            : items // ignore: cast_nullable_to_non_nullable
                  as List<AnnouncementItemPayload>?,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$AnnouncementPayloadImpl implements _AnnouncementPayload {
  const _$AnnouncementPayloadImpl({
    required this.enabled,
    required this.html,
    this.publishedAt,
    final List<AnnouncementItemPayload>? items,
  }) : _items = items;

  factory _$AnnouncementPayloadImpl.fromJson(Map<String, dynamic> json) =>
      _$$AnnouncementPayloadImplFromJson(json);

  @override
  final bool enabled;
  @override
  final String html;
  @override
  final String? publishedAt;
  final List<AnnouncementItemPayload>? _items;
  @override
  List<AnnouncementItemPayload>? get items {
    final value = _items;
    if (value == null) return null;
    if (_items is EqualUnmodifiableListView) return _items;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(value);
  }

  @override
  String toString() {
    return 'AnnouncementPayload(enabled: $enabled, html: $html, publishedAt: $publishedAt, items: $items)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$AnnouncementPayloadImpl &&
            (identical(other.enabled, enabled) || other.enabled == enabled) &&
            (identical(other.html, html) || other.html == html) &&
            (identical(other.publishedAt, publishedAt) ||
                other.publishedAt == publishedAt) &&
            const DeepCollectionEquality().equals(other._items, _items));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
    runtimeType,
    enabled,
    html,
    publishedAt,
    const DeepCollectionEquality().hash(_items),
  );

  /// Create a copy of AnnouncementPayload
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$AnnouncementPayloadImplCopyWith<_$AnnouncementPayloadImpl> get copyWith =>
      __$$AnnouncementPayloadImplCopyWithImpl<_$AnnouncementPayloadImpl>(
        this,
        _$identity,
      );

  @override
  Map<String, dynamic> toJson() {
    return _$$AnnouncementPayloadImplToJson(this);
  }
}

abstract class _AnnouncementPayload implements AnnouncementPayload {
  const factory _AnnouncementPayload({
    required final bool enabled,
    required final String html,
    final String? publishedAt,
    final List<AnnouncementItemPayload>? items,
  }) = _$AnnouncementPayloadImpl;

  factory _AnnouncementPayload.fromJson(Map<String, dynamic> json) =
      _$AnnouncementPayloadImpl.fromJson;

  @override
  bool get enabled;
  @override
  String get html;
  @override
  String? get publishedAt;
  @override
  List<AnnouncementItemPayload>? get items;

  /// Create a copy of AnnouncementPayload
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$AnnouncementPayloadImplCopyWith<_$AnnouncementPayloadImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

HomeProps _$HomePropsFromJson(Map<String, dynamic> json) {
  return _HomeProps.fromJson(json);
}

/// @nodoc
mixin _$HomeProps {
  String get sort => throw _privateConstructorUsedError;
  List<TabItemPayload> get tabs => throw _privateConstructorUsedError;
  List<TopicPayload> get topics => throw _privateConstructorUsedError;
  PaginationPayload get pagination => throw _privateConstructorUsedError;
  AnnouncementPayload get announcement => throw _privateConstructorUsedError;

  /// Serializes this HomeProps to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of HomeProps
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $HomePropsCopyWith<HomeProps> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $HomePropsCopyWith<$Res> {
  factory $HomePropsCopyWith(HomeProps value, $Res Function(HomeProps) then) =
      _$HomePropsCopyWithImpl<$Res, HomeProps>;
  @useResult
  $Res call({
    String sort,
    List<TabItemPayload> tabs,
    List<TopicPayload> topics,
    PaginationPayload pagination,
    AnnouncementPayload announcement,
  });

  $PaginationPayloadCopyWith<$Res> get pagination;
  $AnnouncementPayloadCopyWith<$Res> get announcement;
}

/// @nodoc
class _$HomePropsCopyWithImpl<$Res, $Val extends HomeProps>
    implements $HomePropsCopyWith<$Res> {
  _$HomePropsCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of HomeProps
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? sort = null,
    Object? tabs = null,
    Object? topics = null,
    Object? pagination = null,
    Object? announcement = null,
  }) {
    return _then(
      _value.copyWith(
            sort: null == sort
                ? _value.sort
                : sort // ignore: cast_nullable_to_non_nullable
                      as String,
            tabs: null == tabs
                ? _value.tabs
                : tabs // ignore: cast_nullable_to_non_nullable
                      as List<TabItemPayload>,
            topics: null == topics
                ? _value.topics
                : topics // ignore: cast_nullable_to_non_nullable
                      as List<TopicPayload>,
            pagination: null == pagination
                ? _value.pagination
                : pagination // ignore: cast_nullable_to_non_nullable
                      as PaginationPayload,
            announcement: null == announcement
                ? _value.announcement
                : announcement // ignore: cast_nullable_to_non_nullable
                      as AnnouncementPayload,
          )
          as $Val,
    );
  }

  /// Create a copy of HomeProps
  /// with the given fields replaced by the non-null parameter values.
  @override
  @pragma('vm:prefer-inline')
  $PaginationPayloadCopyWith<$Res> get pagination {
    return $PaginationPayloadCopyWith<$Res>(_value.pagination, (value) {
      return _then(_value.copyWith(pagination: value) as $Val);
    });
  }

  /// Create a copy of HomeProps
  /// with the given fields replaced by the non-null parameter values.
  @override
  @pragma('vm:prefer-inline')
  $AnnouncementPayloadCopyWith<$Res> get announcement {
    return $AnnouncementPayloadCopyWith<$Res>(_value.announcement, (value) {
      return _then(_value.copyWith(announcement: value) as $Val);
    });
  }
}

/// @nodoc
abstract class _$$HomePropsImplCopyWith<$Res>
    implements $HomePropsCopyWith<$Res> {
  factory _$$HomePropsImplCopyWith(
    _$HomePropsImpl value,
    $Res Function(_$HomePropsImpl) then,
  ) = __$$HomePropsImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({
    String sort,
    List<TabItemPayload> tabs,
    List<TopicPayload> topics,
    PaginationPayload pagination,
    AnnouncementPayload announcement,
  });

  @override
  $PaginationPayloadCopyWith<$Res> get pagination;
  @override
  $AnnouncementPayloadCopyWith<$Res> get announcement;
}

/// @nodoc
class __$$HomePropsImplCopyWithImpl<$Res>
    extends _$HomePropsCopyWithImpl<$Res, _$HomePropsImpl>
    implements _$$HomePropsImplCopyWith<$Res> {
  __$$HomePropsImplCopyWithImpl(
    _$HomePropsImpl _value,
    $Res Function(_$HomePropsImpl) _then,
  ) : super(_value, _then);

  /// Create a copy of HomeProps
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? sort = null,
    Object? tabs = null,
    Object? topics = null,
    Object? pagination = null,
    Object? announcement = null,
  }) {
    return _then(
      _$HomePropsImpl(
        sort: null == sort
            ? _value.sort
            : sort // ignore: cast_nullable_to_non_nullable
                  as String,
        tabs: null == tabs
            ? _value._tabs
            : tabs // ignore: cast_nullable_to_non_nullable
                  as List<TabItemPayload>,
        topics: null == topics
            ? _value._topics
            : topics // ignore: cast_nullable_to_non_nullable
                  as List<TopicPayload>,
        pagination: null == pagination
            ? _value.pagination
            : pagination // ignore: cast_nullable_to_non_nullable
                  as PaginationPayload,
        announcement: null == announcement
            ? _value.announcement
            : announcement // ignore: cast_nullable_to_non_nullable
                  as AnnouncementPayload,
      ),
    );
  }
}

/// @nodoc
@JsonSerializable()
class _$HomePropsImpl implements _HomeProps {
  const _$HomePropsImpl({
    required this.sort,
    required final List<TabItemPayload> tabs,
    required final List<TopicPayload> topics,
    required this.pagination,
    required this.announcement,
  }) : _tabs = tabs,
       _topics = topics;

  factory _$HomePropsImpl.fromJson(Map<String, dynamic> json) =>
      _$$HomePropsImplFromJson(json);

  @override
  final String sort;
  final List<TabItemPayload> _tabs;
  @override
  List<TabItemPayload> get tabs {
    if (_tabs is EqualUnmodifiableListView) return _tabs;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_tabs);
  }

  final List<TopicPayload> _topics;
  @override
  List<TopicPayload> get topics {
    if (_topics is EqualUnmodifiableListView) return _topics;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_topics);
  }

  @override
  final PaginationPayload pagination;
  @override
  final AnnouncementPayload announcement;

  @override
  String toString() {
    return 'HomeProps(sort: $sort, tabs: $tabs, topics: $topics, pagination: $pagination, announcement: $announcement)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$HomePropsImpl &&
            (identical(other.sort, sort) || other.sort == sort) &&
            const DeepCollectionEquality().equals(other._tabs, _tabs) &&
            const DeepCollectionEquality().equals(other._topics, _topics) &&
            (identical(other.pagination, pagination) ||
                other.pagination == pagination) &&
            (identical(other.announcement, announcement) ||
                other.announcement == announcement));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
    runtimeType,
    sort,
    const DeepCollectionEquality().hash(_tabs),
    const DeepCollectionEquality().hash(_topics),
    pagination,
    announcement,
  );

  /// Create a copy of HomeProps
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$HomePropsImplCopyWith<_$HomePropsImpl> get copyWith =>
      __$$HomePropsImplCopyWithImpl<_$HomePropsImpl>(this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$$HomePropsImplToJson(this);
  }
}

abstract class _HomeProps implements HomeProps {
  const factory _HomeProps({
    required final String sort,
    required final List<TabItemPayload> tabs,
    required final List<TopicPayload> topics,
    required final PaginationPayload pagination,
    required final AnnouncementPayload announcement,
  }) = _$HomePropsImpl;

  factory _HomeProps.fromJson(Map<String, dynamic> json) =
      _$HomePropsImpl.fromJson;

  @override
  String get sort;
  @override
  List<TabItemPayload> get tabs;
  @override
  List<TopicPayload> get topics;
  @override
  PaginationPayload get pagination;
  @override
  AnnouncementPayload get announcement;

  /// Create a copy of HomeProps
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$HomePropsImplCopyWith<_$HomePropsImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

import '../../current_user.dart';
import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:ui_kit/ui_kit.dart';
import '../../../l10n/app_localizations.dart';
import '../../providers.dart';
import '../../server_messages.dart';
import '../../asset_url.dart';
import '../../widgets/user_decoration.dart';

class DecorationStrings {
  DecorationStrings(BuildContext context)
    : language = Localizations.localeOf(context).languageCode;
  final String language;
  String _s(String zh, String en, String de, String ja) => switch (language) {
    'zh' => zh,
    'de' => de,
    'ja' => ja,
    _ => en,
  };
  String get title => _s(
    '头像框与多彩昵称',
    'Avatar frames & colorful names',
    'Avatarrahmen & farbige Namen',
    'アバターフレームとカラーネーム',
  );
  String get frame => _s('头像框', 'Avatar frame', 'Avatarrahmen', 'アバターフレーム');
  String get nickname => _s('多彩昵称', 'Colorful name', 'Farbiger Name', 'カラーネーム');
  String get grant => _s(
    '由管理员授予，未拥有的装饰仅供预览。',
    'Granted by administrators. Unowned styles are preview only.',
    'Vergabe durch Administratoren. Nicht erworbene Stile dienen nur als Vorschau.',
    '管理者が付与します。未所有の装飾はプレビューのみです。',
  );
  String get none => _s('不装备', 'None', 'Keine', '装備しない');
  String get preview => _s('预览', 'Preview', 'Vorschau', 'プレビュー');
  String get equip => _s('装备', 'Equip', 'Anlegen', '装備');
  String get locked => _s('尚未拥有', 'Not owned', 'Nicht erworben', '未所有');
  String get equipped => _s('已装备', 'Equipped', 'Angelegt', '装備中');
}

class DecorationPage extends ConsumerStatefulWidget {
  const DecorationPage({super.key, this.avatarUrl = '', this.name = ''});
  final String avatarUrl, name;
  @override
  ConsumerState<DecorationPage> createState() => _DecorationPageState();
}

class _DecorationPageState extends ConsumerState<DecorationPage> {
  DecorationState? _state;
  String? _error;
  bool _saving = false;
  String _frame = '', _nickname = '';
  late final int _epoch;
  bool get _current => mounted && ref.read(offlineCacheEpochProvider) == _epoch;
  @override
  void initState() {
    super.initState();
    _epoch = ref.read(offlineCacheEpochProvider);
    _load();
  }

  Future<void> _load() async {
    try {
      final state = await ref.read(userRepositoryProvider).getDecorations();
      if (_current) {
        setState(() {
          _state = state;
          _frame = state.avatarFrame;
          _nickname = state.nicknameStyle;
          _error = null;
        });
      }
    } catch (error) {
      if (_current) {
        setState(
          () =>
              _error = resolveErrorMessage(AppLocalizations.of(context), error),
        );
      }
    }
  }

  Future<void> _equip(String kind, String code) async {
    if (!_current || _saving) return;
    setState(() {
      _saving = true;
      _error = null;
    });
    try {
      final state = await ref
          .read(userRepositoryProvider)
          .equipDecorations(
            avatarFrame: kind == 'avatarFrame' ? code : _state!.avatarFrame,
            nicknameStyle: kind == 'nicknameStyle'
                ? code
                : _state!.nicknameStyle,
          );
      if (_current) {
        ref.invalidate(currentUserProvider);
        setState(() {
          _state = state;
          _frame = state.avatarFrame;
          _nickname = state.nicknameStyle;
        });
      }
    } catch (error) {
      if (_current) {
        setState(
          () =>
              _error = resolveErrorMessage(AppLocalizations.of(context), error),
        );
      }
    } finally {
      if (_current) setState(() => _saving = false);
    }
  }

  DecorationStyle? _style(String code, String kind) {
    for (final style in _state?.catalog ?? <DecorationStyle>[]) {
      if (style.code == code && style.kind == kind && style.isEnabled) {
        return style;
      }
    }
    return null;
  }

  @override
  Widget build(BuildContext context) {
    final epoch = ref.watch(offlineCacheEpochProvider);
    if (epoch != _epoch) return const SizedBox.shrink();
    final s = DecorationStrings(context);
    final l = AppLocalizations.of(context);
    final state = _state;
    return Scaffold(
      appBar: GfAppBar(title: Text(s.title)),
      body: SafeArea(
        child: ListView(
          padding: const EdgeInsets.all(16),
          children: [
            Text(s.grant),
            const SizedBox(height: 16),
            Row(
              children: [
                GfAvatar(
                  src: resolveApiAssetUrl(widget.avatarUrl),
                  size: 64,
                  frameColors: decorationColors(
                    context,
                    _style(_frame, 'avatarFrame'),
                  ),
                  frameAssetUrl: decorationFrameAssetUrl(
                    _style(_frame, 'avatarFrame'),
                  ),
                  frameScale: _style(_frame, 'avatarFrame')?.assetScale ?? 1.25,
                ),
                const SizedBox(width: 16),
                Expanded(
                  child: GfDecoratedName(
                    widget.name,
                    colors: decorationColors(
                      context,
                      _style(_nickname, 'nicknameStyle'),
                    ),
                    style: GfTheme.typographyOf(context).title3,
                  ),
                ),
              ],
            ),
            if (_error != null) ...[
              Text(
                _error!,
                style: TextStyle(color: GfTheme.colorsOf(context).error),
              ),
              if (state == null)
                GfButton(label: l.commonRetry, onPressed: _load),
            ],
            if (state == null && _error == null)
              const LinearProgressIndicator(),
            if (state != null)
              for (final kind in ['avatarFrame', 'nicknameStyle']) ...[
                const SizedBox(height: 24),
                Text(
                  kind == 'avatarFrame' ? s.frame : s.nickname,
                  style: GfTheme.typographyOf(context).title3,
                ),
                ListTile(
                  title: Text(s.none),
                  trailing: TextButton(
                    onPressed: _saving ? null : () => _equip(kind, ''),
                    child: Text(s.equip),
                  ),
                ),
                for (final style in state.catalog.where(
                  (style) => style.kind == kind && style.isEnabled,
                ))
                  ListTile(
                    title: Text(style.name),
                    subtitle: Text(
                      !state.ownedCodes.contains(style.code)
                          ? s.locked
                          : (kind == 'avatarFrame'
                                    ? state.avatarFrame
                                    : state.nicknameStyle) ==
                                style.code
                          ? s.equipped
                          : '',
                    ),
                    trailing: Wrap(
                      children: [
                        TextButton(
                          onPressed: () => setState(() {
                            if (kind == 'avatarFrame') {
                              _frame = style.code;
                            } else {
                              _nickname = style.code;
                            }
                          }),
                          child: Text(s.preview),
                        ),
                        TextButton(
                          onPressed:
                              _saving || !state.ownedCodes.contains(style.code)
                              ? null
                              : () => _equip(kind, style.code),
                          child: Text(s.equip),
                        ),
                      ],
                    ),
                  ),
              ],
          ],
        ),
      ),
    );
  }
}

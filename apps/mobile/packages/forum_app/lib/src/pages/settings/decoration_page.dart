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

class DecorationPage extends ConsumerStatefulWidget {
  const DecorationPage({
    super.key,
    this.avatarUrl = '',
    this.name = '',
    this.username = '',
  });
  final String avatarUrl, name, username;
  @override
  ConsumerState<DecorationPage> createState() => _DecorationPageState();
}

class _DecorationPageState extends ConsumerState<DecorationPage> {
  DecorationState? _state;
  String? _error;
  bool _saving = false;
  String _kind = 'avatarFrame';
  String _frame = '', _nickname = '';
  late final int _epoch;
  bool get _current => mounted && ref.read(offlineCacheEpochProvider) == _epoch;
  String get _selected => _kind == 'avatarFrame' ? _frame : _nickname;
  String get _equipped =>
      _kind == 'avatarFrame' ? _state!.avatarFrame : _state!.nicknameStyle;
  bool get _owned =>
      _selected.isEmpty ||
      (_style(_selected, _kind) != null &&
          _state!.ownedCodes.contains(_selected));

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

  Future<void> _equip() async {
    if (!_current ||
        _saving ||
        _state == null ||
        !_owned ||
        _selected == _equipped) {
      return;
    }
    final kind = _kind;
    final previous = _state!;
    final code = _selected;
    setState(() {
      _saving = true;
      _error = null;
    });
    try {
      // Confirm only this tab's choice. A preview in the other slot is never
      // written implicitly, including an unowned style being tried on there.
      final state = await ref
          .read(userRepositoryProvider)
          .equipDecorations(
            avatarFrame: kind == 'avatarFrame' ? code : previous.avatarFrame,
            nicknameStyle: kind == 'nicknameStyle'
                ? code
                : previous.nicknameStyle,
          );
      if (_current) {
        ref.invalidate(currentUserProvider);
        setState(() {
          _state = state;
          if (kind == 'avatarFrame' || _frame == previous.avatarFrame) {
            _frame = state.avatarFrame;
          }
          if (kind == 'nicknameStyle' || _nickname == previous.nicknameStyle) {
            _nickname = state.nicknameStyle;
          }
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

  Widget _choice(DecorationStyle? style, AppLocalizations l) {
    final state = _state!;
    final code = style?.code ?? '';
    final selected = _selected == code;
    final equipped = _equipped == code;
    final owned = code.isEmpty || state.ownedCodes.contains(code);
    final colors = GfTheme.colorsOf(context);
    final background = selected
        ? Color.alphaBlend(
            colors.primary.withValues(alpha: .07),
            colors.surfaceGrouped,
          )
        : colors.surfaceGrouped;
    final label = style?.name ?? l.decorationNone;
    return MergeSemantics(
      child: Semantics(
        selected: selected,
        button: true,
        enabled: !_saving,
        child: Material(
          color: background,
          child: InkWell(
            key: ValueKey('decoration-option-$_kind-$code'),
            onTap: _saving
                ? null
                : () => setState(() {
                    _error = null;
                    if (_kind == 'avatarFrame') {
                      _frame = code;
                    } else {
                      _nickname = code;
                    }
                  }),
            child: ConstrainedBox(
              constraints: const BoxConstraints(minHeight: 88),
              child: Padding(
                padding: const EdgeInsets.symmetric(
                  horizontal: 16,
                  vertical: 12,
                ),
                child: Row(
                  children: [
                    if (_kind == 'avatarFrame') ...[
                      SizedBox(
                        width: 80,
                        child: _DecorationAvatar(
                          avatarUrl: widget.avatarUrl,
                          style: style,
                          size: 40,
                        ),
                      ),
                      const SizedBox(width: 12),
                    ],
                    Expanded(
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          if (_kind == 'nicknameStyle')
                            GfDecoratedName(
                              label,
                              colors: decorationColors(
                                context,
                                style,
                                background: background,
                              ),
                              style: GfTheme.typographyOf(context).title3,
                            )
                          else
                            GfText(
                              label,
                              style: GfTheme.typographyOf(context).body,
                            ),
                          const SizedBox(height: 4),
                          GfText(
                            equipped
                                ? l.decorationEquipped
                                : owned
                                ? l.decorationAvailable
                                : '${l.decorationLocked} · ${l.decorationPreviewOnly}',
                            style: GfTheme.typographyOf(
                              context,
                            ).caption.copyWith(color: colors.textSecondary),
                          ),
                        ],
                      ),
                    ),
                    const SizedBox(width: 12),
                    GfSymbol(
                      equipped
                          ? 'circle-check'
                          : owned
                          ? 'circle'
                          : 'lock-filled',
                      size: 20,
                      color: equipped ? colors.primary : colors.textSecondary,
                    ),
                  ],
                ),
              ),
            ),
          ),
        ),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final epoch = ref.watch(offlineCacheEpochProvider);
    if (epoch != _epoch) return const SizedBox.shrink();
    final l = AppLocalizations.of(context);
    final colors = GfTheme.colorsOf(context);
    final state = _state;
    final error = _error == null
        ? null
        : Semantics(
            liveRegion: true,
            child: GfText(
              _error!,
              key: const Key('decoration-save-error'),
              style: TextStyle(color: colors.error),
            ),
          );
    final previewing =
        state != null &&
        (_frame != state.avatarFrame || _nickname != state.nicknameStyle);
    return Scaffold(
      backgroundColor: colors.canvasGrouped,
      appBar: GfAppBar(
        title: GfText(l.decorationAppearance),
        backgroundColor: colors.canvasGrouped,
      ),
      bottomNavigationBar: state == null
          ? null
          : SafeArea(
              top: false,
              minimum: const EdgeInsets.fromLTRB(16, 12, 16, 12),
              child: Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  if (error != null) ...[error, const SizedBox(height: 12)],
                  GfButton(
                    key: const Key('decoration-apply'),
                    size: GfButtonSize.large,
                    label: !_owned
                        ? l.decorationPreviewOnly
                        : _selected == _equipped
                        ? l.decorationEquipped
                        : l.decorationEquip,
                    expanded: true,
                    loading: _saving,
                    onPressed: _saving || !_owned || _selected == _equipped
                        ? null
                        : _equip,
                  ),
                  const SizedBox(height: 8),
                  GfText(
                    !_owned ? l.decorationLockedHint : l.decorationGrant,
                    textAlign: TextAlign.center,
                    style: GfTheme.typographyOf(
                      context,
                    ).caption.copyWith(color: colors.textSecondary),
                  ),
                ],
              ),
            ),
      body: SafeArea(
        top: false,
        bottom: state == null,
        child: ListView(
          padding: const EdgeInsets.fromLTRB(16, 12, 16, 24),
          children: [
            Padding(
              padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 8),
              child: Row(
                children: [
                  _DecorationAvatar(
                    avatarUrl: widget.avatarUrl,
                    style: _style(_frame, 'avatarFrame'),
                    size: 64,
                  ),
                  const SizedBox(width: 16),
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        GfDecoratedName(
                          widget.name,
                          key: const Key('decoration-preview-name'),
                          colors: decorationColors(
                            context,
                            _style(_nickname, 'nicknameStyle'),
                            background: colors.canvasGrouped,
                          ),
                          style: GfTheme.typographyOf(context).title3,
                        ),
                        if (widget.username.isNotEmpty) ...[
                          const SizedBox(height: 4),
                          GfText(
                            '@${widget.username}',
                            style: GfTheme.typographyOf(
                              context,
                            ).caption.copyWith(color: colors.textSecondary),
                          ),
                        ],
                        if (state != null) ...[
                          const SizedBox(height: 4),
                          GfText(
                            previewing ? l.decorationPreview : l.decorationCurrent,
                            style: GfTheme.typographyOf(context).caption
                                .copyWith(
                                  color: previewing
                                      ? colors.primary
                                      : colors.textSecondary,
                                ),
                          ),
                        ],
                      ],
                    ),
                  ),
                ],
              ),
            ),
            const SizedBox(height: 16),
            if (error != null && state == null) ...[
              error,
              GfButton(label: l.commonRetry, onPressed: _load),
              const SizedBox(height: 16),
            ],
            if (state == null && _error == null)
              const LinearProgressIndicator(),
            if (state != null) ...[
              Semantics(
                enabled: !_saving,
                child: GfSegmented<String>(
                  segments: [
                    (l.decorationFrame, 'avatarFrame'),
                    (l.decorationNickname, 'nicknameStyle'),
                  ],
                  selected: _kind,
                  onSelected: (kind) {
                    if (!_saving) {
                      setState(() {
                        _kind = kind;
                        _error = null;
                      });
                    }
                  },
                ),
              ),
              const SizedBox(height: 20),
              GfListSection(
                margin: EdgeInsets.zero,
                children: [
                  _choice(null, l),
                  for (final style in state.catalog.where(
                    (style) => style.kind == _kind && style.isEnabled,
                  ))
                    _choice(style, l),
                ],
              ),
            ],
          ],
        ),
      ),
    );
  }
}

/// Reserve the artwork's painted bounds, including the largest supported scale,
/// so trying an image frame never overlaps the name or the neighbouring row.
class _DecorationAvatar extends StatelessWidget {
  const _DecorationAvatar({
    required this.avatarUrl,
    required this.style,
    required this.size,
  });
  final String avatarUrl;
  final DecorationStyle? style;
  final double size;

  @override
  Widget build(BuildContext context) {
    final asset = decorationFrameAssetUrl(style);
    final rawScale = style?.assetScale ?? 1.25;
    final scale = rawScale.isFinite ? rawScale.clamp(.5, 2.0).toDouble() : 1.25;
    return Padding(
      padding: const EdgeInsets.all(4),
      child: SizedBox.square(
        dimension: asset.isEmpty ? size : size * scale.clamp(1, 2),
        child: Center(
          child: GfAvatar(
            src: resolveApiAssetUrl(avatarUrl),
            size: size,
            frameColors: decorationColors(context, style),
            frameAssetUrl: asset,
            frameScale: scale,
          ),
        ),
      ),
    );
  }
}

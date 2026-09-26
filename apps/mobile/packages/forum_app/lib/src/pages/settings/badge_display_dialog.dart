import 'package:core/core.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:ui_kit/ui_kit.dart';
import '../../../l10n/app_localizations.dart';
import '../../server_messages.dart';
import '../../widgets/confirm_discard_edit.dart';
import '../../widgets/user_badge.dart';

/// One ordered list owns both inclusion and display order. Disabled badges stay
/// available below the selected badges; saving an empty list is intentional.
class BadgeDisplayDialog extends StatefulWidget {
  const BadgeDisplayDialog({
    super.key,
    required this.badges,
    required this.selected,
    required this.onSave,
  });
  final List<UserBadgePayload> badges;
  final List<UserBadgePayload> selected;
  final Future<void> Function(List<String>) onSave;
  @override
  State<BadgeDisplayDialog> createState() => _BadgeDisplayDialogState();
}

class _BadgeDisplayDialogState extends State<BadgeDisplayDialog> {
  late final List<String> _initial = widget.selected
      .map((b) => b.code)
      .where((code) => widget.badges.any((b) => b.code == code))
      .toSet()
      .take(5)
      .toList();
  late final List<String> _codes = List.of(_initial);
  bool _saving = false, _allowPop = false, _closing = false;
  String? _error;
  void _move(int from, int to) {
    if (_saving ||
        from < 0 ||
        from >= _codes.length ||
        to < 0 ||
        to >= _codes.length) {
      return;
    }
    setState(() => _codes.insert(to, _codes.removeAt(from)));
  }

  void _pop(bool saved) {
    setState(() => _allowPop = true);
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (mounted) Navigator.pop(context, saved);
    });
  }

  Future<void> _close() async {
    if (_saving || _closing) return;
    _closing = true;
    final leave =
        listEquals(_initial, _codes) || await confirmDiscardEdit(context);
    _closing = false;
    if (mounted && leave) _pop(false);
  }

  Future<void> _save() async {
    if (_saving) return;
    setState(() {
      _saving = true;
      _error = null;
    });
    try {
      await widget.onSave(List.of(_codes));
      if (mounted) _pop(true);
    } catch (error) {
      if (mounted) {
        setState(
          () =>
              _error = resolveErrorMessage(AppLocalizations.of(context), error),
        );
      }
    } finally {
      if (mounted) setState(() => _saving = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final l = AppLocalizations.of(context);
    final colors = GfTheme.colorsOf(context);
    final ordered = [
      for (final code in _codes)
        widget.badges.firstWhere((b) => b.code == code),
      ...widget.badges.where((b) => !_codes.contains(b.code)),
    ];
    return PopScope(
      canPop: _allowPop || (!_saving && listEquals(_initial, _codes)),
      onPopInvokedWithResult: (didPop, _) {
        if (!didPop) _close();
      },
      child: Scaffold(
        appBar: GfAppBar(
          title: Text(l.badgeDisplayTitle),
          leading: IconButton(
            tooltip: l.commonBack,
            onPressed: _saving ? null : _close,
            icon: const GfSymbol('chevron-left'),
          ),
          actions: [
            Padding(
              padding: const EdgeInsets.only(right: 12),
              child: GfButton(
                label: l.commonSave,
                size: GfButtonSize.small,
                loading: _saving,
                onPressed: _save,
              ),
            ),
          ],
        ),
        body: SafeArea(
          top: false,
          child: Align(
            alignment: Alignment.topCenter,
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 640),
              child: ReorderableListView.builder(
                buildDefaultDragHandles: false,
                padding: const EdgeInsets.fromLTRB(16, 12, 16, 24),
                header: Padding(
                  padding: const EdgeInsets.only(bottom: 16),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        l.badgeDisplaySelectedCount(_codes.length),
                        style: GfTheme.typographyOf(context).title3,
                      ),
                      const SizedBox(height: 6),
                      Text(
                        l.badgeDisplayHint,
                        style: GfTheme.typographyOf(
                          context,
                        ).caption.copyWith(color: colors.iconMuted),
                      ),
                      const SizedBox(height: 6),
                      Text(
                        l.badgeDisplayReorder,
                        style: GfTheme.typographyOf(
                          context,
                        ).caption.copyWith(color: colors.iconMuted),
                      ),
                      if (_error != null)
                        Padding(
                          padding: const EdgeInsets.only(top: 12),
                          child: Semantics(
                            liveRegion: true,
                            child: Text(
                              _error!,
                              style: TextStyle(color: colors.error),
                            ),
                          ),
                        ),
                    ],
                  ),
                ),
                itemCount: ordered.length,
                onReorderItem: (from, to) {
                  if (_codes.isNotEmpty) {
                    _move(from, to.clamp(0, _codes.length - 1));
                  }
                },
                itemBuilder: (context, index) {
                  final badge = ordered[index];
                  final selected = _codes.contains(badge.code);
                  final canToggle = !_saving && (selected || _codes.length < 5);
                  void toggle(bool? checked) => setState(() {
                    if (checked == true) {
                      _codes.add(badge.code);
                    } else {
                      _codes.remove(badge.code);
                    }
                  });
                  return Material(
                    key: ValueKey('badge-row-${badge.code}'),
                    color: colors.base100,
                    child: ConstrainedBox(
                      constraints: const BoxConstraints(minHeight: 64),
                      child: Row(
                        children: [
                          Expanded(
                            child: MergeSemantics(
                              child: InkWell(
                                onTap: canToggle
                                    ? () => toggle(!selected)
                                    : null,
                                child: Padding(
                                  padding: const EdgeInsets.symmetric(
                                    vertical: 8,
                                  ),
                                  child: Row(
                                    children: [
                                      Checkbox(
                                        key: Key('badge-select-${badge.code}'),
                                        value: selected,
                                        onChanged: canToggle ? toggle : null,
                                      ),
                                      GfBadgeMedallion(
                                        icon: UserBadgeArtwork(badge, size: 24),
                                        color: userBadgeColor(badge),
                                        size: 40,
                                      ),
                                      const SizedBox(width: 10),
                                      Expanded(
                                        child: Text(
                                          badge.name,
                                          style: GfTheme.typographyOf(
                                            context,
                                          ).bodyStrong,
                                        ),
                                      ),
                                      if (selected)
                                        Padding(
                                          padding: const EdgeInsets.symmetric(
                                            horizontal: 8,
                                          ),
                                          child: Text(
                                            '${index + 1}',
                                            key: ValueKey(
                                              'badge-position-${badge.code}',
                                            ),
                                            style: GfTheme.typographyOf(context)
                                                .caption
                                                .copyWith(
                                                  color: colors.iconMuted,
                                                ),
                                          ),
                                        ),
                                    ],
                                  ),
                                ),
                              ),
                            ),
                          ),
                          if (selected)
                            ReorderableDragStartListener(
                              index: index,
                              enabled: !_saving,
                              child: Semantics(
                                label: l.badgeDisplayReorder,
                                value: '${index + 1}',
                                increasedValue: '${index + 2}',
                                decreasedValue: '$index',
                                onIncrease:
                                    _saving || index == _codes.length - 1
                                    ? null
                                    : () => _move(index, index + 1),
                                onDecrease: _saving || index == 0
                                    ? null
                                    : () => _move(index, index - 1),
                                child: Tooltip(
                                  message: l.badgeDisplayReorder,
                                  child: SizedBox(
                                    width: 48,
                                    height: 48,
                                    child: ColoredBox(
                                      color: Colors.transparent,
                                      child: Center(
                                        child: GfSymbol(
                                          'grip-vertical',
                                          size: 20,
                                          color: colors.iconMuted,
                                        ),
                                      ),
                                    ),
                                  ),
                                ),
                              ),
                            )
                          else
                            const SizedBox(width: 48),
                        ],
                      ),
                    ),
                  );
                },
              ),
            ),
          ),
        ),
      ),
    );
  }
}

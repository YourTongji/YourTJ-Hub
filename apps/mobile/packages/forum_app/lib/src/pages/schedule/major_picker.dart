import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:ui_kit/ui_kit.dart';
import '../../../l10n/app_localizations.dart';

/// Single selection within one term/grade; typing never changes the plan.
class ScheduleMajorPicker extends StatefulWidget {
  const ScheduleMajorPicker({
    super.key,
    required this.items,
    required this.selectedCode,
    required this.contextLabel,
  });
  final List<PkMajor> items;
  final String? selectedCode;
  final String contextLabel;
  @override
  State<ScheduleMajorPicker> createState() => _ScheduleMajorPickerState();
}

class _ScheduleMajorPickerState extends State<ScheduleMajorPicker> {
  final _search = TextEditingController();
  @override
  void dispose() {
    _search.dispose();
    super.dispose();
  }

  void _clear() => setState(_search.clear);

  String _displayName(PkMajor item) {
    // PK may encode a display label as year(code name). Strip only that
    // recognized wrapper with the same code; retain nested name parentheses.
    final match = RegExp(r'^\d{4}\((\S+)\s+(.+)\)$').firstMatch(item.name);
    return match != null && match.group(1) == item.code
        ? match.group(2)!
        : item.name;
  }

  @override
  Widget build(BuildContext context) {
    final l = AppLocalizations.of(context);
    final query = _search.text.trim().toLowerCase();
    final items = widget.items
        .where(
          (item) =>
              _displayName(item).toLowerCase().contains(query) ||
              item.code.toLowerCase().contains(query),
        )
        .toList();
    return SafeArea(
      top: false,
      child: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 8, 8, 8),
            child: Row(
              children: [
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      GfText(
                        l.scheduleMajor,
                        style: GfTheme.typographyOf(context).title3,
                      ),
                      GfText(
                        widget.contextLabel,
                        style: GfTheme.typographyOf(context).small,
                      ),
                    ],
                  ),
                ),
                GfIconButton(
                  symbol: 'x',
                  tooltip: l.commonClose,
                  onPressed: () => Navigator.pop(context),
                ),
              ],
            ),
          ),
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 16),
            child: GfSearchField(
              key: const Key('schedule-major-search'),
              controller: _search,
              hintText: l.scheduleMajorSearchHint,
              clearLabel: l.courseCopyClearSearch,
              onChanged: (_) => setState(() {}),
            ),
          ),
          const SizedBox(height: 8),
          Flexible(
            child: items.isEmpty
                ? GfEmpty(
                    message: query.isEmpty
                        ? l.courseCopyNoOptions
                        : l.scheduleMajorNoMatch,
                    action: query.isEmpty
                        ? null
                        : TextButton(
                            onPressed: _clear,
                            child: GfText(l.courseCopyClearSearch),
                          ),
                  )
                : ListView.builder(
                    shrinkWrap: true,
                    keyboardDismissBehavior:
                        ScrollViewKeyboardDismissBehavior.onDrag,
                    itemCount: items.length,
                    itemBuilder: (context, index) {
                      final item = items[index];
                      final selected = item.code == widget.selectedCode;
                      return ListTile(
                        key: ValueKey('schedule-major-option-${item.code}'),
                        title: GfText(_displayName(item)),
                        subtitle: GfText(item.code),
                        selected: selected,
                        trailing: selected ? const GfSymbol('check') : null,
                        onTap: () => Navigator.pop(context, item),
                      );
                    },
                  ),
          ),
        ],
      ),
    );
  }
}

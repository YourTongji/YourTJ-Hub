import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:ui_kit/ui_kit.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:intl/intl.dart';
import '../../../l10n/app_localizations.dart';
import '../../providers.dart';
import 'campus_helpers.dart';

class CampusCalendarRulesView extends ConsumerStatefulWidget {
  const CampusCalendarRulesView({super.key});
  @override
  ConsumerState<CampusCalendarRulesView> createState() =>
      _CampusCalendarRulesViewState();
}

class _CampusCalendarRulesViewState
    extends ConsumerState<CampusCalendarRulesView> {
  CancelToken? _cancel;
  CampusCalendarRules? _rules;
  Object? _error;
  bool _loading = false;
  bool _expanded = false;
  int _request = 0;
  Future<void> _load(bool expanded) async {
    _expanded = expanded;
    _cancel?.cancel();
    final request = ++_request;
    if (!expanded) {
      setState(() {
        _rules = null;
        _loading = false;
        _error = null;
      });
      return;
    }
    final token = _cancel = CancelToken();
    setState(() {
      _loading = true;
      _error = null;
      _rules = null;
    });
    try {
      final settings = await ref
          .read(campusRepositoryProvider)
          .calendarRules(cancelToken: token);
      if (mounted && request == _request) {
        setState(() => _rules = settings.rules);
      }
    } catch (e) {
      if (mounted && request == _request) setState(() => _error = e);
    } finally {
      if (mounted && request == _request) setState(() => _loading = false);
    }
  }

  @override
  void dispose() {
    _cancel?.cancel();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final l = AppLocalizations.of(context);
    String date(String value) =>
        DateFormat.yMMMEd(l.localeName).format(DateTime.parse(value));
    final rules = _rules;
    return ExpansionTile(
      tilePadding: EdgeInsets.zero,
      title: Text(l.campusCalendarRules),
      trailing: AnimatedRotation(
        turns: _expanded ? .5 : 0,
        duration: MediaQuery.disableAnimationsOf(context)
            ? Duration.zero
            : const Duration(milliseconds: 200),
        child: const GfSymbol('chevron-down', size: 20),
      ),
      onExpansionChanged: _load,
      children: [
        if (_loading) const LinearProgressIndicator(),
        if (_error != null) Text(campusError(l, _error!)),
        if (rules != null) ...[
          Padding(
            padding: const EdgeInsets.symmetric(vertical: 8),
            child: Text(l.campusCalendarRulesHint),
          ),
          if (rules.holidays.isEmpty && rules.moves.isEmpty)
            Text(l.campusNoCalendarRules),
          for (final h in rules.holidays)
            ListTile(
              contentPadding: EdgeInsets.zero,
              leading: const GfSymbol('calendar-x'),
              title: Text(h.name),
              subtitle: Text('${date(h.startDate)} – ${date(h.endDate)}'),
            ),
          for (final m in rules.moves)
            ListTile(
              contentPadding: EdgeInsets.zero,
              leading: const GfSymbol('calendar-days'),
              title: Text(m.name),
              subtitle: Text(
                l.campusMakeupDate(date(m.fromDate), date(m.toDate)),
              ),
            ),
        ],
      ],
    );
  }
}

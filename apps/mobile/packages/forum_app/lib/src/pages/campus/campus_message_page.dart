import 'dart:async';
import 'package:core/core.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:ui_kit/ui_kit.dart';
import 'package:url_launcher/url_launcher.dart';
import '../../../l10n/app_localizations.dart';
import '../../providers.dart';
import '../../widgets/status_views.dart';
import 'campus_connection.dart';
import 'campus_helpers.dart';
import 'campus_private_surface.dart';
import 'campus_state.dart';

class CampusMessagePage extends StatelessWidget {
  const CampusMessagePage({super.key, required this.id});
  final String id;
  @override
  Widget build(BuildContext context) =>
      CampusPrivateSurface(builder: (_) => _MessageBody(id: id));
}

class _MessageBody extends ConsumerStatefulWidget {
  const _MessageBody({required this.id});
  final String id;
  @override
  ConsumerState<_MessageBody> createState() => _MessageBodyState();
}

class _MessageBodyState extends ConsumerState<_MessageBody> {
  CancelToken _cancel = CancelToken();
  CampusMessageDetail? _detail;
  Object? _error;
  String? _revision;
  @override
  void dispose() {
    _cancel.cancel();
    super.dispose();
  }

  Future<void> _load() async {
    _cancel.cancel();
    final cancel = _cancel = CancelToken();
    setState(() {
      _detail = null;
      _error = null;
    });
    try {
      final detail = await ref
          .read(campusRepositoryProvider)
          .message(widget.id, cancelToken: cancel);
      if (mounted && !cancel.isCancelled) setState(() => _detail = detail);
    } catch (e) {
      if (mounted && !cancel.isCancelled) {
        ref.read(campusControllerProvider.notifier).invalidateForError(e);
        setState(() => _error = e);
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final l = AppLocalizations.of(context);
    final state = ref.watch(campusControllerProvider);
    final revision = state.status?.binding?.revision;
    // Status refresh/unlink invalidates the displayed body immediately. The next
    // confirmed revision must perform another membership-checked detail request.
    if (revision != _revision) {
      _revision = revision;
      _cancel.cancel();
      _detail = null;
      _error = null;
      if (revision != null) {
        WidgetsBinding.instance.addPostFrameCallback((_) {
          if (mounted) unawaited(_load());
        });
      }
    }
    return Scaffold(
      appBar: GfAppBar(title: Text(l.campusMessages)),
      body: SafeArea(
        top: false,
        child: ListView(
          padding: const EdgeInsets.all(20),
          children: [
            if (state.status?.candidate != null ||
                state.needsAuthorization ||
                isCampusAuthorizationError(_error))
              CampusConnection(compact: true, onConfirmed: _load)
            else if (state.loading ||
                (revision != null && _detail == null && _error == null))
              const GfLoading()
            else if (_error != null)
              GfErrorRetry(message: campusError(l, _error), onRetry: _load)
            else if (revision == null)
              const CampusConnection()
            else if (_detail case final detail?) ...[
              Text(detail.title, style: GfTheme.typographyOf(context).title2),
              const SizedBox(height: 12),
              Text(
                [
                  detail.publisher,
                  detail.publishedAt,
                ].where((s) => s.isNotEmpty).join(' · '),
                style: GfTheme.typographyOf(context).caption,
              ),
              const SizedBox(height: 24),
              SelectableText(
                detail.content.isEmpty ? l.campusNoData : detail.content,
                style: GfTheme.typographyOf(context).body.copyWith(height: 1.6),
              ),
              const SizedBox(height: 20),
              for (final link in detail.links)
                if (_safeLink(link.url) case final uri?)
                  Padding(
                    padding: const EdgeInsets.only(bottom: 8),
                    child: GfButton(
                      label: link.label.isEmpty ? uri.host : link.label,
                      variant: GfButtonVariant.outline,
                      onPressed: () async {
                        try {
                          if (!await launchUrl(
                            uri,
                            mode: LaunchMode.externalApplication,
                          )) {
                            throw StateError('Could not open link');
                          }
                        } catch (_) {
                          if (context.mounted) {
                            showGfToast(
                              context,
                              l.campusUnavailable,
                              error: true,
                            );
                          }
                        }
                      },
                    ),
                  ),
            ],
          ],
        ),
      ),
    );
  }
}

Uri? _safeLink(String url) {
  final uri = Uri.tryParse(url);
  return uri != null &&
          ['http', 'https'].contains(uri.scheme) &&
          uri.host.isNotEmpty &&
          uri.userInfo.isEmpty
      ? uri
      : null;
}

import 'package:flutter/material.dart';
import 'package:flutter/scheduler.dart';
import 'package:ui_kit/ui_kit.dart';

import 'media_repository.dart';

/// The app injects resolved account/origin/language scope; ui_kit remains storage
/// agnostic. Scope changes and explicit clears remove decoded frames as well as
/// fencing downloads. Local picker files never pass through this host.
class MediaHost extends StatefulWidget {
  const MediaHost({
    super.key,
    required this.repository,
    required this.scopeKey,
    required this.apiOrigin,
    this.imageErrorBuilder,
    required this.child,
  });
  final MediaRepository repository;
  final String scopeKey;
  final String apiOrigin;

  /// Optional host-owned failure state for shared media components; the app
  /// injects the localized, actionable fallback here.
  final GfImageErrorBuilder? imageErrorBuilder;
  final Widget child;

  @override
  State<MediaHost> createState() => _MediaHostState();
}

class _MediaHostState extends State<MediaHost> with WidgetsBindingObserver {
  bool _updating = false;
  bool _queued = false;
  bool _blockAutomaticRefresh = false;

  void _clearDecoded() {
    final cache = PaintingBinding.instance.imageCache;
    cache.maximumSizeBytes = 48 * 1024 * 1024;
    cache.clear();
    cache.clearLiveImages();
  }

  void _invalidated() {
    if (widget.repository.isSuspended) _blockAutomaticRefresh = true;
    _clearDecoded();
    if (!mounted || _updating) return;
    if (SchedulerBinding.instance.schedulerPhase ==
        SchedulerPhase.persistentCallbacks) {
      if (_queued) return;
      _queued = true;
      WidgetsBinding.instance.addPostFrameCallback((_) {
        _queued = false;
        if (mounted) setState(() {});
      });
    } else {
      setState(() {});
    }
  }

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
    _clearDecoded();
    widget.repository.addListener(_invalidated);
  }

  @override
  void didUpdateWidget(MediaHost oldWidget) {
    super.didUpdateWidget(oldWidget);
    _updating = true;
    try {
      if (oldWidget.repository != widget.repository) {
        oldWidget.repository.removeListener(_invalidated);
        oldWidget.repository.invalidate();
        widget.repository.addListener(_invalidated);
        _clearDecoded();
      } else if (oldWidget.scopeKey != widget.scopeKey ||
          oldWidget.apiOrigin != widget.apiOrigin) {
        widget.repository.invalidate();
      }
    } finally {
      _updating = false;
    }
  }

  @override
  void didHaveMemoryPressure() => _clearDecoded();

  @override
  Widget build(BuildContext context) {
    final repository = widget.repository;
    final scope = widget.scopeKey;
    final origin = widget.apiOrigin;
    final generation = repository.generation;
    if (_blockAutomaticRefresh) {
      WidgetsBinding.instance.addPostFrameCallback((_) {
        _blockAutomaticRefresh = false;
      });
    }
    return GfMediaScope(
      identity: (repository, scope, origin, generation),
      factory:
          (
            url, {
            width,
            height,
            policy = ResizeImagePolicy.exact,
            allowedOrigins,
          }) {
            // Capture this per provider, not per inherited scope: mounted images
            // clear to a placeholder, while a later navigation/retry can load anew.
            final blocked = _blockAutomaticRefresh;
            return GfBytesImage(
              identity: (
                repository,
                scope,
                origin,
                generation,
                allowedOrigins == null
                    ? null
                    : (allowedOrigins.toList()..sort()).join(','),
              ),
              url: url,
              width: width,
              height: height,
              policy: policy,
              isCurrent: () => repository.isCurrent(generation),
              onDecodeError: () => repository.discard(
                url,
                scopeKey: scope,
                apiOrigin: origin,
                generation: generation,
              ),
              load: () async {
                if (blocked || repository.isSuspended) {
                  throw StateError('Media loading suspended during clear');
                }
                final bytes = await repository.load(
                  url,
                  scopeKey: scope,
                  apiOrigin: origin,
                  allowedOrigins: allowedOrigins,
                );
                return GfMediaData(
                  bytes,
                  cacheIdentity: repository.decodedIdentity(
                    url,
                    scopeKey: scope,
                    apiOrigin: origin,
                  ),
                );
              },
            );
          },
      imageErrorBuilder: widget.imageErrorBuilder,
      child: widget.child,
    );
  }

  @override
  void dispose() {
    widget.repository.removeListener(_invalidated);
    WidgetsBinding.instance.removeObserver(this);
    super.dispose();
  }
}

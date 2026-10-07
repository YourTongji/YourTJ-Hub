import 'package:core/core.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:ui_kit/ui_kit.dart';
import '../../../l10n/app_localizations.dart';
import '../../asset_url.dart';
import '../../providers.dart';
import '../../server_messages.dart';

class AnonymousProfilePage extends ConsumerStatefulWidget {
  const AnonymousProfilePage({super.key, required this.uid});
  final String uid;
  @override
  ConsumerState<AnonymousProfilePage> createState() =>
      _AnonymousProfilePageState();
}

class _AnonymousProfilePageState extends ConsumerState<AnonymousProfilePage> {
  Map<String, dynamic>? data;
  Object? error;
  int page = 1;
  int revision = 0;
  @override
  void initState() {
    super.initState();
    load();
  }

  Future<void> load() async {
    final request = ++revision;
    try {
      final next = await ref
          .read(apiClientProvider)
          .get<Map<String, dynamic>>(
            '/a/${widget.uid}',
            queryParameters: {'page': page},
            headers: {GfApiClient.pageRequestHeader: 'true'},
            parser: (j) =>
                (j as Map<String, dynamic>)['props'] as Map<String, dynamic>,
          );
      if (mounted && request == revision) {
        setState(() {
          data = next;
          error = null;
        });
      }
    } catch (e) {
      if (mounted && request == revision) setState(() => error = e);
    }
  }

  @override
  Widget build(BuildContext context) {
    final l = AppLocalizations.of(context);
    final d = data;
    return Scaffold(
      appBar: AppBar(title: Text(l.anonymousIdentity)),
      body: error != null
          ? Center(
              child: TextButton(
                onPressed: load,
                child: Text(resolveErrorMessage(l, error!)),
              ),
            )
          : d == null
          ? const Center(child: CircularProgressIndicator())
          : ListView(
              padding: const EdgeInsets.all(20),
              children: [
                GfAvatar(
                  src: resolveApiAssetUrl(
                    (d['persona'] as Map)['avatarUrl'] as String,
                  ),
                  size: 64,
                ),
                Text(
                  (d['persona'] as Map)['name'] as String,
                  style: Theme.of(context).textTheme.headlineSmall,
                ),
                Text(l.anonymousPersonaLabel),
                Text(
                  '${l.profileTopics}: ${d['topicCount']} · ${l.profileReplies}: ${d['replyCount']}',
                ),
                for (final raw in d['topics'] as List)
                  ListTile(
                    title: Text(
                      TopicPayload.fromJson(raw as Map<String, dynamic>).title,
                    ),
                    onTap: () => context.push('/p/post/${raw['id']}'),
                  ),
                for (final raw in d['replies'] as List)
                  ListTile(
                    title: Text(raw['excerpt'] as String),
                    onTap: () => context.push(raw['url'] as String),
                  ),
                if (page > 1)
                  TextButton(
                    onPressed: () {
                      setState(() {
                        page--;
                        data = null;
                      });
                      load();
                    },
                    child: const Icon(Icons.chevron_left),
                  ),
                if (d['hasNext'] == true)
                  TextButton(
                    onPressed: () {
                      setState(() {
                        page++;
                        data = null;
                      });
                      load();
                    },
                    child: Text(l.anonymousNext),
                  ),
              ],
            ),
    );
  }
}

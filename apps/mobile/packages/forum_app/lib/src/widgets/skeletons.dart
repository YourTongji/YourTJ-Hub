import 'package:flutter/material.dart';
import 'package:ui_kit/ui_kit.dart';

/// Loading state for the home feed. Its shape mirrors the final topic rows so
/// content does not jump from a centred spinner into a dense list.
class GfTopicFeedSkeleton extends StatelessWidget {
  const GfTopicFeedSkeleton({super.key, this.itemCount = 5});

  final int itemCount;

  @override
  Widget build(BuildContext context) {
    return ExcludeSemantics(
      child: ListView.separated(
        physics: const NeverScrollableScrollPhysics(),
        padding: const EdgeInsets.symmetric(vertical: 8),
        itemCount: itemCount,
        separatorBuilder: (_, _) => const GfDivider(),
        itemBuilder: (_, _) => const _TopicRowSkeleton(),
      ),
    );
  }
}

/// Loading state for a topic detail page: author, title, prose, action strip
/// and comments are all represented using the same vertical rhythm as the
/// resolved page.
class GfTopicDetailSkeleton extends StatelessWidget {
  const GfTopicDetailSkeleton({super.key});

  @override
  Widget build(BuildContext context) {
    return ExcludeSemantics(
      child: ListView(
        physics: const NeverScrollableScrollPhysics(),
        padding: const EdgeInsets.only(bottom: 24),
        children: const <Widget>[
          Padding(
            padding: EdgeInsets.fromLTRB(16, 16, 16, 20),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: <Widget>[
                _IdentitySkeleton(avatarSize: 40),
                SizedBox(height: 14),
                GfSkeleton(width: 96, height: 22, radius: 6),
                SizedBox(height: 14),
                GfSkeleton(height: 28, radius: 6),
                SizedBox(height: 8),
                GfSkeleton(width: 244, height: 28, radius: 6),
                SizedBox(height: 18),
                _ParagraphSkeleton(lines: 5),
                SizedBox(height: 18),
                GfSkeleton(height: 48, radius: 8),
              ],
            ),
          ),
          GfDivider(),
          Padding(
            padding: EdgeInsets.fromLTRB(16, 18, 16, 10),
            child: GfSkeleton(width: 96, height: 20, radius: 6),
          ),
          _PostSkeleton(),
          GfDivider(),
          _PostSkeleton(),
        ],
      ),
    );
  }
}

/// Loading state for the profile page, mirroring the cover, overlapping
/// avatar, identity copy, stats, tabs and first activity rows.
class GfProfileSkeleton extends StatelessWidget {
  const GfProfileSkeleton({super.key});

  @override
  Widget build(BuildContext context) {
    return ExcludeSemantics(
      child: ListView(
        physics: const NeverScrollableScrollPhysics(),
        children: <Widget>[
          const GfSkeleton(height: 80, radius: 0),
          Transform.translate(
            offset: const Offset(0, -36),
            child: const Padding(
              padding: EdgeInsets.symmetric(horizontal: 16),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: <Widget>[
                  GfSkeleton(width: 96, height: 96, radius: 999),
                  SizedBox(height: 10),
                  GfSkeleton(width: 132, height: 22, radius: 6),
                  SizedBox(height: 6),
                  GfSkeleton(width: 88, height: 14, radius: 5),
                  SizedBox(height: 10),
                  GfSkeleton(width: 232, height: 16, radius: 5),
                  SizedBox(height: 20),
                  Row(
                    children: <Widget>[
                      Expanded(child: GfSkeleton(height: 46, radius: 6)),
                      SizedBox(width: 8),
                      Expanded(child: GfSkeleton(height: 46, radius: 6)),
                      SizedBox(width: 8),
                      Expanded(child: GfSkeleton(height: 46, radius: 6)),
                      SizedBox(width: 8),
                      Expanded(child: GfSkeleton(height: 46, radius: 6)),
                    ],
                  ),
                ],
              ),
            ),
          ),
          const GfDivider(),
          const Padding(
            padding: EdgeInsets.symmetric(horizontal: 12, vertical: 12),
            child: Row(
              children: <Widget>[
                Expanded(child: GfSkeleton(height: 24, radius: 6)),
                SizedBox(width: 12),
                Expanded(child: GfSkeleton(height: 24, radius: 6)),
                SizedBox(width: 12),
                Expanded(child: GfSkeleton(height: 24, radius: 6)),
              ],
            ),
          ),
          const GfDivider(),
          const _TopicRowSkeleton(),
          const GfDivider(),
          const _TopicRowSkeleton(),
        ],
      ),
    );
  }
}

/// Loading state for settings, matching the section label and grouped rows
/// used by the resolved profile and account tabs.
class GfSettingsSkeleton extends StatelessWidget {
  const GfSettingsSkeleton({super.key});

  @override
  Widget build(BuildContext context) {
    return const ExcludeSemantics(
      child: Padding(
        padding: EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: <Widget>[
            GfSkeleton(width: 84, height: 14, radius: 5),
            SizedBox(height: 8),
            GfPanel(
              child: Column(
                children: <Widget>[
                  _SettingsRowSkeleton(),
                  GfDivider(),
                  _SettingsRowSkeleton(),
                  GfDivider(),
                  _SettingsRowSkeleton(),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _SettingsRowSkeleton extends StatelessWidget {
  const _SettingsRowSkeleton();

  @override
  Widget build(BuildContext context) {
    return const Padding(
      padding: EdgeInsets.symmetric(horizontal: 16, vertical: 14),
      child: Row(
        children: <Widget>[
          GfSkeleton(width: 24, height: 24, radius: 6),
          SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: <Widget>[
                GfSkeleton(width: 112, height: 15, radius: 5),
                SizedBox(height: 7),
                GfSkeleton(width: 196, height: 12, radius: 5),
              ],
            ),
          ),
          SizedBox(width: 12),
          GfSkeleton(width: 18, height: 18, radius: 5),
        ],
      ),
    );
  }
}

class _TopicRowSkeleton extends StatelessWidget {
  const _TopicRowSkeleton();

  @override
  Widget build(BuildContext context) {
    return const Padding(
      padding: EdgeInsets.symmetric(horizontal: 16, vertical: 12),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: <Widget>[
          GfSkeleton(width: 246, height: 20, radius: 6),
          SizedBox(height: 8),
          GfSkeleton(height: 14, radius: 5),
          SizedBox(height: 10),
          Row(
            children: <Widget>[
              GfSkeleton(width: 28, height: 28, radius: 999),
              SizedBox(width: 8),
              GfSkeleton(width: 92, height: 12, radius: 5),
              Spacer(),
              GfSkeleton(width: 46, height: 12, radius: 5),
            ],
          ),
        ],
      ),
    );
  }
}

class _PostSkeleton extends StatelessWidget {
  const _PostSkeleton();

  @override
  Widget build(BuildContext context) {
    return const Padding(
      padding: EdgeInsets.all(16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: <Widget>[
          _IdentitySkeleton(avatarSize: 32),
          SizedBox(height: 14),
          _ParagraphSkeleton(lines: 3),
          SizedBox(height: 12),
          GfSkeleton(width: 150, height: 14, radius: 5),
        ],
      ),
    );
  }
}

class _IdentitySkeleton extends StatelessWidget {
  const _IdentitySkeleton({required this.avatarSize});

  final double avatarSize;

  @override
  Widget build(BuildContext context) {
    return Row(
      children: <Widget>[
        GfSkeleton(width: avatarSize, height: avatarSize, radius: 999),
        const SizedBox(width: 10),
        const Expanded(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: <Widget>[
              GfSkeleton(width: 116, height: 16, radius: 5),
              SizedBox(height: 6),
              GfSkeleton(width: 172, height: 12, radius: 5),
            ],
          ),
        ),
      ],
    );
  }
}

class _ParagraphSkeleton extends StatelessWidget {
  const _ParagraphSkeleton({required this.lines});

  final int lines;

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: <Widget>[
        for (int index = 0; index < lines; index++) ...<Widget>[
          FractionallySizedBox(
            widthFactor: index == lines - 1 ? 0.62 : 1,
            child: const GfSkeleton(height: 14, radius: 5),
          ),
          if (index < lines - 1) const SizedBox(height: 8),
        ],
      ],
    );
  }
}

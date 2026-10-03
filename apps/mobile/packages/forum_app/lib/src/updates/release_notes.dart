import 'dart:convert';

class ReleaseNote {
  const ReleaseNote({
    required this.id,
    required this.title,
    required this.summary,
    required this.platforms,
    required this.kind,
    this.required = false,
  });

  final String id;
  final String title;
  final String summary;
  final Set<String> platforms;
  final String kind;
  final bool required;

  bool includesChannel(String channel) =>
      platforms.contains(channel) ||
      (platforms.contains('ios') &&
          const {'ios-app-store', 'ios-testflight'}.contains(channel));

  static ReleaseNote? parse(Object? value, {bool required = false}) {
    if (value is! Map) return null;
    final id = value['id'];
    final title = value['title'];
    final summary = value['summary'];
    final platforms = value['platforms'];
    final kind = value['kind'];
    if (id is! String ||
        !RegExp(r'^[a-z0-9][a-z0-9-]{0,79}$').hasMatch(id) ||
        title is! String ||
        title.trim().isEmpty ||
        title.length > 160 ||
        summary is! String ||
        summary.trim().isEmpty ||
        summary.length > 1000 ||
        platforms is! List ||
        platforms.isEmpty ||
        platforms.any(
          (p) => !const {
            'android',
            'ios',
            'ios-app-store',
            'ios-testflight',
          }.contains(p),
        ) ||
        !const {'feature', 'improvement', 'fix', 'security'}.contains(kind)) {
      return null;
    }
    final explicitRequired = value['required'];
    if (explicitRequired != null && explicitRequired is! bool) return null;
    return ReleaseNote(
      id: id,
      title: title.trim(),
      summary: summary.trim(),
      platforms: platforms.cast<String>().toSet(),
      kind: kind as String,
      required: required || explicitRequired == true,
    );
  }
}

List<ReleaseNote> promptReleaseNotes(List<ReleaseNote> notes) => [
  ...notes.where((note) => note.required),
  ...notes.where((note) => !note.required).take(5),
];

class ReleaseNoteVersion {
  const ReleaseNoteVersion({
    required this.version,
    required this.buildNumber,
    required this.channels,
    required this.highlights,
    required this.breaking,
    required this.requiredActions,
    this.testflightNotes = const [],
  });

  final String version;
  final int buildNumber;
  final Set<String> channels;
  final List<ReleaseNote> highlights;
  final List<ReleaseNote> breaking;
  final List<ReleaseNote> requiredActions;
  final List<ReleaseNote> testflightNotes;

  static ReleaseNoteVersion? parse(Object? value) {
    if (value is! Map) return null;
    final version = value['version'];
    final build = value['buildNumber'];
    final rawChannels = value['channels'];
    const allowedChannels = {'android', 'ios-app-store', 'ios-testflight'};
    if (version is! String ||
        !RegExp(r'^\d+\.\d+\.\d+$').hasMatch(version) ||
        build is! int ||
        build <= 0 ||
        build > 2100000000 ||
        rawChannels is! List ||
        rawChannels.isEmpty ||
        rawChannels.any((channel) => !allowedChannels.contains(channel))) {
      return null;
    }
    List<ReleaseNote>? notes(Object? raw, {bool required = false}) {
      if (raw is! List || raw.length > 100) return null;
      final result = <ReleaseNote>[];
      for (final entry in raw) {
        final parsed = ReleaseNote.parse(entry, required: required);
        if (parsed == null) return null;
        result.add(parsed);
      }
      return result;
    }

    final highlights = notes(value['highlights']);
    final breaking = notes(value['breaking'], required: true);
    final actions = notes(value['requiredActions'], required: true);
    final rawTestflightNotes = value['testflightNotes'];
    if (rawTestflightNotes is! List ||
        rawTestflightNotes.length > 100 ||
        rawChannels.toSet().length != rawChannels.length) {
      return null;
    }
    final testflightNotes = <ReleaseNote>[];
    for (final entry in rawTestflightNotes) {
      if (entry is! Map ||
          entry['id'] is! String ||
          !RegExp(
            r'^[a-z0-9][a-z0-9-]{0,79}$',
          ).hasMatch(entry['id'] as String) ||
          entry['text'] is! String ||
          (entry['text'] as String).trim().isEmpty ||
          (entry['text'] as String).length > 1000) {
        return null;
      }
      testflightNotes.add(
        ReleaseNote(
          id: entry['id'] as String,
          title: '',
          summary: (entry['text'] as String).trim(),
          platforms: const {'ios-testflight'},
          kind: 'feature',
        ),
      );
    }
    if (highlights == null || breaking == null || actions == null) return null;
    return ReleaseNoteVersion(
      version: version,
      buildNumber: build,
      channels: rawChannels.cast<String>().toSet(),
      highlights: highlights,
      breaking: breaking,
      requiredActions: actions,
      testflightNotes: List.unmodifiable(testflightNotes),
    );
  }
}

class ReleaseNoteCoverage {
  const ReleaseNoteCoverage({
    required this.completeFromBuild,
    required this.throughBuild,
    required this.coveredBuilds,
  });

  final int completeFromBuild;
  final int throughBuild;
  final Set<int> coveredBuilds;
}

class ReleaseNoteCatalog {
  const ReleaseNoteCatalog(
    this.releases, {
    this.source,
    this.channelCoverage = const {},
  });
  final List<ReleaseNoteVersion> releases;
  final String? source;
  final Map<String, ReleaseNoteCoverage> channelCoverage;

  static ReleaseNoteCatalog? decode(Object? value) {
    if (value is! Map || value['schemaVersion'] != 1) return null;
    final coverage = value['historyCoverage'];
    String? source;
    final coverageByChannel = <String, ReleaseNoteCoverage>{};
    if (coverage != null) {
      if (coverage is! Map) return null;
      source = coverage['source'];
      final publishedAt = coverage['publishedAt'];
      final channels = coverage['byChannel'];
      if (source is! String ||
          source != 'github-release-receipts' ||
          publishedAt is! String ||
          DateTime.tryParse(publishedAt) == null ||
          channels is! Map) {
        return null;
      }
      const allowedChannels = {'android', 'ios-app-store', 'ios-testflight'};
      if (channels.keys.any((key) => !allowedChannels.contains(key))) {
        return null;
      }
      for (final entry in channels.entries) {
        final channel = entry.value;
        if (channel is! Map) return null;
        final floor = channel['completeFromBuild'];
        final through = channel['throughBuild'];
        final rawBuilds = channel['coveredBuilds'];
        if (floor is! int ||
            through is! int ||
            floor < 0 ||
            through < floor ||
            through > 2100000000 ||
            rawBuilds is! List ||
            rawBuilds.length > 300 ||
            rawBuilds.any(
              (build) => build is! int || build < floor || build > through,
            )) {
          return null;
        }
        final builds = rawBuilds.cast<int>().toSet();
        if (builds.length != rawBuilds.length) return null;
        coverageByChannel[entry.key as String] = ReleaseNoteCoverage(
          completeFromBuild: floor,
          throughBuild: through,
          coveredBuilds: builds,
        );
      }
    } else {
      // Read older cached catalogs during rollout; these do not claim provenance.
      if (value['historyFromBuild'] != null) return null;
    }
    final raw = value['releases'];
    if (raw is! List || raw.length > 300) return null;
    final parsed = <ReleaseNoteVersion>[];
    final builds = <int>{};
    for (final item in raw) {
      final release = ReleaseNoteVersion.parse(item);
      if (release == null || !builds.add(release.buildNumber)) return null;
      parsed.add(release);
    }
    for (final channel in coverageByChannel.entries) {
      if (!channel.value.coveredBuilds.every(
        (build) => parsed.any(
          (release) =>
              release.buildNumber == build &&
              release.channels.contains(channel.key),
        ),
      )) {
        return null;
      }
    }
    parsed.sort((a, b) => a.buildNumber.compareTo(b.buildNumber));
    return ReleaseNoteCatalog(
      List.unmodifiable(parsed),
      source: source,
      channelCoverage: Map.unmodifiable(coverageByChannel),
    );
  }

  factory ReleaseNoteCatalog.fromJson(String source) =>
      ReleaseNoteCatalog.decode(jsonDecode(source)) ??
      (throw const FormatException('Invalid mobile release notes'));

  List<ReleaseNote> forRange({
    required int installedBuild,
    required int targetBuild,
    required String platform,
  }) {
    final merged = <String, ReleaseNote>{};
    final versions = releases.where(
      (r) =>
          r.buildNumber > installedBuild &&
          r.buildNumber <= targetBuild &&
          r.channels.contains(platform),
    );
    for (final release in versions) {
      final structured = [
        ...release.highlights,
        ...release.breaking,
        ...release.requiredActions,
      ];
      // A TestFlight disclosure is also a required entry; keep that titled copy.
      final structuredIds = {
        for (final note in structured)
          if (note.includesChannel(platform)) note.id,
      };
      for (final note in [
        ...structured,
        ...release.testflightNotes.where(
          (note) => !structuredIds.contains(note.id),
        ),
      ]) {
        if (!note.includesChannel(platform)) {
          continue;
        }
        final previous = merged[note.id];
        merged[note.id] =
            previous == null || !previous.required || note.required
            ? note
            : ReleaseNote(
                id: note.id,
                title: note.title,
                summary: note.summary,
                platforms: note.platforms,
                kind: note.kind,
                required: true,
              );
      }
    }
    final result = merged.values.toList()
      ..sort((a, b) {
        final required = (b.required ? 1 : 0).compareTo(a.required ? 1 : 0);
        if (required != 0) return required;
        const priority = {
          'security': 0,
          'feature': 1,
          'improvement': 2,
          'fix': 3,
        };
        return (priority[a.kind] ?? 4).compareTo(priority[b.kind] ?? 4);
      });
    return List.unmodifiable(result);
  }

  bool hasCompleteRange({
    required int installedBuild,
    required int targetBuild,
    required String channel,
  }) {
    final coverage = channelCoverage[channel];
    if (coverage == null ||
        installedBuild < coverage.completeFromBuild ||
        targetBuild > coverage.throughBuild ||
        !coverage.coveredBuilds.contains(targetBuild)) {
      return false;
    }
    return true;
  }

  List<ReleaseNote> notesForUpdate({
    required int installedBuild,
    required int targetBuild,
    required String platform,
    required String channel,
  }) {
    final complete = hasCompleteRange(
      installedBuild: installedBuild,
      targetBuild: targetBuild,
      channel: channel,
    );
    return forRange(
      installedBuild: complete ? installedBuild : targetBuild - 1,
      targetBuild: targetBuild,
      platform: platform,
    );
  }
}

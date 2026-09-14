/// Hand-maintained mirror of packages/api-contract/components/status.yaml.
/// Missing source data stays null; it must never be presented as zero or healthy.

class StatusSnapshot {
  const StatusSnapshot({
    required this.range,
    required this.serverRange,
    required this.refreshAfter,
    required this.server,
    required this.traffic,
    required this.uptime,
  });

  final String range;
  final String serverRange;
  final int refreshAfter;
  final StatusServerSource server;
  final StatusTrafficSource traffic;
  final StatusUptimeSource uptime;

  factory StatusSnapshot.fromJson(Map<String, dynamic> json) => StatusSnapshot(
    range: json['range'] as String,
    serverRange: json['serverRange'] as String,
    refreshAfter: (json['refreshAfter'] as num).toInt(),
    server: StatusServerSource.fromJson(
      Map<String, dynamic>.from(json['server'] as Map),
    ),
    uptime: StatusUptimeSource.fromJson(
      Map<String, dynamic>.from(json['uptime'] as Map),
    ),
    traffic: StatusTrafficSource.fromJson(
      Map<String, dynamic>.from(json['traffic'] as Map),
    ),
  );
}

class StatusServerSource {
  const StatusServerSource({
    required this.state,
    this.fetchedAt,
    required this.data,
  });

  final String state;
  final String? fetchedAt;
  final StatusServer? data;

  factory StatusServerSource.fromJson(Map<String, dynamic> json) =>
      StatusServerSource(
        state: json['state'] as String,
        fetchedAt: json['fetchedAt'] as String?,
        data: json['data'] == null
            ? null
            : StatusServer.fromJson(
                Map<String, dynamic>.from(json['data'] as Map),
              ),
      );
}

class StatusTrafficSource {
  const StatusTrafficSource({
    required this.state,
    this.fetchedAt,
    required this.data,
  });

  final String state;
  final String? fetchedAt;
  final StatusTraffic? data;

  factory StatusTrafficSource.fromJson(Map<String, dynamic> json) =>
      StatusTrafficSource(
        state: json['state'] as String,
        fetchedAt: json['fetchedAt'] as String?,
        data: json['data'] == null
            ? null
            : StatusTraffic.fromJson(
                Map<String, dynamic>.from(json['data'] as Map),
              ),
      );
}

class StatusServer {
  const StatusServer({
    required this.name,
    required this.region,
    required this.cpuCores,
    required this.current,
    required this.history,
    required this.historyAvailable,
  });

  final String name;
  final String region;
  final int cpuCores;
  final StatusSample? current;
  final List<StatusLoadPoint> history;
  final bool historyAvailable;

  factory StatusServer.fromJson(Map<String, dynamic> json) => StatusServer(
    name: json['name'] as String,
    region: json['region'] as String,
    cpuCores: (json['cpuCores'] as num).toInt(),
    current: json['current'] == null
        ? null
        : StatusSample.fromJson(
            Map<String, dynamic>.from(json['current'] as Map),
          ),
    history: (json['history'] as List)
        .map(
          (item) =>
              StatusLoadPoint.fromJson(Map<String, dynamic>.from(item as Map)),
        )
        .toList(),
    historyAvailable: json['historyAvailable'] as bool,
  );
}

class StatusSample {
  const StatusSample({
    required this.observedAt,
    required this.cpu,
    required this.memoryUsed,
    required this.memoryTotal,
    required this.diskUsed,
    required this.diskTotal,
    required this.networkUp,
    required this.networkDown,
    required this.uptime,
  });

  final String observedAt;
  final double cpu;
  final int memoryUsed;
  final int memoryTotal;
  final int diskUsed;
  final int diskTotal;
  final int networkUp;
  final int networkDown;
  final int uptime;

  factory StatusSample.fromJson(Map<String, dynamic> json) => StatusSample(
    observedAt: json['observedAt'] as String,
    cpu: (json['cpu'] as num).toDouble(),
    memoryUsed: (json['memoryUsed'] as num).toInt(),
    memoryTotal: (json['memoryTotal'] as num).toInt(),
    diskUsed: (json['diskUsed'] as num).toInt(),
    diskTotal: (json['diskTotal'] as num).toInt(),
    networkUp: (json['networkUp'] as num).toInt(),
    networkDown: (json['networkDown'] as num).toInt(),
    uptime: (json['uptime'] as num).toInt(),
  );
}

class StatusLoadPoint {
  const StatusLoadPoint({
    required this.time,
    required this.cpu,
    required this.memoryPercent,
  });

  final String time;
  final double cpu;
  final double memoryPercent;

  factory StatusLoadPoint.fromJson(Map<String, dynamic> json) =>
      StatusLoadPoint(
        time: json['time'] as String,
        cpu: (json['cpu'] as num).toDouble(),
        memoryPercent: (json['memoryPercent'] as num).toDouble(),
      );
}

class StatusTraffic {
  const StatusTraffic({
    required this.startAt,
    required this.endAt,
    required this.visitors,
    required this.pageviews,
    required this.visits,
    required this.bounceRate,
    required this.averageDuration,
    required this.activeVisitors,
    required this.series,
    required this.seriesAvailable,
  });

  final String startAt;
  final String endAt;
  final int visitors;
  final int pageviews;
  final int visits;
  final double? bounceRate;
  final double? averageDuration;
  final int? activeVisitors;
  final List<StatusTrafficPoint> series;
  final bool seriesAvailable;

  factory StatusTraffic.fromJson(Map<String, dynamic> json) => StatusTraffic(
    startAt: json['startAt'] as String,
    endAt: json['endAt'] as String,
    visitors: (json['visitors'] as num).toInt(),
    pageviews: (json['pageviews'] as num).toInt(),
    visits: (json['visits'] as num).toInt(),
    bounceRate: json['bounceRate'] == null
        ? null
        : (json['bounceRate'] as num).toDouble(),
    averageDuration: json['averageDuration'] == null
        ? null
        : (json['averageDuration'] as num).toDouble(),
    activeVisitors: json['activeVisitors'] == null
        ? null
        : (json['activeVisitors'] as num).toInt(),
    series: (json['series'] as List)
        .map(
          (item) => StatusTrafficPoint.fromJson(
            Map<String, dynamic>.from(item as Map),
          ),
        )
        .toList(),
    seriesAvailable: json['seriesAvailable'] as bool,
  );
}

class StatusTrafficPoint {
  const StatusTrafficPoint({
    required this.time,
    required this.pageviews,
    required this.visitors,
  });

  final String time;
  final int pageviews;
  final int visitors;

  factory StatusTrafficPoint.fromJson(Map<String, dynamic> json) =>
      StatusTrafficPoint(
        time: json['time'] as String,
        pageviews: (json['pageviews'] as num).toInt(),
        visitors: (json['visitors'] as num).toInt(),
      );
}

class StatusUptimeSource {
  const StatusUptimeSource({
    required this.state,
    this.fetchedAt,
    required this.data,
  });
  final String state;
  final String? fetchedAt;
  final StatusUptime? data;
  factory StatusUptimeSource.fromJson(Map<String, dynamic> json) =>
      StatusUptimeSource(
        state: json['state'] as String,
        fetchedAt: json['fetchedAt'] as String?,
        data: json['data'] == null
            ? null
            : StatusUptime.fromJson(
                Map<String, dynamic>.from(json['data'] as Map),
              ),
      );
}

class StatusUptime {
  const StatusUptime({required this.statusPageUrl, required this.monitors});
  final String statusPageUrl;
  final List<StatusUptimeMonitor> monitors;
  factory StatusUptime.fromJson(Map<String, dynamic> json) => StatusUptime(
    statusPageUrl: json['statusPageUrl'] as String,
    monitors: (json['monitors'] as List)
        .map(
          (item) => StatusUptimeMonitor.fromJson(
            Map<String, dynamic>.from(item as Map),
          ),
        )
        .toList(),
  );
}

class StatusUptimeMonitor {
  const StatusUptimeMonitor({
    required this.id,
    required this.name,
    required this.type,
    required this.uptime24h,
    required this.current,
    required this.history,
  });
  final int id;
  final String name;
  final String type;
  final double? uptime24h;
  final StatusUptimeHeartbeat? current;
  final List<StatusUptimeHeartbeat> history;
  factory StatusUptimeMonitor.fromJson(Map<String, dynamic> json) =>
      StatusUptimeMonitor(
        id: (json['id'] as num).toInt(),
        name: json['name'] as String,
        type: json['type'] as String,
        uptime24h: (json['uptime24h'] as num?)?.toDouble(),
        current: json['current'] == null
            ? null
            : StatusUptimeHeartbeat.fromJson(
                Map<String, dynamic>.from(json['current'] as Map),
              ),
        history: (json['history'] as List)
            .map(
              (item) => StatusUptimeHeartbeat.fromJson(
                Map<String, dynamic>.from(item as Map),
              ),
            )
            .toList(),
      );
}

class StatusUptimeHeartbeat {
  const StatusUptimeHeartbeat({
    required this.time,
    required this.status,
    required this.ping,
  });
  final String time;
  final String status;
  final double? ping;
  factory StatusUptimeHeartbeat.fromJson(Map<String, dynamic> json) =>
      StatusUptimeHeartbeat(
        time: json['time'] as String,
        status: json['status'] as String,
        ping: (json['ping'] as num?)?.toDouble(),
      );
}

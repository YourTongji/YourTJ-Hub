import {
  Any, FlutterPlugin, FlutterPluginBinding, MethodCall, MethodCallHandler,
  MethodChannel, MethodResult, EventChannel, EventSink
} from '@ohos/flutter_ohos';
import { connection } from '@kit.NetworkKit';

export default class ConnectivityPlugin implements FlutterPlugin, MethodCallHandler {
  private channel: MethodChannel | null = null;
  private events: EventChannel | null = null;
  private sink: EventSink | null = null;
  private monitor: connection.NetConnection | null = null;

  getUniqueClassName(): string { return 'YourTJConnectivity'; }

  onAttachedToEngine(binding: FlutterPluginBinding): void {
    this.channel = new MethodChannel(binding.getBinaryMessenger(), 'dev.fluttercommunity.plus/connectivity');
    this.channel.setMethodCallHandler(this);
    this.events = new EventChannel(binding.getBinaryMessenger(), 'dev.fluttercommunity.plus/connectivity_status');
    this.events.setStreamHandler(this);
  }

  onDetachedFromEngine(_binding: FlutterPluginBinding): void {
    this.onCancel(null);
    this.channel?.setMethodCallHandler(null);
    this.events?.setStreamHandler(null);
  }

  private async current(): Promise<string[]> {
    const network = await connection.getDefaultNet();
    if (network.netId === 0) return ['none'];
    const capabilities = await connection.getNetCapabilities(network);
    const types: string[] = [];
    for (const bearer of capabilities.bearerTypes) {
      switch (bearer) {
        case connection.NetBearType.BEARER_WIFI: types.push('wifi'); break;
        case connection.NetBearType.BEARER_CELLULAR: types.push('mobile'); break;
        case connection.NetBearType.BEARER_ETHERNET: types.push('ethernet'); break;
        case connection.NetBearType.BEARER_BLUETOOTH: types.push('bluetooth'); break;
        case connection.NetBearType.BEARER_VPN: types.push('vpn'); break;
        default: types.push('other');
      }
    }
    return types.length ? types : ['other'];
  }

  private async emit(): Promise<void> {
    try { this.sink?.success(await this.current()); }
    catch (_) { this.sink?.error('network_query_failed', 'Could not query network state', null); }
  }

  onListen(_args: Any, sink: EventSink): void {
    this.onCancel(null);
    this.sink = sink;
    this.monitor = connection.createNetConnection();
    this.monitor.on('netAvailable', () => { this.emit(); });
    this.monitor.on('netLost', () => { this.emit(); });
    this.monitor.on('netCapabilitiesChange', () => { this.emit(); });
    this.monitor.on('netUnavailable', () => { this.emit(); });
    this.monitor.register((error) => {
      if (error) this.sink?.error('network_monitor_failed', 'Could not monitor network state', null);
      else this.emit();
    });
  }

  onCancel(_args: Any): void {
    this.sink = null;
    this.monitor?.unregister(() => {});
    this.monitor = null;
  }

  async onMethodCall(call: MethodCall, result: MethodResult): Promise<void> {
    if (call.method !== 'check') { result.notImplemented(); return; }
    try { result.success(await this.current()); }
    catch (_) { result.error('network_query_failed', 'Could not query network state', null); }
  }
}

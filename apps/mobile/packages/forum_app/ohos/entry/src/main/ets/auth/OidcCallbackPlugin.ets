import {
  Any, FlutterPlugin, FlutterPluginBinding, MethodChannel, MethodCall,
  MethodResult, MethodCallHandler, EventChannel, EventSink
} from '@ohos/flutter_ohos';
import { common, Want } from '@kit.AbilityKit';
import { uri } from '@kit.ArkTS';

/** Callbacks stay in memory; Dart owns the PKCE/state/redirect validation. */
export default class OidcCallbackPlugin implements FlutterPlugin, MethodCallHandler {
  private channel: MethodChannel | null = null;
  private events: EventChannel | null = null;
  private sink: EventSink | null = null;
  private initialCallback: string | null = null;
  private context: common.UIAbilityContext;

  constructor(context: common.UIAbilityContext) { this.context = context; }

  getUniqueClassName(): string { return 'YourTJOidcCallback'; }

  onAttachedToEngine(binding: FlutterPluginBinding): void {
    this.channel = new MethodChannel(binding.getBinaryMessenger(), 'yourtj/oidc');
    this.channel.setMethodCallHandler(this);
    this.events = new EventChannel(binding.getBinaryMessenger(), 'yourtj/oidc/callbacks');
    this.events.setStreamHandler(this);
  }

  onDetachedFromEngine(_binding: FlutterPluginBinding): void {
    this.channel?.setMethodCallHandler(null);
    this.events?.setStreamHandler(null);
    this.sink = null;
    this.initialCallback = null;
  }

  onListen(_args: Any, sink: EventSink): void { this.sink = sink; }
  onCancel(_args: Any): void { this.sink = null; }

  acceptWant(want: Want): void {
    const callback = want.uri;
    if (!callback || !/^yourtj:\/\/callback(?:[?#]|$)/.test(callback)) return;
    if (this.sink) this.sink.success(callback);
    else this.initialCallback = callback;
  }

  async onMethodCall(call: MethodCall, result: MethodResult): Promise<void> {
    if (call.method === 'openAuthorization') {
      try {
        const args = call.args as Map<string, string>;
        const url = args.get('url');
        if (typeof url !== 'string') throw new Error('Invalid authorization URL');
        const parsed = new uri.URI(url);
        if (parsed.scheme !== 'https' || !parsed.host || parsed.authority.includes('@')) {
          throw new Error('Invalid authorization URL');
        }
        await this.context.startAbility({ action: 'ohos.want.action.viewData', uri: url });
        result.success(true);
      } catch (_) {
        result.error('authorization_launch_failed', 'Could not open the authorization browser', null);
      }
      return;
    }
    if (call.method !== 'getInitialCallback') {
      result.notImplemented();
      return;
    }
    result.success(this.initialCallback);
    this.initialCallback = null;
  }
}

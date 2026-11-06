import { Any, EventChannel, EventSink, FlutterPlugin, FlutterPluginBinding } from '@ohos/flutter_ohos';
import { Want } from '@kit.AbilityKit';

const EMAIL_VERIFIED = 'yourtj://account/email-verified';

export default class AccountReturnPlugin implements FlutterPlugin {
  private channel: EventChannel | null = null;
  private sink: EventSink | null = null;
  private pending: boolean = false;

  getUniqueClassName(): string { return 'YourTJAccountReturn'; }

  onAttachedToEngine(binding: FlutterPluginBinding): void {
    this.channel = new EventChannel(binding.getBinaryMessenger(), 'yourtj/account_return');
    this.channel.setStreamHandler(this);
  }

  onDetachedFromEngine(_binding: FlutterPluginBinding): void {
    this.channel?.setStreamHandler(null);
    this.sink = null;
  }

  acceptWant(want: Want): void {
    if (want.uri !== EMAIL_VERIFIED) return;
    if (this.sink) this.sink.success(EMAIL_VERIFIED);
    else this.pending = true;
  }

  onListen(_args: Any, sink: EventSink): void {
    this.sink = sink;
    if (this.pending) {
      this.pending = false;
      sink.success(EMAIL_VERIFIED);
    }
  }

  onCancel(_args: Any): void { this.sink = null; }
}

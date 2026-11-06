import {
  FlutterPlugin, FlutterPluginBinding, MethodCall, MethodCallHandler, MethodChannel, MethodResult
} from '@ohos/flutter_ohos';
import { common } from '@kit.AbilityKit';
import { systemShare } from '@kit.ShareKit';
import { uniformTypeDescriptor as utd } from '@kit.ArkData';
import { fileUri } from '@kit.CoreFileKit';

export default class SharePlugin implements FlutterPlugin, MethodCallHandler {
  private channel: MethodChannel | null = null;
  private controller: systemShare.ShareController | null = null;
  private pending: MethodResult | null = null;
  private completed = (_value: systemShare.ShareOperationResult): void => { this.finish('shared'); };
  private dismissed = (): void => { this.finish(''); };

  private context: common.UIAbilityContext;
  constructor(context: common.UIAbilityContext) { this.context = context; }
  getUniqueClassName(): string { return 'YourTJShare'; }

  onAttachedToEngine(binding: FlutterPluginBinding): void {
    this.channel = new MethodChannel(binding.getBinaryMessenger(), 'dev.fluttercommunity.plus/share');
    this.channel.setMethodCallHandler(this);
  }

  onDetachedFromEngine(_binding: FlutterPluginBinding): void {
    this.finish('');
    this.channel?.setMethodCallHandler(null);
  }

  private cleanup(): void {
    this.controller?.off('shareCompleted', this.completed);
    this.controller?.off('dismiss', this.dismissed);
    this.controller = null;
  }

  private finish(value: string): void {
    const result = this.pending;
    this.pending = null;
    this.cleanup();
    result?.success(value);
  }

  async onMethodCall(call: MethodCall, result: MethodResult): Promise<void> {
    if (call.method !== 'share') { result.notImplemented(); return; }
    if (this.pending) { result.error('share_busy', 'A share panel is already open', null); return; }
    try {
      const args = call.args as Map<string, string | string[]>;
      const records: systemShare.SharedRecord[] = [];
      const text = args.get('text') as string;
      const uri = args.get('uri') as string;
      const title = (args.get('subject') ?? args.get('title') ?? '') as string;
      if (text) records.push({ utd: utd.UniformDataType.PLAIN_TEXT, content: text, title: title });
      if (uri) records.push({ utd: utd.UniformDataType.HYPERLINK, content: uri, title: title });
      const paths = (args.get('paths') ?? []) as string[];
      const mimeTypes = (args.get('mimeTypes') ?? []) as string[];
      for (let i = 0; i < paths.length; i++) {
        if (!paths[i].startsWith('/')) throw new Error('Expected an absolute file path');
        records.push({
          utd: utd.getUniformDataTypeByMIMEType(mimeTypes[i] ?? 'application/octet-stream'),
          uri: fileUri.getUriFromPath(paths[i]), title: title
        });
      }
      if (records.length === 0) throw new Error('Empty share data');
      const data = new systemShare.SharedData(records[0]);
      for (let i = 1; i < records.length; i++) data.addRecord(records[i]);
      this.controller = new systemShare.ShareController(data);
      this.pending = result;
      this.controller.on('shareCompleted', this.completed);
      this.controller.on('dismiss', this.dismissed);
      await this.controller.show(this.context, {});
    } catch (_) {
      this.pending = null;
      this.cleanup();
      result.error('share_failed', 'Could not open the system share panel', null);
    }
  }
}

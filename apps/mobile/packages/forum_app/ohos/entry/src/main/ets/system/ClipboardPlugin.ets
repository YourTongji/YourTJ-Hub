import {
  FlutterPlugin, FlutterPluginBinding, MethodCall, MethodCallHandler, MethodChannel, MethodResult
} from '@ohos/flutter_ohos';
import { abilityAccessCtrl, common } from '@kit.AbilityKit';
import { pasteboard, BusinessError } from '@kit.BasicServicesKit';
import { image } from '@kit.ImageKit';

const MAX_IMAGE_BYTES: number = 16 * 1024 * 1024;
const MAX_PIXELS: number = 16 * 1024 * 1024;

export default class ClipboardPlugin implements FlutterPlugin, MethodCallHandler {
  private channel: MethodChannel | null = null;
  private tail: Promise<void> = Promise.resolve();
  private board: pasteboard.SystemPasteboard = pasteboard.getSystemPasteboard();
  private context: common.UIAbilityContext;
  constructor(context: common.UIAbilityContext) { this.context = context; }
  getUniqueClassName(): string { return 'YourTJClipboard'; }

  onAttachedToEngine(binding: FlutterPluginBinding): void {
    this.channel = new MethodChannel(binding.getBinaryMessenger(), 'yourtj/clipboard');
    this.channel.setMethodCallHandler(this);
  }
  onDetachedFromEngine(_binding: FlutterPluginBinding): void {
    this.channel?.setMethodCallHandler(null);
    this.channel = null;
  }
  onMethodCall(call: MethodCall, result: MethodResult): void {
    this.tail = this.tail.then(() => this.handle(call, result)).catch((error: BusinessError) => {
      result.error(error.code === 201 ? 'clipboard_permission_denied' : 'clipboard_failed',
        'Clipboard operation could not be completed', null);
    });
  }

  private async write(data: pasteboard.PasteData): Promise<void> {
    const property = data.getProperty();
    property.shareOption = pasteboard.ShareOption.LOCALDEVICE;
    data.setProperty(property);
    await this.board.setData(data);
  }

  private async read(type: string, result: MethodResult): Promise<pasteboard.PasteData | null> {
    // Check MIME metadata before asking for access; never read content while polling.
    if (!this.board.hasDataType(type)) { result.success(null); return null; }
    const permission = await abilityAccessCtrl.createAtManager().requestPermissionsFromUser(
      this.context, ['ohos.permission.READ_PASTEBOARD']);
    if (permission.authResults.length !== 1 || permission.authResults[0] !== 0) {
      result.error('clipboard_permission_denied', 'Clipboard access was declined', null);
      return null;
    }
    return this.board.getData();
  }

  private async handle(call: MethodCall, result: MethodResult): Promise<void> {
    switch (call.method) {
      case 'hasData':
        result.success([pasteboard.MIMETYPE_TEXT_PLAIN, pasteboard.MIMETYPE_TEXT_HTML,
          pasteboard.MIMETYPE_PIXELMAP, 'image/gif'].some(type => this.board.hasDataType(type)));
        return;
      case 'copyHtml': {
        const html = call.args as string;
        if (typeof html !== 'string' || html.length === 0 || html.length > 1024 * 1024) {
          throw new Error('Invalid clipboard HTML');
        }
        await this.write(pasteboard.createData(pasteboard.MIMETYPE_TEXT_HTML, html));
        result.success(null);
        return;
      }
      case 'getHtml': {
        const data = await this.read(pasteboard.MIMETYPE_TEXT_HTML, result);
        if (!data) return;
        const html = data.getPrimaryHtml();
        if (html.length > 1024 * 1024) throw new Error('Clipboard HTML exceeds size limit');
        result.success(html || null);
        return;
      }
      case 'copyImage': {
        const bytes = call.args as Uint8Array;
        if (!(bytes instanceof Uint8Array) || bytes.byteLength === 0 || bytes.byteLength > MAX_IMAGE_BYTES) {
          throw new Error('Invalid clipboard image');
        }
        const buffer = bytes.buffer.slice(bytes.byteOffset, bytes.byteOffset + bytes.byteLength);
        const source = image.createImageSource(buffer);
        try {
          const info = await source.getImageInfo();
          if (info.size.width * info.size.height > MAX_PIXELS) throw new Error('Clipboard image exceeds pixel limit');
          const pixelMap = await source.createPixelMap();
          try {
            const data = pasteboard.createData(pasteboard.MIMETYPE_PIXELMAP, pixelMap);
            if (isGif(bytes)) data.addRecord('image/gif', buffer);
            await this.write(data);
          } finally { await pixelMap.release(); }
        } finally { await source.release(); }
        result.success(null);
        return;
      }
      case 'getImage': {
        const data = await this.read(pasteboard.MIMETYPE_PIXELMAP, result);
        if (!data) return;
        const pixelMap = data.getPrimaryPixelMap();
        if (!pixelMap) { result.success(null); return; }
        const packer = image.createImagePacker();
        try {
          const info = await pixelMap.getImageInfo();
          if (info.size.width * info.size.height > MAX_PIXELS) throw new Error('Clipboard image exceeds pixel limit');
          const buffer = await packer.packToData(pixelMap, { format: 'image/png', quality: 100 });
          if (buffer.byteLength > MAX_IMAGE_BYTES) throw new Error('Clipboard image exceeds size limit');
          result.success(new Uint8Array(buffer));
        } finally { await packer.release(); await pixelMap.release(); }
        return;
      }
      case 'getGif': {
        const data = await this.read('image/gif', result);
        if (!data) return;
        for (let index = 0; index < data.getRecordCount(); index++) {
          const record = data.getRecordAt(index);
          if (!record.getValidTypes(['image/gif']).includes('image/gif')) continue;
          const value = await record.getData('image/gif');
          if (value instanceof ArrayBuffer && value.byteLength <= MAX_IMAGE_BYTES) {
            const bytes = new Uint8Array(value);
            if (isGif(bytes)) { result.success(bytes); return; }
          }
        }
        result.success(null);
        return;
      }
      default: result.notImplemented();
    }
  }
}

export function isGif(bytes: Uint8Array): boolean {
  return bytes.length >= 6 && bytes[0] === 71 && bytes[1] === 73 && bytes[2] === 70 &&
    bytes[3] === 56 && (bytes[4] === 55 || bytes[4] === 57) && bytes[5] === 97;
}

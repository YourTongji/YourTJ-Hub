import {
  FlutterPlugin, FlutterPluginBinding, MethodCall, MethodCallHandler, MethodChannel, MethodResult
} from '@ohos/flutter_ohos';
import { common } from '@kit.AbilityKit';
import { fileIo as fs, fileUri } from '@kit.CoreFileKit';
import { photoAccessHelper } from '@kit.MediaLibraryKit';
import { util } from '@kit.ArkTS';
import { BusinessError } from '@kit.BasicServicesKit';

export default class GalleryPlugin implements FlutterPlugin, MethodCallHandler {
  private channel: MethodChannel | null = null;
  private saving: boolean = false;

  private context: common.UIAbilityContext;
  constructor(context: common.UIAbilityContext) { this.context = context; }
  getUniqueClassName(): string { return 'YourTJGallery'; }

  onAttachedToEngine(binding: FlutterPluginBinding): void {
    this.channel = new MethodChannel(binding.getBinaryMessenger(), 'gal');
    this.channel.setMethodCallHandler(this);
  }

  onDetachedFromEngine(_binding: FlutterPluginBinding): void { this.channel?.setMethodCallHandler(null); }

  async onMethodCall(call: MethodCall, result: MethodResult): Promise<void> {
    // Every save requests consent in the system creation dialog; no album-wide permission is needed.
    if (call.method === 'hasAccess' || call.method === 'requestAccess') { result.success(true); return; }
    if (call.method !== 'putImageBytes') { result.notImplemented(); return; }
    if (this.saving) { result.error('UNEXPECTED', 'An image save is already active', null); return; }
    this.saving = true;
    let temporary = '';
    let helper: photoAccessHelper.PhotoAccessHelper | null = null;
    try {
      const args = call.args as Map<string, string | Uint8Array>;
      const bytes = args.get('bytes') as Uint8Array;
      const extension = imageExtension(bytes);
      if (!extension) { result.error('NOT_SUPPORTED_FORMAT', 'Unsupported image encoding', null); return; }
      const rawName = (args.get('name') ?? 'YourTJ') as string;
      const title = rawName.replace(/[\\/:*?"'`<>|{}\[\]]/g, '_').slice(0, 180) || 'YourTJ';
      temporary = `${this.context.cacheDir}/yourtj-save-${util.generateRandomUUID()}.${extension}`;
      const source = await fs.open(temporary, fs.OpenMode.CREATE | fs.OpenMode.READ_WRITE);
      try {
        const buffer = bytes.buffer.slice(bytes.byteOffset, bytes.byteOffset + bytes.byteLength);
        let written = 0;
        while (written < buffer.byteLength) {
          const count = await fs.write(source.fd, buffer.slice(written));
          if (count <= 0) throw new Error('Incomplete image write');
          written += count;
        }
      } finally { await fs.close(source); }
      helper = photoAccessHelper.getPhotoAccessHelper(this.context);
      const uris = await helper.showAssetsCreationDialog([fileUri.getUriFromPath(temporary)], [{
        title: title, fileNameExtension: extension, photoType: photoAccessHelper.PhotoType.IMAGE
      }]);
      if (uris.length !== 1) { result.error('ACCESS_DENIED', 'Image save was declined', null); return; }
      const destination = await fs.open(uris[0], fs.OpenMode.READ_WRITE);
      try { await fs.copyFile(temporary, destination.fd); }
      finally { await fs.close(destination); }
      result.success(null);
    } catch (error) {
      result.error('UNEXPECTED', 'Could not save the image', (error as BusinessError).code ?? null);
    }
    finally {
      if (temporary) { try { await fs.unlink(temporary); } catch (_) {} }
      if (helper) { try { await helper.release(); } catch (_) {} }
      this.saving = false;
    }
  }
}

export function imageExtension(bytes: Uint8Array): string | null {
  if (!bytes || bytes.length < 12) return null;
  if (bytes[0] === 0x89 && bytes[1] === 0x50 && bytes[2] === 0x4e && bytes[3] === 0x47 &&
    bytes[4] === 0x0d && bytes[5] === 0x0a && bytes[6] === 0x1a && bytes[7] === 0x0a) return 'png';
  if (bytes[0] === 0xff && bytes[1] === 0xd8 && bytes[2] === 0xff) return 'jpg';
  if (bytes[0] === 0x47 && bytes[1] === 0x49 && bytes[2] === 0x46 && bytes[3] === 0x38 &&
    (bytes[4] === 0x37 || bytes[4] === 0x39) && bytes[5] === 0x61) return 'gif';
  if (bytes[0] === 0x52 && bytes[1] === 0x49 && bytes[2] === 0x46 && bytes[3] === 0x46 &&
    bytes[8] === 0x57 && bytes[9] === 0x45 && bytes[10] === 0x42 && bytes[11] === 0x50) return 'webp';
  return null;
}

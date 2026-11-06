import { common } from '@kit.AbilityKit';
import { asset } from '@kit.AssetStoreKit';
import { preferences } from '@kit.ArkData';
import { cryptoFramework } from '@kit.CryptoArchitectureKit';
import { util } from '@kit.ArkTS';
import { BusinessError } from '@kit.BasicServicesKit';

export class SecureStore {
  private store: preferences.Preferences;
  private encoder: util.TextEncoder = new util.TextEncoder();
  private decoder: util.TextDecoder = new util.TextDecoder('utf-8', { fatal: true });
  private base64: util.Base64Helper = new util.Base64Helper();
  private key: cryptoFramework.SymKey | null = null;

  constructor(context: common.Context) {
    this.store = preferences.getPreferencesSync(context, { name: 'yourtj_secure_storage_v1' });
  }

  private async loadKey(create: boolean): Promise<cryptoFramework.SymKey> {
    if (this.key) return this.key;
    const query: asset.AssetMap = new Map();
    query.set(asset.Tag.ALIAS, this.encoder.encodeInto('yourtj.secure-storage.key.v1'));
    query.set(asset.Tag.RETURN_TYPE, asset.ReturnType.ALL);
    let bytes: Uint8Array | null = null;
    try {
      const found = await asset.query(query);
      bytes = found[0]?.get(asset.Tag.SECRET) as Uint8Array;
    } catch (error) {
      if ((error as BusinessError).code !== 24000002) {
        throw new Error(`Asset Store query failed (${(error as BusinessError).code})`);
      }
    }
    if (!bytes) {
      if (!create || Object.keys(this.store.getAllSync()).length > 0) {
        throw new Error('Secure storage encryption key unavailable');
      }
      bytes = (await cryptoFramework.createRandom().generateRandom(32)).data;
      const attributes: asset.AssetMap = new Map();
      attributes.set(asset.Tag.ALIAS, this.encoder.encodeInto('yourtj.secure-storage.key.v1'));
      attributes.set(asset.Tag.SECRET, bytes);
      attributes.set(asset.Tag.ACCESSIBILITY, asset.Accessibility.DEVICE_FIRST_UNLOCKED);
      attributes.set(asset.Tag.SYNC_TYPE, asset.SyncType.NEVER);
      try {
        await asset.add(attributes);
      } catch (error) {
        bytes.fill(0);
        // Another engine may have created the same device-local key.
        if ((error as BusinessError).code !== 24000003) {
          throw new Error(`Asset Store write failed (${(error as BusinessError).code})`);
        }
        const found = await asset.query(query);
        bytes = found[0]?.get(asset.Tag.SECRET) as Uint8Array;
      }
    }
    if (!bytes || bytes.length !== 32) throw new Error('Invalid secure storage encryption key');
    try {
      this.key = await cryptoFramework.createSymKeyGenerator('AES256').convertKey({ data: bytes });
      return this.key;
    } finally {
      bytes.fill(0);
    }
  }

  async write(name: string, value: string): Promise<void> {
    const key = await this.loadKey(true);
    const iv = await cryptoFramework.createRandom().generateRandom(12);
    const params: cryptoFramework.GcmParamsSpec = {
      algName: 'GcmParamsSpec', iv: iv,
      aad: { data: this.encoder.encodeInto(name) }, authTag: { data: new Uint8Array(16) }
    };
    const cipher = cryptoFramework.createCipher('AES256|GCM|NoPadding');
    await cipher.init(cryptoFramework.CryptoMode.ENCRYPT_MODE, key, params);
    const plaintext = value.length ? this.encoder.encodeInto(value) : new Uint8Array(0);
    let encrypted: cryptoFramework.DataBlob;
    try {
      encrypted = await cipher.doFinal(plaintext.length ? { data: plaintext } : null);
    } finally { plaintext.fill(0); }
    await this.store.put(`item:${name}`,
      `1.${this.base64.encodeToStringSync(iv.data)}.${this.base64.encodeToStringSync(encrypted.data)}`);
    await this.store.flush();
  }

  async read(name: string): Promise<string | null> {
    const entry = `item:${name}`;
    if (!this.store.hasSync(entry)) return null;
    const encoded = this.store.getSync(entry, '') as string;
    const parts = encoded.split('.');
    if (parts.length !== 3 || parts[0] !== '1') throw new Error('Invalid encrypted storage record');
    const iv = this.base64.decodeSync(parts[1]);
    const data = this.base64.decodeSync(parts[2]);
    if (iv.length !== 12 || data.length < 16) throw new Error('Invalid encrypted storage record');
    const key = await this.loadKey(false);
    const params: cryptoFramework.GcmParamsSpec = {
      algName: 'GcmParamsSpec', iv: { data: iv },
      aad: { data: this.encoder.encodeInto(name) }, authTag: { data: data.slice(-16) }
    };
    const cipher = cryptoFramework.createCipher('AES256|GCM|NoPadding');
    await cipher.init(cryptoFramework.CryptoMode.DECRYPT_MODE, key, params);
    const plaintext = await cipher.doFinal(data.length > 16 ? { data: data.slice(0, -16) } : null);
    if (!plaintext) {
      if (data.length === 16) return '';
      throw new Error('Secure storage decryption returned no data');
    }
    try {
      return plaintext.data?.length ? this.decoder.decodeWithStream(plaintext.data, { stream: false }) : '';
    } finally {
      plaintext.data?.fill(0);
    }
  }

  containsKey(name: string): boolean { return this.store.hasSync(`item:${name}`); }

  async delete(name: string): Promise<void> {
    await this.store.delete(`item:${name}`);
    await this.store.flush();
  }

  async readAll(): Promise<Map<string, string>> {
    const result = new Map<string, string>();
    for (const entry of Object.keys(this.store.getAllSync())) {
      if (!entry.startsWith('item:')) continue;
      const name = entry.slice(5);
      const value = await this.read(name);
      if (value !== null) result.set(name, value);
    }
    return result;
  }

  async deleteAll(): Promise<void> {
    await this.store.clear();
    await this.store.flush();
  }
}

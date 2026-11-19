import { describe, it, expect } from '@ohos/hypium';
import { common } from '@kit.AbilityKit';
import { preferences } from '@kit.ArkData';
import { SecureStore } from '../../../main/ets/storage/SecureStore';

export default function secureStoreTest(context: common.UIAbilityContext): void {
  describe('SecureStore', () => {
    it('roundTripsLongUnicodeAndEmptyValuesAcrossInstances', 0, async () => {
      const store = new SecureStore(context);
      const name = 'test.harmony.roundtrip';
      const value = '课表 · sample '.repeat(512);
      let stage = 'write long';
      try {
        await store.write(name, value);
        stage = 'read long';
        expect(await new SecureStore(context).read(name)).assertEqual(value);
        stage = 'write empty';
        await store.write(name, '');
        stage = 'read empty';
        expect(await store.read(name)).assertEqual('');
      } catch (error) {
        throw new Error(`Secure roundtrip failed at ${stage}: ${(error as Error).message}`);
      } finally {
        await store.delete(name);
      }
      expect(await store.read(name)).assertNull();
    });

    it('rejectsTamperingWithoutErasingTheRecord', 0, async () => {
      const store = new SecureStore(context);
      const disk = preferences.getPreferencesSync(context, { name: 'yourtj_secure_storage_v1' });
      const name = 'test.harmony.tamper';
      try {
        await store.write(name, '');
        const encrypted = disk.getSync(`item:${name}`, '') as string;
        expect(encrypted.startsWith('1.')).assertTrue();
        const parts = encrypted.split('.');
        const first = parts[2].charAt(0) === 'A' ? 'B' : 'A';
        const damaged = `1.${parts[1]}.${first}${parts[2].slice(1)}`;
        await disk.put(`item:${name}`, damaged);
        await disk.flush();
        let failed = false;
        try { await store.read(name); } catch (_) { failed = true; }
        expect(failed).assertTrue();
        expect(disk.getSync(`item:${name}`, '')).assertEqual(damaged);
      } finally {
        await store.delete(name);
      }
    });

    it('bindsCiphertextToItsLogicalKey', 0, async () => {
      const store = new SecureStore(context);
      const disk = preferences.getPreferencesSync(context, { name: 'yourtj_secure_storage_v1' });
      const first = 'test.harmony.aad.first';
      const second = 'test.harmony.aad.second';
      try {
        await store.write(first, 'sample');
        await disk.put(`item:${second}`, disk.getSync(`item:${first}`, ''));
        await disk.flush();
        let failed = false;
        try { await store.read(second); } catch (_) { failed = true; }
        expect(failed).assertTrue();
        expect(await store.read(first)).assertEqual('sample');
      } finally {
        await store.delete(first);
        await store.delete(second);
      }
    });
  });
}

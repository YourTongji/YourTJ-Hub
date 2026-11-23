import { common } from '@kit.AbilityKit';
import scheduleFormStoreTest from './ScheduleFormStore.test';
import accountReturnTest from './AccountReturn.test';
import galleryTest from './Gallery.test';
import clipboardTest from './Clipboard.test';
import scheduleDataTest from './ScheduleData.test'
import secureStoreTest from './SecureStore.test'
import passkeyTest from './Passkey.test';

export default function testsuite(context: common.UIAbilityContext) {
  scheduleFormStoreTest(context);
  accountReturnTest();
  galleryTest();
  clipboardTest(context);
  scheduleDataTest();
  secureStoreTest(context);
  passkeyTest(context);
}

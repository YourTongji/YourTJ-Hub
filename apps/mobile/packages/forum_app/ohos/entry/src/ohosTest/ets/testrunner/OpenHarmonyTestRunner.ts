import TestRunner from '@ohos.application.testRunner';
import { abilityDelegatorRegistry } from '@kit.TestKit';

export default class OpenHarmonyTestRunner implements TestRunner {
  onPrepare(): void {}

  async onRun(): Promise<void> {
    const delegator = abilityDelegatorRegistry.getAbilityDelegator();
    const args = abilityDelegatorRegistry.getArguments();
    try {
      const result = await delegator.executeShellCommand(`aa start -a TestAbility -b ${args.bundleName}`);
      if (result.exitCode !== 0) await delegator.finishTest('Could not start the native test ability', -1);
    } catch (_) {
      await delegator.finishTest('Native test ability startup failed', -1);
    }
  }
}

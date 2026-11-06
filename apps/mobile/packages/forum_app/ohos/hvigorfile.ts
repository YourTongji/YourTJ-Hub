import path from 'path'
import { appTasks, OhosHapContext, OhosPluginId } from '@ohos/hvigor-ohos-plugin';
import { hvigor, HvigorNode } from '@ohos/hvigor';
import { flutterHvigorPlugin } from 'flutter-hvigor-plugin';

// Flutter copies native assets for the requested ABI but leaves earlier ABIs in libs/.
hvigor.nodesEvaluated(() => {
    const requested = hvigor.getParameter().getExtParam('TARGET_PLATFORM')?.split(',');
    if (!requested) return;
    const architectures = new Map([['ohos-arm64', 'arm64-v8a'], ['ohos-x64', 'x86_64']]);
    if (requested.some(platform => !architectures.has(platform))) {
        throw new Error('Unsupported YourTJ HarmonyOS target architecture');
    }
    const selected = requested.map(platform => architectures.get(platform));
    const excludes = [...architectures.values()]
        .filter(abi => !selected.includes(abi)).map(abi => `**/${abi}/*.so`);
    hvigor.getRootNode().subNodes((node: HvigorNode) => {
        const context = node.getContext(OhosPluginId.OHOS_HAP_PLUGIN) as OhosHapContext;
        if (!context) return;
        const profile = context.getBuildProfileOpt();
        const options = profile.buildOption ??= {};
        const native = options.nativeLib ??= {};
        const filter = native.filter ??= {};
        filter.excludes = [...(filter.excludes ?? []), ...excludes];
        context.setBuildProfileOpt(profile);
    });
});

export default {
    system: appTasks,  /* Built-in plugin of Hvigor. It cannot be modified. */
    plugins:[flutterHvigorPlugin(path.dirname(__dirname))]         /* Custom plugin to extend the functionality of Hvigor. */
}
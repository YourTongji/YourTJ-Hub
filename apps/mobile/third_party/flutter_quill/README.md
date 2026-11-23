# Local flutter_quill adaptation

Source: flutter_quill 11.6.0 from pub.dev, under the included MIT license.

The editor's tap-outside handler accepts additional mobile platforms, including
OpenHarmony, while retaining the existing rendering and touch-focus behavior.

On HarmonyOS, the native provider uses YourTJ's `yourtj/clipboard` Pasteboard
bridge for HTML, image and GIF operations. Toolbar polling queries supported
MIME metadata rather than reading clipboard contents or requesting permission.
Explicit paste requests use the native permission flow; a refusal leaves rich
paste empty, while other platform errors remain visible to callers. Copies are
restricted to the local device by the native bridge. Other platforms retain
the upstream native provider and clipboard behavior.

Remove the corresponding local changes when upstream supports these HarmonyOS
contracts without changing the supported UI or permission boundaries.

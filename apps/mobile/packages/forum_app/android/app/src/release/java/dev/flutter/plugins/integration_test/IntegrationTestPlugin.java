package dev.flutter.plugins.integration_test;

import io.flutter.embedding.engine.plugins.FlutterPlugin;

/**
 * Flutter 3.44.9 leaves dev-only plugins in the generated registrant but
 * excludes their projects from the release classpath. Integration tests keep
 * using the real plugin in debug/profile builds; release needs no test hooks.
 */
public final class IntegrationTestPlugin implements FlutterPlugin {
  @Override
  public void onAttachedToEngine(FlutterPluginBinding binding) {}

  @Override
  public void onDetachedFromEngine(FlutterPluginBinding binding) {}
}

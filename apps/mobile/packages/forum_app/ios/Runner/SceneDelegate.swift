import Flutter
import UIKit

class SceneDelegate: FlutterSceneDelegate {
  private var reservedRegionsBridge: YourTJReservedRegionsBridge?

  override func scene(
    _ scene: UIScene,
    willConnectTo session: UISceneSession,
    options connectionOptions: UIScene.ConnectionOptions
  ) {
    super.scene(scene, willConnectTo: session, options: connectionOptions)
    for context in connectionOptions.urlContexts {
      (UIApplication.shared.delegate as? AppDelegate)?.accountReturnBridge.handle(context.url)
    }
    if let controller = window?.rootViewController as? FlutterViewController {
      reservedRegionsBridge = YourTJReservedRegionsBridge(controller: controller)
    }
  }

  override func scene(_ scene: UIScene, openURLContexts URLContexts: Set<UIOpenURLContext>) {
    let bridge = (UIApplication.shared.delegate as? AppDelegate)?.accountReturnBridge
    let other = Set(URLContexts.filter { bridge?.handle($0.url) != true })
    if !other.isEmpty { super.scene(scene, openURLContexts: other) }
  }

  override func sceneDidDisconnect(_ scene: UIScene) {
    reservedRegionsBridge?.dispose()
    reservedRegionsBridge = nil
    super.sceneDidDisconnect(scene)
  }
}

// Uses active occlusion regions in this view's coordinates, as described by
// https://developer.apple.com/videos/play/tech-talks/111463/
private final class YourTJReservedRegionsBridge: NSObject, FlutterStreamHandler {
  private weak var view: UIView?
  private let channel: FlutterEventChannel
  private var stopUpdates: (() -> Void)?
  private var sink: FlutterEventSink?
  private var previous: NSDictionary?

  init(controller: FlutterViewController) {
    view = controller.view
    channel = FlutterEventChannel(
      name: "yourtj/window_reserved_regions",
      binaryMessenger: controller.binaryMessenger
    )
    super.init()
    channel.setStreamHandler(self)
  }

  func onListen(withArguments arguments: Any?, eventSink events: @escaping FlutterEventSink) -> FlutterError? {
    sink = events
    previous = nil
    guard let view else { return nil }
    // Passive observation: do not request display frames just to poll geometry.
    // Region changes can occur without a window-size change (system chrome).
    #if compiler(>=6.4)
      if #available(iOS 27.1, *) {
        let link = UIUpdateLink(view: view) { [weak self] _, _ in self?.publish() }
        link.requiresContinuousUpdates = false
        link.isEnabled = true
        stopUpdates = { link.isEnabled = false }
        publish()
        return nil
      }
    #endif
    // An older SDK/OS has no authoritative outside-safe-area geometry.
    events(nil)
    return nil
  }

  func onCancel(withArguments arguments: Any?) -> FlutterError? {
    stopUpdates?()
    stopUpdates = nil
    sink = nil
    previous = nil
    return nil
  }

  func dispose() {
    _ = onCancel(withArguments: nil)
    channel.setStreamHandler(nil)
  }

  #if compiler(>=6.4)
  @available(iOS 27.1, *)
  private func publish() {
    guard let view, let window = view.window, let scene = window.windowScene, let sink else { return }
    let bounds = view.bounds
    var frames = view.reservedRegions(kind: .occlusion)
      .filter { $0.isActive }.map(\.frame)
    if let statusBar = scene.statusBarManager, !statusBar.isStatusBarHidden {
      frames.append(scene.coordinateSpace.convert(statusBar.statusBarFrame, to: view))
    }
    let occlusions = frames.compactMap { frame -> [Double]? in
      let visible = frame.intersection(bounds)
      guard !visible.isNull, !visible.isEmpty else { return nil }
      return [Double(visible.minX - bounds.minX), Double(visible.minY - bounds.minY),
              Double(visible.width), Double(visible.height)]
    }
    let safe = view.safeAreaInsets
    let message: NSDictionary = [
      "width": Double(bounds.width), "height": Double(bounds.height),
      "safeArea": [Double(safe.left), Double(safe.top), Double(safe.right), Double(safe.bottom)],
      "occlusions": occlusions
    ]
    guard previous != message else { return }
    previous = message
    sink(message)
  }
  #endif
}

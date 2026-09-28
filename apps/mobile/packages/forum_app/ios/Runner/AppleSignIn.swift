import AuthenticationServices
import CryptoKit
import Flutter
import UIKit

final class YourTJAppleAuth: NSObject, ASAuthorizationControllerDelegate, ASAuthorizationControllerPresentationContextProviding {
  private let channel: FlutterMethodChannel
  private var completion: FlutterResult?
  private var controller: ASAuthorizationController?
  private var anchor: UIWindow?
  private var expectedState: String?
  private var observer: NSObjectProtocol?

  init(registrar: FlutterPluginRegistrar) {
    channel = FlutterMethodChannel(name: "yourtj/apple-auth", binaryMessenger: registrar.messenger())
    super.init()
    channel.setMethodCallHandler { [weak self] call, result in self?.handle(call, result) }
    registrar.register(AppleButtonFactory(messenger: registrar.messenger()), withId: "yourtj/apple-sign-in-button")
    observer = NotificationCenter.default.addObserver(forName: ASAuthorizationAppleIDProvider.credentialRevokedNotification, object: nil, queue: .main) { [weak self] _ in
      self?.channel.invokeMethod("credentialsChanged", arguments: nil)
    }
  }
  deinit { if let observer { NotificationCenter.default.removeObserver(observer) } }

  private func handle(_ call: FlutterMethodCall, _ result: @escaping FlutterResult) {
    let args = call.arguments as? [String: Any]
    switch call.method {
    case "authorize":
      guard completion == nil else { result(FlutterError(code: "busy", message: "Apple authorization already in progress", details: nil)); return }
      guard let nonce = args?["nonce"] as? String, (32...128).contains(nonce.utf8.count),
        let window = UIApplication.shared.connectedScenes.compactMap({ $0 as? UIWindowScene }).filter({ $0.activationState == .foregroundActive }).flatMap({ $0.windows }).first(where: { $0.isKeyWindow }) else {
        result(FlutterError(code: "unavailable", message: "Apple authorization unavailable", details: nil)); return
      }
      let request = ASAuthorizationAppleIDProvider().createRequest()
      request.requestedScopes = [] // No name or email is needed to bind an identity.
      request.nonce = SHA256.hash(data: Data(nonce.utf8)).map { String(format: "%02x", $0) }.joined()
      let state = UUID().uuidString
      request.state = state
      expectedState = state
      anchor = window
      completion = result
      let flow = ASAuthorizationController(authorizationRequests: [request])
      controller = flow
      flow.delegate = self
      flow.presentationContextProvider = self
      flow.performRequests()
    case "credentialState":
      guard let user = args?["userIdentifier"] as? String, !user.isEmpty, user.count <= 255 else {
        result(FlutterError(code: "invalid", message: "Invalid Apple identity", details: nil)); return
      }
      ASAuthorizationAppleIDProvider().getCredentialState(forUserID: user) { state, error in
        DispatchQueue.main.async {
          if error != nil { result(FlutterError(code: "unavailable", message: "Unable to check Apple authorization", details: nil)); return }
          switch state {
          case .authorized: result("authorized")
          case .revoked: result("revoked")
          case .notFound: result("notFound")
          case .transferred: result("transferred")
          @unknown default: result("unknown")
          }
        }
      }
    default: result(FlutterMethodNotImplemented)
    }
  }

  func presentationAnchor(for controller: ASAuthorizationController) -> ASPresentationAnchor { anchor ?? ASPresentationAnchor() }
  func authorizationController(controller: ASAuthorizationController, didCompleteWithAuthorization authorization: ASAuthorization) {
    guard controller === self.controller else { return }
    guard let credential = authorization.credential as? ASAuthorizationAppleIDCredential,
      credential.state == expectedState,
      let codeData = credential.authorizationCode, let code = String(data: codeData, encoding: .utf8), !code.isEmpty,
      let tokenData = credential.identityToken, let token = String(data: tokenData, encoding: .utf8), !token.isEmpty,
      !credential.user.isEmpty else {
      finish(FlutterError(code: "invalid", message: "Incomplete Apple authorization", details: nil)); return
    }
    finish(["authorizationCode": code, "identityToken": token, "userIdentifier": credential.user])
  }
  func authorizationController(controller: ASAuthorizationController, didCompleteWithError error: Error) {
    guard controller === self.controller else { return }
    let cancelled = (error as? ASAuthorizationError)?.code == .canceled
    finish(FlutterError(code: cancelled ? "cancelled" : "unavailable", message: cancelled ? "Apple authorization cancelled" : "Apple authorization failed", details: nil))
  }
  private func finish(_ value: Any) {
    let callback = completion
    completion = nil; controller = nil; anchor = nil; expectedState = nil
    callback?(value)
  }
}

private final class AppleButtonFactory: NSObject, FlutterPlatformViewFactory {
  let messenger: FlutterBinaryMessenger
  init(messenger: FlutterBinaryMessenger) { self.messenger = messenger }
  func createArgsCodec() -> FlutterMessageCodec & NSObjectProtocol { FlutterStandardMessageCodec.sharedInstance() }
  func create(withFrame frame: CGRect, viewIdentifier viewId: Int64, arguments args: Any?) -> FlutterPlatformView {
    let params = args as? [String: Any]
    return AppleButtonView(
      frame: frame,
      id: viewId,
      dark: params?["dark"] as? Bool ?? false,
      radius: params?["radius"] as? Double,
      height: params?["height"] as? Double,
      messenger: messenger)
  }
}
private final class AppleButtonView: NSObject, FlutterPlatformView {
  let button: ASAuthorizationAppleIDButton
  let channel: FlutterMethodChannel
  init(frame: CGRect, id: Int64, dark: Bool, radius: Double?, height: Double?, messenger: FlutterBinaryMessenger) {
    button = ASAuthorizationAppleIDButton(type: .signIn, style: dark ? .white : .black)
    channel = FlutterMethodChannel(name: "yourtj/apple-button/\(id)", binaryMessenger: messenger)
    super.init()
    button.frame = frame
    button.cornerRadius = AppleButtonView.cornerRadius(radius: radius, height: height)
    button.addTarget(self, action: #selector(pressed), for: .touchUpInside)
  }
  /// The frame and the corner radius are the only adjustable properties of the
  /// official control. The factory is called with a zero frame and the engine
  /// only resizes the view afterwards through an autoresizing mask, so the pill
  /// radius is capped against the row height Flutter renders instead of the
  /// frame, which would collapse the button to a square.
  static func cornerRadius(radius: Double?, height: Double?) -> CGFloat {
    let requested = max(0, CGFloat(radius ?? 6))
    guard let rowHeight = height, rowHeight > 0 else { return requested }
    return min(requested, CGFloat(rowHeight) / 2)
  }
  func view() -> UIView { button }
  @objc private func pressed() { channel.invokeMethod("pressed", arguments: nil) }
}

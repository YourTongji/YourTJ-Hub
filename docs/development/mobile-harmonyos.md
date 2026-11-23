# 原生鸿蒙开发与验证

> Doc type: reference
>
> Status: Active
>
> Owner: Platform maintainers
>
> Last verified: 2026-11-20

`Partial`：应用通过 Flutter OH 和 ArkTS UIAbility 构建原生 HarmonyOS HAP，复用现有
Flutter 页面和设计系统。鸿蒙沿用手机端品牌启动层、减弱动态效果和存储就绪门控。
系统调用及桌面服务卡片位于
[原生模块](../../apps/mobile/packages/forum_app/ohos/entry/src/main/module.json5)。
签名真机、完整视觉与交互验收、Passkey 和应用市场分发尚未完成。
AppTest 邀请测试及固定群组更新入口见[发布文档](../operations/mobile-releases.md#群组分发与更新)。

## 应用身份

HarmonyOS 开发者中心的原生应用名称为 `YourTJ`，APP ID 为 `6917621118093079103`，
包名为 `tj.yourtj.forumapp`，支持手机、平板和 PC/2in1。Android 的
`tj.yourtj.forum_app` 与 iOS 的 `tj.yourtj.forumApp` 保持各自身份；华为要求原生鸿蒙
包名不能与 Android 相同，也不能使用 `harmonyos`、`ohos` 保留词，见
[应用创建规范](https://developer.huawei.com/consumer/cn/doc/app/agc-help-create-app-0000002247955506)。
公共品牌、应用介绍和隐私政策沿用现有移动端资料；邀请测试不代表应用已上架。

## 工具与空间边界

当前构建使用 Flutter OH `3.44.9+ohos-1.0.0-beta`（Flutter 提交 `c2fb91fb38c40a6b97b5ae3d9f71107b7dae7ad2`）、
华为 Linux Command Line Tools `26.0.0.865` 和 API 26 SDK。命令行工具从
[华为官方下载中心](https://developer.huawei.com/consumer/cn/download/command-line-tools-for-hmos)获取。
它与 Android/iOS 的标准 Flutter SDK 分开使用，不修改全局 Flutter 或 shell 配置。

在构建 shell 中将 Flutter OH、工具包自带 Node、命令行工具和 SDK toolchains 加入 PATH；
`NODE_HOME` 指向工具包的 `tool/node`，`DEVECO_SDK_HOME` 指向工具包的 `sdk`。
JDK 使用本机已安装且符合工具要求的版本。不要把 SDK、缓存、镜像、凭据或 HAP 加入源码。

手机验收只需一份 x64 手机系统镜像，各设备实例共享镜像。直板机实例使用 2 GB 内存、
4 GB 稀疏存储并关闭快照；不需要额外安装 Android 镜像或完整 DevEco Studio。
官方模拟器 CLI 的 `-help` 提供创建、启动、停止和删除实例命令。

## 构建与 ABI 检查

以下 Bash 命令在仓库根目录运行，复用既有环境参数生成器，生成未签名的 dev Release HAP：

```bash
set -euo pipefail
HARMONY_DEFINE_TEXT="$(MOBILE_ENVIRONMENT=dev MOBILE_PREVIEW=true \
  SOURCE_SHA="$(git rev-parse HEAD)" python3 scripts/mobile-release/build_environment.py)"
mapfile -t HARMONY_DEFINES <<< "$HARMONY_DEFINE_TEXT"
cd apps/mobile/packages/forum_app
flutter pub get
flutter build hap --release --no-pub --no-codesign \
  --target-platform=ohos-arm64 "${HARMONY_DEFINES[@]}"
python3 ohos/test/check_hap_abi.py \
  build/ohos/hap/entry-default-unsigned.hap --abi arm64-v8a --release
```

x64 模拟器改用 `--debug --target-platform=ohos-x64`，ABI 检查改为 `--abi x86_64`，
不传 `--release`。切换架构无需删除缓存；Hvigor 根据本次 `TARGET_PLATFORM` 排除其他 ABI，
防止上次 native-assets 构建留下的库混入 HAP。检查脚本验证库是否齐全、ELF 架构是否匹配，
并拒绝 glibc 依赖。构建成功不能代替这些产物检查，也不能证明签名或真机可用。

## 本地依赖来源与差异边界

`apps/mobile/third_party` 的三份源码以 pub.dev 对应版本的发布归档为基线。
升级时从版本 API 的 `archive_url` 获取归档，先校验 `archive_sha256`，再逐文件比较；
不要以版本号相同代替源码保真检查。

| 包 | 基线版本 | pub.dev 归档 SHA-256 |
|---|---|---|
| flutter_quill | 11.6.0 | `1f773f5f75e2628d80361c2c840c592791a0a157ebd587c9e37cf2cb8b8519b7` |
| flutter_math_fork | 0.7.4 | `6d5f2f1aa57ae539ffb0a04bb39d2da67af74601d685a161aff7ce5bda5fa407` |
| sqlite3 | 3.5.2 | `4c7fe79840389aaeaf05fd093f795b631b5a98e2bd28d54e555c100f4a9c7a1c` |

预期运行源码差异如下，包内 README 说明行为与撤除条件：

- Quill 修改 `raw_editor_state.dart` 的平台分支、`quill_native_provider.dart` 的桥选择、
  `default_clipboard_service.dart` 的元数据查询，新增 `harmony_quill_native_bridge.dart`。
- Math 修改 `line_editable.dart`、`selectable.dart` 和
  `gesture_detector_builder_selectable.dart` 的平台默认分支；共四处 switch 回退。
- SQLite 修改 `hook/build.dart`，新增 `hook/ohos.dart` 与 `assets/sqlite3mc/`；
  `pubspec.yaml` 移除上游 workspace resolution。其他 Dart 运行源码保持上游内容。

其余差异为 README 替换及少量行尾空格、空行清理；未省略 `lib/`、`hook/` 或上游运行资产。
为控制体积，副本省略上游示例、测试、开发文档与工具；Quill 还省略独立的 extensions/test
子包，这些不由当前本地包导出。副本不代表上游完整开发仓库，升级仍需运行应用侧回归。

SQLite3MC 归档来源和摘要见[本地 SQLite README](../../apps/mobile/third_party/sqlite3/README.md)。
其中主 C 源、`sqlite3mc_amalgamation.h`、`sqlite3.h` 与该归档逐字节一致；
`sqlite3ext.h` 仅三行注释的行尾空格被清理，归档 README 保持原样。

卡片使用系统声明的 30 分钟刷新（每天 48 次）与午夜刷新，不叠加五分钟轮询；同步和退出按目标类型刷新，
单卡失败不阻断其余卡片清空，仅明确不存在的 formId 会从注册表移除。系统后台刷新不保证课程切换即时到达。

## 原生与交互验证

[Hypium 测试](../../apps/mobile/packages/forum_app/ohos/entry/src/ohosTest/ets/test/List.test.ets)
覆盖课表投影、账号返回 URI、图片格式和加密存储。卡片测试通过真实原生存储验证三类卡片的
非空课程字段与退出后的数据清空，并在结束时恢复原存储值；不替代桌面添加、拖拽和尺寸视觉验收。
在应用的 `ohos` 目录运行：

```bash
hvigorw --mode module -p module=entry@ohosTest -p product=default \
  -p buildMode=test assembleHap --no-daemon
hdc install -r entry/build/default/outputs/ohosTest/entry-ohosTest-unsigned.hap
hdc shell aa test -b tj.yourtj.forumapp -m entry_test \
  -s unittest OpenHarmonyTestRunner -w 60
```

检查报告中的 Failure、Error 和 Pass，不能仅依赖 `aa test` 的退出码。
[Flutter 原生桥接测试](../../apps/mobile/packages/forum_app/integration_test/harmony_native_bridge_test.dart)
另行覆盖实际加密数据库、跨 isolate 重开和 Dart/ArkTS 调用。

[系统弹窗测试页面](../../apps/mobile/packages/forum_app/integration_test/harmony_system_dialog_probe.dart)
使用普通 Flutter 事件循环和合成数据，可作为 `flutter build hap` 的 `--target`。
分别点击保存图片后的“禁止”和“允许”，核对返回状态及 `cleanup=true`；
分享面板关闭应返回 `share:dismissed`。该页面只用于设备验收，测试后必须重新安装正常入口的 HAP。
自动化测试返回取消结果不足以证明系统弹窗实际显示。
该测试页还可开启、取消并重新开启网络监听，只显示连接类型枚举及事件次数，
不显示网络名称或地址。虚拟以太网下的订阅成功不能替代实际断网与恢复验收。

头像和封面拍照复用图片选择插件的系统 CameraPicker，不声明应用的 `CAMERA` 权限。
该系统界面由用户确认拍摄，无需应用直接操作摄像头，见
[CameraPicker 官方指南](https://developer.huawei.com/consumer/cn/doc/doccenter-capabilities/camera-picker)。
系统测试页提供独立拍照按钮，只解码返回图片并删除测试临时文件，不提交账号修改。
当前 Mate 70 模拟器未安装系统相机，调用返回 `camera_permission_error`，
插件内部出现 Promise 返回值缺失；拍照成功及取消流程仍须在具有相机能力的设备验证。

时间线卡片的日期按钮通过 FormExtension 消息切换本卡片的日期，不启动应用；
卡片正文通过既有今日校园或课表深链进入应用。日期选择按 form ID 保存，
不同卡片不共用选择状态。空课表文案不绑定“今天”，避免明天视图显示错误日期。
三类卡片均声明 `resizable: true`，支持在各自的 `supportDimensions` 中调整尺寸；
仅声明多个尺寸不会自动启用桌面拖拽，见
[ArkTS 卡片配置](https://developer.huawei.com/consumer/cn/doc/doccenter-capabilities/arkts-ui-widget-configuration)。

学校 WebView、键盘、系统返回、前后台恢复、主题及字号、卡片添加/刷新/日期切换/退出清理，
均需要独立交互证据。前后台恢复只重新绑定 Flutter 绘图缓冲区，不重建 WebView 或清空表单。
`settings_editors_test.dart` 使用仓库内的真实字体，对资料编辑、密码修改和账号关闭界面
逐像素比较 Android 与鸿蒙平台分支，覆盖明暗主题、320/390 逻辑宽度及窄屏 130% 系统字号。
该检查验证共享 Flutter 渲染的一致性，不代替设备字体、系统窗口和全部页面的视觉验收。

## 编辑器剪贴板

公式渲染沿用共享编辑器的 `flutter_math_fork` 0.7.4。本地
[适配副本](../../apps/mobile/third_party/flutter_math_fork/README.md) 仅为平台选择分支补上
Material 回退，以支持 Flutter OH 的额外平台；原有公式字体和布局保持不变。
升级该依赖时，需同时用标准 Flutter 和 Flutter OH 运行 Markdown 与富内容测试。

鸿蒙编辑器复用原有 Quill UI，通过原生 Pasteboard 处理 HTML、图片和 GIF。
工具栏只查询 MIME 元数据，不能定时读取正文或触发授权。用户粘贴时才申请
`READ_PASTEBOARD`，拒绝后保持空的富文本结果；其他原生错误不伪装成无数据。
复制限定为 `LOCALDEVICE`，不启用跨设备共享，桥接层不记录剪贴板正文。

图片输入和输出上限为 16 MiB，解码前检查至多 16 Mi 像素；超限失败而非静默截断。
GIF 保留独立的原始 MIME 记录以保存动画；未提供 GIF 记录的来源不能被当成 GIF。
文件型 HTML/Markdown 的剪贴板路径不声明支持，与现有 Android 插件能力保持一致。

系统测试页包含合成 HTML、PNG 像素和双帧 GIF 字节的往返检查，以及显式剪贴板权限
设置按钮。拒绝后不会在同一次粘贴中自动强行进入设置。恢复授权使用官方
[指定权限设置对话框](https://developer.huawei.com/consumer/cn/doc/doccenter-capabilities/request-user-authorization-second)。
通用应用详情采用 API 26 的官方接口，较低版本使用系统应用详情 Want；
API 26 的 Mate 70 模拟器已通过独立按钮验证，实际显示 YourTJ 系统应用详情及通知、网络、
剪贴板权限入口。API 18–25 的 Want 分支仍需对应系统验证。
系统测试页分别提供应用详情和剪贴板权限按钮，并显示等待、调用返回、缺少桥接和超时状态。
`returned` 只表示桥接调用完成；页面验收仍须确认系统界面实际出现。

## 品牌资源

鸿蒙原生启动标志使用 `assets/splash/brand_mark.png` 的透明副本 `startup_brand.png`；
卡片的 `widget_brand.png` 分别复用 Android `drawable`、`drawable-night` 的透明标志，
映射到鸿蒙 `base`、`dark` 资源目录。应用图标 `icon.png` 带白底，不能用于这两个位置。
`launcher_icons_test.dart` 检查资源引用、透明通道及副本与源文件一致；更新品牌源图时须同步副本。

## 外部接入边界

Push Kit 的设备注册和实际投递需要 AGC 应用、客户端配置及服务端凭据；
未配置时不得以模拟器测试代替真实推送验收。凭据仅通过本地文件配置，不写入仓库或客户端。
Passkey 必须先完成 AGC App Linking 开通、服务器域名关联、AGC 关联域名及模块的域名配置，
见[华为通行密钥开发准备](https://developer.huawei.com/consumer/cn/doc/doccenter-capabilities/onlineauthentication-passwordless-auth-preparation)。
不应为了使未配置的客户端通过验收而放宽服务端 WebAuthn origin 校验。
Passkey、域名关联、正式签名及 AppGallery 发布不能由未签名 HAP 构建证明完成。

`Partial`：登录和账号安全页已接入原生 `OnlineAuthenticationKit.fido2` 桥接，
复用现有通行密钥界面、密码/TOTP 再认证和服务端挑战流程。API 20 以下或无 FIDO2
系统能力时保持不可用；能力查询失败保留重试，不冒充“不支持”。同一原生实例只允许一个
认证操作，用户取消、拒绝和超时复用现有安全提示，不向界面暴露系统诊断。
接口及错误码以 [ArkTS FIDO2 API](https://developer.huawei.com/consumer/cn/doc/harmonyos-references/onlineauthentication-passkey-api)
和固定 SDK 声明为准。

桥接将服务器的 Base64URL 字段转换成系统要求的字节数组，将响应中的 `clientDataJson`
字段映射为 WebAuthn 的 `clientDataJSON`。转换只调整二进制编码，不解析或重写签名数据。
单元测试覆盖断言、注册、取消、响应格式和空扩展；Hypium 覆盖原生解码及长度边界。

网站 `/.well-known/applinking.json` 由论坛按实例配置提供。在
`deploy/instances/<env>.json` 的 `passkeys` 对象中设置 `harmony_app_identifier`，
渲染器会写入对应 TOML 字段。该值默认留空，端点返回 404；配置后仅关联指定应用。
公开应用标识不是密钥，但填写前必须核对签名 profile 中的 `app-identifier`。
生产实例已配置与发布 profile 一致的公开标识 `6917621118093079103`，dev 留空。
网站关联文件须随正常部署生效，AGC App Linking 和关联域名仍须独立验证。

原生模块已为 `f.yourtj.de` 和 `dev.yourtj.de` 声明 `domainVerify`，
路径限定为 `/auth/passkey`，不接管普通论坛网页链接。系统能力查询成功不能证明域名验证通过。
关联文件应使用已签名应用的 `appIdentifier`，不得用包名或 Android 证书指纹代替，见
[App Linking 指南](https://developer.huawei.com/consumer/en/doc/HarmonyOS-Guides/app-linking-startup/)。
必须在域名验证为 `success` 后，用签名包验证注册及登录，再决定真实 `origin` 是否需要
显式增加到服务端允许列表。官方 ArkTS 接口没有调用方指定 `origin` 的参数，
当前未扩展服务端来源白名单，也未改动生产认证配置。域名及签名身份准备好后，用
`hdc shell hidumper -s AppDomainVerifyManager` 检查对应域名的验证结果。

## 服务端推送鉴权与拒绝处理

原生鸿蒙使用 Push Kit v3 场景化推送：服务账号以 PS256 签出 JWT，直接放入
`Authorization: Bearer <JWT>`；`aud` 的 OAuth 地址是 JWT 声明，不代表需要交换 access token。
HarmonyOS 5 起该接口不再支持 OAuth 2.0 鉴权，见
[官方请求头规范](https://developer.huawei.com/consumer/en/doc/doccenter-references/api/push-scenariozed-api-request-struct/)
和[服务账号 JWT 指南](https://developer.huawei.com/consumer/en/doc/doccenter-capabilities/push-jwt-token/)。

服务端不会仅凭 `80300007` 删除设备注册：该码同时涵盖项目不匹配、推送权限缺失、
地区限制与 token 格式错误等原因，见
[官方响应码说明](https://developer.huawei.com/consumer/en/doc/doccenter-references/api/push-scenariozed-api-response/)。
拒绝按 best-effort 失败处理，当前通知不自动重试；保留注册也意味着无效 token 可能继续占用
每用户设备配额。排查时应先核对项目、权限与客户端重新注册，不把该码等同于 APNs 的永久失效。
原生推送仍为 `Partial`：协议契约测试不代表真实设备注册、送达或点击已验收。

## 原生 CI 与发布验收

`CI / native HarmonyOS` 对相关同仓库 PR 使用固定 Flutter OH 和经 SHA-256 校验的官方 SDK，
分别编译 ARM64/x64 无签名 Release HAP，运行 Flutter 检查和原生投影模型测试，并核验各包 ABI。
它不使用发布证书或服务账号；SDK 下载 Secret 缺失时明确失败，不能把未编译写成通过。
共享 Flutter 改动同样触发鸿蒙检查。标准 Flutter 的测试不能代替此原生编译。

该 CI 尚未运行系统模拟器或 Hypium 设备测试。SDK 编译、签名验签、模拟器交互和真机安装是
不同证据；相机、真实推送、覆盖升级与 AppTest 自动更新仍需设备验收。
统一 RC、dev、送审、恢复和历史目录的事实源见[移动发布流程](../operations/mobile-releases.md#harmonyos-app-test)。

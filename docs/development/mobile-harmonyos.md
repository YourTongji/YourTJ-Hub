# 原生鸿蒙开发与验证

> Doc type: reference
>
> Status: Active
>
> Owner: Platform maintainers
>
> Last verified: 2026-11-07

`Partial`：应用通过 Flutter OH 和 ArkTS UIAbility 构建原生 HarmonyOS HAP，复用现有
Flutter 页面和设计系统。鸿蒙沿用手机端品牌启动层、减弱动态效果和存储就绪门控。
系统调用及桌面服务卡片位于
[原生模块](../../apps/mobile/packages/forum_app/ohos/entry/src/main/module.json5)。
签名真机、完整视觉与交互验收、Passkey 和应用市场分发尚未完成。

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

## 原生与交互验证

[Hypium 测试](../../apps/mobile/packages/forum_app/ohos/entry/src/ohosTest/ets/test/List.test.ets)
覆盖课表投影、账号返回 URI、图片格式和加密存储。在应用的 `ohos` 目录运行：

```bash
hvigorw --mode module -p module=entry@ohosTest -p product=default \
  -p buildMode=test assembleHap --no-daemon
hdc install -r entry/build/default/outputs/ohosTest/entry-ohosTest-unsigned.hap
hdc shell aa test -b tj.yourtj.forum_app -m entry_test \
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
当前模拟器未打开该详情页面，不能把调用返回当作页面验收通过。

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

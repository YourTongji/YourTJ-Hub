import 'package:flutter/widgets.dart';

/// campus 隐私内容的前台判定：仅 paused/hidden/detached 视为被覆盖——此时
/// 丢弃隐私子树、suspend 数据、冲刷内存缓存。inactive（通知栏、控制中心、
/// 来电等瞬时遮挡，内容仍在屏上）不触发重建，避免每次遮挡都整页骨架屏
/// 重载。
///
/// 取舍：Flutter 自 hidden/paused 起停用帧调度，卸载产生的空白帧不会绘制，
/// 应用切换器缩略图可能保留含隐私内容的最后一帧；恢复前台仍先核验绑定再
/// 取数。取舍与备选方案见 MADR 0092。
bool isCampusForeground(AppLifecycleState? state) =>
    state == null ||
    state == AppLifecycleState.resumed ||
    state == AppLifecycleState.inactive;

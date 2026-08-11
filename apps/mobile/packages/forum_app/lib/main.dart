import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'src/app.dart';
import 'src/router.dart';

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();
  // 装载跨重启缓存门禁:标记存在(上次缓存清理失败且令牌可能残留)时,
  // appRouter 会强制重定向到登录页,残留会话无法进入 shell。
  await initStartupGate();
  runApp(const ProviderScope(child: GfApp()));
}

import 'dart:io';

import 'package:code_assets/code_assets.dart';
import 'package:hooks/hooks.dart';
import 'package:sqlite3/src/hook/compile/description.dart';

bool isOhosCompiler(Uri? compiler) =>
    compiler?.path
        .replaceAll('\\', '/')
        .contains('/openharmony/native/llvm/bin/') ??
    false;

Future<void> buildOhosSqlite(
  BuildInput input,
  BuildOutputBuilder output,
) async {
  final compiler = input.config.code.cCompiler!.compiler!;
  final nativeRoot = compiler.resolve('../../');
  final triple = switch (input.config.code.targetArchitecture) {
    Architecture.arm64 => 'aarch64-linux-ohos',
    Architecture.x64 => 'x86_64-linux-ohos',
    _ => throw UnsupportedError('Unsupported HarmonyOS SQLite architecture'),
  };
  final source = input.packageRoot.resolve(
    'assets/sqlite3mc/sqlite3mc_amalgamation.c',
  );
  final library = input.outputDirectory.resolve('libsqlite3mc.so');
  final defines = CompilerDefines.parse(input.userDefines, OS.linux);
  final result = await Process.run(compiler.toFilePath(), [
    '--target=$triple',
    '--sysroot=${nativeRoot.resolve('sysroot').toFilePath()}',
    '-shared',
    '-fPIC',
    '-O2',
    '-D_GNU_SOURCE=1',
    for (final entry in defines.entries)
      '-D${entry.key}${entry.value == null ? '' : '=${entry.value}'}',
    '-Wl,-Bsymbolic',
    '-Wl,-soname,libsqlite3mc.so',
    source.toFilePath(),
    '-o',
    library.toFilePath(),
    '-lm',
    '-ldl',
    '-pthread',
  ]);
  if (result.exitCode != 0)
    throw StateError('HarmonyOS SQLite compilation failed: ${result.stderr}');
  output.dependencies.addAll([
    source,
    source.resolve('sqlite3mc_amalgamation.h'),
    source.resolve('sqlite3.h'),
    source.resolve('sqlite3ext.h'),
  ]);
  output.assets.code.add(
    CodeAsset(
      package: 'sqlite3',
      name: 'src/ffi/libsqlite3.g.dart',
      linkMode: DynamicLoadingBundled(),
      file: library,
    ),
  );
}

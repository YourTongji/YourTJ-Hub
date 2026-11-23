import 'package:flutter_test/flutter_test.dart';

import '../../../third_party/sqlite3/hook/ohos.dart';

void main() {
  test(
    'only an OpenHarmony SDK compiler selects the native encrypted build',
    () {
      expect(isOhosCompiler(null), isFalse);
      expect(isOhosCompiler(Uri.file('/usr/bin/clang')), isFalse);
      expect(
        isOhosCompiler(Uri.file('/sdk/android/ndk/llvm/bin/clang')),
        isFalse,
      );
      expect(
        isOhosCompiler(
          Uri.file('/sdk/default/openharmony/native/llvm/bin/clang'),
        ),
        isTrue,
      );
      expect(
        isOhosCompiler(
          Uri.parse(
            'file:///C:/Sdk/default/openharmony/native/llvm/bin/clang.exe',
          ),
        ),
        isTrue,
      );
    },
  );
}

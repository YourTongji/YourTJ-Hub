import 'dart:async';
import 'dart:io';

/// Windows refuses to delete a file while any other process still has it open,
/// and antivirus or indexer scans of files a test just wrote are enough to hold
/// one for a moment. The delete of the whole directory then fails with a
/// sharing violation (`PathAccessException`, errno 32) that clears by itself —
/// see issue #1138. Delays double so a lock that clears quickly costs the suite
/// almost nothing, while a slow scan is still covered.
const _lockBackoff = <Duration>[
  Duration(milliseconds: 25),
  Duration(milliseconds: 50),
  Duration(milliseconds: 100),
  Duration(milliseconds: 200),
  Duration(milliseconds: 400),
  Duration(milliseconds: 800),
  Duration(milliseconds: 1600),
  Duration(milliseconds: 2000),
  Duration(milliseconds: 2000),
  Duration(milliseconds: 2000),
  Duration(milliseconds: 2000),
];

/// Windows `ERROR_SHARING_VIOLATION`, the errno a delete reports while another
/// process holds a file in the directory open.
const _sharingViolation = 32;

/// Removes a test-owned temporary directory and everything in it.
///
/// Cleanup is scaffolding, not something a test asserts, so this helper never
/// fails the test that owns the directory: a delete the OS refuses is retried
/// while it reports a sharing violation (the antivirus or indexer scan of
/// issue #1138), for about eleven seconds, and then — like any other failure —
/// it is printed and the directory is left behind. That print is what keeps the
/// failure mode visible: a handle the test itself leaked is exactly what makes
/// the delete fail, and it still shows up in the output, it just does not turn
/// into a test failure. A test that needs to prove its own handles are released
/// should assert that directly instead of riding on delete semantics — macOS
/// and Linux unlink open files happily, so this delete never had anything to
/// say there anyway.
///
/// A directory that is already gone counts as cleaned up and prints nothing.
Future<void> deleteTempDirectory(
  Directory directory, {
  List<Duration> backoff = _lockBackoff,
}) async {
  for (var attempt = 0; ; attempt++) {
    try {
      await directory.delete(recursive: true);
      return;
    } on FileSystemException catch (error) {
      if (error is PathNotFoundException) return;
      if (!_isSharingViolation(error) || attempt >= backoff.length) {
        // Cleanup reports belong in the test output, not in an app logger.
        // ignore: avoid_print
        print(
          'deleteTempDirectory: leaving ${directory.path} behind after '
          'cleanup failed ($error); the test result is not affected.',
        );
        return;
      }
      await Future<void>.delayed(backoff[attempt]);
    }
  }
}

bool _isSharingViolation(FileSystemException error) =>
    error is PathAccessException &&
    error.osError?.errorCode == _sharingViolation;

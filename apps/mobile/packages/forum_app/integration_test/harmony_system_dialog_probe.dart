import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'package:flutter/material.dart';
import 'package:connectivity_plus/connectivity_plus.dart';
import 'package:flutter/services.dart';
import 'package:flutter_quill/flutter_quill.dart';
import 'package:image/image.dart' as images;
import 'package:image_picker/image_picker.dart';
import 'package:gal/gal.dart';
import 'package:path_provider/path_provider.dart';
import 'package:share_plus/share_plus.dart';

/// Manual device acceptance entrypoint; contains only synthetic local data.
void main() => runApp(const MaterialApp(home: SystemDialogProbe()));

class SystemDialogProbe extends StatefulWidget {
  const SystemDialogProbe({super.key});
  @override
  State<SystemDialogProbe> createState() => _SystemDialogProbeState();
}

class _SystemDialogProbeState extends State<SystemDialogProbe> {
  String _status = 'ready';
  bool _busy = false;
  StreamSubscription<List<ConnectivityResult>>? _network;
  int _networkEvents = 0;

  Future<void> _toggleNetwork() async {
    final previous = _network;
    if (previous != null) {
      _network = null;
      await previous.cancel();
      if (mounted) setState(() => _status = 'network:stopped');
      return;
    }
    _networkEvents = 0;
    try {
      final current = await Connectivity().checkConnectivity();
      if (!mounted) return;
      setState(
        () => _status = 'network:check=${current.map((v) => v.name).join(",")}',
      );
      _network = Connectivity().onConnectivityChanged.listen(
        (values) {
          if (!mounted) return;
          _networkEvents++;
          setState(
            () => _status =
                'network:events=$_networkEvents:${values.map((v) => v.name).join(",")}',
          );
        },
        onError: (Object error) {
          if (mounted) {
            setState(() => _status = 'network:failure:${error.runtimeType}');
          }
        },
      );
    } on PlatformException catch (error) {
      if (mounted) setState(() => _status = 'network:${error.code}');
    }
  }

  @override
  void dispose() {
    _network?.cancel();
    super.dispose();
  }

  Future<void> _camera() async {
    setState(() {
      _busy = true;
      _status = 'camera:waiting';
    });
    String status;
    try {
      final photo = await ImagePicker().pickImage(source: ImageSource.camera);
      if (photo == null) {
        status = 'camera:cancelled';
      } else {
        try {
          final decoded = images.decodeImage(await photo.readAsBytes());
          status =
              'camera:decoded=${decoded != null && decoded.width > 0 && decoded.height > 0}';
        } finally {
          // The probe owns the returned temporary capture, never the user's gallery.
          await File(photo.path).delete();
        }
      }
    } on PlatformException catch (error) {
      status = 'camera:${error.code}';
    } catch (error) {
      status = 'camera:failure:${error.runtimeType}';
    }
    if (mounted) {
      setState(() {
        _busy = false;
        _status = status;
      });
    }
  }

  Future<void> _run(bool gallery) async {
    setState(() {
      _busy = true;
      _status = 'waiting';
    });
    String status;
    try {
      if (gallery) {
        final cache = await getTemporaryDirectory();
        Future<Set<String>> saves() async => {
          await for (final entry in cache.list())
            if (entry.path.split('/').last.startsWith('yourtj-save-'))
              entry.path,
        };
        final before = await saves();
        try {
          await Gal.putImageBytes(
            base64Decode(
              'iVBORw0KGgoAAAANSUhEUgAAAAIAAAACCAIAAAD91JpzAAAAEElEQVR4nGP8zwACTGCSAQANHQEDgslx/wAAAABJRU5ErkJggg==',
            ),
            name: 'YourTJ synthetic acceptance',
          );
          status = 'gallery:saved';
        } on GalException catch (error) {
          status =
              'gallery:${error.type.name}:${error.platformException.details}';
        }
        final after = await saves();
        status +=
            ':cleanup=${before.length == after.length && before.containsAll(after)}';
      } else {
        final result = await SharePlus.instance.share(
          ShareParams(
            text: 'YourTJ synthetic share acceptance',
            title: 'YourTJ native test',
          ),
        );
        status = 'share:${result.status.name}';
      }
    } catch (error) {
      status = 'failure:${error.runtimeType}';
    }
    if (mounted) {
      setState(() {
        _busy = false;
        _status = status;
      });
    }
  }

  Future<void> _clipboard() async {
    setState(() {
      _busy = true;
      _status = 'clipboard:waiting';
    });
    const channel = MethodChannel('yourtj/clipboard');
    const html = '<p>YourTJ <b>synthetic clipboard</b></p>';
    String status;
    try {
      await channel.invokeMethod<void>('copyHtml', html);
      final present = await channel.invokeMethod<bool>('hasData');
      final copiedHtml = await channel.invokeMethod<String>('getHtml');
      final controller = QuillController.basic();
      bool editorMatches;
      try {
        // Acceptance covers the editor's experimental paste entrypoint.
        // ignore: experimental_member_use
        final pasted = await controller.clipboardPaste();
        editorMatches =
            pasted &&
            controller.document.toPlainText().trimRight() ==
                'YourTJ synthetic clipboard' &&
            controller.document.toDelta().toJson().any(
              (operation) =>
                  operation['insert'] == 'synthetic clipboard' &&
                  (operation['attributes'] as Map?)?['bold'] == true,
            );
      } finally {
        controller.dispose();
      }
      final png = base64Decode(
        'iVBORw0KGgoAAAANSUhEUgAAAAIAAAACCAIAAAD91JpzAAAAEElEQVR4nGP8zwACTGCSAQANHQEDgslx/wAAAABJRU5ErkJggg==',
      );
      await channel.invokeMethod<void>('copyImage', png);
      final copiedPng = await channel.invokeMethod<Uint8List>('getImage');
      final decoded = copiedPng == null ? null : images.decodeImage(copiedPng);
      final pngMatches =
          decoded != null &&
          decoded.width == 2 &&
          decoded.height == 2 &&
          decoded.getPixel(0, 0).r == 255 &&
          decoded.getPixel(0, 0).g == 0;
      final gif = base64Decode(
        'R0lGODlhAgACAIEAAP8AAAAAAAAAAAAAACH/C05FVFNDQVBFMi4wAwEAAAAh+QQACgAAACwAAAAAAgACAAAIBgABCAQQEAAh+QQBCgABACwAAAAAAgACAIEAAP8AAAAAAAAAAAAIBgABCAQQEAA7',
      );
      await channel.invokeMethod<void>('copyImage', gif);
      final copiedGif = await channel.invokeMethod<Uint8List>('getGif');
      final gifMatches =
          copiedGif != null && base64Encode(copiedGif) == base64Encode(gif);
      status =
          'clipboard:html=${present == true && copiedHtml == html}:editor=$editorMatches:png=$pngMatches:gif=$gifMatches';
    } on PlatformException catch (error) {
      status = 'clipboard:${error.code}';
    } catch (error) {
      status = 'clipboard:failure:${error.runtimeType}';
    }
    if (mounted) {
      setState(() {
        _busy = false;
        _status = status;
      });
    }
  }

  Future<void> _settings() async {
    try {
      await const MethodChannel(
        'com.spencerccf.app_settings/methods',
      ).invokeMethod<void>('openSettings', {'type': 'clipboard'});
      if (mounted) setState(() => _status = 'settings:opened');
    } on PlatformException catch (error) {
      if (mounted) setState(() => _status = 'settings:${error.code}');
    }
  }

  @override
  Widget build(BuildContext context) => Scaffold(
    appBar: AppBar(title: const Text('YourTJ system acceptance')),
    body: SafeArea(
      child: Column(
        children: [
          TextButton(
            onPressed: _busy ? null : () => _run(false),
            child: const Text('Test native share'),
          ),
          TextButton(
            onPressed: _busy ? null : () => _run(true),
            child: const Text('Test native image save'),
          ),
          TextButton(
            onPressed: _busy ? null : _clipboard,
            child: const Text('Test native clipboard'),
          ),
          TextButton(
            onPressed: _busy ? null : _settings,
            child: const Text('Test clipboard permission settings'),
          ),
          TextButton(
            onPressed: _busy ? null : _camera,
            child: const Text('Test native camera'),
          ),
          TextButton(
            onPressed: _busy ? null : _toggleNetwork,
            child: const Text('Toggle native network listener'),
          ),
          Text(_status),
        ],
      ),
    ),
  );
}

import 'package:flutter/material.dart';

/// The existing YourTJ cat mark, shared with the native launcher artwork.
class YourTjMark extends StatelessWidget {
  const YourTjMark({super.key});

  @override
  Widget build(BuildContext context) => ClipRRect(
    borderRadius: BorderRadius.circular(10),
    child: Image.asset(
      'assets/launcher/icon.png',
      width: 40,
      height: 40,
      fit: BoxFit.contain,
      cacheWidth: (40 * MediaQuery.devicePixelRatioOf(context)).ceil(),
      semanticLabel: 'YourTJ',
    ),
  );
}

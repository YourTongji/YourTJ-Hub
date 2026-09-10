import 'package:flutter/material.dart';

import 'gf_input.dart';

/// Multi-line text area aligned with web `.gf-textarea` (components.css):
/// 1px line border, radius field, base-100 fill, 12px padding, 14px text.
/// Same focus ring behavior as [GfInput].
class GfTextarea extends GfInput {
  const GfTextarea({
    super.key,
    super.controller,
    super.focusNode,
    super.hintText,
    super.labelText,
    super.onChanged,
    super.onSubmitted,
    super.maxLength,
    super.enabled,
    super.autofocus,
    super.textAlignVertical,
    super.minLines = 4,
    super.maxLines = 8,
  }) : super(
         prefixIcon: null,
         suffixIcon: null,
         obscureText: false,
         keyboardType: TextInputType.multiline,
         textInputAction: TextInputAction.newline,
       );
}

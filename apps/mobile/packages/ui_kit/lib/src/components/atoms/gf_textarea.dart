import 'package:flutter/material.dart';

import 'gf_input.dart';

/// Multi-line form field sharing the rounded surface and states of [GfInput].
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

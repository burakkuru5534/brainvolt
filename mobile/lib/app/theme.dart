import 'package:flutter/material.dart';

/// Brand colour; Material 3 derives the rest of the scheme from it.
const brandSeed = Color(0xFF5B3FD9);

/// Light and dark themes.
ThemeData buildTheme(Brightness brightness) => ThemeData(
  colorScheme: ColorScheme.fromSeed(
    seedColor: brandSeed,
    brightness: brightness,
  ),
  useMaterial3: true,
);

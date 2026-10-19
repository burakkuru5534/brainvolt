import 'package:flutter/material.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

import 'providers.dart';

part 'theme.g.dart';

/// Brand purple; Material 3 derives the remaining roles from it.
const brandSeed = Color(0xFF6A3BE4);

/// The bundled typeface (Nunito, SIL OFL; assets/fonts).
const brandFont = 'Nunito';

/// Design tokens of the BrainVolt look (D-085) beyond Material's roles.
@immutable
class BvColors extends ThemeExtension<BvColors> {
  const BvColors({
    required this.background,
    required this.card,
    required this.cardBorder,
    required this.ink,
    required this.muted,
    required this.badge,
    required this.badgeInk,
    required this.gradientStart,
    required this.gradientEnd,
    required this.selected,
    required this.success,
    required this.danger,
    required this.glow,
  });

  static const light = BvColors(
    background: Color(0xFFF5F2FF),
    card: Colors.white,
    cardBorder: Color(0xFFE6DEFA),
    ink: Color(0xFF22114D),
    muted: Color(0xFF8574B4),
    badge: Color(0xFFEEE7FF),
    badgeInk: Color(0xFF6A3BE4),
    gradientStart: Color(0xFF8B5CF6),
    gradientEnd: Color(0xFF5B21D6),
    selected: Color(0xFF2A1166),
    success: Color(0xFF22C55E),
    danger: Color(0xFFEF4444),
    glow: Color(0x4D7C3AED),
  );

  static const dark = BvColors(
    background: Color(0xFF0B0716),
    card: Color(0xFF151024),
    cardBorder: Color(0xFF3B2A6B),
    ink: Colors.white,
    muted: Color(0xFFB3A3E6),
    badge: Color(0xFF241A3F),
    badgeInk: Color(0xFFB794FF),
    gradientStart: Color(0xFF8B5CF6),
    gradientEnd: Color(0xFF5B21D6),
    selected: Color(0xFF8B5CF6),
    success: Color(0xFF22C55E),
    danger: Color(0xFFF87171),
    glow: Color(0x807C3AED),
  );

  /// Page background.
  final Color background;
  final Color card;
  final Color cardBorder;

  /// Headings and body text.
  final Color ink;

  /// Secondary text.
  final Color muted;

  /// Round icon badges.
  final Color badge;
  final Color badgeInk;

  /// Primary buttons and hero cards.
  final Color gradientStart;
  final Color gradientEnd;

  /// A chosen tile.
  final Color selected;
  final Color success;
  final Color danger;

  /// Shadow under gradient buttons.
  final Color glow;

  LinearGradient get gradient => LinearGradient(
    begin: Alignment.topLeft,
    end: Alignment.bottomRight,
    colors: [gradientStart, gradientEnd],
  );

  static BvColors of(BuildContext context) =>
      Theme.of(context).extension<BvColors>() ?? light;

  @override
  BvColors copyWith() => this;

  @override
  BvColors lerp(BvColors? other, double t) =>
      other == null || t < 0.5 ? this : other;
}

/// [style] in [weight]: the variable font needs the `wght` axis set too.
TextStyle weighted(TextStyle style, FontWeight weight) => style.copyWith(
  fontWeight: weight,
  fontVariations: [FontVariation.weight(weight.value.toDouble())],
);

TextTheme _text(TextTheme base, Color ink, Color muted) {
  TextStyle s(TextStyle? t, FontWeight w, [Color? c]) =>
      weighted(t!, w).copyWith(color: c ?? ink, fontFamily: brandFont);
  return base.copyWith(
    displaySmall: s(base.displaySmall, FontWeight.w900),
    headlineLarge: s(base.headlineLarge, FontWeight.w900),
    headlineMedium: s(base.headlineMedium, FontWeight.w900),
    headlineSmall: s(base.headlineSmall, FontWeight.w800),
    titleLarge: s(base.titleLarge, FontWeight.w800),
    titleMedium: s(base.titleMedium, FontWeight.w800),
    titleSmall: s(base.titleSmall, FontWeight.w700),
    bodyLarge: s(base.bodyLarge, FontWeight.w600),
    bodyMedium: s(base.bodyMedium, FontWeight.w600),
    bodySmall: s(base.bodySmall, FontWeight.w600, muted),
    labelLarge: s(base.labelLarge, FontWeight.w800),
    labelMedium: s(base.labelMedium, FontWeight.w700),
    labelSmall: s(base.labelSmall, FontWeight.w700, muted),
  );
}

/// Light and dark themes (D-085).
ThemeData buildTheme(Brightness brightness) {
  final bv = brightness == Brightness.light ? BvColors.light : BvColors.dark;
  final scheme =
      ColorScheme.fromSeed(
        seedColor: brandSeed,
        brightness: brightness,
      ).copyWith(
        primary: brightness == Brightness.light
            ? brandSeed
            : const Color(0xFF9B6BFF),
        surface: bv.card,
        onSurface: bv.ink,
        error: bv.danger,
      );
  final base = ThemeData(
    colorScheme: scheme,
    brightness: brightness,
    useMaterial3: true,
    fontFamily: brandFont,
  );
  const radius = BorderRadius.all(Radius.circular(20));
  const stadium = WidgetStatePropertyAll<OutlinedBorder>(StadiumBorder());
  final label = WidgetStatePropertyAll(
    weighted(
      const TextStyle(fontSize: 16, fontFamily: brandFont),
      FontWeight.w800,
    ),
  );
  return base.copyWith(
    scaffoldBackgroundColor: bv.background,
    canvasColor: bv.background,
    textTheme: _text(base.textTheme, bv.ink, bv.muted),
    extensions: [bv],
    appBarTheme: AppBarTheme(
      backgroundColor: bv.background,
      surfaceTintColor: Colors.transparent,
      foregroundColor: bv.ink,
      elevation: 0,
      centerTitle: true,
      titleTextStyle: weighted(
        TextStyle(fontSize: 22, color: bv.ink, fontFamily: brandFont),
        FontWeight.w900,
      ),
    ),
    cardTheme: CardThemeData(
      color: bv.card,
      surfaceTintColor: Colors.transparent,
      elevation: 0,
      shape: RoundedRectangleBorder(
        borderRadius: radius,
        side: BorderSide(color: bv.cardBorder),
      ),
    ),
    filledButtonTheme: FilledButtonThemeData(
      style: ButtonStyle(
        shape: stadium,
        minimumSize: const WidgetStatePropertyAll(Size(64, 52)),
        textStyle: label,
      ),
    ),
    outlinedButtonTheme: OutlinedButtonThemeData(
      style: ButtonStyle(
        shape: stadium,
        minimumSize: const WidgetStatePropertyAll(Size(64, 52)),
        textStyle: label,
        foregroundColor: WidgetStatePropertyAll(bv.ink),
        backgroundColor: WidgetStatePropertyAll(bv.card),
        side: WidgetStatePropertyAll(
          BorderSide(color: bv.cardBorder, width: 1.5),
        ),
      ),
    ),
    textButtonTheme: TextButtonThemeData(style: ButtonStyle(textStyle: label)),
    inputDecorationTheme: InputDecorationTheme(
      filled: true,
      fillColor: bv.card,
      border: OutlineInputBorder(
        borderRadius: const BorderRadius.all(Radius.circular(16)),
        borderSide: BorderSide(color: bv.cardBorder),
      ),
      enabledBorder: OutlineInputBorder(
        borderRadius: const BorderRadius.all(Radius.circular(16)),
        borderSide: BorderSide(color: bv.cardBorder),
      ),
    ),
    bottomSheetTheme: BottomSheetThemeData(
      backgroundColor: bv.background,
      surfaceTintColor: Colors.transparent,
      showDragHandle: true,
      dragHandleColor: bv.cardBorder,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(32)),
      ),
    ),
    dialogTheme: DialogThemeData(
      backgroundColor: bv.card,
      surfaceTintColor: Colors.transparent,
      shape: const RoundedRectangleBorder(borderRadius: radius),
    ),
    listTileTheme: ListTileThemeData(iconColor: bv.badgeInk),
    progressIndicatorTheme: ProgressIndicatorThemeData(
      color: scheme.primary,
      linearTrackColor: bv.badge,
      linearMinHeight: 8,
      borderRadius: const BorderRadius.all(Radius.circular(8)),
    ),
    chipTheme: ChipThemeData(
      backgroundColor: bv.badge,
      side: BorderSide.none,
      shape: const StadiumBorder(),
      labelStyle: weighted(
        TextStyle(color: bv.badgeInk, fontFamily: brandFont),
        FontWeight.w800,
      ),
    ),
    dividerTheme: DividerThemeData(color: bv.cardBorder),
  );
}

const _themePrefsKey = 'theme_mode';

/// The user's light / dark choice; system by default (D-085).
@Riverpod(keepAlive: true)
class ThemeModeSetting extends _$ThemeModeSetting {
  @override
  ThemeMode build() {
    final v = ref.watch(sharedPreferencesProvider).getString(_themePrefsKey);
    return ThemeMode.values.where((m) => m.name == v).firstOrNull ??
        ThemeMode.system;
  }

  Future<void> set(ThemeMode mode) async {
    await ref
        .read(sharedPreferencesProvider)
        .setString(_themePrefsKey, mode.name);
    state = mode;
  }
}

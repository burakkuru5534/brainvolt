import 'package:flutter/cupertino.dart' show CupertinoPageTransitionsBuilder;
import 'package:flutter/material.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

import 'providers.dart';

part 'theme.g.dart';

/// Brand purple; Material 3 derives the remaining roles from it.
const brandSeed = Color(0xFF7C3AED);

/// The bundled typeface (Nunito, SIL OFL; assets/fonts).
const brandFont = 'Nunito';

/// Design tokens of the BrainVolt look (D-099) beyond Material's roles.
@immutable
class BvColors extends ThemeExtension<BvColors> {
  const BvColors({
    required this.background,
    required this.backgroundTop,
    required this.backgroundBottom,
    required this.surface,
    required this.card,
    required this.cardBorder,
    required this.ink,
    required this.muted,
    required this.badge,
    required this.badgeInk,
    required this.gradientStart,
    required this.gradientMid,
    required this.gradientEnd,
    required this.accent,
    required this.gold,
    required this.selected,
    required this.success,
    required this.danger,
    required this.glow,
  });

  /// BrainVolt: deep navy star field, frosted glass, purple → orange.
  static const witclash = BvColors(
    background: Color(0xFF0D0B26),
    backgroundTop: Color(0xFF16124A),
    backgroundBottom: Color(0xFF07061A),
    surface: Color(0xFF1A1640),
    card: Color(0x17FFFFFF),
    cardBorder: Color(0x26FFFFFF),
    ink: Colors.white,
    muted: Color(0xFFA9A4D8),
    badge: Color(0x339B6BFF),
    badgeInk: Color(0xFFCDB8FF),
    gradientStart: Color(0xFF7C3AED),
    gradientMid: Color(0xFFD9468F),
    gradientEnd: Color(0xFFFF8A3D),
    accent: Color(0xFFFF8A3D),
    gold: Color(0xFFFBBF24),
    selected: Color(0xFF7C3AED),
    success: Color(0xFF22C55E),
    danger: Color(0xFFF43F5E),
    glow: Color(0x667C3AED),
  );

  /// BrainVolt has one look; both names keep older call sites working.
  static const light = witclash;
  static const dark = witclash;

  /// Page background (solid fallback under the star field).
  final Color background;
  final Color backgroundTop;
  final Color backgroundBottom;

  /// Opaque panels: dialogs, sheets, menus, snack bars.
  final Color surface;

  /// Frosted glass cards.
  final Color card;
  final Color cardBorder;

  /// Headings and body text.
  final Color ink;

  /// Secondary text.
  final Color muted;

  /// Round icon badges.
  final Color badge;
  final Color badgeInk;

  /// Primary buttons and hero cards: purple → magenta → orange.
  final Color gradientStart;
  final Color gradientMid;
  final Color gradientEnd;

  /// Highlights (timers, streaks, the "Clash" half of the word mark).
  final Color accent;

  /// Ranks, crowns, rewards.
  final Color gold;

  /// A chosen tile.
  final Color selected;
  final Color success;
  final Color danger;

  /// Shadow under gradient buttons.
  final Color glow;

  LinearGradient get gradient => LinearGradient(
    begin: Alignment.centerLeft,
    end: Alignment.centerRight,
    colors: [gradientStart, gradientMid, gradientEnd],
  );

  /// A quieter gradient for large hero cards.
  LinearGradient get heroGradient => LinearGradient(
    begin: Alignment.topLeft,
    end: Alignment.bottomRight,
    colors: [
      gradientStart.withAlpha(230),
      gradientMid.withAlpha(200),
      gradientEnd.withAlpha(210),
    ],
  );

  static BvColors of(BuildContext context) =>
      Theme.of(context).extension<BvColors>() ?? witclash;

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
    displayLarge: s(base.displayLarge, FontWeight.w900),
    displayMedium: s(base.displayMedium, FontWeight.w900),
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

/// Wraps every routed page in the BrainVolt background (D-099), so each
/// route paints its own star field during transitions and no screen can
/// fall back to a plain colour.
class WcPageTransitionsBuilder extends PageTransitionsBuilder {
  const WcPageTransitionsBuilder(this.inner, this.background);

  final PageTransitionsBuilder inner;
  final Widget Function(Widget child) background;

  @override
  Widget buildTransitions<T>(
    PageRoute<T> route,
    BuildContext context,
    Animation<double> animation,
    Animation<double> secondaryAnimation,
    Widget child,
  ) => inner.buildTransitions(
    route,
    context,
    animation,
    secondaryAnimation,
    background(child),
  );
}

/// The BrainVolt theme (D-099): dark only. [brightness] is accepted for
/// older call sites and ignored. [background] wraps every page (see
/// [WcPageTransitionsBuilder]); the app passes `WcBackground`.
ThemeData buildTheme([
  Brightness brightness = Brightness.dark,
  Widget Function(Widget child)? background,
]) {
  const bv = BvColors.witclash;
  final scheme =
      ColorScheme.fromSeed(
        seedColor: brandSeed,
        brightness: Brightness.dark,
      ).copyWith(
        primary: const Color(0xFF9B6BFF),
        onPrimary: Colors.white,
        primaryContainer: const Color(0xFF3B2585),
        onPrimaryContainer: Colors.white,
        secondary: bv.accent,
        onSecondary: Colors.white,
        secondaryContainer: const Color(0xFF4A2A1E),
        onSecondaryContainer: const Color(0xFFFFD2B5),
        tertiary: bv.gold,
        surface: bv.surface,
        onSurface: bv.ink,
        onSurfaceVariant: bv.muted,
        surfaceContainerLowest: const Color(0xFF100D2C),
        surfaceContainerLow: const Color(0xFF151236),
        surfaceContainer: bv.surface,
        surfaceContainerHigh: const Color(0xFF211C4D),
        surfaceContainerHighest: const Color(0xFF2A2459),
        outline: const Color(0x4DFFFFFF),
        outlineVariant: bv.cardBorder,
        error: bv.danger,
        onError: Colors.white,
        errorContainer: const Color(0xFF4C1426),
        onErrorContainer: const Color(0xFFFFC9D2),
        inverseSurface: Colors.white,
        onInverseSurface: bv.background,
        surfaceTint: Colors.transparent,
      );
  final base = ThemeData(
    colorScheme: scheme,
    brightness: Brightness.dark,
    useMaterial3: true,
    fontFamily: brandFont,
  );
  const radius = BorderRadius.all(Radius.circular(22));
  const stadium = WidgetStatePropertyAll<OutlinedBorder>(StadiumBorder());
  final label = WidgetStatePropertyAll(
    weighted(
      const TextStyle(fontSize: 16, fontFamily: brandFont),
      FontWeight.w800,
    ),
  );
  final text = _text(base.textTheme, bv.ink, bv.muted);
  const panelSide = BorderSide(color: Color(0x2EFFFFFF));
  const inputRadius = BorderRadius.all(Radius.circular(16));
  OutlineInputBorder inputBorder(Color c, [double w = 1]) => OutlineInputBorder(
    borderRadius: inputRadius,
    borderSide: BorderSide(color: c, width: w),
  );
  final transitions = background == null
      ? null
      : PageTransitionsTheme(
          builders: {
            for (final p in TargetPlatform.values)
              p: WcPageTransitionsBuilder(
                p == TargetPlatform.iOS || p == TargetPlatform.macOS
                    ? const CupertinoPageTransitionsBuilder()
                    : const FadeForwardsPageTransitionsBuilder(),
                background,
              ),
          },
        );
  return base.copyWith(
    scaffoldBackgroundColor: Colors.transparent,
    canvasColor: bv.surface,
    textTheme: text,
    primaryTextTheme: text,
    extensions: const [bv],
    pageTransitionsTheme: transitions,
    iconTheme: IconThemeData(color: bv.ink),
    appBarTheme: AppBarTheme(
      backgroundColor: Colors.transparent,
      surfaceTintColor: Colors.transparent,
      foregroundColor: bv.ink,
      elevation: 0,
      scrolledUnderElevation: 0,
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
        backgroundColor: WidgetStateProperty.resolveWith(
          (s) => s.contains(WidgetState.disabled)
              ? Colors.white.withAlpha(30)
              : bv.gradientStart,
        ),
        foregroundColor: WidgetStateProperty.resolveWith(
          (s) => s.contains(WidgetState.disabled)
              ? Colors.white.withAlpha(110)
              : Colors.white,
        ),
      ),
    ),
    elevatedButtonTheme: ElevatedButtonThemeData(
      style: ButtonStyle(
        shape: stadium,
        minimumSize: const WidgetStatePropertyAll(Size(64, 52)),
        textStyle: label,
        elevation: const WidgetStatePropertyAll(0),
        backgroundColor: WidgetStatePropertyAll(bv.card),
        foregroundColor: WidgetStatePropertyAll(bv.ink),
        side: WidgetStatePropertyAll(BorderSide(color: bv.cardBorder)),
      ),
    ),
    outlinedButtonTheme: OutlinedButtonThemeData(
      style: ButtonStyle(
        shape: stadium,
        minimumSize: const WidgetStatePropertyAll(Size(64, 52)),
        textStyle: label,
        foregroundColor: WidgetStateProperty.resolveWith(
          (s) => s.contains(WidgetState.disabled)
              ? Colors.white.withAlpha(110)
              : bv.ink,
        ),
        backgroundColor: WidgetStatePropertyAll(bv.card),
        side: WidgetStatePropertyAll(
          BorderSide(color: bv.cardBorder, width: 1.5),
        ),
      ),
    ),
    textButtonTheme: TextButtonThemeData(
      style: ButtonStyle(
        textStyle: label,
        foregroundColor: WidgetStatePropertyAll(bv.badgeInk),
      ),
    ),
    iconButtonTheme: IconButtonThemeData(
      style: ButtonStyle(foregroundColor: WidgetStatePropertyAll(bv.ink)),
    ),
    inputDecorationTheme: InputDecorationTheme(
      filled: true,
      fillColor: bv.card,
      labelStyle: TextStyle(color: bv.muted),
      hintStyle: TextStyle(color: bv.muted.withAlpha(180)),
      helperStyle: TextStyle(color: bv.muted),
      prefixIconColor: bv.muted,
      suffixIconColor: bv.muted,
      border: inputBorder(bv.cardBorder),
      enabledBorder: inputBorder(bv.cardBorder),
      focusedBorder: inputBorder(bv.gradientStart, 2),
      errorBorder: inputBorder(bv.danger),
      focusedErrorBorder: inputBorder(bv.danger, 2),
      disabledBorder: inputBorder(bv.cardBorder.withAlpha(20)),
    ),
    textSelectionTheme: TextSelectionThemeData(
      cursorColor: bv.accent,
      selectionColor: bv.gradientStart.withAlpha(110),
      selectionHandleColor: bv.accent,
    ),
    bottomSheetTheme: BottomSheetThemeData(
      backgroundColor: bv.surface,
      modalBackgroundColor: bv.surface,
      surfaceTintColor: Colors.transparent,
      showDragHandle: true,
      dragHandleColor: Colors.white.withAlpha(70),
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(32)),
        side: panelSide,
      ),
    ),
    dialogTheme: DialogThemeData(
      backgroundColor: bv.surface,
      surfaceTintColor: Colors.transparent,
      barrierColor: const Color(0xB3050414),
      titleTextStyle: text.titleLarge,
      contentTextStyle: text.bodyMedium,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.all(Radius.circular(28)),
        side: panelSide,
      ),
    ),
    snackBarTheme: SnackBarThemeData(
      backgroundColor: const Color(0xFF241E55),
      contentTextStyle: text.bodyMedium,
      actionTextColor: bv.accent,
      behavior: SnackBarBehavior.floating,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.all(Radius.circular(16)),
        side: panelSide,
      ),
    ),
    popupMenuTheme: PopupMenuThemeData(
      color: bv.surface,
      surfaceTintColor: Colors.transparent,
      textStyle: text.bodyMedium,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.all(Radius.circular(16)),
        side: panelSide,
      ),
    ),
    menuTheme: MenuThemeData(
      style: MenuStyle(
        backgroundColor: WidgetStatePropertyAll(bv.surface),
        surfaceTintColor: const WidgetStatePropertyAll(Colors.transparent),
      ),
    ),
    tooltipTheme: TooltipThemeData(
      decoration: BoxDecoration(
        color: const Color(0xFF241E55),
        borderRadius: BorderRadius.circular(10),
        border: Border.all(color: bv.cardBorder),
      ),
      textStyle: text.bodySmall?.copyWith(color: bv.ink),
    ),
    listTileTheme: ListTileThemeData(
      iconColor: bv.badgeInk,
      textColor: bv.ink,
      titleTextStyle: text.titleMedium,
      subtitleTextStyle: text.bodyMedium?.copyWith(color: bv.muted),
    ),
    progressIndicatorTheme: ProgressIndicatorThemeData(
      color: bv.accent,
      circularTrackColor: Colors.transparent,
      linearTrackColor: Colors.white.withAlpha(30),
      linearMinHeight: 8,
      borderRadius: const BorderRadius.all(Radius.circular(8)),
    ),
    chipTheme: ChipThemeData(
      backgroundColor: bv.card,
      selectedColor: bv.gradientStart,
      disabledColor: Colors.white.withAlpha(12),
      checkmarkColor: Colors.white,
      side: BorderSide(color: bv.cardBorder),
      shape: const StadiumBorder(),
      iconTheme: IconThemeData(color: bv.badgeInk, size: 18),
      labelStyle: weighted(
        TextStyle(color: bv.ink, fontFamily: brandFont),
        FontWeight.w800,
      ),
      secondaryLabelStyle: weighted(
        const TextStyle(color: Colors.white, fontFamily: brandFont),
        FontWeight.w800,
      ),
    ),
    segmentedButtonTheme: SegmentedButtonThemeData(
      style: ButtonStyle(
        backgroundColor: WidgetStateProperty.resolveWith(
          (s) => s.contains(WidgetState.selected) ? bv.gradientStart : bv.card,
        ),
        foregroundColor: WidgetStatePropertyAll(bv.ink),
        side: WidgetStatePropertyAll(BorderSide(color: bv.cardBorder)),
        textStyle: label,
      ),
    ),
    switchTheme: SwitchThemeData(
      thumbColor: WidgetStateProperty.resolveWith(
        (s) => s.contains(WidgetState.selected)
            ? Colors.white
            : const Color(0xFFB9B5DD),
      ),
      trackColor: WidgetStateProperty.resolveWith(
        (s) => s.contains(WidgetState.selected)
            ? bv.gradientStart
            : Colors.white.withAlpha(25),
      ),
      trackOutlineColor: WidgetStateProperty.resolveWith(
        (s) => s.contains(WidgetState.selected)
            ? Colors.transparent
            : bv.cardBorder,
      ),
    ),
    radioTheme: RadioThemeData(
      fillColor: WidgetStateProperty.resolveWith(
        (s) => s.contains(WidgetState.selected) ? bv.accent : bv.muted,
      ),
    ),
    checkboxTheme: CheckboxThemeData(
      fillColor: WidgetStateProperty.resolveWith(
        (s) => s.contains(WidgetState.selected)
            ? bv.gradientStart
            : Colors.transparent,
      ),
      checkColor: const WidgetStatePropertyAll(Colors.white),
      side: BorderSide(color: bv.muted, width: 1.5),
    ),
    tabBarTheme: TabBarThemeData(
      labelColor: Colors.white,
      unselectedLabelColor: bv.muted,
      labelStyle: label.value,
      unselectedLabelStyle: label.value,
      dividerColor: Colors.transparent,
      indicatorSize: TabBarIndicatorSize.tab,
      indicator: BoxDecoration(
        gradient: bv.gradient,
        borderRadius: BorderRadius.circular(40),
      ),
      overlayColor: const WidgetStatePropertyAll(Colors.transparent),
    ),
    navigationBarTheme: NavigationBarThemeData(
      backgroundColor: const Color(0xE6120F33),
      surfaceTintColor: Colors.transparent,
      indicatorColor: bv.gradientStart.withAlpha(90),
      iconTheme: WidgetStateProperty.resolveWith(
        (s) => IconThemeData(
          color: s.contains(WidgetState.selected) ? Colors.white : bv.muted,
        ),
      ),
      labelTextStyle: WidgetStateProperty.resolveWith(
        (s) => weighted(
          TextStyle(
            fontSize: 12,
            fontFamily: brandFont,
            color: s.contains(WidgetState.selected) ? Colors.white : bv.muted,
          ),
          FontWeight.w800,
        ),
      ),
    ),
    badgeTheme: BadgeThemeData(backgroundColor: bv.accent),
    dividerTheme: DividerThemeData(color: bv.cardBorder),
    expansionTileTheme: ExpansionTileThemeData(
      iconColor: bv.ink,
      collapsedIconColor: bv.muted,
      textColor: bv.ink,
      collapsedTextColor: bv.ink,
    ),
    floatingActionButtonTheme: FloatingActionButtonThemeData(
      backgroundColor: bv.gradientStart,
      foregroundColor: Colors.white,
      elevation: 4,
      shape: const StadiumBorder(),
      extendedTextStyle: weighted(
        const TextStyle(fontSize: 16, fontFamily: brandFont),
        FontWeight.w800,
      ),
    ),
  );
}

const _themePrefsKey = 'theme_mode';

/// The former light / dark choice (D-085). BrainVolt is dark only
/// (D-099); the stored value is kept but no longer applied.
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

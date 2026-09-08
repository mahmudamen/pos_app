import 'package:flash/flash.dart';
import 'package:flutter/material.dart';
import 'package:google_fonts/google_fonts.dart';

class AppColors {
  // 🎨 Primary Color Palette (Blue Tones)
  static const Color background = Color(
    0xFFE0EBF5,
  ); // General background light blue
  static const Color header = Color(0xFF2B5D8A); // Header dark blue
  static const Color text = Color(0xFF1A2A45); // Dark text dark blue
  static const Color tabActive = Color(0xFF2B5D8A); // Active tab dark blue
  static const Color refresh = Color(
    0xFF5A9BD4,
  ); // Interaction buttons medium blue
  static const Color lineTotal = Color(
    0xFF7AA6C1,
  ); // Secondary elements blue gray
  static const Color grandTotal = Color(
    0xFF2B5D8A,
  ); // Important titles dark blue
  static const Color send = Color(0xFF5A9BD4); // Send button medium blue
  static const Color rowHighlight = Color(
    0xFFB0C7E0,
  ); // Row highlight light blue
  static const Color tableHeader = Color(
    0xFFB0C7E0,
  ); // Table headers light blue
  static const Color gridCard = Color(
    0xFFE0EBF5,
  ); // Grid card background light blue
  static const Color warnAmber = Color(0x26FF0000);
  static const Color card = Color(0xFFFFFFFF); // Card background
  static const Color bottomBar = Color(0xFF1F3E6D); // Bottom bar dark blue
  static const Color fabBg = Color(0xFF5A9BD4); // Floating button medium blue
  static const Color fabIcon = Color(0xFFFFFFFF); // Floating button icon white
  static const Color cardBorderBase = Color(
    0xFFB0C7E0,
  ); // Card border light blue
  static const Color cardShadowBase = Color(0xFF000000); // Shadows

  // 🌚 Dark Mode Colors
  static const Color darkBackground = Color(0xFF1A1C1E);
  static const Color darkSurface = Color(0xFF242627);
  static const Color darkText = Color(0xFFE5E7EB);
  static const Color darkHeader = Color(0xFF2B4A51);
  static const Color darkBottomBar = Color(0xFF2F3F42);
}

/// 🌞 Light Theme
ThemeData buildAppTheme() {
  final base = ThemeData(
    useMaterial3: true,
    colorScheme: ColorScheme.fromSeed(seedColor: AppColors.tabActive),
    extensions: [
      FlashBarTheme(
        backgroundColor: AppColors.card,
        surfaceTintColor: AppColors.card,
        shadowColor: AppColors.cardShadowBase.withOpacity(0.15),
        titleTextStyle: GoogleFonts.cairo(
          color: AppColors.text,
          fontWeight: FontWeight.bold,
          fontSize: 16,
        ),
        contentTextStyle: GoogleFonts.cairo(
          color: AppColors.text,
          fontSize: 14,
        ),
        iconColor: AppColors.tabActive,
      ),
      FlashToastTheme(
        backgroundColor: AppColors.header,
        textStyle: GoogleFonts.cairo(color: Colors.white, fontSize: 14),
        alignment: Alignment.bottomCenter,
        margin: const EdgeInsets.all(12),
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
      ),
    ],
  );

  return base.copyWith(
    scaffoldBackgroundColor: AppColors.background,
    appBarTheme: const AppBarTheme(
      backgroundColor: AppColors.header,
      foregroundColor: Colors.white,
      elevation: 0,
    ),
    iconTheme: const IconThemeData(color: AppColors.text),
    floatingActionButtonTheme: const FloatingActionButtonThemeData(
      backgroundColor: AppColors.fabBg,
      foregroundColor: AppColors.fabIcon,
    ),
    dividerTheme: DividerThemeData(
      color: AppColors.text.withOpacity(0.12),
      thickness: 1,
      space: 1,
    ),
    cardTheme: CardThemeData(
      color: AppColors.card,
      elevation: 3,
      shadowColor: AppColors.cardShadowBase.withOpacity(0.08),
      surfaceTintColor: Colors.transparent,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(12),
        side: const BorderSide(color: AppColors.cardBorderBase, width: 1),
      ),
    ),
    textTheme: GoogleFonts.cairoTextTheme(
      base.textTheme,
    ).apply(bodyColor: AppColors.text, displayColor: AppColors.text),
    inputDecorationTheme: const InputDecorationTheme(
      border: OutlineInputBorder(),
    ),
    progressIndicatorTheme: const ProgressIndicatorThemeData(
      color: AppColors.tabActive,
    ),
    elevatedButtonTheme: ElevatedButtonThemeData(
      style: ElevatedButton.styleFrom(
        backgroundColor: AppColors.tabActive,
        foregroundColor: Colors.white,
      ),
    ),
  );
}

/// 🌙 Dark Theme
ThemeData buildDarkAppTheme() {
  final base = ThemeData.dark(useMaterial3: true);

  return base.copyWith(
    scaffoldBackgroundColor: AppColors.darkBackground,
    appBarTheme: const AppBarTheme(
      backgroundColor: AppColors.darkHeader,
      foregroundColor: Colors.white,
      elevation: 0,
    ),
    iconTheme: const IconThemeData(color: AppColors.darkText),
    floatingActionButtonTheme: const FloatingActionButtonThemeData(
      backgroundColor: AppColors.refresh,
      foregroundColor: Colors.white,
    ),
    cardTheme: CardThemeData(
      color: AppColors.darkSurface,
      elevation: 3,
      shadowColor: Colors.black.withOpacity(0.2),
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
    ),
    textTheme: GoogleFonts.cairoTextTheme(
      base.textTheme,
    ).apply(bodyColor: AppColors.darkText, displayColor: AppColors.darkText),
    bottomAppBarTheme: const BottomAppBarThemeData(
      color: AppColors.darkBottomBar,
    ),
    progressIndicatorTheme: const ProgressIndicatorThemeData(
      color: AppColors.refresh,
    ),
  );
}

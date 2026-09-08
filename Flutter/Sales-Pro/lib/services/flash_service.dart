import 'package:sales_pro/theme/app_theme.dart';
import 'package:flash/flash.dart';
import 'package:flash/flash_helper.dart';
import 'package:flutter/material.dart';
import 'package:google_fonts/google_fonts.dart';

enum FlashMessageType { success, warning, error, info, noInternet }

/// Color and icon maps (from app_colors.dart)
final Map<FlashMessageType, Color> flashColors = {
  FlashMessageType.success: const Color(0xFF4CAF50),
  FlashMessageType.warning: const Color(0xFFFF9800),
  FlashMessageType.error: const Color(0xFFF44336),
  FlashMessageType.info: const Color(0xFF2196F3),
  FlashMessageType.noInternet:
      AppColors.warnAmber, // We used a color from AppColors
};

final Map<FlashMessageType, IconData> flashIcons = {
  FlashMessageType.success: Icons.check_circle_outline,
  FlashMessageType.warning: Icons.warning_amber_outlined,
  FlashMessageType.error: Icons.error_outline,
  FlashMessageType.info: Icons.info_outline,
  FlashMessageType.noInternet: Icons.wifi_off,
};

/// Unified function
void showAppFlash(
  BuildContext context, {
  required FlashMessageType type,
  String? message,
  Duration duration = const Duration(seconds: 3),
}) {
  if (!context.mounted) return;

  final theme = Theme.of(context);
  final isDark = theme.brightness == Brightness.dark;

  // Default text
  final String displayMessage =
      message ??
      (type == FlashMessageType.noInternet
          ? 'You must be connected to the internet to continue'
          : type == FlashMessageType.success
          ? 'Success'
          : type == FlashMessageType.error
          ? 'Error'
          : type == FlashMessageType.warning
          ? 'Warning'
          : 'Info');

  // Use FlashBarTheme from theme
  final flashBarTheme =
      theme.extension<FlashBarTheme>() ??
      FlashBarTheme(
        backgroundColor: isDark ? AppColors.darkSurface : AppColors.card,
        shadowColor: AppColors.cardShadowBase.withOpacity(0.15),
      );

  context.showFlash(
    duration: duration,
    builder: (context, controller) {
      return FlashBar(
        controller: controller,
        position: FlashPosition.bottom,
        behavior: FlashBehavior.floating,
        margin: const EdgeInsets.all(16),
        clipBehavior: Clip.antiAlias,
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
        backgroundColor:
            flashColors[type] ??
            (isDark ? AppColors.darkSurface : AppColors.card),
        indicatorColor: flashColors[type]?.withOpacity(0.8),
        shadowColor: flashBarTheme.shadowColor,
        content: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
          child: Row(
            children: [
              Icon(flashIcons[type], color: Colors.white, size: 24),
              const SizedBox(width: 12),
              Expanded(
                child: Text(
                  displayMessage,
                  style: GoogleFonts.cairo(
                    color: Colors.white,
                    fontSize: 15,
                    fontWeight: FontWeight.w500,
                  ),
                ),
              ),
            ],
          ),
        ),
        primaryAction: TextButton(
          onPressed: controller.dismiss,
          child: Text(
            'Close',
            style: GoogleFonts.cairo(
              color: Colors.white,
              fontWeight: FontWeight.bold,
            ),
          ),
        ),
      );
    },
  );
}

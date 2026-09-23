import 'package:flutter/services.dart';

/// Lightweight audio/haptic feedback for UI selections and confirmations.
///
/// Uses Flutter's platform channels only (no bundled audio assets or native
/// plugin) so it keeps the app dependency-free and works on Android out of
/// the box: [click] short tap sound, [select] tactile tick, [confirm] a
/// distinct confirm tap + light impact. All calls are no-ops when
/// [enabled] is false and are safe to invoke from tests (the platform
/// channel is a no-op in the test harness).
class UiFeedback {
  /// Global mute flag; wired to the persisted `app_sound_enabled` setting.
  static bool enabled = true;

  /// Short click feedback for a button/option tap.
  static void click() {
    if (!enabled) return;
    SystemSound.play(SystemSoundType.click);
    HapticFeedback.lightImpact();
  }

  /// Tactile selection tick (option highlighted, not yet applied).
  static void select() {
    if (!enabled) return;
    HapticFeedback.selectionClick();
  }

  /// Confirm feedback for a choice applied/saved.
  static void confirm() {
    if (!enabled) return;
    SystemSound.play(SystemSoundType.click);
    HapticFeedback.mediumImpact();
  }
}
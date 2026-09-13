import 'package:flutter/services.dart';

/// Controls POS "focus mode": hides the status/navigation bars (so the
/// notification shade is out of reach), keeps the screen awake, and, when the
/// user has granted Do-Not-Disturb access, silences incoming calls and
/// notifications. All native calls fail softly so the widget tests stay green.
class FocusMode {
  FocusMode._();

  static const MethodChannel _channel = MethodChannel('com.xamltech.pos_go/focus');

  /// Applies/removes focus mode. Returns true when DND is actually active.
  static Future<bool> apply(bool enabled) async {
    if (enabled) {
      SystemChrome.setEnabledSystemUIMode(SystemUiMode.immersiveSticky);
    } else {
      SystemChrome.setEnabledSystemUIMode(SystemUiMode.edgeToEdge);
    }
    try {
      await _channel.invokeMethod<void>('set', {'enabled': enabled});
    } catch (_) {
      // Non-Android hosts / tests: nothing else to do.
    }
    return enabled && await dndGranted();
  }

  static Future<bool> dndGranted() async {
    try {
      return await _channel.invokeMethod<bool>('dndGranted') ?? false;
    } catch (_) {
      return false;
    }
  }

  static Future<void> openDndSettings() async {
    try {
      await _channel.invokeMethod<void>('openDndSettings');
    } catch (_) {
      // Nothing to open off-device.
    }
  }
}
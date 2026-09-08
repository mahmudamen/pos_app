import 'dart:async';
import 'package:connectivity_plus/connectivity_plus.dart';
import 'package:flutter/foundation.dart';

class ConnectivityService {
  // Notifier for connection status (online/offline)
  final ValueNotifier<bool> isOnline = ValueNotifier<bool>(true);

  // Stream subscription for connectivity
  StreamSubscription<dynamic>? _sub;

  // Debounce to avoid rapid updates
  Timer? _debounce;

  // To store last connection status
  bool _lastState = true;

  /// Start service and monitor connection
  Future<void> start() async {
    final conn = Connectivity();

    // Check initial connection status
    final firstResult = await conn.checkConnectivity();
    isOnline.value = _toOnlineResult(firstResult);
    _lastState = isOnline.value;

    // Listen for connection changes
    _sub = conn.onConnectivityChanged.listen(_handleConnectivityChange);
  }

  /// Convert result to online/offline
  bool _toOnline(ConnectivityResult r) =>
      r == ConnectivityResult.mobile ||
      r == ConnectivityResult.wifi ||
      r == ConnectivityResult.ethernet;

  /// Support single or list results
  bool _toOnlineResult(dynamic result) {
    if (result is List<ConnectivityResult>) {
      return result.any(_toOnline);
    } else if (result is ConnectivityResult) {
      return _toOnline(result);
    } else {
      return false;
    }
  }

  /// Handle connection changes with debounce
  void _handleConnectivityChange(dynamic result) {
    final newState = _toOnlineResult(result);

    if (newState != _lastState) {
      _debounce?.cancel();
      _debounce = Timer(const Duration(milliseconds: 500), () {
        isOnline.value = newState;
        _lastState = newState;
      });
    }
  }

  /// تنظيف الموارد
  Future<void> dispose() async {
    await _sub?.cancel();
    _debounce?.cancel();
    isOnline.dispose();
  }
}

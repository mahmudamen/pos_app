import 'dart:async';

import 'package:flutter/foundation.dart';

import 'api_client.dart';
import 'session_store.dart';

/// Best-effort client telemetry: forwards crashes and lifecycle/action events
/// to `POST /v1/client/events`.
///
/// Deliberately fire-and-forget — a missing session (e.g. a crash before login)
/// or a network failure drops the event rather than disturbing the POS flow.
class Telemetry {
  Telemetry._();

  static final Telemetry instance = Telemetry._();

  /// Build identifier reported with every event; keep in sync with pubspec.
  static const String buildVersion = '0.1.0+1';

  ApiClient? _api;
  Session? _session;
  bool _installed = false;

  void configure(ApiClient? api) => _api = api;

  /// Tracks the active session so events are attributed to the right tenant.
  /// Callers pass `null` on logout.
  void setSession(Session? session) => _session = session;

  /// Installs the global crash/error hooks. Idempotent.
  void install() {
    if (_installed) return;
    _installed = true;
    final previous = FlutterError.onError;
    FlutterError.onError = (FlutterErrorDetails details) {
      reportEvent(
        'crash',
        stackTrace: details.stack?.toString(),
        payload: {
          'exception': details.exceptionAsString(),
          'library': details.library,
        },
      );
      if (previous != null) {
        previous(details);
      } else {
        FlutterError.presentError(details);
      }
    };
    PlatformDispatcher.instance.onError = (Object error, StackTrace stack) {
      reportEvent(
        'platform_error',
        stackTrace: stack.toString(),
        payload: {'exception': '$error'},
      );
      return false;
    };
  }

  void reportScreen(String screen) => reportEvent('screen', screen: screen);

  /// Queues an event. No-op without a configured client + active session.
  void reportEvent(
    String event, {
    String? screen,
    String? stackTrace,
    Map<String, dynamic> payload = const {},
  }) {
    final api = _api;
    final session = _session;
    if (api == null || session == null) return;
    unawaited(
      api
          .postClientEvent(
            session,
            event,
            appVersion: buildVersion,
            screen: screen,
            stackTrace: stackTrace,
            payload: payload,
          )
          .catchError((Object _) {}),
    );
  }

  @visibleForTesting
  void reset() {
    _api = null;
    _session = null;
    _installed = false;
  }
}
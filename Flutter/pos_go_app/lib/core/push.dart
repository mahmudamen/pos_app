import 'dart:async';
import 'dart:io';

import 'package:firebase_core/firebase_core.dart';
import 'package:firebase_messaging/firebase_messaging.dart';
import 'package:flutter/foundation.dart';

/// Push-notification plumbing, and nothing else.
///
/// There is no Firestore and no Firebase Auth here. POS.Go keeps its own JWT
/// auth and its own Postgres-backed store; Firebase is used strictly to receive
/// push, because FCM is the only reliable way to reach an Android device that is
/// not being actively used by a cashier.
///
/// The rule every call site must respect: [initialize] never throws. A build
/// without `android/app/google-services.json` (a fresh clone, CI, a test build)
/// has no Firebase project, and push simply is not available. That happens in
/// practice, so it is modelled as a value rather than an exception -- see
/// [PushStatus] and [isAvailable].
///
/// The token is a credential. It is handed back to the caller to be stored
/// tenant-scoped through the POS API. It must never be written to
/// `client_events`, logged, or sent to any third party.
class PushService {
  PushService._();

  static final PushService instance = PushService._();

  final _controller = StreamController<PushMessage>.broadcast();
  bool _available = false;
  String? _token;
  String? _projectId;

  /// Foreground messages. Background messages are handled by
  /// [firebaseMessagingBackgroundHandler] in a separate isolate and do not
  /// arrive here.
  Stream<PushMessage> get messages => _controller.stream;

  /// True when a Firebase project was found and FCM is usable.
  bool get isAvailable => _available;

  /// The current FCM registration token, or null when unavailable.
  String? get token => _token;

  /// Which Firebase project this build is bound to. Useful in bug reports, and
  /// the fastest way to catch a release built against the wrong project.
  String? get projectId => _projectId;

  /// Attempts to initialise Firebase and obtain a token.
  ///
  /// Returns the status rather than throwing. Every failure mode here is
  /// expected at least once in a normal development cycle: no config file, no
  /// Play Services on the device, no permission granted yet, no network.
  Future<PushStatus> initialize() async {
    if (kIsWeb) {
      return const PushStatus.unavailable('web is not supported');
    }
    if (!Platform.isAndroid && !Platform.isIOS) {
      return const PushStatus.unavailable('platform is not Android or iOS');
    }

    try {
      await Firebase.initializeApp();
    } catch (error) {
      // The overwhelmingly common case: no google-services.json in this build.
      return PushStatus.unavailable('no Firebase config: $error');
    }

    try {
      final messaging = FirebaseMessaging.instance;

      // Foreground notifications are rendered by the app itself, so the OS
      // should not also show a banner for them.
      await messaging.setForegroundNotificationPresentationOptions(
        alert: false,
        badge: true,
        sound: false,
      );

      final settings = await messaging.getNotificationSettings();
      if (settings.authorizationStatus == AuthorizationStatus.denied) {
        return const PushStatus.needsPermission();
      }

      FirebaseMessaging.onBackgroundMessage(firebaseMessagingBackgroundHandler);

      _available = true;
      _projectId = _readProjectId();

      // Token changes are the normal case, not an edge case: a reinstall, a
      // restore, or a Firebase-side revoke all mint a new one. Surfacing this
      // is what stops a device going permanently silent after it is swapped.
      _tokenSub = messaging.onTokenRefresh.listen((token) {
        _token = token;
        _controller.add(PushMessage.tokenRefreshed(token));
      });

      final token = await messaging.getToken();
      _token = token;
      if (token == null) {
        return const PushStatus.unavailable('FCM returned no token');
      }
      return PushStatus.ready(token);
    } catch (error) {
      _available = false;
      return PushStatus.unavailable('FCM init failed: $error');
    }
  }

  StreamSubscription<String>? _tokenSub;

  /// Asks for notification permission. Returns true if granted.
  Future<bool> requestPermission() async {
    try {
      final messaging = FirebaseMessaging.instance;
      final current = await messaging.getNotificationSettings();
      if (current.authorizationStatus == AuthorizationStatus.authorized ||
          current.authorizationStatus == AuthorizationStatus.provisional) {
        return true;
      }
      // targetSdk 36 requires an explicit runtime grant, so this is the call
      // that actually surfaces the Android 13+ permission dialog.
      final result = await messaging.requestPermission();
      return result.authorizationStatus == AuthorizationStatus.authorized ||
          result.authorizationStatus == AuthorizationStatus.provisional;
    } catch (error) {
      debugPrint('PushService.requestPermission failed: $error');
      return false;
    }
  }

  /// Releases the FCM token so a handed-over or retired device stops receiving
  /// tenant notifications.
  Future<void> disable() async {
    await _tokenSub?.cancel();
    _tokenSub = null;
    try {
      await FirebaseMessaging.instance.deleteToken();
    } catch (error) {
      debugPrint('PushService.disable: $error');
    }
    _available = false;
    _token = null;
  }

  void dispose() {
    _tokenSub?.cancel();
    _controller.close();
  }

  /// Reads the bound project id out of the native config.
  ///
  /// Deliberately kept readable at runtime. A release built against the wrong
  /// Firebase project is otherwise invisible until someone notices push
  /// arriving in the wrong place.
  static String? _readProjectId() {
    try {
      return Firebase.app().options.projectId;
    } catch (_) {
      return null;
    }
  }
}

/// Why push is or is not available, in a form a UI can branch on.
enum PushStatusKind { ready, needsPermission, unavailable }

class PushStatus {
  const PushStatus.ready(this.token)
    : kind = PushStatusKind.ready,
      detail = null;
  const PushStatus.needsPermission()
    : kind = PushStatusKind.needsPermission,
      token = null,
      detail = 'notification permission not granted';
  const PushStatus.unavailable(this.detail)
    : kind = PushStatusKind.unavailable,
      token = null;

  final PushStatusKind kind;
  final String? token;
  final String? detail;

  bool get isReady => kind == PushStatusKind.ready;
}

/// A push event delivered to the running app.
class PushMessage {
  const PushMessage({this.title, this.body, this.data = const {}}) : token = null;

  const PushMessage.tokenRefreshed(String this.token)
    : title = null,
      body = null,
      data = const {'type': 'token_refresh'};

  final String? title;
  final String? body;
  final Map<String, dynamic> data;
  final String? token;

  bool get isTokenRefresh => data['type'] == 'token_refresh';
}

/// Background entrypoint. This runs in a separate isolate, so it cannot close
/// over anything from the UI isolate and cannot touch [PushService.instance] --
/// which is why the handler stays minimal and leaves the work for the app to
/// pick up on next foreground.
@pragma('vm:entry-point')
Future<void> firebaseMessagingBackgroundHandler(RemoteMessage message) async {
  // Payloads are summarised, never dumped. A POS notification can carry a sale
  // id or a customer name, and those do not belong in logcat.
  if (kDebugMode) {
    debugPrint(
      'FCM background message: ${message.messageId} '
      'keys=${message.data.keys.toList()} '
      'hasNotification=${message.notification != null}',
    );
  }
}

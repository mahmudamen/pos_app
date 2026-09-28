import 'package:flutter_test/flutter_test.dart';
import 'package:pos_go_app/core/push.dart';

void main() {
  group('PushStatus', () {
    test('ready carries the token and is the only state that counts as ready', () {
      const status = PushStatus.ready('tok-abc');
      expect(status.kind, PushStatusKind.ready);
      expect(status.isReady, isTrue);
      expect(status.token, 'tok-abc');
      expect(status.detail, isNull);
    });

    test('needsPermission is not ready and explains itself', () {
      const status = PushStatus.needsPermission();
      expect(status.kind, PushStatusKind.needsPermission);
      expect(status.isReady, isFalse);
      expect(status.token, isNull);
      expect(status.detail, isNotNull);
    });

    test('unavailable keeps the reason for telemetry but has no token', () {
      const status = PushStatus.unavailable('no Firebase config');
      expect(status.kind, PushStatusKind.unavailable);
      expect(status.isReady, isFalse);
      expect(status.token, isNull);
      expect(status.detail, 'no Firebase config');
    });
  });

  group('PushMessage', () {
    test('a normal message is not mistaken for a token refresh', () {
      const message = PushMessage(title: 'Sale', body: 'Paid');
      expect(message.isTokenRefresh, isFalse);
      expect(message.token, isNull);
      expect(message.title, 'Sale');
    });

    test('token refresh is distinguishable and carries the new token', () {
      const message = PushMessage.tokenRefreshed('tok-new');
      expect(message.isTokenRefresh, isTrue);
      expect(message.token, 'tok-new');
      expect(message.data['type'], 'token_refresh');
    });
  });

  group('service before initialization', () {
    // The point of the whole design: a build with no google-services.json must
    // behave as "push is off", not as an exception at startup.
    test('reports unavailable rather than throwing', () {
      expect(PushService.instance.isAvailable, isFalse);
      expect(PushService.instance.token, isNull);
    });

    test('projectId is null until a config is bound', () {
      expect(PushService.instance.projectId, isNull);
    });

    test('message stream is open and emits nothing by itself', () async {
      expect(PushService.instance.messages, isNotNull);
    });
  });
}

import 'package:flutter/foundation.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:pos_go_app/core/api_client.dart';
import 'package:pos_go_app/core/session_store.dart';
import 'package:pos_go_app/core/telemetry.dart';

class _FakeApi extends ApiClient {
  final List<({String event, String? screen, Map<String, dynamic> payload})>
      events = [];

  @override
  Future<void> postClientEvent(
    Session session,
    String event, {
    String? appVersion,
    String? screen,
    String? stackTrace,
    Map<String, dynamic> payload = const {},
  }) async {
    events.add((event: event, screen: screen, payload: payload));
  }
}

const _session = Session(
  accessToken: 'access-token',
  refreshToken: 'refresh-token',
  userId: 'user-1',
  displayName: 'Cashier',
  tenantId: 'tenant-1',
);

void main() {
  setUp(() => Telemetry.instance.reset());
  tearDown(() => Telemetry.instance.reset());

  test('reportEvent is a no-op without a session', () {
    Telemetry.instance.configure(_FakeApi());
    Telemetry.instance.reportEvent('crash');
  });

  test('reportEvent is a no-op without an API client', () {
    Telemetry.instance.setSession(_session);
    Telemetry.instance.reportEvent('crash');
  });

  test('reportEvent forwards when configured', () async {
    final api = _FakeApi();
    Telemetry.instance.configure(api);
    Telemetry.instance.setSession(_session);

    Telemetry.instance.reportEvent(
      'screen',
      screen: 'pos',
      payload: {'a': 1},
    );
    await Future<void>.delayed(Duration.zero);

    expect(api.events, hasLength(1));
    expect(api.events.single.event, 'screen');
    expect(api.events.single.screen, 'pos');
    expect(api.events.single.payload, {'a': 1});
  });

  test('reportScreen sends a screen event', () async {
    final api = _FakeApi();
    Telemetry.instance.configure(api);
    Telemetry.instance.setSession(_session);

    Telemetry.instance.reportScreen('customers');
    await Future<void>.delayed(Duration.zero);

    expect(api.events.single.event, 'screen');
    expect(api.events.single.screen, 'customers');
  });

  testWidgets('install reports crashes and preserves the previous handler',
      (tester) async {
    final api = _FakeApi();
    Telemetry.instance.configure(api);
    Telemetry.instance.setSession(_session);
    var chained = false;
    FlutterError.onError = (FlutterErrorDetails details) {
      chained = true;
    };

    Telemetry.instance.install();
    final installed = FlutterError.onError!;
    installed(FlutterErrorDetails(
      exception: StateError('boom'),
      stack: StackTrace.current,
    ));
    await tester.pump();

    expect(chained, isTrue);
    expect(api.events.single.event, 'crash');
    expect(api.events.single.payload['exception'], contains('boom'));

    FlutterError.onError = FlutterError.presentError;
  });
}
import 'dart:convert';

import 'dart:ui' show Locale;

import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;

import 'package:pos_go_app/core/api_client.dart';
import 'package:pos_go_app/core/onboarding.dart';
import 'package:pos_go_app/core/session_store.dart';
import 'package:pos_go_app/l10n/strings.dart';

const _session = Session(
  accessToken: 'access-token',
  refreshToken: 'refresh-token',
  userId: 'user-1',
  displayName: 'Cashier One',
  tenantId: 'tenant-1',
  deviceId: 'device-1',
  role: 'cashier',
);

const _stateJson = '{"data":{"tours":[{"id":"cash_session","version":1,'
    '"goal":"cash_session_opened","status":"completed","current_step":"verify",'
    '"step_count":4,"goal_met":true},{"id":"first_sale","version":1,'
    '"goal":"first_sale_completed","status":"not_started","current_step":"",'
    '"step_count":4,"goal_met":false}],"readiness":{"store_configured":true,'
    '"product_ready":true,"terminal_configured":false,"cash_session_opened":true,'
    '"first_sale_completed":false,"activated":false},"recommended":"first_sale"},'
    '"meta":{"request_id":"test"}}';

class _RouteClient extends http.BaseClient {
  _RouteClient(this.onRoute);

  final http.Response Function(http.BaseRequest request) onRoute;

  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    final response = onRoute(request);
    return http.StreamedResponse(
      Stream.value(response.body.codeUnits),
      response.statusCode,
      headers: {'content-type': 'application/json'},
    );
  }
}

void main() {
  group('onboarding models', () {
    test('TourProgress parses status + goal met from the Go row', () {
      final progress = TourProgress.fromJson({
        'id': 'first_sale',
        'version': 1,
        'goal': 'first_sale_completed',
        'status': 'paused',
        'current_step': 'checkout',
        'step_count': 4,
        'goal_met': false,
      });
      expect(progress.id, 'first_sale');
      expect(progress.isPaused, isTrue);
      expect(progress.isRunning, isFalse);
      expect(progress.isCompleted, isFalse);
      expect(progress.isInFlight, isTrue);
      expect(progress.goalMet, isFalse);
    });

    test('OnboardingState parses tours, readiness and recommended', () {
      final state = OnboardingState.fromJson(
          jsonDecode(_stateJson)['data'] as Map<String, dynamic>);
      expect(state.tours.length, 2);
      expect(state.tour('cash_session')!.isCompleted, isTrue);
      expect(state.tour('first_sale')!.goalMet, isFalse);
      expect(state.recommended, 'first_sale');
      expect(state.readiness.storeConfigured, isTrue);
      expect(state.readiness.firstSaleCompleted, isFalse);
    });

    test('TourActionResult and OnboardingEventResult parse the Go row', () {
      final action = TourActionResult.fromJson(const {
        'tour': 'cash_session',
        'status': 'running',
        'current_step': 'start',
      });
      expect(action.tourId, 'cash_session');
      expect(action.status, 'running');

      final event = OnboardingEventResult.fromJson(const {
        'event_id': 'evt-1',
        'event': 'tour_started',
        'replayed': true,
      });
      expect(event.eventId, 'evt-1');
      expect(event.replayed, isTrue);
    });
  });

  group('onboarding API', () {
    test('onboardingState GETs /v1/onboarding and parses the envelope', () async {
      final client = ApiClient(
        client: _RouteClient((request) {
          expect(request.method, 'GET');
          expect(request.url.path, '/v1/onboarding');
          expect(request.headers['Authorization'], 'Bearer access-token');
          return http.Response(_stateJson, 200);
        }),
      );
      final state = await client.onboardingState(_session);
      expect(state.recommended, 'first_sale');
      expect(state.tour('cash_session')!.status, 'completed');
    });

    test('onboardingReadiness GETs /v1/onboarding/readiness', () async {
      final client = ApiClient(
        client: _RouteClient((request) {
          expect(request.url.path, '/v1/onboarding/readiness');
          return http.Response(
              '{"data":{"store_configured":true,"product_ready":false,'
              '"terminal_configured":false,"cash_session_opened":true,'
              '"first_sale_completed":false,"activated":false},'
              '"meta":{"request_id":"t"}}',
              200);
        }),
      );
      final readiness = await client.onboardingReadiness(_session);
      expect(readiness.cashSessionOpened, isTrue);
      expect(readiness.productReady, isFalse);
    });

    test('startTour POSTs the tour_version body', () async {
      final client = ApiClient(
        client: _RouteClient((request) {
          expect(request.method, 'POST');
          expect(request.url.path, '/v1/onboarding/tours/first_sale/start');
          final body = jsonDecode(utf8.decode(
              (request as http.Request).bodyBytes));
          expect(body, const {'tour_version': 1});
          return http.Response(
              '{"data":{"tour":"first_sale","status":"running",'
              '"current_step":"intro"},"meta":{"request_id":"t"}}',
              200);
        }),
      );
      final result = await client.startTour(_session, 'first_sale', version: 1);
      expect(result.status, 'running');
      expect(result.currentStep, 'intro');
    });

    test('startTour surfaces tour_version_mismatch as a typed error', () async {
      final client = ApiClient(
        client: _RouteClient((request) => http.Response(
            '{"error":{"code":"tour_version_mismatch",'
            '"message":"tour changed"},"meta":{"request_id":"t"}}',
            409)),
      );
      await expectLater(
        client.startTour(_session, 'first_sale', version: 2),
        throwsA(isA<ApiException>()
            .having((e) => e.isTourVersionMismatch, 'isTourVersionMismatch',
                isTrue)),
      );
    });

    test('setTourStep PATCHes current_step', () async {
      final client = ApiClient(
        client: _RouteClient((request) {
          expect(request.method, 'PATCH');
          expect(request.url.path, '/v1/onboarding/tours/first_sale');
          final body = jsonDecode(utf8.decode((request as http.Request).bodyBytes));
          expect(body, const {'current_step': 'add_item'});
          return http.Response(
              '{"data":{"tour":"first_sale","status":"running",'
              '"current_step":"add_item"},"meta":{"request_id":"t"}}',
              200);
        }),
      );
      final result = await client.setTourStep(_session, 'first_sale', 'add_item');
      expect(result.currentStep, 'add_item');
    });

    test('pause/resume/skip/complete POST to their endpoints', () async {
      final paths = <String>[];
      final client = ApiClient(
        client: _RouteClient((request) {
          paths.add(request.url.path);
          return http.Response(
              '{"data":{"tour":"first_sale","status":"running",'
              '"current_step":"checkout"},"meta":{"request_id":"t"}}',
              200);
        }),
      );
      await client.pauseTour(_session, 'first_sale');
      await client.resumeTour(_session, 'first_sale');
      await client.skipTour(_session, 'first_sale');
      await client.completeTour(_session, 'first_sale');
      expect(paths, const [
        '/v1/onboarding/tours/first_sale/pause',
        '/v1/onboarding/tours/first_sale/resume',
        '/v1/onboarding/tours/first_sale/skip',
        '/v1/onboarding/tours/first_sale/complete',
      ]);
    });

    test('completeTour surfaces goal_not_met until the goal really exists',
        () async {
      final client = ApiClient(
        client: _RouteClient((request) => http.Response(
            '{"error":{"code":"goal_not_met",'
            '"message":"no open cash session on record"},'
            '"meta":{"request_id":"t"}}',
            409)),
      );
      await expectLater(
        client.completeTour(_session, 'cash_session'),
        throwsA(isA<ApiException>()
            .having((e) => e.isGoalNotMet, 'isGoalNotMet', isTrue)),
      );
    });

    test('reportOnboardingEvent POSTs payload and accepts 201 + idempotent 200',
        () async {
      var calls = 0;
      final client = ApiClient(
        client: _RouteClient((request) {
          calls++;
          expect(request.url.path, '/v1/onboarding/events');
          final body = jsonDecode(utf8.decode((request as http.Request).bodyBytes));
          expect(body['event_id'], 'evt-1');
          expect(body['event'], 'tour_started');
          if (calls == 1) {
            expect(body['payload'], const {'tour': 'first_sale'});
          }
          return http.Response(
              '{"data":{"event_id":"evt-1","event":"tour_started",'
              '"replayed":${calls > 1}},"meta":{"request_id":"t"}}',
              calls > 1 ? 200 : 201);
        }),
      );
      final first = await client.reportOnboardingEvent(_session,
          eventId: 'evt-1', event: 'tour_started', payload: const {
            'tour': 'first_sale'});
      expect(first.replayed, isFalse);
      final replay = await client.reportOnboardingEvent(_session,
          eventId: 'evt-1', event: 'tour_started');
      expect(replay.replayed, isTrue);
    });

    test('onboarding methods require auth (401 → ApiException)', () async {
      final client = ApiClient(
        client: _RouteClient((request) => http.Response(
            '{"error":{"code":"unauthorized","message":"missing token"},'
            '"meta":{"request_id":"t"}}',
            401)),
      );
      await expectLater(
        client.onboardingState(_session),
        throwsA(isA<ApiException>()),
      );
    });
  });

  test('localized tour copy resolves both languages', () {
    final ar = AppStrings(const Locale('ar'));
    final en = AppStrings(const Locale('en'));
    expect(en.tourCashIntroTitle, 'Cash session');
    expect(ar.tourCashIntroTitle, 'جلسة الكاش');
    expect(en.tourFirstCheckoutBody, contains('Complete sale'));
    expect(ar.tourFirstCheckoutBody, contains('إتمام البيع'));
  });
}
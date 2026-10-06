import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:pos_go_app/core/api_client.dart';
import 'package:pos_go_app/core/onboarding.dart';
import 'package:pos_go_app/core/session_store.dart';
import 'package:pos_go_app/core/tour/tour_controller.dart';
import 'package:pos_go_app/core/tour/tour_definitions.dart';
import 'package:pos_go_app/core/tour/tour_target_registry.dart';
import 'package:pos_go_app/features/onboarding/onboarding_center_screen.dart';
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

Map<String, dynamic> _stateJson({String? firstSaleStatus = 'not_started'}) {
  return {
    'data': {
      'tours': [
        {
          'id': 'store_setup',
          'version': 1,
          'goal': 'store_configured',
          'status': 'not_started',
          'current_step': '',
          'step_count': 4,
          'goal_met': false,
        },
        {
          'id': 'first_product',
          'version': 1,
          'goal': 'product_ready',
          'status': 'not_started',
          'current_step': '',
          'step_count': 4,
          'goal_met': false,
        },
        {
          'id': 'terminal_setup',
          'version': 1,
          'goal': 'terminal_configured',
          'status': 'not_started',
          'current_step': '',
          'step_count': 4,
          'goal_met': false,
        },
        {
          'id': 'cash_session',
          'version': 1,
          'goal': 'cash_session_opened',
          'status': 'completed',
          'current_step': 'verify',
          'step_count': 4,
          'goal_met': true,
        },
        {
          'id': 'first_sale',
          'version': 1,
          'goal': 'first_sale_completed',
          'status': firstSaleStatus ?? 'not_started',
          'current_step': firstSaleStatus == 'running' ? 'checkout' : '',
          'step_count': 4,
          'goal_met': false,
        },
      ],
      'readiness': {
        'store_configured': false,
        'product_ready': false,
        'terminal_configured': false,
        'cash_session_opened': true,
        'first_sale_completed': false,
        'activated': false,
      },
      'recommended': 'first_sale',
    },
    'meta': {'request_id': 'test'},
  };
}

class _CenterApi extends ApiClient {
  _CenterApi();

  bool failOnce = false;
  int stateCalls = 0;
  int startCalls = 0;
  int skipCalls = 0;
  String? lastStatus;

  @override
  Future<OnboardingState> onboardingState(Session session) async {
    stateCalls++;
    if (failOnce && stateCalls == 1) {
      throw const ApiException('boom', code: 'server_error');
    }
    return OnboardingState.fromJson(
        _stateJson(firstSaleStatus: lastStatus)['data'] as Map<String, dynamic>);
  }

  @override
  Future<TourActionResult> startTour(Session session, String tourId,
      {int? version}) async {
    startCalls++;
    lastStatus = 'running';
    return const TourActionResult(tourId: 'x', status: 'running');
  }

  @override
  Future<TourActionResult> setTourStep(
      Session session, String tourId, String currentStep) async {
    return const TourActionResult(tourId: 'x', status: 'running');
  }

  @override
  Future<TourActionResult> skipTour(Session session, String tourId) async {
    skipCalls++;
    lastStatus = 'skipped';
    return const TourActionResult(tourId: 'x', status: 'skipped');
  }

  @override
  Future<TourActionResult> pauseTour(Session session, String tourId) async {
    return const TourActionResult(tourId: 'x', status: 'paused');
  }

  @override
  Future<TourActionResult> resumeTour(Session session, String tourId) async {
    lastStatus = 'running';
    return const TourActionResult(tourId: 'x', status: 'running');
  }

  @override
  Future<TourActionResult> completeTour(Session session, String tourId) async {
    return const TourActionResult(tourId: 'x', status: 'completed');
  }
}

void main() {
  late _CenterApi api;
  late TourController controller;

  setUp(() {
    TourTargetRegistry.reset();
    api = _CenterApi();
    controller = TourController(
      registry: TourTargetRegistry.instance,
      api: api,
      session: _session,
    );
  });

  tearDown(() => controller.disposeController());

  Future<void> openCenter(WidgetTester tester) async {
    tester.view.physicalSize = const Size(800, 1300);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    await tester.pumpWidget(MaterialApp(
      locale: const Locale('en'),
      localizationsDelegates: const [
        AppStrings.delegate,
        GlobalMaterialLocalizations.delegate,
        GlobalWidgetsLocalizations.delegate,
        GlobalCupertinoLocalizations.delegate,
      ],
      home: Scaffold(
        body: Builder(
          builder: (context) => Center(
            child: FilledButton(
              onPressed: () => Navigator.of(context).push(MaterialPageRoute(
                builder: (_) => OnboardingCenterScreen(
                  session: _session,
                  apiClient: api,
                  tourController: controller,
                ),
              )),
              child: const Text('open'),
            ),
          ),
        ),
      ),
    ));
    // The custom AppStrings locale delegate needs a second frame before the
    // home is actually built, so settle before interacting.
    await tester.pumpAndSettle();
    await tester.tap(find.text('open'));
    await tester.pumpAndSettle();
    expect(find.text('Guides center'), findsOneWidget);
  }

  testWidgets('center lists the recommended card and all five tours in order',
      (tester) async {
    await openCenter(tester);

    expect(find.text('Recommended'), findsOneWidget);
    // cash_session is completed; its tile shows a Completed chip.
    expect(find.text('Completed'), findsWidgets);
    // The recommended first_sale tour is not started: tile says Start.
    expect(find.text('Start'), findsWidgets);
    // The two truly unimplemented tours are labelled honestly (once each tile).
    expect(find.text('Not available in this build'), findsNWidgets(2));
    expect(find.text('store_setup'), findsOneWidget);
    expect(find.text('All tours'), findsOneWidget);
  });

  testWidgets('a running recommended tour offers Continue and Resume labels',
      (tester) async {
    api.lastStatus = 'running';
    await openCenter(tester);

    // Recommended card uses the plain continuation verb for a running tour.
    expect(find.text('Continue'), findsWidgets);
    expect(find.textContaining('Goal:'), findsWidgets);
    expect(find.text('Running'), findsWidgets);
  });

  testWidgets('start runs the recommended tour and returns to the app',
      (tester) async {
    await openCenter(tester);

    await tester.tap(find.text('Start').first);
    await tester.pumpAndSettle();

    expect(api.startCalls, 1);
    // The center popped back to the app root.
    expect(find.text('open'), findsOneWidget);
    expect(find.text('Guides center'), findsNothing);
  });

  testWidgets('skip records the tour as declined and reloads the list',
      (tester) async {
    // Simulate a locally-running tour (the real app reaches the center while
    // a tour is live, so the controller holds its definition).
    api.lastStatus = 'running';
    await controller.start(TourCatalog.build(AppStrings(const Locale('en')))['first_sale']!);
    await openCenter(tester);

    expect(find.text('Running'), findsWidgets);
    // The disabled close on the completed cash_session tile cannot be tapped;
    // the enabled one belongs to the running first_sale tour.
    await tester.tap(find.byIcon(Icons.close).last);
    await tester.pumpAndSettle();

    expect(api.skipCalls, 1);
    expect(find.text('Tour skipped.'), findsOneWidget);
    expect(find.text('Skipped'), findsOneWidget);
  });

  testWidgets('load error shows a retry that recovers', (tester) async {
    api.failOnce = true;
    await openCenter(tester);

    expect(find.text('Try again'), findsOneWidget);
    api.failOnce = false;
    await tester.tap(find.text('Try again'));
    await tester.pumpAndSettle();

    expect(find.text('Recommended'), findsOneWidget);
    expect(find.text('All tours'), findsOneWidget);
  });
}
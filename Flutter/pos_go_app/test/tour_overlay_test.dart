import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:pos_go_app/core/api_client.dart';
import 'package:pos_go_app/core/onboarding.dart';
import 'package:pos_go_app/core/session_store.dart';
import 'package:pos_go_app/core/tour/tour_controller.dart';
import 'package:pos_go_app/core/tour/tour_definitions.dart';
import 'package:pos_go_app/core/tour/tour_overlay.dart';
import 'package:pos_go_app/core/tour/tour_target.dart';
import 'package:pos_go_app/core/tour/tour_target_registry.dart';
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

class _TourApi extends ApiClient {
  _TourApi({this.failCompleteWith});

  String? failCompleteWith;
  int startCalls = 0;
  int completeCalls = 0;
  int stepCalls = 0;
  int pauseCalls = 0;
  int skipCalls = 0;

  @override
  Future<TourActionResult> startTour(Session session, String tourId,
      {int? version}) async {
    startCalls++;
    return const TourActionResult(tourId: 'x', status: 'running');
  }

  @override
  Future<TourActionResult> setTourStep(
      Session session, String tourId, String currentStep) async {
    stepCalls++;
    return const TourActionResult(tourId: 'x', status: 'running');
  }

  @override
  Future<TourActionResult> pauseTour(Session session, String tourId) async {
    pauseCalls++;
    return const TourActionResult(tourId: 'x', status: 'paused');
  }

  @override
  Future<TourActionResult> resumeTour(Session session, String tourId) async {
    return const TourActionResult(tourId: 'x', status: 'running');
  }

  @override
  Future<TourActionResult> skipTour(Session session, String tourId) async {
    skipCalls++;
    return const TourActionResult(tourId: 'x', status: 'skipped');
  }

  @override
  Future<TourActionResult> completeTour(Session session, String tourId) async {
    completeCalls++;
    if (failCompleteWith != null) {
      throw ApiException(failCompleteWith!, code: failCompleteWith);
    }
    return const TourActionResult(tourId: 'x', status: 'completed');
  }
}

Widget _host(TourController controller, Widget target) {
  return MaterialApp(
    locale: const Locale('en'),
    localizationsDelegates: const [
      AppStrings.delegate,
      GlobalMaterialLocalizations.delegate,
      GlobalWidgetsLocalizations.delegate,
      GlobalCupertinoLocalizations.delegate,
    ],
    home: Scaffold(body: TourOverlayHost(controller: controller, child: target)),
  );
}

void main() {
  setUp(TourTargetRegistry.reset);

  testWidgets('overlay shows the bubble next to a registered target',
      (tester) async {
    final api = _TourApi();
    final controller = TourController(
      registry: TourTargetRegistry.instance,
      api: api,
      session: _session,
    );
    addTearDown(controller.disposeController);
    final def = TourCatalog.build(AppStrings(const Locale('en')))['cash_session']!;

    await tester.pumpWidget(_host(
      controller,
      Scaffold(
        body: Align(
          alignment: Alignment.topCenter,
          child: TourTarget(
            id: 'session_bar',
            child: Container(
              width: 300,
              height: 64,
              color: Colors.blue,
              child: const Text('session bar'),
            ),
          ),
        ),
      ),
    ));
    // The custom AppStrings locale delegate renders a placeholder on the very
    // first frame, so settle before the TourTargets register their keys.
    await tester.pumpAndSettle();
    await controller.start(def);
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 300));

    expect(find.text('Cash session'), findsOneWidget);
    expect(find.text('1/4'), findsOneWidget);
    expect(find.text('Next'), findsOneWidget);
    // The spotlight keeps the target visible while the bubble floats below it.
    expect(tester.getTopLeft(find.text('Cash session')).dy,
        greaterThan(tester.getTopLeft(find.text('session bar')).dy));
  });

  testWidgets('overlay advances to the next step on Next and completes on the '
      'last one', (tester) async {
    final api = _TourApi();
    final controller = TourController(
      registry: TourTargetRegistry.instance,
      api: api,
      session: _session,
    );
    addTearDown(controller.disposeController);
    final def = TourCatalog.build(AppStrings(const Locale('en')))['first_sale']!;

    await tester.pumpWidget(_host(
      controller,
      Scaffold(
        body: Center(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              TourTarget(
                  id: 'cart_panel',
                  child: Container(width: 200, height: 60, color: Colors.grey)),
              const SizedBox(height: 200),
              TourTarget(
                  id: 'product_grid',
                  child: Container(width: 200, height: 200, color: Colors.grey)),
              const TourTarget(
                  id: 'checkout_button',
                  child: FilledButton(
                      onPressed: null,
                      child: Text('Complete sale')),
              ),
            ],
          ),
        ),
      ),
    ));
    await tester.pumpAndSettle();
    await controller.start(def);
    await tester.pumpAndSettle();
    expect(find.text('Your first sale'), findsOneWidget);

    await tester.tap(find.text('Next'));
    await tester.pumpAndSettle();
    expect(find.text('Add a product'), findsOneWidget);

    await tester.tap(find.text('Next'));
    await tester.pumpAndSettle();
    expect(find.text('Complete the sale'), findsOneWidget);

    // Last step flips the primary to Finish; completing hits the server.
    await tester.tap(find.text('Next'));
    await tester.pumpAndSettle();
    expect(find.text('Verify the sale'), findsOneWidget);
    expect(find.text('Finish'), findsOneWidget);

    await tester.tap(find.text('Finish'));
    await tester.pumpAndSettle();
    expect(api.completeCalls, 1);
    expect(find.text('Verify the sale'), findsNothing);
    expect(controller.phase, TourPhase.done);
  });

  testWidgets('missing target falls back to the guided card with retry',
      (tester) async {
    final api = _TourApi();
    final controller = TourController(
      registry: TourTargetRegistry.instance,
      api: api,
      session: _session,
    );
    addTearDown(controller.disposeController);
    final def = TourCatalog.build(AppStrings(const Locale('en')))['cash_session']!;

    // No 'session_bar' target registered: the spotlight cannot resolve.
    await tester.pumpWidget(
        _host(controller, const Scaffold(body: SizedBox())));
    await tester.pumpAndSettle();
    await controller.start(def);
    await tester.pumpAndSettle();

    expect(find.text('Try again'), findsOneWidget);
    expect(find.text('Skip step'), findsOneWidget);
    expect(find.text('Exit tour'), findsOneWidget);

    // Registering the target makes the overlay re-resolve the rect and the
    // bubble replaces the fallback on the next frame.
    await tester.pumpWidget(_host(
      controller,
      Scaffold(
        body: Center(
          child: TourTarget(
            id: 'session_bar',
            child: Container(width: 200, height: 60, color: Colors.blue),
          ),
        ),
      ),
    ));
    await tester.pumpAndSettle();
    expect(find.text('Cash session'), findsOneWidget);
    expect(find.text('Try again'), findsNothing);
  });

  testWidgets('goal-not-met refusal surfaces in the bubble with Check again',
      (tester) async {
    final api = _TourApi(failCompleteWith: 'goal_not_met');
    final controller = TourController(
      registry: TourTargetRegistry.instance,
      api: api,
      session: _session,
    );
    addTearDown(controller.disposeController);
    final def = TourCatalog.build(AppStrings(const Locale('en')))['first_sale']!;

    await tester.pumpWidget(_host(
      controller,
      Scaffold(
        body: Center(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              TourTarget(
                  id: 'cart_panel',
                  child: Container(width: 200, height: 60, color: Colors.grey)),
              const SizedBox(height: 200),
              const TourTarget(
                  id: 'checkout_button',
                  child: FilledButton(onPressed: null, child: Text('x')),
              ),
            ],
          ),
        ),
      ),
    ));
    await tester.pumpAndSettle();
    await controller.start(def);
    await tester.pumpAndSettle();
    // Walk every step so the final one triggers the complete.
    await controller.next();
    await controller.next();
    await controller.next();
    await controller.next();
    await tester.pumpAndSettle();
    expect(controller.goalNotMet, isTrue);
    expect(find.text('Check again'), findsOneWidget);
    expect(find.text('Keep going'), findsOneWidget);

    api.failCompleteWith = null;
    await controller.complete();
    await tester.pumpAndSettle();
    expect(controller.phase, TourPhase.done);
    expect(find.text('Check again'), findsNothing);
  });

  testWidgets('the overlay host renders the app untouched when idle',
      (tester) async {
    final api = _TourApi();
    final controller = TourController(
      registry: TourTargetRegistry.instance,
      api: api,
      session: _session,
    );
    addTearDown(controller.disposeController);
    await tester.pumpWidget(
        _host(controller, const Scaffold(body: Text('plain app'))));
    await tester.pumpAndSettle();
    expect(find.text('plain app'), findsOneWidget);
    expect(find.text('Next'), findsNothing);
  });
}
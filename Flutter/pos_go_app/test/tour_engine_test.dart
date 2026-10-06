import 'package:flutter/widgets.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:pos_go_app/core/api_client.dart';
import 'package:pos_go_app/core/onboarding.dart';
import 'package:pos_go_app/core/session_store.dart';
import 'package:pos_go_app/core/tour/tour_controller.dart';
import 'package:pos_go_app/core/tour/tour_definitions.dart';
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

TourCatalog _catalog() => TourCatalog.build(AppStrings(const Locale('en')));

class _TourApi extends ApiClient {
  _TourApi();

  String? failStartWith;
  String? failCompleteWith;
  final List<String> calls = [];
  final List<String> steps = [];

  @override
  Future<TourActionResult> startTour(Session session, String tourId,
      {int? version}) async {
    calls.add('start:$tourId');
    if (failStartWith != null) {
      throw ApiException(failStartWith!, code: failStartWith);
    }
    return const TourActionResult(tourId: 'x', status: 'running');
  }

  @override
  Future<TourActionResult> setTourStep(
      Session session, String tourId, String currentStep) async {
    calls.add('step:$currentStep');
    steps.add(currentStep);
    return TourActionResult(tourId: tourId, status: 'running');
  }

  @override
  Future<TourActionResult> pauseTour(Session session, String tourId) async {
    calls.add('pause:$tourId');
    return const TourActionResult(tourId: 'x', status: 'paused');
  }

  @override
  Future<TourActionResult> resumeTour(Session session, String tourId) async {
    calls.add('resume:$tourId');
    return const TourActionResult(tourId: 'x', status: 'running');
  }

  @override
  Future<TourActionResult> skipTour(Session session, String tourId) async {
    calls.add('skip:$tourId');
    return const TourActionResult(tourId: 'x', status: 'skipped');
  }

  @override
  Future<TourActionResult> completeTour(Session session, String tourId) async {
    calls.add('complete:$tourId');
    if (failCompleteWith != null) {
      throw ApiException(failCompleteWith!, code: failCompleteWith);
    }
    return const TourActionResult(tourId: 'x', status: 'completed');
  }
}

void main() {
  late _TourApi api;
  late TourController controller;

  setUp(() {
    TourTargetRegistry.reset();
    api = _TourApi();
    controller = TourController(
      registry: TourTargetRegistry.instance,
      api: api,
      session: _session,
    );
  });

  tearDown(() => controller.disposeController());

  test('start runs the catalog tour, persists the first step and notifies',
      () async {
    TourTargetRegistry.instance.register('session_bar', GlobalKey());
    final err = await controller.start(_catalog()['cash_session']!);
    expect(err, isNull);
    expect(controller.isRunning, isTrue);
    expect(controller.currentStep!.id, 'intro');
    expect(controller.definition!.id, 'cash_session');
    expect(api.steps, contains('intro'));
  });

  test('start refusal is returned as a machine code, phase stays idle',
      () async {
    api.failStartWith = 'tour_version_mismatch';
    final err = await controller.start(_catalog()['first_sale']!);
    expect(err, 'tour_version_mismatch');
    expect(controller.isIdle, isTrue);
  });

  test('start on an empty (unimplemented) tour flags the target missing',
      () async {
    final def = _catalog()['store_setup']!;
    expect(def.locallyImplementable, isFalse);
    final err = await controller.start(def);
    expect(err, isNull);
    expect(controller.isRunning, isTrue);
    expect(controller.targetMissing, isTrue);
  });

  test('start runs the terminal_setup tour with its three printer steps',
      () async {
    final def = _catalog()['terminal_setup']!;
    expect(def.locallyImplementable, isTrue);
    expect(def.steps.map((e) => e.target),
        ['printers_button', 'printers_enable', 'printers_test']);
    final err = await controller.start(def);
    expect(err, isNull);
    expect(controller.currentStep!.id, 'intro');
    await controller.next(); // enable
    expect(controller.currentStep!.target, 'printers_enable');
    expect(controller.currentStep!.optional, isFalse);
    await controller.next(); // verify
    expect(controller.currentStep!.target, 'printers_test');
    expect(controller.currentStep!.optional, isTrue);
    await controller.next(); // completes on the last step
    expect(controller.phase, TourPhase.done);
    expect(api.calls, contains('complete:terminal_setup'));
  });

  test('next advances and persists each step; on the last step it completes',
      () async {
    final err = await controller.start(_catalog()['cash_session']!);
    expect(err, isNull);
    await controller.next();
    expect(controller.currentStep!.id, 'start');
    await controller.next();
    expect(controller.currentStep!.id, 'starting_cash');
    await controller.next();
    expect(controller.currentStep!.id, 'verify');
    await controller.next();
    expect(controller.phase, TourPhase.done);
    expect(api.calls, contains('complete:cash_session'));
  });

  test('complete refuses goal_not_met and stays running for retry', () async {
    api.failCompleteWith = 'goal_not_met';
    await controller.start(_catalog()['cash_session']!);
    await controller.next(); // start
    await controller.next(); // starting_cash
    await controller.next(); // verify
    final err = await controller.complete();
    expect(err, 'goal_not_met');
    expect(controller.goalNotMet, isTrue);
    expect(controller.isRunning, isTrue);

    api.failCompleteWith = null;
    final ok = await controller.complete();
    expect(ok, isNull);
    expect(controller.phase, TourPhase.done);
  });

  test('complete on an already-completed tour returns null (idempotent)',
      () async {
    api.failCompleteWith = 'tour_completed';
    await controller.start(_catalog()['cash_session']!);
    expect(await controller.complete(), isNull);
    expect(controller.phase, TourPhase.done);
  });

  test('skipStep skips ahead; on the last step it completes', () async {
    await controller.start(_catalog()['cash_session']!);
    await controller.next(); // start
    await controller.skipStep(); // starting_cash
    expect(controller.currentStep!.id, 'starting_cash');
    await controller.skipStep(); // verify (last)
    expect(controller.currentStep!.id, 'verify');
    await controller.skipStep();
    expect(controller.phase, TourPhase.done);
  });

  test('pause then exit drops the overlay without skipping server-side',
      () async {
    await controller.start(_catalog()['first_sale']!);
    expect(await controller.pause(), isNull);
    expect(controller.isPaused, isTrue);
    expect(api.calls, contains('pause:first_sale'));
    expect(await controller.exit(), isNull);
    expect(controller.isIdle, isTrue);
    expect(api.calls.where((c) => c.startsWith('skip:')), isEmpty);
  });

  test('resume from a cold start adopts the definition and restarts the tour',
      () async {
    // Simulate a server-side paused tour reached from the center screen with
    // no locally-held definition (fresh app start).
    final err = await controller.resume(_catalog()['first_sale']!);
    expect(err, isNull);
    expect(controller.isRunning, isTrue);
    expect(controller.definition!.id, 'first_sale');
    expect(api.calls, contains('resume:first_sale'));
  });

  test('skip records the whole tour as declined and clears the definition',
      () async {
    await controller.start(_catalog()['first_sale']!);
    expect(await controller.skip(), isNull);
    expect(controller.isIdle, isTrue);
    expect(api.calls, contains('skip:first_sale'));
    expect(controller.definition, isNull);
  });

  test('handleBusinessAction jumps to the last step and verifies (sale)',
      () async {
    await controller.start(_catalog()['first_sale']!);
    expect(controller.currentStep!.id, 'intro');
    await controller.handleBusinessAction('sale');
    expect(controller.phase, TourPhase.done);
    expect(api.calls, contains('complete:first_sale'));
  });

  test('handleBusinessAction ignores other tours and actions', () async {
    await controller.start(_catalog()['cash_session']!);
    await controller.handleBusinessAction('sale');
    expect(controller.isRunning, isTrue);
    expect(api.calls.where((c) => c.startsWith('complete:')), isEmpty);
  });

  test('optional targets missing are not reported as missing', () async {
    await controller.start(_catalog()['first_sale']!);
    await controller.next(); // add_item is optional
    expect(controller.currentStep!.id, 'add_item');
    expect(controller.currentStep!.optional, isTrue);
    expect(controller.targetMissing, isFalse);
  });

  test('refreshTarget reports a non-optional missing target', () async {
    await controller.start(_catalog()['cash_session']!);
    controller.refreshTarget();
    expect(controller.targetMissing, isTrue);
    controller.retryResolution();
    expect(controller.targetMissing, isFalse);
  });

  test('targetExists resolves registry keys', () {
    expect(controller.targetExists('session_bar'), isFalse);
    TourTargetRegistry.instance.register('session_bar', GlobalKey());
    expect(controller.targetExists('session_bar'), isTrue);
  });

  test('background persists the current step then server-pauses', () async {
    await controller.start(_catalog()['cash_session']!);
    await controller.next(); // start
    await controller.background();
    expect(api.steps, contains('start'));
    expect(api.calls, contains('pause:cash_session'));
  });
}
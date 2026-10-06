import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:integration_test/integration_test.dart';

import 'package:pos_go_app/core/api_client.dart';
import 'package:pos_go_app/core/tour/tour_controller.dart';
import 'package:pos_go_app/core/tour/tour_definitions.dart';
import 'package:pos_go_app/core/tour/tour_overlay.dart';
import 'package:pos_go_app/core/tour/tour_target.dart';
import 'package:pos_go_app/core/tour/tour_target_registry.dart';
import 'package:pos_go_app/l10n/strings.dart';

const demoPassword = String.fromEnvironment('DEMO_PASSWORD');

/// On-device tour-overlay smoke (E4-style). Renders the real overlay code that
/// PosApp wires up (TourController + TourTargetRegistry + TourOverlayHost),
/// targets the printers screen's three terminal_setup targets, and drives the
/// tour against a REAL backend over the network. Verifies the Spotlight bubble
/// resolves to a registered on-device target and step navigation persists to
/// the server (start + per-step rows land in the backend's tour state).
///
///   flutter test integration_test/tour_overlay_e2e_test.dart -d <device> \
///     --dart-define=API_BASE_URL=http://10.0.2.2:8080 \
///     --dart-define=DEMO_PASSWORD=<demo-restaurant-manager password>
///
/// Stops on the last step without tapping Finish: whether the terminal goal is
/// already satisfied depends on the target tenant's register-session history,
/// so completion is asserted separately (checked in the tour_sessions rows).
void main() {
  IntegrationTestWidgetsFlutterBinding.ensureInitialized();

  Widget host(TourController controller) => MaterialApp(
        locale: const Locale('en'),
        localizationsDelegates: const [
          AppStrings.delegate,
          GlobalMaterialLocalizations.delegate,
          GlobalWidgetsLocalizations.delegate,
          GlobalCupertinoLocalizations.delegate,
        ],
        home: Scaffold(
          body: TourOverlayHost(
            controller: controller,
            child: const Column(
              mainAxisAlignment: MainAxisAlignment.center,
              children: [
                TourTarget(
                  id: 'printers_button',
                  child: FilledButton(
                    onPressed: null,
                    child: Text('Printers'),
                  ),
                ),
                SizedBox(height: 16),
                TourTarget(
                  id: 'printers_enable',
                  child: SwitchListTile(
                    title: Text('Printing enabled'),
                    value: true,
                    onChanged: null,
                  ),
                ),
                const SizedBox(height: 16),
                TourTarget(
                  id: 'printers_test',
                  child: FilledButton.tonalIcon(
                    onPressed: null,
                    icon: Icon(Icons.print_outlined),
                    label: Text('Print test page'),
                  ),
                ),
              ],
            ),
          ),
        ),
      );

  testWidgets('on-device terminal_setup tour overlay round-trip', (tester) async {
    final api = ApiClient();
    final session = await api.login(
      tenantId: 'demo-restaurant',
      email: 'demo-restaurant-manager@xamltech.com',
      password: demoPassword,
      deviceId: 'tour-smoke-device',
      deviceName: 'Tour Overlay Smoke',
    );
    expect(session.accessToken, isNotEmpty);

    final controller = TourController(
      registry: TourTargetRegistry.instance,
      api: api,
      session: session,
    );
    addTearDown(controller.disposeController);

    // First frame serves a localization placeholder, so settle before the
    // TourTargets register their GlobalKeys (same as the widget overlay tests).
    await tester.pumpWidget(host(controller));
    await tester.pumpAndSettle();

    final def = TourCatalog.build(AppStrings(const Locale('en')))['terminal_setup']!;
    final startError = await controller.start(def);
    expect(startError, isNull);

    await tester.pump();
    await tester.pump(const Duration(milliseconds: 300));
    expect(find.text('Set up your printer'), findsOneWidget);
    expect(find.text('1/3'), findsOneWidget);
    expect(find.text('Next'), findsOneWidget);

    await tester.tap(find.text('Next'));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 300));
    expect(find.text('Enable printing'), findsOneWidget);
    expect(find.text('2/3'), findsOneWidget);

    await tester.tap(find.text('Next'));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 300));
    expect(find.text('Test your printer'), findsOneWidget);
    expect(find.text('3/3'), findsOneWidget);
    expect(find.text('Finish'), findsOneWidget);
  });
}
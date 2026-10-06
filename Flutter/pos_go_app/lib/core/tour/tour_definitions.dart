import '../../l10n/strings.dart';

/// A single guidance screen inside a tour.
class TourStepDef {
  const TourStepDef({
    required this.id,
    required this.title,
    required this.body,
    required this.target,
    this.optional = false,
  });

  final String id;
  final String title;
  final String body;

  /// The [TourTargetRegistry] key the spotlight homes in on.
  final String target;

  /// When true a missing target is skipped silently instead of offering the
  /// Try again / Skip step / Exit fallback (TOUR_ENGINE.md: missing targets
  /// must never crash the tour, and genuinely optional targets must not nag).
  final bool optional;
}

/// A server-known tour with the local UI steps that teach it.
class TourDefinition {
  const TourDefinition({
    required this.id,
    required this.version,
    required this.goal,
    required this.steps,
  });

  final String id;
  final int version;
  final String goal;
  final List<TourStepDef> steps;

  bool get locallyImplementable => steps.isNotEmpty;
  int get stepCount => steps.length;
}

/// The tour library. Copy is resolved from [AppStrings] at build time so the
/// engine stays locale-agnostic; steps are the *local* UI steps (each maps to
/// a TourTarget in the POS screen). Tours the backend knows about but this
/// build cannot teach yet carry zero steps, which the onboarding center shows
/// as "not on this build" instead of lying that they are done.
class TourCatalog {
  const TourCatalog._(this.byId);

  final Map<String, TourDefinition> byId;

  TourDefinition? operator [](String id) => byId[id];

  static TourCatalog build(AppStrings s) => TourCatalog._({
        'store_setup': const TourDefinition(
          id: 'store_setup',
          version: 1,
          goal: 'store_configured',
          steps: [],
        ),
        'first_product': const TourDefinition(
          id: 'first_product',
          version: 1,
          goal: 'product_ready',
          steps: [],
        ),
        'terminal_setup': TourDefinition(
          id: 'terminal_setup',
          version: 1,
          goal: 'terminal_configured',
          steps: [
            TourStepDef(
              id: 'intro',
              title: s.tourTermIntroTitle,
              body: s.tourTermIntroBody,
              target: 'printers_button',
            ),
            TourStepDef(
              id: 'enable',
              title: s.tourTermEnableTitle,
              body: s.tourTermEnableBody,
              target: 'printers_enable',
            ),
            TourStepDef(
              id: 'verify',
              title: s.tourTermVerifyTitle,
              body: s.tourTermVerifyBody,
              target: 'printers_test',
              // The test button only renders once printing is enabled, so a
              // genuinely absent target must not nag.
              optional: true,
            ),
          ],
        ),
        'cash_session': TourDefinition(
          id: 'cash_session',
          version: 1,
          goal: 'cash_session_opened',
          steps: [
            TourStepDef(
              id: 'intro',
              title: s.tourCashIntroTitle,
              body: s.tourCashIntroBody,
              target: 'session_bar',
            ),
            TourStepDef(
              id: 'start',
              title: s.tourCashStartTitle,
              body: s.tourCashStartBody,
              target: 'session_bar_open_button',
            ),
            TourStepDef(
              id: 'starting_cash',
              title: s.tourCashCashTitle,
              body: s.tourCashCashBody,
              target: 'session_starting_cash_field',
              optional: true,
            ),
            TourStepDef(
              id: 'verify',
              title: s.tourCashVerifyTitle,
              body: s.tourCashVerifyBody,
              target: 'session_bar',
            ),
          ],
        ),
        'first_sale': TourDefinition(
          id: 'first_sale',
          version: 1,
          goal: 'first_sale_completed',
          steps: [
            TourStepDef(
              id: 'intro',
              title: s.tourFirstIntroTitle,
              body: s.tourFirstIntroBody,
              target: 'cart_panel',
            ),
            TourStepDef(
              id: 'add_item',
              title: s.tourFirstAddTitle,
              body: s.tourFirstAddBody,
              target: 'product_grid',
              optional: true,
            ),
            TourStepDef(
              id: 'checkout',
              title: s.tourFirstCheckoutTitle,
              body: s.tourFirstCheckoutBody,
              target: 'checkout_button',
            ),
            TourStepDef(
              id: 'first_sale',
              title: s.tourFirstVerifyTitle,
              body: s.tourFirstVerifyBody,
              target: 'cart_panel',
            ),
          ],
        ),
      });
}
/// Server-authoritative onboarding models (migration 054).
///
/// The backend decides which tours exist, whether a business goal is met and
/// which tour is recommended; clients only render state. See
/// `docs/onboarding/API_CONTRACT.md`.
library;

class OnboardingReadiness {
  const OnboardingReadiness({
    required this.storeConfigured,
    required this.productReady,
    required this.terminalConfigured,
    required this.cashSessionOpened,
    required this.firstSaleCompleted,
    required this.activated,
  });

  factory OnboardingReadiness.fromJson(Map<String, dynamic> json) =>
      OnboardingReadiness(
        storeConfigured: json['store_configured'] as bool? ?? false,
        productReady: json['product_ready'] as bool? ?? false,
        terminalConfigured: json['terminal_configured'] as bool? ?? false,
        cashSessionOpened: json['cash_session_opened'] as bool? ?? false,
        firstSaleCompleted: json['first_sale_completed'] as bool? ?? false,
        activated: json['activated'] as bool? ?? false,
      );

  final bool storeConfigured;
  final bool productReady;
  final bool terminalConfigured;
  final bool cashSessionOpened;
  final bool firstSaleCompleted;
  final bool activated;
}

class TourProgress {
  const TourProgress({
    required this.id,
    required this.version,
    required this.goal,
    required this.status,
    required this.currentStep,
    required this.stepCount,
    required this.goalMet,
    this.startedAt,
    this.pausedAt,
    this.completedAt,
    this.updatedAt,
  });

  factory TourProgress.fromJson(Map<String, dynamic> json) => TourProgress(
        id: json['id'] as String,
        version: (json['version'] as num?)?.toInt() ?? 1,
        goal: json['goal'] as String? ?? '',
        status: json['status'] as String? ?? 'not_started',
        currentStep: json['current_step'] as String? ?? '',
        stepCount: (json['step_count'] as num?)?.toInt() ?? 0,
        goalMet: json['goal_met'] as bool? ?? false,
        startedAt: json['started_at'] as String?,
        pausedAt: json['paused_at'] as String?,
        completedAt: json['completed_at'] as String?,
        updatedAt: json['updated_at'] as String?,
      );

  final String id;
  final int version;
  final String goal;
  final String status;
  final String currentStep;
  final int stepCount;
  final bool goalMet;
  final String? startedAt;
  final String? pausedAt;
  final String? completedAt;
  final String? updatedAt;

  bool get isRunning => status == 'running';
  bool get isPaused => status == 'paused';
  bool get isCompleted => status == 'completed';
  bool get isSkipped => status == 'skipped';
  bool get isInFlight => isRunning || isPaused;
}

class OnboardingState {
  const OnboardingState({
    required this.tours,
    required this.readiness,
    this.recommended = '',
  });

  factory OnboardingState.fromJson(Map<String, dynamic> json) =>
      OnboardingState(
        tours: (json['tours'] as List<dynamic>? ?? [])
            .map((row) => TourProgress.fromJson(row as Map<String, dynamic>))
            .toList(),
        readiness: OnboardingReadiness.fromJson(
            json['readiness'] as Map<String, dynamic>? ?? const {}),
        recommended: json['recommended'] as String? ?? '',
      );

  final List<TourProgress> tours;
  final OnboardingReadiness readiness;
  final String recommended;

  TourProgress? tour(String id) {
    for (final row in tours) {
      if (row.id == id) return row;
    }
    return null;
  }
}

class TourActionResult {
  const TourActionResult({
    required this.tourId,
    this.status = '',
    this.currentStep,
  });

  factory TourActionResult.fromJson(Map<String, dynamic> json) =>
      TourActionResult(
        tourId: json['tour'] as String? ?? '',
        status: json['status'] as String? ?? '',
        currentStep: json['current_step'] as String?,
      );

  final String tourId;
  final String status;
  final String? currentStep;
}

class OnboardingEventResult {
  const OnboardingEventResult({
    required this.eventId,
    required this.event,
    required this.replayed,
  });

  factory OnboardingEventResult.fromJson(Map<String, dynamic> json) =>
      OnboardingEventResult(
        eventId: json['event_id'] as String? ?? '',
        event: json['event'] as String? ?? '',
        replayed: json['replayed'] as bool? ?? false,
      );

  final String eventId;
  final String event;
  final bool replayed;
}
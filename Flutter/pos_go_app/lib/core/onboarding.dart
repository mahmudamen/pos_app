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
    this.reward = const OnboardingReward(),
  });

  factory OnboardingState.fromJson(Map<String, dynamic> json) =>
      OnboardingState(
        tours: (json['tours'] as List<dynamic>? ?? [])
            .map((row) => TourProgress.fromJson(row as Map<String, dynamic>))
            .toList(),
        readiness: OnboardingReadiness.fromJson(
            json['readiness'] as Map<String, dynamic>? ?? const {}),
        recommended: json['recommended'] as String? ?? '',
        reward: OnboardingReward.fromJson(
            json['reward'] as Map<String, dynamic>? ?? const {}),
      );

  final List<TourProgress> tours;
  final OnboardingReadiness readiness;
  final String recommended;
  final OnboardingReward reward;

  TourProgress? tour(String id) {
    for (final row in tours) {
      if (row.id == id) return row;
    }
    return null;
  }
}

/// The activation reward (EPIC 11). Eligibility is server-authoritative: it is
/// true only once the tenant activated with a completed, paid sale.
class OnboardingReward {
  const OnboardingReward({
    this.eligible = false,
    this.granted = false,
    this.rewardType = '',
    this.grantedAt,
    this.replayed = false,
  });

  factory OnboardingReward.fromJson(Map<String, dynamic> json) =>
      OnboardingReward(
        eligible: json['eligible'] as bool? ?? false,
        granted: json['granted'] as bool? ?? false,
        rewardType: json['reward_type'] as String? ?? '',
        grantedAt: json['granted_at'] as String?,
        replayed: json['replayed'] as bool? ?? false,
      );

  final bool eligible;
  final bool granted;
  final String rewardType;

  /// ISO-8601 time of the grant; null when not yet granted.
  final String? grantedAt;
  final bool replayed;
}

/// Tenant-wide onboarding + activation aggregates (EPIC 13). Manager-only.
class OnboardingSummary {
  const OnboardingSummary({
    required this.totals,
    required this.completionRate,
    required this.skipRate,
    required this.events,
    required this.rewardGrants,
    required this.activated,
    this.timeToFirstSaleSeconds,
    this.tours = const [],
  });

  factory OnboardingSummary.fromJson(Map<String, dynamic> json) {
    final totals =
        json['totals'] as Map<String, dynamic>? ?? const <String, dynamic>{};
    final rates =
        json['rates'] as Map<String, dynamic>? ?? const <String, dynamic>{};
    return OnboardingSummary(
      totals: TourTotals.fromJson(totals),
      completionRate: (rates['completion_rate'] as num?)?.toDouble() ?? 0,
      skipRate: (rates['skip_rate'] as num?)?.toDouble() ?? 0,
      events: (json['events'] as Map<String, dynamic>? ?? const {})
          .map((key, value) => MapEntry(key, (value as num?)?.toInt() ?? 0)),
      rewardGrants: (json['reward_grants'] as num?)?.toInt() ?? 0,
      activated: json['activated'] as bool? ?? false,
      timeToFirstSaleSeconds:
          (json['time_to_first_sale_seconds'] as num?)?.toInt(),
      tours: (json['tours'] as List<dynamic>? ?? [])
          .map((row) =>
              TourAggregate.fromJson(row as Map<String, dynamic>))
          .toList(),
    );
  }

  final TourTotals totals;
  final double completionRate;
  final double skipRate;
  final Map<String, int> events;
  final int rewardGrants;
  final bool activated;
  final int? timeToFirstSaleSeconds;
  final List<TourAggregate> tours;
}

class TourTotals {
  const TourTotals({
    this.started = 0,
    this.completed = 0,
    this.skipped = 0,
    this.inProgress = 0,
  });

  factory TourTotals.fromJson(Map<String, dynamic> json) => TourTotals(
        started: (json['started'] as num?)?.toInt() ?? 0,
        completed: (json['completed'] as num?)?.toInt() ?? 0,
        skipped: (json['skipped'] as num?)?.toInt() ?? 0,
        inProgress: (json['in_progress'] as num?)?.toInt() ?? 0,
      );

  final int started;
  final int completed;
  final int skipped;
  final int inProgress;
}

class TourAggregate {
  const TourAggregate({
    required this.id,
    this.started = 0,
    this.completed = 0,
    this.skipped = 0,
    this.inProgress = 0,
  });

  factory TourAggregate.fromJson(Map<String, dynamic> json) => TourAggregate(
        id: json['id'] as String? ?? '',
        started: (json['started'] as num?)?.toInt() ?? 0,
        completed: (json['completed'] as num?)?.toInt() ?? 0,
        skipped: (json['skipped'] as num?)?.toInt() ?? 0,
        inProgress: (json['in_progress'] as num?)?.toInt() ?? 0,
      );

  final String id;
  final int started;
  final int completed;
  final int skipped;
  final int inProgress;
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
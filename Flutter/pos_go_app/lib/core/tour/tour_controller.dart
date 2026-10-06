import 'package:flutter/foundation.dart';

import '../api_client.dart';
import '../session_store.dart';
import 'tour_definitions.dart';
import 'tour_target_registry.dart';

/// Lifecycle phases of a guided tour (TOUR_ENGINE.md):
/// LOAD → ELIGIBILITY → START → RESOLVE STEP → NAVIGATE → WAIT TARGET →
/// SHOW GUIDANCE → WAIT ACTION → VERIFY → NEXT → COMPLETE.
enum TourPhase { idle, running, paused, completing, done }

/// Drives a single guided tour. The controller owns the server state (start /
/// step / pause / resume / skip / complete), the overlay widget renders it.
/// A tour can only complete when the backend's goal verification succeeds —
/// the client cannot lie about a business outcome.
class TourController extends ChangeNotifier {
  TourController({
    required TourTargetRegistry registry,
    required ApiClient api,
    required Session session,
  })  : _registry = registry,
        _api = api,
        _session = session;

  final TourTargetRegistry _registry;
  final ApiClient _api;
  final Session _session;

  TourPhase _phase = TourPhase.idle;
  TourDefinition? _def;
  int _stepIndex = 0;
  bool _targetMissing = false;
  bool _goalNotMet = false;
  String? _lastError;

  void disposeController() {
    _registry.clear();
    dispose();
  }

  TourPhase get phase => _phase;
  bool get isIdle => _phase == TourPhase.idle;
  bool get isActive => _phase == TourPhase.running || _phase == TourPhase.paused;
  bool get isRunning => _phase == TourPhase.running;
  bool get isPaused => _phase == TourPhase.paused;

  TourDefinition? get definition => _def;
  TourStepDef? get currentStep {
    if (_def == null || _stepIndex < 0 || _stepIndex >= _def!.steps.length) {
      return null;
    }
    return _def!.steps[_stepIndex];
  }

  int get stepIndex => _stepIndex;
  bool get onLastStep => _def != null && _stepIndex >= _def!.steps.length - 1;

  /// True when the spotlight target is not currently in the registry — the
  /// overlay shows the guided fallback instead of crashing the tour.
  bool get targetMissing => _targetMissing;

  /// True after a [complete] refused with `goal_not_met`: the user did the
  /// action but the business fact is not there yet, so we tell the truth.
  bool get goalNotMet => _goalNotMet;

  String? get lastError => _lastError;

  bool targetExists(String id) => _registry.has(id);

  /// START. Returns null on success or a machine error code on refusal
  /// (tour_version_mismatch, tour_completed, …).
  Future<String?> start(TourDefinition def) async {
    try {
      await _api.startTour(_session, def.id, version: def.version);
    } on ApiException catch (e) {
      _lastError = e.code ?? 'start_failed';
      return _lastError;
    } catch (_) {
      _lastError = 'start_failed';
      return _lastError;
    }
    _def = def;
    _stepIndex = 0;
    _phase = TourPhase.running;
    _targetMissing = def.steps.isEmpty;
    _goalNotMet = false;
    _lastError = null;
    await _persistStep();
    notifyListeners();
    return null;
  }

  Future<void> _persistStep() async {
    final step = currentStep;
    if (_def == null || step == null) return;
    try {
      await _api.setTourStep(_session, _def!.id, step.id);
      // Best-effort: the backend keeps the resume position; if this fails the
      // tour still runs locally and the mismatch resolves on the next step.
    } catch (_) {}
  }

  /// NAVIGATE → next step, or COMPLETE when already on the last step.
  Future<void> next() async {
    if (!isActive || _def == null) return;
    if (_stepIndex < _def!.steps.length - 1) {
      _stepIndex++;
      _targetMissing = false;
      _goalNotMet = false;
      await _persistStep();
      notifyListeners();
      return;
    }
    await complete();
  }

  /// Recompute the spotlight after the user waited / scrolled (fallback Retry).
  void retryResolution() {
    _targetMissing = false;
    notifyListeners();
  }

  /// Skip the current step (fallback action when its target never appeared).
  Future<void> skipStep() async {
    if (_stepIndex < _def!.steps.length - 1) {
      _stepIndex++;
      _targetMissing = false;
      await _persistStep();
      notifyListeners();
      return;
    }
    await complete();
  }

  /// Local pause without a server call; the overlay calls [resume] to replay
  /// the current step. Used when the target is temporarily hidden.
  void softPause() {
    if (_phase == TourPhase.running) {
      _phase = TourPhase.paused;
      notifyListeners();
    }
  }

  /// VERIFY + COMPLETE. The server re-checks the business goal and answers
  /// `goal_not_met` until the fact really exists; that refusal is surfaced
  /// (returned as `goal_not_met`, [goalNotMet] = true) and the tour stays
  /// running so the user can finish the work.
  Future<String?> complete() async {
    if (_def == null) return 'not_running';
    _phase = TourPhase.completing;
    notifyListeners();
    try {
      await _api.completeTour(_session, _def!.id);
      _phase = TourPhase.done;
      _goalNotMet = false;
      _lastError = null;
      notifyListeners();
      return null;
    } on ApiException catch (e) {
      _phase = TourPhase.running;
      if (e.isGoalNotMet) {
        _goalNotMet = true;
        notifyListeners();
        return 'goal_not_met';
      }
      if (e.isTourCompleted) {
        _phase = TourPhase.done;
        notifyListeners();
        return null;
      }
      _lastError = e.code ?? 'complete_failed';
      notifyListeners();
      return _lastError;
    } catch (_) {
      _phase = TourPhase.running;
      _lastError = 'complete_failed';
      notifyListeners();
      return _lastError;
    }
  }

  /// PAUSE from the overlay: the server remembers the position so the tour can
  /// be resumed later (even from another device).
  Future<String?> pause() async {
    if (_def == null) return 'not_running';
    try {
      await _api.pauseTour(_session, _def!.id);
    } on ApiException catch (e) {
      _lastError = e.code ?? 'pause_failed';
      return _lastError;
    } catch (_) {
      _lastError = 'pause_failed';
      return _lastError;
    }
    _phase = TourPhase.paused;
    _lastError = null;
    notifyListeners();
    return null;
  }

  Future<String?> resume(TourDefinition def) async {
    _def ??= def;
    if (_def == null) return 'not_running';
    try {
      await _api.resumeTour(_session, _def!.id);
    } on ApiException catch (e) {
      _lastError = e.code ?? 'resume_failed';
      return _lastError;
    } catch (_) {
      _lastError = 'resume_failed';
      return _lastError;
    }
    _phase = TourPhase.running;
    _goalNotMet = false;
    _lastError = null;
    notifyListeners();
    return null;
  }

  /// SKIP: the server records the whole tour as declined.
  Future<String?> skip() async {
    if (_def == null) return 'not_running';
    try {
      await _api.skipTour(_session, _def!.id);
    } on ApiException catch (e) {
      _lastError = e.code ?? 'skip_failed';
      return _lastError;
    } catch (_) {
      _lastError = 'skip_failed';
      return _lastError;
    }
    _def = null;
    _stepIndex = 0;
    _phase = TourPhase.idle;
    _lastError = null;
    notifyListeners();
    return null;
  }

  /// EXIT: pause the tour and drop the overlay (no server skip).
  Future<String?> exit() async {
    if (_def == null) return 'not_running';
    final err = await pause();
    _phase = TourPhase.idle;
    _goalNotMet = false;
    notifyListeners();
    return err;
  }

  /// The tour keeps its progress when the app goes to background / screen
  /// closes: persist the current step, then pause on the server.
  Future<void> background() async {
    if (!isActive || _def == null) return;
    await _persistStep();
    try {
      await _api.pauseTour(_session, _def!.id);
    } catch (_) {}
  }

  /// WAIT ACTION hook: called by the app when the business action a step asks
  /// for actually happened (a session opened, a sale completed). Guarantees
  /// the tour advances past the actionable step and then lets the server
  /// verify the goal.
  Future<void> handleBusinessAction(String action) async {
    if (_def == null || !isRunning) return;
    final matches =
        (_def!.id == 'cash_session' && action == 'open') ||
        (_def!.id == 'first_sale' && action == 'sale');
    if (!matches) return;
    if (!onLastStep) {
      _stepIndex = _def!.steps.length - 1;
      await _persistStep();
    }
    await complete();
  }

  /// Recompute whether the current target exists (after rebuilds / navigation).
  void refreshTarget() {
    if (_def == null) return;
    final step = currentStep;
    if (step == null) return;
    final missing = step.optional == false && !_registry.has(step.target);
    if (missing != _targetMissing) {
      _targetMissing = missing;
      notifyListeners();
    }
  }
}
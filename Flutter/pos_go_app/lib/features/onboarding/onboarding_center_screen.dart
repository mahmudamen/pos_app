import 'package:flutter/material.dart';

import '../../core/api_client.dart';
import '../../core/onboarding.dart';
import '../../core/session_store.dart';
import '../../core/tour/tour_controller.dart';
import '../../core/tour/tour_definitions.dart';
import '../../l10n/strings.dart';

/// Onboarding center (EPIC 12): lists every server-known tour with its goal and
/// status, highlights the server-recommended one, and starts / resumes /
/// skips a tour. Tours the current build cannot teach yet are shown as "not
/// available in this build" instead of pretending they exist.
class OnboardingCenterScreen extends StatefulWidget {
  const OnboardingCenterScreen({
    super.key,
    required this.session,
    required this.apiClient,
    required this.tourController,
  });

  final Session session;
  final ApiClient apiClient;
  final TourController tourController;

  @override
  State<OnboardingCenterScreen> createState() => _OnboardingCenterScreenState();
}

class _OnboardingCenterScreenState extends State<OnboardingCenterScreen> {
  static const _order = [
    'store_setup',
    'first_product',
    'terminal_setup',
    'cash_session',
    'first_sale',
  ];

  OnboardingState? _state;
  String? _error;
  bool _loading = true;
  bool _busy = false;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    try {
      final state = await widget.apiClient.onboardingState(widget.session);
      if (!mounted) return;
      setState(() {
        _state = state;
        _error = null;
        _loading = false;
      });
    } catch (error) {
      if (!mounted) return;
      setState(() {
        _error = error.toString();
        _loading = false;
      });
    }
  }

  void _snack(String message) {
    ScaffoldMessenger.of(context)
        .showSnackBar(SnackBar(content: Text(message)));
  }

  String _actionError(String? code) {
    final s = AppStrings.of(context);
    switch (code) {
      case 'tour_version_mismatch':
        return s.tourVersionMismatch;
      case 'tour_completed':
        return s.tourAlreadyDone;
      case 'tour_not_running':
        return s.tourNotRunning;
      default:
        return s.tourActionFailed;
    }
  }

  Future<void> _start(TourDefinition def) async {
    final err = await widget.tourController.start(def);
    if (!mounted) return;
    if (err == null) {
      Navigator.of(context).pop();
      return;
    }
    _snack(_actionError(err));
  }

  Future<void> _resume(TourDefinition def) async {
    final err = await widget.tourController.resume(def);
    if (!mounted) return;
    if (err == null) {
      Navigator.of(context).pop();
      return;
    }
    _snack(_actionError(err));
  }

  Future<void> _skip(TourDefinition def, TourProgress progress) async {
    if (progress.isCompleted) {
      _snack(AppStrings.of(context).tourAlreadyDone);
      return;
    }
    if (_busy) return;
    setState(() => _busy = true);
    String? err;
    try {
      err = await widget.tourController.skip();
    } finally {
      if (mounted) setState(() => _busy = false);
    }
    if (!mounted) return;
    if (err == null) _snack(AppStrings.of(context).tourSkipped);
    await _load();
  }

  @override
  Widget build(BuildContext context) {
    final s = AppStrings.of(context);
    final catalog = TourCatalog.build(s);
    final state = _state;

    return Scaffold(
      appBar: AppBar(title: Text(s.onboardingCenterTitle)),
      body: _loading
          ? const Center(child: CircularProgressIndicator())
          : _error != null
              ? Center(
                  child: Column(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      Text(_error!),
                      const SizedBox(height: 16),
                      FilledButton(
                        onPressed: () {
                          setState(() {
                            _loading = true;
                            _error = null;
                          });
                          _load();
                        },
                        child: Text(s.tourTryAgain),
                      ),
                    ],
                  ),
                )
              : RefreshIndicator(
                  onRefresh: _load,
                  child: ListView(
                    padding: const EdgeInsets.fromLTRB(16, 8, 16, 24),
                    children: [
                      Text(s.onboardingCenterSubtitle,
                          style: Theme.of(context).textTheme.bodyMedium),
                      const SizedBox(height: 16),
                      if (state != null && state.recommended.isNotEmpty)
                        _RecommendedCard(
                          strings: s,
                          definition: catalog[state.recommended],
                          progress: state.tour(state.recommended),
                          busy: _busy,
                          onStart: _start,
                          onResume: _resume,
                        ),
                      const SizedBox(height: 16),
                      Text(
                        s.tourAllTours,
                        style: Theme.of(context).textTheme.titleMedium,
                      ),
                      const SizedBox(height: 8),
                      for (final id in _order)
                        if (catalog[id] != null)
                          _TourTile(
                            strings: s,
                            definition: catalog[id]!,
                            progress: state?.tour(id),
                            busy: _busy,
                            onStart: _start,
                            onResume: _resume,
                            onSkip: _skip,
                          ),
                    ],
                  ),
                ),
    );
  }
}

class _RecommendedCard extends StatelessWidget {
  const _RecommendedCard({
    required this.strings,
    required this.definition,
    required this.progress,
    required this.busy,
    required this.onStart,
    required this.onResume,
  });

  final AppStrings strings;
  final TourDefinition? definition;
  final TourProgress? progress;
  final bool busy;
  final void Function(TourDefinition) onStart;
  final void Function(TourDefinition) onResume;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final def = definition;
    final p = progress;
    if (def == null || p == null) return const SizedBox.shrink();

    return Card(
      color: theme.colorScheme.primaryContainer,
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Icon(Icons.star_rounded, color: theme.colorScheme.primary),
                const SizedBox(width: 8),
                Text(
                  strings.tourRecommended,
                  style: theme.textTheme.titleSmall?.copyWith(
                    color: theme.colorScheme.primary,
                    fontWeight: FontWeight.w700,
                  ),
                ),
              ],
            ),
            const SizedBox(height: 8),
            Text(def.steps.isEmpty
                ? strings.tourRecommendedHint
                : def.steps.first.title,
                style: theme.textTheme.titleMedium),
            const SizedBox(height: 4),
            Text(
              def.steps.isEmpty
                  ? strings.tourNotOnBuild
                  : def.goal,
              style: theme.textTheme.bodySmall,
            ),
            const SizedBox(height: 12),
            Align(
              alignment: AlignmentDirectional.centerEnd,
              child: _ActionButton(
                strings: strings,
                progress: p,
                busy: busy,
                onStart: onStart,
                onResume: onResume,
                definition: def,
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _TourTile extends StatelessWidget {
  const _TourTile({
    required this.strings,
    required this.definition,
    required this.progress,
    required this.busy,
    required this.onStart,
    required this.onResume,
    required this.onSkip,
  });

  final AppStrings strings;
  final TourDefinition definition;
  final TourProgress? progress;
  final bool busy;
  final void Function(TourDefinition) onStart;
  final void Function(TourDefinition) onResume;
  final void Function(TourDefinition, TourProgress) onSkip;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final p = progress;
    final implementable = definition.steps.isNotEmpty;

    return Card(
      margin: const EdgeInsets.only(bottom: 8),
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Expanded(
                  child: Text(
                    implementable
                        ? definition.steps.first.title
                        : definition.id,
                    style: theme.textTheme.titleMedium,
                  ),
                ),
                if (p != null) _StatusChip(strings: strings, progress: p),
              ],
            ),
            const SizedBox(height: 4),
            Text(
              '${strings.tourGoalLabel}: ${definition.goal}',
              style: theme.textTheme.bodySmall,
            ),
            if (!implementable) ...[
              const SizedBox(height: 4),
              Text(
                strings.tourNotOnBuild,
                style: theme.textTheme.bodySmall?.copyWith(
                  color: theme.colorScheme.onSurfaceVariant,
                ),
              ),
            ],
            if (implementable) ...[
              const SizedBox(height: 12),
              Row(
                mainAxisAlignment: MainAxisAlignment.end,
                children: [
                  IconButton(
                    tooltip: strings.tourSkipTourAction,
                    onPressed:
                        busy || p == null || p.isCompleted || !implementable
                            ? null
                            : () => onSkip(definition, p),
                    icon: const Icon(Icons.close),
                  ),
                  _ActionButton(
                    strings: strings,
                    progress: p,
                    busy: busy,
                    onStart: onStart,
                    onResume: onResume,
                    definition: definition,
                  ),
                ],
              ),
            ],
          ],
        ),
      ),
    );
  }
}

class _ActionButton extends StatelessWidget {
  const _ActionButton({
    required this.strings,
    required this.progress,
    required this.busy,
    required this.onStart,
    required this.onResume,
    required this.definition,
  });

  final AppStrings strings;
  final TourProgress? progress;
  final bool busy;
  final void Function(TourDefinition) onStart;
  final void Function(TourDefinition) onResume;
  final TourDefinition definition;

  @override
  Widget build(BuildContext context) {
    final p = progress;
    if (definition.steps.isEmpty) {
      return FilledButton.tonal(
        onPressed: null,
        child: Text(strings.tourNotOnBuild),
      );
    }
    if (p?.isCompleted == true) {
      return FilledButton.tonalIcon(
        onPressed: null,
        icon: const Icon(Icons.check),
        label: Text(strings.tourCompletedLabel),
      );
    }
    if (p == null || (!p.isRunning && !p.isPaused)) {
      return FilledButton(
        onPressed: busy ? null : () => onStart(definition),
        child: Text(strings.tourStartAction),
      );
    }
    return FilledButton(
      onPressed: busy ? null : () => onResume(definition),
      child: Text(p.isPaused
          ? strings.tourResumeAction
          : strings.tourContinueAction),
    );
  }
}

class _StatusChip extends StatelessWidget {
  const _StatusChip({required this.strings, required this.progress});

  final AppStrings strings;
  final TourProgress progress;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    final (label, color) = switch (progress.status) {
      'completed' => (strings.tourCompletedLabel, Colors.green.shade700),
      'running' => (strings.tourRunningLabel, scheme.primary),
      'paused' => (strings.tourPausedLabel, scheme.tertiary),
      'skipped' => (strings.tourSkippedLabel, scheme.outline),
      _ => (strings.tourAvailableLabel, scheme.outline),
    };
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 3),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.12),
        borderRadius: BorderRadius.circular(999),
      ),
      child: Text(
        label,
        style: Theme.of(context).textTheme.labelSmall?.copyWith(color: color),
      ),
    );
  }
}
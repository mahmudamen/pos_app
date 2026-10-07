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
  OnboardingSummary? _summary;
  OnboardingReward? _reward;
  String? _error;
  bool _loading = true;
  bool _busy = false;
  bool _claiming = false;

  @override
  void initState() {
    super.initState();
    _load();
    _loadSummary();
  }

  Future<void> _load() async {
    try {
      final state = await widget.apiClient.onboardingState(widget.session);
      if (!mounted) return;
      setState(() {
        _state = state;
        _reward = state.reward;
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

  Future<void> _loadSummary() async {
    try {
      final summary = await widget.apiClient.onboardingSummary(widget.session);
      if (!mounted) return;
      setState(() => _summary = summary);
    } catch (_) {
      // Analytics are manager-only: a guest / permission error simply hides
      // the card instead of failing the whole center.
    }
  }

  Future<void> _claimReward() async {
    if (_claiming) return;
    setState(() => _claiming = true);
    try {
      final reward = await widget.apiClient.claimReward(widget.session);
      if (!mounted) return;
      setState(() {
        _claiming = false;
        _reward = reward;
      });
      _snack(reward.replayed
          ? AppStrings.of(context).rewardReplayedHint
          : AppStrings.of(context).rewardGranted);
      await _loadSummary();
    } catch (e) {
      if (!mounted) return;
      setState(() => _claiming = false);
      _snack(e.toString());
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
                  onRefresh: () async {
                    await _load();
                    await _loadSummary();
                  },
                  child: ListView(
                    padding: const EdgeInsets.fromLTRB(16, 8, 16, 24),
                    children: [
                      Text(s.onboardingCenterSubtitle,
                          style: Theme.of(context).textTheme.bodyMedium),
                      const SizedBox(height: 12),
                      _RewardCard(
                        strings: s,
                        reward: _reward,
                        claiming: _claiming,
                        onClaim: _claimReward,
                      ),
                      if (_summary != null) ...[
                        const SizedBox(height: 12),
                        _AnalyticsCard(strings: s, summary: _summary!),
                      ],
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

/// EPIC 11 activation reward: server-authoritative milestone with an idempotent
/// claim. The card explains why it is (not yet) available and never lets the
/// client decide eligibility.
class _RewardCard extends StatelessWidget {
  const _RewardCard({
    required this.strings,
    required this.reward,
    required this.claiming,
    required this.onClaim,
  });

  final AppStrings strings;
  final OnboardingReward? reward;
  final bool claiming;
  final VoidCallback onClaim;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final r = reward;
    if (r == null) return const SizedBox.shrink();
    final granted = r.granted;

    return Card(
      color: granted
          ? theme.colorScheme.primaryContainer
          : theme.colorScheme.surfaceContainerHighest,
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Icon(
                  granted ? Icons.emoji_events : Icons.card_giftcard,
                  color: theme.colorScheme.primary,
                ),
                const SizedBox(width: 8),
                Text(
                  strings.rewardTitle,
                  style: theme.textTheme.titleMedium
                      ?.copyWith(fontWeight: FontWeight.w700),
                ),
              ],
            ),
            const SizedBox(height: 8),
            Text(
              granted
                  ? strings.rewardClaimedHint
                  : r.eligible
                      ? strings.rewardUnlocked
                      : strings.rewardBody,
              style: theme.textTheme.bodyMedium,
            ),
            const SizedBox(height: 12),
            Align(
              alignment: AlignmentDirectional.centerEnd,
              child: granted
                  ? Chip(
                      avatar: Icon(Icons.check_circle,
                          size: 18, color: Colors.green.shade700),
                      label: Text(strings.rewardGranted),
                    )
                  : FilledButton.icon(
                      onPressed: (r.eligible && !claiming) ? onClaim : null,
                      icon: claiming
                          ? const SizedBox(
                              width: 16,
                              height: 16,
                              child:
                                  CircularProgressIndicator(strokeWidth: 2),
                            )
                          : const Icon(Icons.redeem),
                      label: Text(strings.claimRewardAction),
                    ),
            ),
          ],
        ),
      ),
    );
  }
}

/// EPIC 13 analytics: tenant-wide onboarding progress (manager-only; the card
/// is only built when the summary could be loaded).
class _AnalyticsCard extends StatelessWidget {
  const _AnalyticsCard({required this.strings, required this.summary});

  final AppStrings strings;
  final OnboardingSummary summary;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final seconds = summary.timeToFirstSaleSeconds;
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Icon(Icons.insights_outlined,
                    color: theme.colorScheme.primary),
                const SizedBox(width: 8),
                Expanded(
                  child: Text(
                    strings.analyticsTitle,
                    style: theme.textTheme.titleMedium
                        ?.copyWith(fontWeight: FontWeight.w700),
                  ),
                ),
                if (summary.activated)
                  Chip(
                    visualDensity: VisualDensity.compact,
                    avatar: Icon(Icons.check_circle,
                        size: 18, color: Colors.green.shade700),
                    label: Text(strings.statActive),
                  ),
              ],
            ),
            const SizedBox(height: 4),
            Text(
              strings.analyticsSubtitle,
              style: theme.textTheme.bodySmall
                  ?.copyWith(color: theme.colorScheme.onSurfaceVariant),
            ),
            const SizedBox(height: 12),
            Wrap(
              spacing: 12,
              runSpacing: 12,
              children: [
                _StatTile(
                  label: strings.statToursStarted,
                  value: '${summary.totals.started}',
                ),
                _StatTile(
                  label: strings.statToursCompleted,
                  value: '${summary.totals.completed}',
                ),
                _StatTile(
                  label: strings.statRewardsGranted,
                  value: '${summary.rewardGrants}',
                ),
                _StatTile(
                  label: strings.statCompletionRate,
                  value: '${(summary.completionRate * 100).round()}%',
                ),
                if (seconds != null)
                  _StatTile(
                    label: strings.statTimeToFirstSale,
                    value: _duration(seconds),
                  ),
              ],
            ),
          ],
        ),
      ),
    );
  }

  String _duration(int seconds) {
    if (seconds < 60) return '${seconds}s';
    final minutes = seconds ~/ 60;
    if (minutes < 60) return '${minutes}m';
    final hours = minutes ~/ 60;
    if (hours < 24) return '${hours}h';
    return '${hours ~/ 24}d';
  }
}

class _StatTile extends StatelessWidget {
  const _StatTile({required this.label, required this.value});

  final String label;
  final String value;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Container(
      width: 150,
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
      decoration: BoxDecoration(
        color: theme.colorScheme.surfaceContainerHighest.withValues(alpha: 0.5),
        borderRadius: BorderRadius.circular(12),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            value,
            style: theme.textTheme.titleLarge
                ?.copyWith(fontWeight: FontWeight.w700),
          ),
          const SizedBox(height: 2),
          Text(
            label,
            style: theme.textTheme.bodySmall
                ?.copyWith(color: theme.colorScheme.onSurfaceVariant),
          ),
        ],
      ),
    );
  }
}
import 'dart:math' as math;

import 'package:flutter/material.dart';

import '../../l10n/strings.dart';
import 'tour_controller.dart';
import 'tour_definitions.dart';
import 'tour_target_registry.dart';

/// Mounts the tour UI above the whole app. Wrap the MaterialApp's `builder`
/// child: when a tour is running a spotlight + guidance bubble overlay the
/// current screen; otherwise the app renders unchanged.
class TourOverlayHost extends StatelessWidget {
  const TourOverlayHost({
    super.key,
    required this.controller,
    required this.child,
  });

  final TourController controller;
  final Widget child;

  @override
  Widget build(BuildContext context) {
    return AnimatedBuilder(
      animation: controller,
      child: child,
      builder: (context, app) {
        final show = controller.isRunning && controller.definition != null;
        return Stack(
          children: [
            app!,
            if (show)
              Positioned.fill(
                child: TourOverlay(controller: controller),
              ),
          ],
        );
      },
    );
  }
}

/// Resolves a target's on-screen rect from its registered GlobalKey. Uses
/// root-global coordinates (via the target box itself) so it also works while
/// the overlay is being mounted for the very first time.
Rect? _targetRect(TourTargetRegistry registry, String id) {
  final key = registry.keyFor(id);
  if (key == null) return null;
  final box = key.currentContext?.findRenderObject();
  if (box is! RenderBox || !box.attached || !box.hasSize) return null;
  // The target was laid out in a previous frame; its transform is up to date,
  // while the overlay's own render object may not be attached yet mid-build.
  final global = box.localToGlobal(Offset.zero);
  return Rect.fromLTWH(global.dx, global.dy, box.size.width, box.size.height);
}

class TourOverlay extends StatefulWidget {
  const TourOverlay({super.key, required this.controller});

  final TourController controller;

  @override
  State<TourOverlay> createState() => _TourOverlayState();
}

class _TourOverlayState extends State<TourOverlay>
    with SingleTickerProviderStateMixin {
  late final AnimationController _fade = AnimationController(
    vsync: this,
    duration: const Duration(milliseconds: 250),
  );

  /// A target may be registered but its box not laid out yet on the very frame
  /// it appears (e.g. a freshly mounted TourTarget); resolve it after layout.
  bool _resolvingTarget = false;

  void _resolveTargetAfterLayout() {
    if (_resolvingTarget) return;
    _resolvingTarget = true;
    WidgetsBinding.instance.addPostFrameCallback((_) {
      _resolvingTarget = false;
      if (mounted) setState(() {});
    });
  }

  @override
  void initState() {
    super.initState();
    _fade.forward();
  }

  @override
  void dispose() {
    _fade.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final controller = widget.controller;
    final definition = controller.definition;
    final step = controller.currentStep;
    final size = MediaQuery.sizeOf(context);

    final targetMissing = step == null ||
        (step.optional == false && !controller.targetExists(step.target));
    final hole =
        step == null ? null : _targetRect(TourTargetRegistry.instance, step.target);
    if (step != null &&
        !targetMissing &&
        hole == null &&
        controller.targetExists(step.target)) {
      // Registered but not yet laid out; resolve the rect on the next frame.
      _resolveTargetAfterLayout();
    }

    return Stack(
      children: [
        if (size.width > 0 && size.height > 0)
          Positioned.fill(
            child: IgnorePointer(
              child: FadeTransition(
                opacity: _fade,
                child: CustomPaint(
                  size: size,
                  painter: _SpotlightPainter(
                    hole: hole,
                    screen: Offset.zero & size,
                    color: Colors.black.withValues(alpha: 0.62),
                    ring: Theme.of(context).colorScheme.primary,
                  ),
                ),
              ),
            ),
          ),
        if (targetMissing)
          _buildFallbackCard(context, controller, definition, step)
        else if (hole != null)
          _PositionedBubble(
            controller: controller,
            step: step,
            hole: hole,
          ),
      ],
    );
  }

  Widget _buildFallbackCard(
    BuildContext context,
    TourController controller,
    TourDefinition? definition,
    TourStepDef? step,
  ) {
    final strings = AppStrings.of(context);
    final theme = Theme.of(context);
    return Center(
      child: SafeArea(
        child: Card(
          margin: const EdgeInsets.all(24),
          child: Padding(
            padding: const EdgeInsets.all(20),
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 420),
              child: Column(
                mainAxisSize: MainAxisSize.min,
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Row(
                    children: [
                      Icon(Icons.tour_outlined, color: theme.colorScheme.primary),
                      const SizedBox(width: 8),
                      Expanded(
                        child: Text(
                          step?.title ?? definition?.id ?? '',
                          style: theme.textTheme.titleMedium,
                        ),
                      ),
                    ],
                  ),
                  const SizedBox(height: 12),
                  Text(
                    step == null
                        ? strings.tourTargetMissing
                        : strings.tourTargetMissingStep(step.id),
                    style: theme.textTheme.bodyMedium,
                  ),
                  const SizedBox(height: 16),
                  Wrap(
                    alignment: WrapAlignment.end,
                    spacing: 8,
                    runSpacing: 4,
                    children: [
                      TextButton(
                        onPressed: controller.exit,
                        child: Text(strings.tourExit),
                      ),
                      if (controller.definition != null &&
                          !controller.onLastStep)
                        TextButton(
                          onPressed: controller.skipStep,
                          child: Text(strings.tourSkipStep),
                        ),
                      FilledButton(
                        onPressed: controller.retryResolution,
                        child: Text(strings.tourTryAgain),
                      ),
                    ],
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }
}

/// Positions the guidance bubble next to the spotlight hole, flipping above
/// the target when there is not enough room below (and always constraining its
/// height so it can never overflow the screen). Must remain a direct child of
/// the overlay's Stack, so the geometry is derived from the screen size rather
/// than a LayoutBuilder.
class _PositionedBubble extends StatelessWidget {
  const _PositionedBubble({
    required this.controller,
    required this.step,
    required this.hole,
  });

  final TourController controller;
  final TourStepDef step;
  final Rect hole;

  @override
  Widget build(BuildContext context) {
    final size = MediaQuery.sizeOf(context);
    const margin = 12.0;
    const gap = 14.0;
    const maxWidth = 420.0;
    final width = math.min(size.width - margin * 2, maxWidth);
    final left =
        (hole.center.dx - width / 2)
            .clamp(margin, math.max(margin, size.width - width - margin))
            .toDouble();

    final spaceBelow = size.height - hole.bottom - gap;
    final below = spaceBelow >= 170;
    final maxHeight = below
        ? math.max(60.0, size.height - hole.bottom - gap - margin)
        : math.max(60.0, hole.top - gap - margin);

    return Positioned(
      left: left,
      width: width,
      top: below ? hole.bottom + gap : null,
      bottom: below ? null : size.height - hole.top + gap,
      child: ConstrainedBox(
        constraints: BoxConstraints(maxHeight: maxHeight),
        child: SingleChildScrollView(
          child: _BubbleCard(controller: controller, step: step),
        ),
      ),
    );
  }
}

class _BubbleCard extends StatelessWidget {
  const _BubbleCard({required this.controller, required this.step});

  final TourController controller;
  final TourStepDef step;

  @override
  Widget build(BuildContext context) {
    final strings = AppStrings.of(context);
    final theme = Theme.of(context);
    final definition = controller.definition;
    final index = controller.stepIndex;
    final count = definition?.stepCount ?? 1;

    Widget actions;
    if (controller.goalNotMet) {
      actions = Wrap(
        alignment: WrapAlignment.end,
        spacing: 8,
        runSpacing: 4,
        children: [
          TextButton(
            onPressed: controller.exit,
            child: Text(strings.tourKeepGoing),
          ),
          FilledButton(
            onPressed: controller.complete,
            child: Text(strings.tourCheckAgain),
          ),
        ],
      );
    } else {
      actions = Wrap(
        alignment: WrapAlignment.end,
        spacing: 8,
        runSpacing: 4,
        children: [
          IconButton(
            tooltip: strings.tourPauseTooltip,
            onPressed: controller.pause,
            icon: const Icon(Icons.pause_circle_outline),
          ),
          if (!controller.onLastStep)
            TextButton(
              onPressed: controller.skipStep,
              child: Text(strings.tourSkipStep),
            ),
          FilledButton(
            onPressed: () {
              if (controller.onLastStep) {
                controller.complete();
              } else {
                controller.next();
              }
            },
            child: Text(controller.onLastStep ? strings.tourFinish : strings.tourNext),
          ),
        ],
      );
    }

    return Card(
      elevation: 6,
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(14)),
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Expanded(
                  child: Text(
                    step.title,
                    style: theme.textTheme.titleMedium?.copyWith(
                      fontWeight: FontWeight.w700,
                    ),
                  ),
                ),
                Container(
                  padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
                  decoration: BoxDecoration(
                    color: theme.colorScheme.surfaceContainerHighest,
                    borderRadius: BorderRadius.circular(999),
                  ),
                  child: Text(
                    '${index + 1}/$count',
                    style: theme.textTheme.labelSmall,
                  ),
                ),
              ],
            ),
            const SizedBox(height: 8),
            Text(step.body, style: theme.textTheme.bodyMedium),
            if (controller.goalNotMet) ...[
              const SizedBox(height: 8),
              Text(
                strings.tourGoalNotMet,
                style: theme.textTheme.bodySmall?.copyWith(
                  color: theme.colorScheme.error,
                ),
              ),
            ],
            const SizedBox(height: 12),
            actions,
          ],
        ),
      ),
    );
  }
}

class _SpotlightPainter extends CustomPainter {
  _SpotlightPainter({
    required this.hole,
    required this.screen,
    required this.color,
    required this.ring,
  });

  final Rect? hole;
  final Rect screen;
  final Color color;
  final Color ring;

  @override
  void paint(Canvas canvas, Size size) {
    final overlayPaint = Paint()..color = color;

    if (hole == null || hole!.isEmpty) {
      canvas.drawRect(screen, overlayPaint);
      return;
    }

    final clamped = hole!.intersect(screen);
    if (clamped.isEmpty) {
      canvas.drawRect(screen, overlayPaint);
      return;
    }

    final outer = Path()..addRect(screen);
    final guide = RRect.fromRectAndRadius(
      clamped.inflate(6),
      const Radius.circular(14),
    );
    final holePath = Path()..addRRect(guide);
    final mask = Path.combine(PathOperation.difference, outer, holePath);
    canvas.drawPath(mask, overlayPaint);

    final ringPaint = Paint()
      ..style = PaintingStyle.stroke
      ..strokeWidth = 2
      ..color = ring;
    canvas.drawRRect(guide, ringPaint);
  }

  @override
  bool shouldRepaint(_SpotlightPainter old) =>
      old.hole != hole || old.color != color || old.ring != ring;
}
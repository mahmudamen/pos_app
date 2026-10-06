import 'package:flutter/widgets.dart';

import 'tour_target_registry.dart';

/// Registers a [GlobalKey] for one tour step target. Wrap the POS element the
/// step points at and the spotlight will find its exact rect. The key lives
/// for the widget's lifetime and is removed on dispose, so a target that left
/// the tree is reported as missing instead of pointing at a dead context.
class TourTarget extends StatefulWidget {
  const TourTarget({super.key, required this.id, required this.child});

  final String id;
  final Widget child;

  @override
  State<TourTarget> createState() => _TourTargetState();
}

class _TourTargetState extends State<TourTarget> {
  final GlobalKey _key = GlobalKey();

  @override
  void initState() {
    super.initState();
    TourTargetRegistry.instance.register(widget.id, _key);
  }

  @override
  void dispose() {
    TourTargetRegistry.instance.unregister(widget.id);
    super.dispose();
  }

  @override
  Widget build(BuildContext context) =>
      KeyedSubtree(key: _key, child: widget.child);
}
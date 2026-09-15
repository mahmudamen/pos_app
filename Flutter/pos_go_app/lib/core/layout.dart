import 'package:flutter/widgets.dart';

/// Cap on content width for list/form surfaces so large tablets don't stretch
/// rows edge-to-edge; phones below this width render unchanged.
const double kMaxContentWidth = 1040;

/// Constrains [child] to at most [maxWidth] and top-centers it, keeping
/// multi-column rows readable and centered on big screens.
class MaxWidthBox extends StatelessWidget {
  const MaxWidthBox({
    super.key,
    required this.child,
    this.maxWidth = kMaxContentWidth,
  });

  final Widget child;
  final double maxWidth;

  @override
  Widget build(BuildContext context) {
    return Align(
      alignment: Alignment.topCenter,
      child: ConstrainedBox(
        constraints: BoxConstraints(maxWidth: maxWidth),
        child: child,
      ),
    );
  }
}
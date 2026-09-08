import 'package:flutter/material.dart';

extension ContextResponsive on BuildContext {
  Size get screenSize => MediaQuery.sizeOf(this);
  double get w => screenSize.width;
  double get h => screenSize.height;
  bool get isSmall => w < 600;
  bool get isTablet => w >= 600 && w < 1024;
  bool get isDesktop => w >= 1024;
  double get scale => (w / 390).clamp(0.85, 1.25);
}

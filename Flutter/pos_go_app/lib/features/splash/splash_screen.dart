import 'dart:async';
import 'dart:math' as math;

import 'package:flutter/material.dart';

import '../../l10n/strings.dart';

class SplashScreen extends StatefulWidget {
  const SplashScreen({super.key, this.onFinished});

  final VoidCallback? onFinished;

  @override
  State<SplashScreen> createState() => _SplashScreenState();
}

class _SplashScreenState extends State<SplashScreen>
    with TickerProviderStateMixin {
  static const _ink = Color(0xff0a1628);
  static const _mid = Color(0xff003355);
  static const _blue = Color(0xff00adee);

  late final AnimationController _enter = AnimationController(
    vsync: this,
    duration: const Duration(milliseconds: 950),
  )..forward();

  late final AnimationController _glow = AnimationController(
    vsync: this,
    duration: const Duration(milliseconds: 1600),
  )..repeat(reverse: true);

  Timer? _timer;

  @override
  void initState() {
    super.initState();
    _timer = Timer(
      const Duration(milliseconds: 1700),
      () => widget.onFinished?.call(),
    );
  }

  late final Animation<double> _fadeIn = CurvedAnimation(
    parent: _enter,
    curve: Curves.easeOut,
  );

  late final Animation<double> _rise = Tween(begin: 26.0, end: 0.0).animate(
    CurvedAnimation(parent: _enter, curve: Curves.easeOutCubic),
  );

  late final Animation<double> _scale = Tween(begin: 0.72, end: 1.0).animate(
    CurvedAnimation(parent: _enter, curve: Curves.elasticOut),
  );

  late final Animation<Offset> _slide = Tween<Offset>(
    begin: const Offset(0, 0.10),
    end: Offset.zero,
  ).animate(_rise);

  @override
  void dispose() {
    _timer?.cancel();
    _enter.dispose();
    _glow.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final s = AppStrings.of(context);
    return Scaffold(
      backgroundColor: _ink,
      body: Stack(
        fit: StackFit.expand,
        children: [
          const DecoratedBox(
            decoration: BoxDecoration(
              gradient: LinearGradient(
                begin: Alignment.topCenter,
                end: Alignment.bottomCenter,
                colors: [_ink, _mid, _ink],
              ),
            ),
          ),
          const Positioned.fill(child: CustomPaint(painter: _MotifPainter())),
          Positioned(
            top: -140,
            left: -140,
            child: Container(
              width: 360,
              height: 360,
              decoration: BoxDecoration(
                shape: BoxShape.circle,
                gradient: RadialGradient(
                  colors: [
                    _blue.withValues(alpha: 0.18),
                    _blue.withValues(alpha: 0.0),
                  ],
                ),
              ),
            ),
          ),
          Positioned(
            bottom: -120,
            right: -120,
            child: Container(
              width: 340,
              height: 340,
              decoration: BoxDecoration(
                shape: BoxShape.circle,
                gradient: RadialGradient(
                  colors: [
                    _blue.withValues(alpha: 0.22),
                    _blue.withValues(alpha: 0.0),
                  ],
                ),
              ),
            ),
          ),
          SafeArea(
            child: Column(
              children: [
                const Spacer(flex: 3),
                AnimatedBuilder(
                  animation: _glow,
                  builder: (context, child) {
                    return Transform.scale(
                      scale: 1.0 + _glow.value * 0.10,
                      child: Opacity(
                        opacity: 0.55 + _glow.value * 0.30,
                        child: child,
                      ),
                    );
                  },
                  child: Container(
                    width: 130,
                    height: 130,
                    decoration: BoxDecoration(
                      shape: BoxShape.circle,
                      boxShadow: [
                        BoxShadow(
                          color: _blue.withValues(alpha: 0.55),
                          blurRadius: 42,
                          spreadRadius: 4,
                        ),
                      ],
                    ),
                  ),
                ),
                const SizedBox(height: 42),
                _XamltechLogo(scale: _scale, blue: _blue),
                const SizedBox(height: 34),
                FadeTransition(
                  opacity: _fadeIn,
                  child: SlideTransition(
                    position: _slide,
                    child: Column(
                      children: [
                        Text(
                          s.splashTitle,
                          style: const TextStyle(
                            color: Colors.white,
                            fontSize: 30,
                            fontWeight: FontWeight.w800,
                            letterSpacing: 0.5,
                          ),
                          textAlign: TextAlign.center,
                        ),
                        const SizedBox(height: 12),
                        Padding(
                          padding: const EdgeInsets.symmetric(horizontal: 40),
                          child: Text(
                            s.splashTagline,
                            style: TextStyle(
                              color: Colors.white.withValues(alpha: 0.72),
                              fontSize: 15,
                              height: 1.6,
                            ),
                            textAlign: TextAlign.center,
                          ),
                        ),
                      ],
                    ),
                  ),
                ),
                const Spacer(),
                _LoadingDots(blue: _blue, glow: _glow),
                const SizedBox(height: 18),
                FadeTransition(
                  opacity: _fadeIn,
                  child: Text(
                    s.poweredByXamltech,
                    style: TextStyle(
                      color: Colors.white.withValues(alpha: 0.62),
                      fontSize: 13,
                      letterSpacing: 1.2,
                    ),
                    textAlign: TextAlign.center,
                  ),
                ),
                const SizedBox(height: 6),
                Directionality(
                  textDirection: TextDirection.ltr,
                  child: Text(
                    'xamltech.com',
                    style: TextStyle(
                      color: _blue.withValues(alpha: 0.85),
                      fontSize: 12,
                      letterSpacing: 1.6,
                      fontWeight: FontWeight.w600,
                    ),
                  ),
                ),
                const Spacer(flex: 2),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

class _XamltechLogo extends StatelessWidget {
  const _XamltechLogo({
    required this.scale,
    required this.blue,
  });

  final Animation<double> scale;
  final Color blue;

  @override
  Widget build(BuildContext context) {
    return ScaleTransition(
      scale: scale,
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          ClipRRect(
            borderRadius: BorderRadius.circular(20),
            child: Image.asset(
              'assets/xamltech_logo.png',
              width: 100,
              height: 100,
              fit: BoxFit.contain,
            ),
          ),
          const SizedBox(height: 14),
          Directionality(
            textDirection: TextDirection.ltr,
            child: Row(
              mainAxisSize: MainAxisSize.min,
              crossAxisAlignment: CrossAxisAlignment.baseline,
              textBaseline: TextBaseline.alphabetic,
              children: [
                const Text(
                  'XAML',
                  style: TextStyle(
                    color: Colors.white,
                    fontSize: 26,
                    fontWeight: FontWeight.w800,
                    letterSpacing: 3.5,
                    height: 1,
                  ),
                ),
                const SizedBox(width: 4),
                Text(
                  'tech',
                  style: TextStyle(
                    color: blue,
                    fontSize: 26,
                    fontWeight: FontWeight.w300,
                    letterSpacing: 1.5,
                    height: 1,
                  ),
                ),
              ],
            ),
          ),
          const SizedBox(height: 6),
          Text(
            'xamltech.com',
            style: TextStyle(
              color: Colors.white.withValues(alpha: 0.6),
              fontSize: 12,
              letterSpacing: 3.0,
              fontWeight: FontWeight.w500,
            ),
          ),
        ],
      ),
    );
  }
}

class _LoadingDots extends StatelessWidget {
  const _LoadingDots({required this.blue, required this.glow});

  final Color blue;
  final Animation<double> glow;

  @override
  Widget build(BuildContext context) {
    return AnimatedBuilder(
      animation: glow,
      builder: (context, child) {
        return Row(
          mainAxisSize: MainAxisSize.min,
          children: List.generate(3, (i) {
            final t = (glow.value + i * 0.28) % 1.0;
            final color = Color.lerp(
              blue.withValues(alpha: 0.25),
              Colors.white,
              t,
            )!;
            return Container(
              width: 8,
              height: 8,
              margin: const EdgeInsets.symmetric(horizontal: 5),
              decoration: BoxDecoration(
                shape: BoxShape.circle,
                color: color,
              ),
            );
          }),
        );
      },
    );
  }
}

class _MotifPainter extends CustomPainter {
  const _MotifPainter();

  @override
  void paint(Canvas canvas, Size size) {
    final arc = Paint()
      ..style = PaintingStyle.stroke
      ..strokeWidth = 2
      ..color = Colors.white.withValues(alpha: 0.04);
    final center = Offset(size.width * 1.02, size.height * 0.16);
    for (var i = 0; i < 5; i++) {
      canvas.drawArc(
        Rect.fromCircle(center: center, radius: 56.0 + i * 44),
        0.0,
        math.pi * 0.62,
        false,
        arc,
      );
    }
    final col = Paint()
      ..strokeWidth = 1
      ..color = Colors.white.withValues(alpha: 0.03);
    for (var x = 0.0; x < size.width * 0.22; x += 38) {
      canvas.drawLine(Offset(x, 0), Offset(x, size.height), col);
    }
  }

  @override
  bool shouldRepaint(_MotifPainter oldDelegate) => false;
}

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
  static const _ink = Color(0xff032931);
  static const _mid = Color(0xff0a4a4e);
  static const _teal = Color(0xff14b8a6);
  static const _gold = Color(0xffe6c06c);

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
                    _gold.withValues(alpha: 0.22),
                    _gold.withValues(alpha: 0.0),
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
                    _teal.withValues(alpha: 0.26),
                    _teal.withValues(alpha: 0.0),
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
                          color: _teal.withValues(alpha: 0.55),
                          blurRadius: 42,
                          spreadRadius: 4,
                        ),
                      ],
                    ),
                  ),
                ),
                const SizedBox(height: 42),
                _XamltechMark(scale: _scale, gold: _gold, teal: _teal),
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
                _LoadingDots(teal: _teal, gold: _gold, glow: _glow),
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
                      color: _gold.withValues(alpha: 0.85),
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

class _XamltechMark extends StatelessWidget {
  const _XamltechMark({
    required this.scale,
    required this.gold,
    required this.teal,
  });

  final Animation<double> scale;
  final Color gold;
  final Color teal;

  @override
  Widget build(BuildContext context) {
    return Directionality(
      textDirection: TextDirection.ltr,
      child: ScaleTransition(
        scale: scale,
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Container(
              width: 84,
              height: 84,
              decoration: BoxDecoration(
                borderRadius: BorderRadius.circular(22),
                gradient: const LinearGradient(
                  begin: Alignment.topLeft,
                  end: Alignment.bottomRight,
                  colors: [Color(0xff1cbfa9), Color(0xff0b6e6b)],
                ),
                boxShadow: [
                  BoxShadow(
                    color: teal.withValues(alpha: 0.45),
                    blurRadius: 26,
                    offset: const Offset(0, 10),
                  ),
                ],
              ),
              child: const CustomPaint(painter: _XMarkPainter()),
            ),
            const SizedBox(height: 14),
            Row(
              mainAxisSize: MainAxisSize.min,
              crossAxisAlignment: CrossAxisAlignment.baseline,
              textBaseline: TextBaseline.alphabetic,
              children: [
                Text(
                  'XAML',
                  style: TextStyle(
                    color: Colors.white,
                    fontSize: 26,
                    fontWeight: FontWeight.w800,
                    letterSpacing: 3.5,
                    height: 1,
                    shadows: [
                      Shadow(
                        color: gold.withValues(alpha: 0.35),
                        blurRadius: 16,
                      ),
                    ],
                  ),
                ),
                const SizedBox(width: 4),
                Text(
                  'tech',
                  style: TextStyle(
                    color: gold,
                    fontSize: 26,
                    fontWeight: FontWeight.w300,
                    letterSpacing: 1.5,
                    height: 1,
                  ),
                ),
              ],
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
      ),
    );
  }
}

class _LoadingDots extends StatelessWidget {
  const _LoadingDots({required this.teal, required this.gold, required this.glow});

  final Color teal;
  final Color gold;
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
              teal.withValues(alpha: 0.25),
              gold,
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

class _XMarkPainter extends CustomPainter {
  const _XMarkPainter();

  @override
  void paint(Canvas canvas, Size size) {
    final thick = Paint()
      ..color = Colors.white
      ..strokeWidth = size.width * 0.17
      ..strokeCap = StrokeCap.round
      ..style = PaintingStyle.stroke;
    final thin = Paint()
      ..color = Colors.white.withValues(alpha: 0.85)
      ..strokeWidth = size.width * 0.10
      ..strokeCap = StrokeCap.round;

    final m = size.width * 0.20;
    final a = Offset(m, m);
    final b = Offset(size.width - m, size.height - m);
    final c = Offset(size.width - m, m);
    final d = Offset(m, size.height - m);

    canvas.drawLine(a, b, thick);
    canvas.drawLine(c, d, thin);

    final dot = Paint()..color = const Color(0xffe6c06c);
    canvas.drawCircle(
      Offset(size.width * 0.72, size.height * 0.30),
      size.width * 0.07,
      dot,
    );
  }

  @override
  bool shouldRepaint(_XMarkPainter oldDelegate) => false;
}

class _MotifPainter extends CustomPainter {
  const _MotifPainter();

  @override
  void paint(Canvas canvas, Size size) {
    final arc = Paint()
      ..style = PaintingStyle.stroke
      ..strokeWidth = 2
      ..color = Colors.white.withValues(alpha: 0.06);
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
      ..color = Colors.white.withValues(alpha: 0.05);
    for (var x = 0.0; x < size.width * 0.22; x += 38) {
      canvas.drawLine(Offset(x, 0), Offset(x, size.height), col);
    }
  }

  @override
  bool shouldRepaint(_MotifPainter oldDelegate) => false;
}
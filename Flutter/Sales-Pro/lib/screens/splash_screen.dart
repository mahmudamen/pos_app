import 'dart:async';
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../state/app_provider.dart';
import '../core/localization/app_localizations.dart';

void _releaseLog(String message) {
  // In release mode, use print for important messages
  if (kReleaseMode) {
    print('[RELEASE] $message');
  } else {
    debugPrint(message);
  }
}

class SplashScreen extends StatefulWidget {
  const SplashScreen({super.key});

  @override
  State<SplashScreen> createState() => _SplashScreenState();
}

class _SplashScreenState extends State<SplashScreen> {
  String _statusMessage = '';
  bool _hasError = false;

  @override
  void initState() {
    super.initState();
    // Initialize message after first frame when context is available
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (mounted) {
        setState(() {
          _statusMessage = AppLocalizations.of(context).loading;
        });
      }
    });
    _checkLoginStatus();
  }

  Future<void> _checkLoginStatus() async {
    final app = context.read<AppProvider>();

    // Simple delay to avoid flickering
    await Future.delayed(const Duration(milliseconds: 800));

    // Add safety timer for auto transition
    final safetyTimer = Timer(const Duration(seconds: 15), () {
      if (mounted) {
        _releaseLog('⏰ Safety timeout - forcing navigation to login');
        Navigator.pushReplacementNamed(context, '/login');
      }
    });

    try {
      // Add timeout to ensure no long wait
      await Future.any([
        _initializeApp(app),
        Future.delayed(const Duration(seconds: 10)),
      ]);

      safetyTimer.cancel(); // Cancel timer if operation succeeded

      if (!mounted) return;

      final nextRoute = app.currentUser != null ? '/home' : '/login';
      _releaseLog('🚀 Navigating to: $nextRoute');
      Navigator.pushReplacementNamed(context, nextRoute);
    } catch (e) {
      safetyTimer.cancel(); // Cancel timer
      _releaseLog('❌ Error in splash screen: $e');
      if (mounted) {
        final loc = AppLocalizations.of(context);
        setState(() {
          _hasError = true;
          _statusMessage = loc.errorConnection;
        });
        // Wait a second then navigate to login
        await Future.delayed(const Duration(seconds: 2));
        if (mounted) {
          Navigator.pushReplacementNamed(context, '/login');
        }
      }
    }
  }

  Future<void> _initializeApp(AppProvider app) async {
    try {
      // In release mode, if operation takes too long, go to login directly
      if (kReleaseMode) {
        // Shorter timeout in release mode
        await _initWithTimeout(app, const Duration(seconds: 5));
      } else {
        await _initWithTimeout(app, const Duration(seconds: 8));
      }
    } catch (e) {
      _releaseLog('❌ Error during app initialization: $e');
      if (mounted) {
        setState(() {
          _hasError = true;
          _statusMessage = 'Error occurred, redirecting to login...';
        });
      }
      // Do not stop here, let it go to login screen
      await Future.delayed(const Duration(seconds: 1));
    }
  }

  Future<void> _initWithTimeout(AppProvider app, Duration timeout) async {
    // Ensure AppProvider is initialized first
    if (mounted) {
      final loc = AppLocalizations.of(context);
      setState(() => _statusMessage = loc.initializingApp);
    }
    _releaseLog('🔧 Initializing AppProvider...');

    await app.initialize().timeout(
      timeout,
      onTimeout: () {
        _releaseLog('⏰ AppProvider initialization timeout');
        throw TimeoutException('AppProvider timeout', timeout);
      },
    );
    _releaseLog('✅ AppProvider initialized');

    if (mounted) {
      final loc = AppLocalizations.of(context);
      setState(() => _statusMessage = loc.loading);
    }
    _releaseLog('🔄 Restoring session...');

    await app.restoreSession().timeout(
      timeout,
      onTimeout: () {
        _releaseLog('⏰ Session restore timeout - continuing to login');
      },
    );
    _releaseLog('✅ Session restored');
  }

  void _skipToLogin() {
    Navigator.pushReplacementNamed(context, '/login');
  }

  @override
  Widget build(BuildContext context) {
    final loc = AppLocalizations.of(context);
    return Scaffold(
      backgroundColor: Theme.of(context).colorScheme.background,
      body: SafeArea(
        child: Center(
          child: Column(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              // Add your logo here
              // Image.asset('assets/logo.png', width: 120),
              const SizedBox(height: 30),
              Text(
                loc.appFullName,
                style: const TextStyle(
                  fontSize: 26,
                  fontWeight: FontWeight.bold,
                ),
              ),
              const SizedBox(height: 20),
              if (!_hasError) ...[
                const CircularProgressIndicator(),
              ] else ...[
                const Icon(
                  Icons.warning_amber_rounded,
                  color: Colors.orange,
                  size: 48,
                ),
              ],
              const SizedBox(height: 16),
              Text(_statusMessage),
              const SizedBox(height: 20),
              if (_hasError) ...[
                ElevatedButton(onPressed: _skipToLogin, child: Text(loc.login)),
              ],
            ],
          ),
        ),
      ),
    );
  }
}

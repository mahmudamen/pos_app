import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';

import 'core/api_client.dart';
import 'core/focus_mode.dart';
import 'core/session_store.dart';
import 'core/storage/local_database.dart';
import 'core/telemetry.dart';
import 'features/auth/login_screen.dart';
import 'features/onboarding/onboarding_screen.dart';
import 'features/pos/pos_screen.dart';
import 'features/splash/splash_screen.dart';
import 'l10n/strings.dart';

class PosApp extends StatefulWidget {
  const PosApp({super.key});

  @override
  State<PosApp> createState() => _PosAppState();
}

class _PosAppState extends State<PosApp> {
  final _sessionStore = SessionStore();
  final _localDatabase = LocalDatabase();
  late final ApiClient _apiClient;
  Session? _session;
  bool _restoring = true;
  bool _splashDone = false;
  bool _showLogin = false;
  Locale _locale = const Locale('ar');
  bool _languageCustomized = false;

  @override
  void initState() {
    super.initState();
    _apiClient = ApiClient(
      onSessionRefreshed: (session) async {
        await _sessionStore.save(session);
        if (mounted) setState(() => _session = session);
      },
    );
    Telemetry.instance.configure(_apiClient);
    Telemetry.instance.install();
    _restoreSession();
  }

  Future<void> _restoreSession() async {
    final session = await _sessionStore.read();
    final language = await _sessionStore.readLanguage();
    final customized = await _sessionStore.hasCustomizedLanguage();
    Telemetry.instance.setSession(session);
    if (!mounted) return;
    setState(() {
      _session = session;
      _locale = Locale(language);
      _languageCustomized = customized;
      _restoring = false;
    });
  }

  Future<void> _setLanguage(String code) async {
    await _sessionStore.saveLanguage(code);
    if (mounted) {
      setState(() {
        _locale = Locale(code);
        _languageCustomized = true;
      });
    }
  }

  Future<void> _authenticated(Session session) async {
    await _sessionStore.save(session);
    if (!_languageCustomized) {
      // Tenant's default language applies only until the user explicitly picks
      // one (login toggle or settings).
      await _sessionStore.applyTenantLanguage(session.defaultLanguage);
      if (mounted) setState(() => _locale = Locale(session.defaultLanguage));
    }
    if (!mounted) return;
    setState(() => _session = session);
    Telemetry.instance.setSession(session);
    Telemetry.instance.reportScreen('pos');
  }

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'POS Go',
      debugShowCheckedModeBanner: false,
      locale: _locale,
      supportedLocales: AppStrings.supportedLocales,
      localizationsDelegates: const [
        AppStrings.delegate,
        GlobalMaterialLocalizations.delegate,
        GlobalWidgetsLocalizations.delegate,
        GlobalCupertinoLocalizations.delegate,
      ],
      theme: ThemeData(
        colorScheme: ColorScheme.fromSeed(seedColor: const Color(0xff00adee)),
        scaffoldBackgroundColor: const Color(0xfff4f7f6),
        useMaterial3: true,
      ),
      home: AnimatedSwitcher(
        duration: const Duration(milliseconds: 500),
        child: _restoring || !_splashDone
            ? SplashScreen(
                key: const ValueKey('splash'),
                onFinished: () {
                  if (mounted) setState(() => _splashDone = true);
                },
              )
            : _session == null && !_showLogin
                ? OnboardingScreen(
                    key: const ValueKey('onboarding'),
                    apiClient: _apiClient,
                    sessionStore: _sessionStore,
                    onLanguageChanged: _setLanguage,
                    onAuthenticated: _authenticated,
                    onOpenLogin: () {
                      if (mounted) {
                        setState(() {
                          _showLogin = true;
                          _splashDone = true;
                        });
                      }
                    },
                  )
                : _session == null
                ? LoginScreen(
                    key: const ValueKey('login'),
                    apiClient: _apiClient,
                    sessionStore: _sessionStore,
                    onLanguageChanged: _setLanguage,
                    onAuthenticated: _authenticated,
                  )
                : PosScreen(
                    key: const ValueKey('pos'),
                    session: _session!,
                  apiClient: _apiClient,
                  localDatabase: _localDatabase,
                  sessionStore: _sessionStore,
                  onLanguageChanged: _setLanguage,
                  onSignOut: () async {
                    await FocusMode.apply(false);
                    try {
                      await _apiClient.logout(_session!);
                    } catch (_) {
                      // Clear local credentials even when the server is offline.
                    }
                    try {
                      await _sessionStore.clear();
                    } catch (_) {
                      // Unlock the UI even if secure storage is transiently unavailable.
                    }
                    if (mounted) setState(() => _session = null);
                    Telemetry.instance.setSession(null);
                  },
                ),
              ),
    );
  }
}
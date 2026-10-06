import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_localizations/flutter_localizations.dart';

import 'core/api_client.dart';
import 'core/app_links.dart';
import 'core/feedback.dart';
import 'core/focus_mode.dart';
import 'core/fonts.dart';
import 'core/session_store.dart';
import 'core/storage/local_database.dart';
import 'core/telemetry.dart';
import 'core/push.dart';
import 'core/theme.dart';
import 'core/tour/tour_controller.dart';
import 'core/tour/tour_overlay.dart';
import 'core/tour/tour_target_registry.dart';
import 'features/auth/login_screen.dart';
import 'features/onboarding/onboarding_screen.dart';
import 'features/pos/pos_screen.dart';
import 'features/splash/splash_screen.dart';
import 'l10n/strings.dart';

class PosApp extends StatefulWidget {
  const PosApp({
    super.key,
    this.appLinkSource,
    this.navigatorKey,
  });

  /// Overridden by tests so no platform channel is involved.
  final AppLinkSource? appLinkSource;

  /// Overridden by tests; a link has to be able to reach the navigator.
  final GlobalKey<NavigatorState>? navigatorKey;

  @override
  State<PosApp> createState() => _PosAppState();
}

class _PosAppState extends State<PosApp> {
  final _sessionStore = SessionStore();
  final _localDatabase = LocalDatabase();
  late final ApiClient _apiClient;
  late final AppLinkService _appLinks;
  late final GlobalKey<NavigatorState> _navigatorKey;
  Session? _session;
  bool _restoring = true;
  bool _splashDone = false;
  bool _showLogin = false;
  Locale _locale = AppStrings.defaultLocale;
  bool _languageCustomized = false;
  FontSetting _fontSetting = const FontSetting();
  ThemeSetting _themeSetting = const ThemeSetting();
  TourController? _tour;

  @override
  void initState() {
    super.initState();
    _apiClient = ApiClient(
      onSessionRefreshed: (session) async {
        await _sessionStore.save(session);
        if (mounted) setState(() => _session = session);
      },
    );
    _navigatorKey = widget.navigatorKey ?? GlobalKey<NavigatorState>();
    _appLinks = widget.appLinkSource == null
        ? AppLinkService()
        : AppLinkService(source: widget.appLinkSource);
    Telemetry.instance.configure(_apiClient);
    Telemetry.instance.install();
    _initPush();
    _initSystemUi();
    _initAppLinks();
    _restoreSession();
  }

  @override
  void dispose() {
    _tour?.disposeController();
    _tour = null;
    _appLinks.dispose();
    super.dispose();
  }

  /// Wires up Android App Links for `https://api.xamltech.com/pos…`.
  ///
  /// A verified link always lands on the POS screen: the POS is the app's home
  /// surface, so the work here is popping whatever sub-screen the user had
  /// pushed (dashboard, customers, community…) back off, which is otherwise
  /// impossible from outside that widget's navigator. Sub-paths are reported to
  /// telemetry with their destination but are not routed to a specific screen
  /// yet — inventing navigation targets that do not exist would be worse than
  /// landing on the POS, and the telemetry is what tells us which targets to
  /// build.
  Future<void> _initAppLinks() async {
    final available = await _appLinks.start();
    if (!available) {
      Telemetry.instance.reportEvent('app_links_unavailable');
      return;
    }
    _appLinks.links.listen(_onAppLink);
  }

  void _onAppLink(PosAppLink link) {
    Telemetry.instance.reportEvent(
      'app_link_open',
      payload: {
        'destination': link.destination.wireName,
        'path': link.uri.path,
        'signed_in': _session != null,
      },
    );
    // Not signed in yet: the link is honoured by showing the POS after login,
    // which is what the app already does, so there is nothing to pop.
    if (_session == null) return;
    revealPosScreen(_navigatorKey);
  }

  /// Draws behind the status and navigation bars, explicitly and from launch.
  ///
  /// Android 15 turns this on by itself at targetSdk 35+ and ignores
  /// `windowOptOutEdgeToEdgeEnforcement` at 36, so opting out is not an option.
  /// Flutter's `PlatformPlugin` makes this exact call
  /// (`WindowCompat.setDecorFitsSystemWindows(window, false)`) — the same one
  /// Android's `enableEdgeToEdge()` makes, which is unavailable here because
  /// `FlutterActivity` extends `Activity`, not `ComponentActivity`.
  ///
  /// Screens must therefore honour the insets: `AppBar` handles the top one,
  /// and bottom-anchored controls need their own padding.
  Future<void> _initSystemUi() async {
    try {
      await SystemChrome.setEnabledSystemUIMode(SystemUiMode.edgeToEdge);
    } catch (_) {
      // Web, desktop and widget tests have no platform channel here.
    }
  }

  /// Wires up FCM, tolerating a build with no Firebase config.
  ///
  /// Fire-and-forget and deliberately not awaited: push is an enhancement, and
  /// a device that cannot register must still reach the login screen. The
  /// status is reported to telemetry (metadata only, never the token) because
  /// "push silently does nothing" is otherwise impossible to diagnose from a
  /// bug report.
  Future<void> _initPush() async {
    try {
      final status = await PushService.instance.initialize();
      if (!status.isReady) {
        Telemetry.instance.reportEvent(
          'push_unavailable',
          payload: {'reason': status.detail ?? status.kind.name},
        );
        return;
      }
      if (status.kind == PushStatusKind.needsPermission) {
        final granted = await PushService.instance.requestPermission();
        Telemetry.instance.reportEvent(
          'push_permission',
          payload: {'granted': granted},
        );
        return;
      }
      Telemetry.instance.reportEvent(
        'push_ready',
        // project_id is a public identifier and is the single most useful thing
        // to see in a bug report. The token never appears here.
        payload: {'project_id': PushService.instance.projectId ?? ''},
      );
    } catch (error) {
      Telemetry.instance.reportEvent(
        'push_error',
        payload: {'detail': error.toString()},
      );
    }
  }

  Future<void> _restoreSession() async {
    final session = await _sessionStore.read();
    final language = await _sessionStore.readLanguage();
    final customized = await _sessionStore.hasCustomizedLanguage();
    UiFeedback.enabled = await _sessionStore.readSoundEnabled();
    final font = FontSetting(
      fontMode: FontMode.fromWire(await _sessionStore.readFontFamily()),
      size: FontSize.fromWire(await _sessionStore.readFontSize()),
    );
    final theme = await _sessionStore.readThemeSetting();
    Telemetry.instance.setSession(session);
    _setTour(session);
    if (!mounted) return;
    setState(() {
      _session = session;
      _locale = Locale(language);
      _languageCustomized = customized;
      _fontSetting = font;
      _themeSetting = theme;
      _restoring = false;
    });
  }

  void _setTour(Session? session) {
    if (session == null) {
      _tour?.disposeController();
      _tour = null;
      return;
    }
    _tour?.disposeController();
    _tour = TourController(
      registry: TourTargetRegistry.instance,
      api: _apiClient,
      session: session,
    );
  }

  Widget _wrapTour(BuildContext context, Widget? child) {
    final tour = _tour;
    if (tour == null || child == null) return child ?? const SizedBox.shrink();
    return TourOverlayHost(controller: tour, child: child);
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

  Future<void> _setFontMode(FontMode mode) async {
    await _sessionStore.saveFontFamily(mode.wire);
    if (mounted) {
      setState(() => _fontSetting = _fontSetting.copyWith(fontMode: mode));
    }
  }

  Future<void> _setFontSize(FontSize size) async {
    await _sessionStore.saveFontSize(size.wire);
    if (mounted) {
      setState(() => _fontSetting = _fontSetting.copyWith(size: size));
    }
  }

  Future<void> _setThemePreference(ThemePreference preference) async {
    await _sessionStore.saveThemePreference(preference.wire);
    if (mounted) {
      setState(() =>
          _themeSetting = _themeSetting.copyWith(preference: preference));
    }
  }

  Future<void> _setThemeAccent(ThemeAccent accent) async {
    await _sessionStore.saveThemeAccent(accent.wire);
    if (mounted) {
      setState(() => _themeSetting = _themeSetting.copyWith(accent: accent));
    }
  }

  Future<void> _authenticated(Session session) async {
    _setTour(session);
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
      navigatorKey: _navigatorKey,
      debugShowCheckedModeBanner: false,
      locale: _locale,
      supportedLocales: AppStrings.supportedLocales,
      localizationsDelegates: const [
        AppStrings.delegate,
        GlobalMaterialLocalizations.delegate,
        GlobalWidgetsLocalizations.delegate,
        GlobalCupertinoLocalizations.delegate,
      ],
      theme: buildPosTheme(
        _themeSetting,
        Brightness.light,
        fontFamily: resolveFonts(
            _fontSetting.fontMode, _locale.languageCode == 'ar'),
        fontSizeFactor: fontScale(_fontSetting.size),
      ),
      darkTheme: buildPosTheme(
        _themeSetting,
        Brightness.dark,
        fontFamily: resolveFonts(
            _fontSetting.fontMode, _locale.languageCode == 'ar'),
        fontSizeFactor: fontScale(_fontSetting.size),
      ),
      themeMode: _themeSetting.preference.toThemeMode,
      builder: (context, child) => _wrapTour(context, child),
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
                    fontSetting: _fontSetting,
                    onFontModeChanged: _setFontMode,
                    onFontSizeChanged: _setFontSize,
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
                    fontSetting: _fontSetting,
                    onFontModeChanged: _setFontMode,
                    onFontSizeChanged: _setFontSize,
                    themeSetting: _themeSetting,
                    onThemePreferenceChanged: _setThemePreference,
                    onThemeAccentChanged: _setThemeAccent,
                    onAuthenticated: _authenticated,
                  )
                : PosScreen(
                    key: const ValueKey('pos'),
                    session: _session!,
                  apiClient: _apiClient,
                  localDatabase: _localDatabase,
                  sessionStore: _sessionStore,
                  fontSetting: _fontSetting,
                  themeSetting: _themeSetting,
                  tourController: _tour,
                  onLanguageChanged: _setLanguage,
                  onFontModeChanged: _setFontMode,
                  onFontSizeChanged: _setFontSize,
                  onThemePreferenceChanged: _setThemePreference,
                  onThemeAccentChanged: _setThemeAccent,
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
                    _setTour(null);
                    if (mounted) setState(() => _session = null);
                    Telemetry.instance.setSession(null);
                  },
                ),
              ),
    );
  }
}
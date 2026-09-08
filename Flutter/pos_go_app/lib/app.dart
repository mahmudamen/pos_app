import 'package:flutter/material.dart';

import 'core/api_client.dart';
import 'core/session_store.dart';
import 'features/auth/login_screen.dart';
import 'features/pos/pos_screen.dart';

class PosApp extends StatefulWidget {
  const PosApp({super.key});

  @override
  State<PosApp> createState() => _PosAppState();
}

class _PosAppState extends State<PosApp> {
  final _sessionStore = SessionStore();
  late final ApiClient _apiClient;
  Session? _session;
  bool _restoring = true;

  @override
  void initState() {
    super.initState();
    _apiClient = ApiClient(
      onSessionRefreshed: (session) async {
        await _sessionStore.save(session);
        if (mounted) setState(() => _session = session);
      },
    );
    _restoreSession();
  }

  Future<void> _restoreSession() async {
    final session = await _sessionStore.read();
    if (!mounted) return;
    setState(() {
      _session = session;
      _restoring = false;
    });
  }

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'POS Go',
      debugShowCheckedModeBanner: false,
      theme: ThemeData(
        colorScheme: ColorScheme.fromSeed(seedColor: const Color(0xff0f766e)),
        scaffoldBackgroundColor: const Color(0xfff4f7f6),
        useMaterial3: true,
      ),
      home: _restoring
          ? const Scaffold(body: Center(child: CircularProgressIndicator()))
          : _session == null
              ? LoginScreen(
                  apiClient: _apiClient,
                  onAuthenticated: (session) async {
                    await _sessionStore.save(session);
                    if (mounted) setState(() => _session = session);
                  },
                )
              : PosScreen(
                  session: _session!,
                  apiClient: _apiClient,
                  onSignOut: () async {
                    try {
                      await _apiClient.logout(_session!);
                    } catch (_) {
                      // Clear local credentials even when the server is offline.
                    }
                    await _sessionStore.clear();
                    if (mounted) setState(() => _session = null);
                  },
                ),
    );
  }
}

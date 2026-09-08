import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:shared_preferences/shared_preferences.dart';
import '../state/app_provider.dart';
import '../theme/app_theme.dart';
import '../core/localization/app_localizations.dart';

class LoginScreen extends StatefulWidget {
  const LoginScreen({super.key});

  @override
  State<LoginScreen> createState() => _LoginScreenState();
}

class _LoginScreenState extends State<LoginScreen>
    with TickerProviderStateMixin {
  final _formKey = GlobalKey<FormState>();
  // Default credentials for demo
  final _nameController = TextEditingController(text: 'admin');
  final _passwordController = TextEditingController(text: '123456');

  late AnimationController _controller;
  late Animation<double> _fadeAnimation;
  late Animation<double> _scaleAnimation;

  // Staggered animation for each item
  late Animation<double> _logoAnimation;
  late Animation<double> _titleAnimation;
  late Animation<double> _subtitleAnimation;
  late Animation<double> _formAnimation;
  late Animation<double> _buttonAnimation;

  bool _obscurePassword = true;
  bool _isShaking = false;
  bool _rememberMe = true;

  @override
  void initState() {
    super.initState();

    _controller = AnimationController(
      vsync: this,
      duration: const Duration(milliseconds: 1800),
    );

    // General Animation
    _fadeAnimation = Tween<double>(begin: 0.0, end: 1.0).animate(
      CurvedAnimation(
        parent: _controller,
        curve: const Interval(0.0, 0.6, curve: Curves.easeOut),
      ),
    );

    _scaleAnimation = Tween<double>(
      begin: 0.8,
      end: 1.0,
    ).animate(CurvedAnimation(parent: _controller, curve: Curves.elasticOut));

    // Staggered Animation
    _logoAnimation = Tween<double>(begin: 0.0, end: 1.0).animate(
      CurvedAnimation(
        parent: _controller,
        curve: const Interval(0.1, 0.5, curve: Curves.easeOutBack),
      ),
    );
    _titleAnimation = Tween<double>(begin: 0.0, end: 1.0).animate(
      CurvedAnimation(
        parent: _controller,
        curve: const Interval(0.3, 0.7, curve: Curves.easeOut),
      ),
    );
    _subtitleAnimation = Tween<double>(begin: 0.0, end: 1.0).animate(
      CurvedAnimation(
        parent: _controller,
        curve: const Interval(0.4, 0.8, curve: Curves.easeOut),
      ),
    );
    _formAnimation = Tween<double>(begin: 0.0, end: 1.0).animate(
      CurvedAnimation(
        parent: _controller,
        curve: const Interval(0.5, 0.9, curve: Curves.easeOutCubic),
      ),
    );
    _buttonAnimation = Tween<double>(begin: 0.0, end: 1.0).animate(
      CurvedAnimation(
        parent: _controller,
        curve: const Interval(0.7, 1.0, curve: Curves.easeOutCubic),
      ),
    );

    _controller.forward();
  }

  @override
  void dispose() {
    _nameController.dispose();
    _passwordController.dispose();
    _controller.dispose();
    super.dispose();
  }

  void _triggerShake() async {
    setState(() => _isShaking = true);
    await Future.delayed(const Duration(milliseconds: 600));
    if (mounted) setState(() => _isShaking = false);
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final app = context.watch<AppProvider>();
    final loc = AppLocalizations.of(context);

    return Scaffold(
      resizeToAvoidBottomInset: true,
      body: Stack(
        children: [
          // Gradient background with light Blur
          Container(
            decoration: BoxDecoration(
              gradient: LinearGradient(
                begin: Alignment.topLeft,
                end: Alignment.bottomRight,
                colors: [
                  theme.colorScheme.primary.withOpacity(0.15),
                  theme.colorScheme.secondary.withOpacity(0.05),
                  theme.colorScheme.background,
                ],
              ),
            ),
          ),

          SafeArea(
            child: Center(
              child: ConstrainedBox(
                constraints: const BoxConstraints(maxWidth: 440),
                child: SingleChildScrollView(
                  padding: const EdgeInsets.all(24),
                  child: ScaleTransition(
                    scale: _scaleAnimation,
                    child: FadeTransition(
                      opacity: _fadeAnimation,
                      child: Card(
                        elevation: 20,
                        shadowColor: theme.colorScheme.primary.withOpacity(
                          0.25,
                        ),
                        shape: RoundedRectangleBorder(
                          borderRadius: BorderRadius.circular(28),
                        ),
                        child: Padding(
                          padding: const EdgeInsets.fromLTRB(32, 40, 32, 40),
                          child: Form(
                            key: _formKey,
                            child: Column(
                              mainAxisSize: MainAxisSize.min,
                              children: [
                                // Logo
                                FadeTransition(
                                  opacity: _logoAnimation,
                                  child: SlideTransition(
                                    position:
                                        Tween<Offset>(
                                          begin: const Offset(0, -0.5),
                                          end: Offset.zero,
                                        ).animate(
                                          CurvedAnimation(
                                            parent: _controller,
                                            curve: const Interval(
                                              0.1,
                                              0.6,
                                              curve: Curves.easeOutBack,
                                            ),
                                          ),
                                        ),
                                    child: Container(
                                      padding: const EdgeInsets.all(20),
                                      decoration: BoxDecoration(
                                        gradient: LinearGradient(
                                          colors: [
                                            theme.colorScheme.primary,
                                            theme.colorScheme.primary
                                                .withOpacity(0.7),
                                          ],
                                        ),
                                        shape: BoxShape.circle,
                                        boxShadow: [
                                          BoxShadow(
                                            color: theme.colorScheme.primary
                                                .withOpacity(0.4),
                                            blurRadius: 20,
                                            offset: const Offset(0, 10),
                                          ),
                                        ],
                                      ),
                                      child: Image.asset(
                                        'assets/icons/Icon.png',
                                        width: 64,
                                        height: 64,
                                        errorBuilder: (_, __, ___) => Icon(
                                          Icons.store,
                                          size: 64,
                                          color: Colors.white,
                                        ),
                                      ),
                                    ),
                                  ),
                                ),
                                const SizedBox(height: 24),

                                // Title
                                FadeTransition(
                                  opacity: _titleAnimation,
                                  child: Text(
                                    loc.appFullName,
                                    style: theme.textTheme.headlineMedium
                                        ?.copyWith(
                                          fontWeight: FontWeight.w900,
                                          color: theme.colorScheme.primary,
                                          letterSpacing: 1.2,
                                        ),
                                    textAlign: TextAlign.center,
                                  ),
                                ),
                                const SizedBox(height: 8),

                                FadeTransition(
                                  opacity: _subtitleAnimation,
                                  child: Text(
                                    loc.welcomeBack,
                                    style: theme.textTheme.bodyLarge?.copyWith(
                                      color: theme.colorScheme.onSurfaceVariant,
                                    ),
                                    textAlign: TextAlign.center,
                                  ),
                                ),
                                const SizedBox(height: 40),

                                // Fields with animation
                                FadeTransition(
                                  opacity: _formAnimation,
                                  child: SlideTransition(
                                    position:
                                        Tween<Offset>(
                                          begin: const Offset(0, 0.3),
                                          end: Offset.zero,
                                        ).animate(
                                          CurvedAnimation(
                                            parent: _controller,
                                            curve: const Interval(
                                              0.5,
                                              0.9,
                                              curve: Curves.easeOutCubic,
                                            ),
                                          ),
                                        ),
                                    child: Column(
                                      children: [
                                        _buildAnimatedTextField(
                                          context: context,
                                          controller: _nameController,
                                          label: loc.enterUsername,
                                          icon: Icons.person_outline_rounded,
                                          validator: (v) =>
                                              v?.trim().isEmpty ?? true
                                              ? loc.enterUsername
                                              : null,
                                        ),
                                        const SizedBox(height: 20),
                                        _buildAnimatedTextField(
                                          context: context,
                                          controller: _passwordController,
                                          label: loc.enterPassword,
                                          icon: Icons.lock_outline_rounded,
                                          obscureText: _obscurePassword,
                                          isPassword: true,
                                          validator: (v) => v?.isEmpty ?? true
                                              ? loc.enterPassword
                                              : null,
                                          onSubmitted: (_) =>
                                              _handleLogin(context),
                                        ),
                                        const SizedBox(height: 10),
                                        // Checkbox for Remember Me
                                        FadeTransition(
                                          opacity: _formAnimation,
                                          child: Row(
                                            children: [
                                              Checkbox(
                                                value: _rememberMe,
                                                activeColor:
                                                    theme.colorScheme.primary,
                                                onChanged: (val) {
                                                  setState(() {
                                                    _rememberMe = val ?? true;
                                                  });
                                                },
                                              ),
                                              Text(
                                                loc.rememberMe,
                                                style: const TextStyle(
                                                  fontWeight: FontWeight.bold,
                                                ),
                                              ),
                                            ],
                                          ),
                                        ),
                                      ],
                                    ),
                                  ),
                                ),

                                const SizedBox(height: 24),

                                // Login button with animation
                                FadeTransition(
                                  opacity: _buttonAnimation,
                                  child: SlideTransition(
                                    position:
                                        Tween<Offset>(
                                          begin: const Offset(0, 0.5),
                                          end: Offset.zero,
                                        ).animate(
                                          CurvedAnimation(
                                            parent: _controller,
                                            curve: const Interval(
                                              0.7,
                                              1.0,
                                              curve: Curves.easeOutCubic,
                                            ),
                                          ),
                                        ),
                                    child: SizedBox(
                                      width: double.infinity,
                                      height: 56,
                                      child: ElevatedButton(
                                        onPressed: app.busy
                                            ? null
                                            : () => _handleLogin(context),
                                        style: ElevatedButton.styleFrom(
                                          elevation: 8,
                                          shadowColor: theme.colorScheme.primary
                                              .withOpacity(0.5),
                                          backgroundColor:
                                              theme.colorScheme.primary,
                                          foregroundColor: Colors.white,
                                          shape: RoundedRectangleBorder(
                                            borderRadius: BorderRadius.circular(
                                              16,
                                            ),
                                          ),
                                        ),
                                        child: app.busy
                                            ? const SizedBox(
                                                width: 28,
                                                height: 28,
                                                child:
                                                    CircularProgressIndicator(
                                                      color: Colors.white,
                                                      strokeWidth: 3,
                                                    ),
                                              )
                                            : Text(
                                                loc.login,
                                                style: const TextStyle(
                                                  fontSize: 17,
                                                  fontWeight: FontWeight.bold,
                                                  letterSpacing: 0.8,
                                                ),
                                              ),
                                      ),
                                    ),
                                  ),
                                ),

                                const SizedBox(height: 24),
                                Text(
                                  loc.appVersion,
                                  style: TextStyle(
                                    fontSize: 12,
                                    color: Colors.grey.shade600,
                                  ),
                                ),
                              ],
                            ),
                          ),
                        ),
                      ),
                    ),
                  ),
                ),
              ),
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildAnimatedTextField({
    required BuildContext context,
    required TextEditingController controller,
    required String label,
    required IconData icon,
    bool obscureText = false,
    bool isPassword = false,
    required String? Function(String?) validator,
    void Function(String)? onSubmitted,
  }) {
    return AnimatedContainer(
      duration: const Duration(milliseconds: 600),
      transform: Matrix4.translationValues(_isShaking ? 8 : 0, 0, 0)
        ..rotateZ(_isShaking ? 0.05 : 0),
      child: TextFormField(
        controller: controller,
        obscureText: obscureText,
        textInputAction: isPassword
            ? TextInputAction.done
            : TextInputAction.next,
        onFieldSubmitted: onSubmitted,
        style: const TextStyle(fontSize: 15.5, fontWeight: FontWeight.w500),
        decoration: InputDecoration(
          labelText: label,
          prefixIcon: Icon(icon, size: 22),
          suffixIcon: isPassword
              ? IconButton(
                  icon: Icon(
                    _obscurePassword
                        ? Icons.visibility_off_outlined
                        : Icons.visibility_outlined,
                  ),
                  onPressed: () =>
                      setState(() => _obscurePassword = !_obscurePassword),
                )
              : null,
          filled: true,
          fillColor: Theme.of(context).colorScheme.surface.withOpacity(0.6),
          border: OutlineInputBorder(
            borderRadius: BorderRadius.circular(16),
            borderSide: BorderSide.none,
          ),
          enabledBorder: OutlineInputBorder(
            borderRadius: BorderRadius.circular(16),
            borderSide: BorderSide(color: Colors.grey.withOpacity(0.3)),
          ),
          focusedBorder: OutlineInputBorder(
            borderRadius: BorderRadius.circular(16),
            borderSide: BorderSide(
              color: Theme.of(context).colorScheme.primary,
              width: 2.5,
            ),
          ),
          errorBorder: OutlineInputBorder(
            borderRadius: BorderRadius.circular(16),
            borderSide: const BorderSide(color: Colors.redAccent, width: 2),
          ),
          contentPadding: const EdgeInsets.symmetric(
            horizontal: 20,
            vertical: 18,
          ),
        ),
        validator: validator,
      ),
    );
  }

  Future<void> _handleLogin(BuildContext context) async {
    if (!_formKey.currentState!.validate()) {
      _triggerShake();
      return;
    }

    try {
      final appProvider = context.read<AppProvider>();
      if (!appProvider.initialized) await appProvider.initialize();

      await appProvider.login(
        _nameController.text.trim(),
        _passwordController.text,
        rememberMe: _rememberMe,
      );
      await appProvider.loadStores();
      await appProvider.loadCustomers();

      final sp = await SharedPreferences.getInstance();
      int storeId = int.tryParse(appProvider.currentUser?.idTStore ?? '') ?? 0;
      final savedStoreId = sp.getInt('default_store_id');
      if (savedStoreId != null && savedStoreId > 0) storeId = savedStoreId;
      if (storeId > 0) await appProvider.loadPrices(storeId);

      await appProvider.refreshBalance();

      if (mounted) {
        Navigator.pushReplacementNamed(context, '/home');
      }
    } catch (e) {
      if (!mounted) return;
      _triggerShake();

      final loc = AppLocalizations.of(context);
      String msg = loc.errorUnexpected(e.toString());
      if (e.toString().contains('timeout'))
        msg = loc.errorTimeout;
      else if (e.toString().contains('401'))
        msg = loc.invalidCredentials;
      else if (e.toString().contains('network'))
        msg = loc.errorConnection;

      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(msg), backgroundColor: Colors.redAccent),
      );
    }
  }

  void _showForgotPassword(BuildContext context) {
    final loc = AppLocalizations.of(context);
    ScaffoldMessenger.of(
      context,
    ).showSnackBar(SnackBar(content: Text(loc.passwordRecoverySoon)));
  }
}

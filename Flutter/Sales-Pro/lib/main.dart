import 'firebase_options.example.dart';
import 'package:sales_pro/repository/api_repository.dart';
import 'package:sales_pro/state/app_provider.dart';
import 'package:firebase_core/firebase_core.dart';
import 'package:firebase_messaging/firebase_messaging.dart';
import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:flutter_localizations/flutter_localizations.dart';

import 'core/localization/app_localizations.dart';
import 'services/language_service.dart';
import 'features/orders/dto/t_bill_dto.dart';
import 'screens/login_screen.dart';
import 'screens/invoice_screen.dart';
import 'screens/barcode_scan_screen.dart';
import 'screens/settings_screen.dart';
import 'screens/home_screen.dart';
import 'screens/splash_screen.dart';
import 'theme/app_theme.dart';

@pragma('vm:entry-point')
Future<void> _firebaseMessagingBackgroundHandler(RemoteMessage message) async {
  await Firebase.initializeApp(options: DefaultFirebaseOptions.currentPlatform);
  // Background message processing (removed debug print for production)
}

void main() async {
  WidgetsFlutterBinding.ensureInitialized();

  try {
    // Initialize Firebase in main with error handling
    await Firebase.initializeApp(
      options: DefaultFirebaseOptions.currentPlatform,
    );
    FirebaseMessaging.onBackgroundMessage(_firebaseMessagingBackgroundHandler);

    // Delay notification initialization for the app itself
    // await NotificationService().init();  // Will be done in AppProvider
    // await retrieveFCMToken();  // Will be done in AppProvider
  } catch (e) {
    print('Firebase initialization error: $e');
    // Continue running the app even if Firebase fails
  }

  runApp(const MomtazApp());
}

Future<void> retrieveFCMToken() async {
  try {
    String? token = await FirebaseMessaging.instance.getToken();
    if (token != null) {
      print('FCM Registration Token: $token');
      // You can show a message or send it to the server
      // Example: SnackBar
      // ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(token)));
    }
  } catch (e) {
    print('Error fetching FCM token: $e');
  }
}

class MomtazApp extends StatelessWidget {
  const MomtazApp({super.key});

  @override
  Widget build(BuildContext context) {
    final theme = buildAppTheme();

    return MultiProvider(
      providers: [
        Provider<ApiRepository>(create: (_) => ApiRepository()),
        ChangeNotifierProvider<AppProvider>(
          create: (ctx) => AppProvider(ctx.read<ApiRepository>()),
        ),
        ChangeNotifierProvider<LanguageService>(
          create: (_) => LanguageService()..init(),
        ),
      ],
      child: Consumer2<AppProvider, LanguageService>(
        builder: (context, app, languageService, child) {
          return MaterialApp(
            debugShowCheckedModeBanner: false,
            theme: theme,
            locale: languageService.currentLocale,
            supportedLocales: const [Locale('en'), Locale('ar')],
            localizationsDelegates: const [
              AppLocalizations.delegate,
              GlobalMaterialLocalizations.delegate,
              GlobalWidgetsLocalizations.delegate,
              GlobalCupertinoLocalizations.delegate,
            ],

            // Start from SplashScreen
            home: const SplashScreen(),

            routes: {
              '/login': (_) => const LoginScreen(),
              '/home': (_) => const HomeScreen(),
              '/scan': (_) => const BarcodeScanScreen(),
              '/settings': (_) => const SettingsScreen(),
            },

            onGenerateRoute: (settings) {
              if (settings.name == '/invoice') {
                final bill = settings.arguments as TBillDto?;
                return MaterialPageRoute(
                  builder: (_) => InvoiceScreen(bill: bill),
                );
              }
              return null;
            },
          );
        },
      ),
    );
  }
}

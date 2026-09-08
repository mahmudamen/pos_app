import 'package:flutter_local_notifications/flutter_local_notifications.dart';
import 'package:firebase_messaging/firebase_messaging.dart';
import 'package:firebase_core/firebase_core.dart';
import '../firebase_options.example.dart';

typedef NotificationCallback = void Function(RemoteMessage message);

class NotificationService {
  static final NotificationService _instance = NotificationService._internal();
  factory NotificationService() => _instance;
  NotificationService._internal();

  final FlutterLocalNotificationsPlugin flutterLocalNotificationsPlugin =
      FlutterLocalNotificationsPlugin();

  bool _initialized = false;
  NotificationCallback? onCustomerUpdate;
  NotificationCallback? onItemUpdate;
  NotificationCallback? onWarehouseUpdate;
  Future<void> init() async {
    if (_initialized) return;
    FirebaseMessaging.onBackgroundMessage(_firebaseMessagingBackgroundHandler);
    // Initialize Firebase
    await Firebase.initializeApp(
      options: DefaultFirebaseOptions.currentPlatform,
    );

    // Create Android notification channel
    const AndroidNotificationChannel channel = AndroidNotificationChannel(
      'main_channel',
      'Main Notifications',
      description: 'Used for important notifications.',
      importance: Importance.high,
    );

    // Register channel
    await flutterLocalNotificationsPlugin
        .resolvePlatformSpecificImplementation<
          AndroidFlutterLocalNotificationsPlugin
        >()
        ?.createNotificationChannel(channel);

    // Initialize local notifications
    const AndroidInitializationSettings androidInit =
        AndroidInitializationSettings('@mipmap/ic_launcher');
    const InitializationSettings initSettings = InitializationSettings(
      android: androidInit,
    );
    await flutterLocalNotificationsPlugin.initialize(initSettings);

    // Request permissions (iOS and Android 13+)
    await _requestPermissions();

    // Subscribe to topics
    await FirebaseMessaging.instance.subscribeToTopic('customers');
    await FirebaseMessaging.instance.subscribeToTopic('items');
    await FirebaseMessaging.instance.subscribeToTopic('warehouses');

    // Receive messages while app is running
    FirebaseMessaging.onMessage.listen(_onMessageReceived);

    // Handle notification tap
    FirebaseMessaging.onMessageOpenedApp.listen((RemoteMessage message) {
      // Notification clicked (debug info removed for production)
      _onMessageReceived(message);
    });

    // التحقق من الرسالة الأولية (إذا تم فتح التطبيق من الإشعار)
    FirebaseMessaging.instance.getInitialMessage().then((
      RemoteMessage? message,
    ) {
      if (message != null) {
        // App opened from notification (debug info removed for production)
        _onMessageReceived(message);
      }
    });

    _initialized = true;
    // NotificationService initialized (debug message removed for production)

    // التحقق من حالة callbacks بعد 2 ثانية لإعطاء وقت للتسجيل
    Future.delayed(const Duration(seconds: 2), () {
      checkCallbacksStatus();
    });
  }

  Future<void> _requestPermissions() async {
    await FirebaseMessaging.instance.requestPermission(
      alert: true,
      badge: true,
      sound: true,
    );
    // Notification permission status handled silently (debug print removed for production)
  }

  void _onMessageReceived(RemoteMessage message) {
    // Message received (debug info removed for production)

    final notification = message.notification;
    final android = message.notification?.android;
    if (notification != null && android != null) {
      flutterLocalNotificationsPlugin.show(
        notification.hashCode,
        notification.title,
        notification.body,
        const NotificationDetails(
          android: AndroidNotificationDetails(
            'main_channel',
            'Main Notifications',
            importance: Importance.max,
            priority: Priority.high,
          ),
        ),
      );
    }

    // تحديد نوع الموضوع من عدة مصادر محتملة
    final topic = message.from;
    final topicFromData = message.data['topic'];
    final typeFromData = message.data['type'];

    // Topic analysis (debug info removed for production)

    // فحص المواضيع بطرق متعددة
    bool isCustomerUpdate =
        topic == '/topics/customers' ||
        topicFromData == 'customers' ||
        typeFromData == 'customer_update';

    bool isItemUpdate =
        topic == '/topics/items' ||
        topicFromData == 'items' ||
        typeFromData == 'item_update';

    bool isWarehouseUpdate =
        topic == '/topics/warehouses' ||
        topicFromData == 'warehouses' ||
        typeFromData == 'warehouse_update';

    if (isCustomerUpdate) {
      onCustomerUpdate?.call(message);
    } else if (isItemUpdate) {
      onItemUpdate?.call(message);
    } else if (isWarehouseUpdate) {
      onWarehouseUpdate?.call(message);
    }
    // Unknown topic/type handled silently
  }

  Future<void> showLocalNotification({
    required String title,
    required String body,
  }) async {
    await flutterLocalNotificationsPlugin.show(
      DateTime.now().millisecondsSinceEpoch ~/ 1000,
      title,
      body,
      const NotificationDetails(
        android: AndroidNotificationDetails(
          'main_channel',
          'Main Notifications',
          importance: Importance.max,
          priority: Priority.high,
        ),
      ),
    );
  }

  /// تحقق من حالة callbacks
  void checkCallbacksStatus() {
    // Callback status check (debug info removed for production)
  }

  /// محاكاة استلام رسالة لاختبار callbacks
  void simulateMessage(String type) {
    // Message simulation (debug info removed for production)

    // محاكاة الرسالة مباشرة بدلاً من إنشاء RemoteMessage
    if (type == 'item' && onItemUpdate != null) {
      // يمكن استدعاء callback مباشرة للاختبار
      // onItemUpdate?.call(mockMessage); // يحتاج RemoteMessage حقيقي
    } else if (type == 'customer' && onCustomerUpdate != null) {
      // Customer callback test
    } else if (type == 'warehouse' && onWarehouseUpdate != null) {
      // Warehouse callback test
    }
    // Callback status handled silently
  }
}

@pragma('vm:entry-point')
Future<void> _firebaseMessagingBackgroundHandler(RemoteMessage message) async {
  await Firebase.initializeApp(options: DefaultFirebaseOptions.currentPlatform);
  // Background message handling (debug info removed for production)

  // يمكن إضافة معالجة خاصة للرسائل في الخلفية هنا إذا لزم الأمر
}

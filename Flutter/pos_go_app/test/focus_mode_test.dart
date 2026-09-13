import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:pos_go_app/core/focus_mode.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  test('FocusMode.apply forwards enabled flag to the native channel', () async {
    final actual = <Map<String, dynamic>>[];
    final messenger =
        TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger;
    const channel = MethodChannel('com.xamltech.pos_go/focus');
    messenger.setMockMethodCallHandler(channel, (call) async {
      if (call.method == 'dndGranted') return true;
      actual.add((call.arguments as Map).cast<String, dynamic>());
      return null;
    });
    try {
      final dndActive = await FocusMode.apply(true);
      expect(dndActive, isTrue);
      await FocusMode.apply(false);
      expect(actual, hasLength(2));
      expect(actual[0]['enabled'], isTrue);
      expect(actual[1]['enabled'], isFalse);
    } finally {
      messenger.setMockMethodCallHandler(channel, null);
    }
  });

  test('FocusMode.apply degrades gracefully on hosts without a channel', () async {
    final messenger =
        TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger;
    messenger.setMockMethodCallHandler(const MethodChannel('pos_go/unmapped'),
        null);
    final dndActive = await FocusMode.apply(true);
    expect(dndActive, isFalse);
    await FocusMode.apply(false);
  });
}
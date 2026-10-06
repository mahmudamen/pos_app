import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:pos_go_app/core/app_links.dart';

void main() {
  group('PosAppLink.parse', () {
    test('accepts the claimed origin and path prefix', () {
      final link = PosAppLink.parse(
        Uri.parse('https://api.xamltech.com/pos'),
      );
      expect(link, isNotNull);
      expect(link!.destination, PosLinkDestination.pos);
    });

    test('accepts a sub-path and classifies it', () {
      final cases = <String, PosLinkDestination>{
        '/pos/sales': PosLinkDestination.sales,
        '/pos/products': PosLinkDestination.products,
        '/pos/catalog': PosLinkDestination.products,
        '/pos/customers': PosLinkDestination.customers,
        '/pos/dashboard': PosLinkDestination.dashboard,
        '/pos/insights': PosLinkDestination.dashboard,
        '/pos/community': PosLinkDestination.community,
        '/pos/whatever': PosLinkDestination.unknown,
        '/pos/a/b/c': PosLinkDestination.unknown,
      };
      cases.forEach((path, destination) {
        final link =
            PosAppLink.parse(Uri.parse('https://api.xamltech.com$path'));
        expect(link, isNotNull, reason: path);
        expect(link!.destination, destination, reason: path);
      });
    });

    test('tolerates trailing slash, query and fragment', () {
      for (final suffix in ['/', '?tab=sales', '#top', '/?tab=sales']) {
        final link = PosAppLink.parse(
          Uri.parse('https://api.xamltech.com/pos$suffix'),
        );
        expect(link, isNotNull, reason: suffix);
        expect(link!.destination, PosLinkDestination.pos, reason: suffix);
      }
    });

    test('ignores a query or fragment on a sub-path', () {
      final link = PosAppLink.parse(
        Uri.parse('https://api.xamltech.com/pos/sales?from=email#x'),
      );
      expect(link!.destination, PosLinkDestination.sales);
    });

    test('is case insensitive on host and path', () {
      final link = PosAppLink.parse(
        Uri.parse('https://API.XamlTech.com/POS/Sales'),
      );
      expect(link, isNotNull);
      expect(link!.destination, PosLinkDestination.sales);
    });

    test('rejects links this app must not handle', () {
      final rejected = <String>[
        // Claiming the whole host would put the app in front of the API and
        // the admin console, so those must never resolve.
        'https://api.xamltech.com/v1/sales',
        'https://api.xamltech.com/admin/',
        'https://api.xamltech.com/',
        // `/positional` must not pass as `/pos`.
        'https://api.xamltech.com/positional',
        'https://api.xamltech.com/pos-foo',
        'https://api.xamltech.com/v2/pos',
        // Other hosts and schemes are not verified for this app.
        'https://posgo.xamltech.com/pos',
        'https://evil.example.com/pos',
        'http://api.xamltech.com/pos',
      ];
      for (final url in rejected) {
        expect(PosAppLink.parse(Uri.parse(url)), isNull, reason: url);
      }
    });

    test('rejects a null link', () {
      expect(PosAppLink.parse(null), isNull);
    });
  });

  group('AppLinkService', () {
    test('replays the launch link and then live links', () async {
      final source = FakeAppLinkSource(
        initial: Uri.parse('https://api.xamltech.com/pos'),
      );
      final service = AppLinkService(source: source);
      final seen = <PosAppLink>[];
      service.links.listen(seen.add);
      await service.start();

      await pumpEventQueue();
      expect(seen.single.destination, PosLinkDestination.pos);

      source.emit(Uri.parse('https://api.xamltech.com/pos/sales'));
      await pumpEventQueue();
      expect(seen, hasLength(2));
      expect(seen.last.destination, PosLinkDestination.sales);
      await service.dispose();
    });

    test('drops links the app does not claim', () async {
      final source = FakeAppLinkSource();
      final service = AppLinkService(source: source);
      final seen = <PosAppLink>[];
      service.links.listen(seen.add);
      await service.start();

      source.emit(Uri.parse('https://api.xamltech.com/v1/sales'));
      source.emit(Uri.parse('https://api.xamltech.com/pos'));
      await pumpEventQueue();

      expect(seen, hasLength(1));
      expect(seen.single.uri.path, '/pos');
      await service.dispose();
    });

    test('start is idempotent', () async {
      final source = FakeAppLinkSource(
        initial: Uri.parse('https://api.xamltech.com/pos'),
      );
      final service = AppLinkService(source: source);
      expect(await service.start(), isTrue);
      expect(await service.start(), isTrue);
      await service.dispose();
    });

    test('reports unavailable instead of throwing when the plugin is absent',
        () async {
      // Web and desktop have no App Links platform channel; the app must keep
      // working, so a missing plugin is a capability result, not an error.
      final service = AppLinkService(source: ThrowingAppLinkSource());
      expect(await service.start(), isFalse);
      await service.dispose();
    });
  });

  group('revealPosScreen', () {
    testWidgets('pops a pushed sub-screen back to the POS home', (tester) async {
      final key = GlobalKey<NavigatorState>();
      await tester.pumpWidget(
        MaterialApp(
          navigatorKey: key,
          home: const Scaffold(body: Text('pos-home')),
        ),
      );
      key.currentState!.push(
        MaterialPageRoute<void>(
          builder: (_) => const Scaffold(body: Text('community-sub-screen')),
        ),
      );
      await tester.pumpAndSettle();
      expect(find.text('community-sub-screen'), findsOneWidget);

      expect(revealPosScreen(key), isTrue);
      await tester.pumpAndSettle();

      expect(find.text('community-sub-screen'), findsNothing);
      expect(find.text('pos-home'), findsOneWidget);
    });

    testWidgets('reports false when already on the POS', (tester) async {
      final key = GlobalKey<NavigatorState>();
      await tester.pumpWidget(
        MaterialApp(
          navigatorKey: key,
          home: const Scaffold(body: Text('pos-home')),
        ),
      );
      await tester.pumpAndSettle();

      expect(revealPosScreen(key), isFalse);
      expect(find.text('pos-home'), findsOneWidget);
    });

    testWidgets('reports false before the navigator exists', (tester) async {
      // A link can arrive before the first frame: popping is not merely
      // pointless, it must not throw.
      expect(revealPosScreen(GlobalKey<NavigatorState>()), isFalse);
    });
  });
}

class FakeAppLinkSource implements AppLinkSource {
  FakeAppLinkSource({this.initial});

  final Uri? initial;
  final StreamController<Uri> _controller = StreamController<Uri>.broadcast();

  @override
  Future<Uri?> initialLink() async => initial;

  @override
  Stream<Uri> get linkStream => _controller.stream;

  void emit(Uri uri) => _controller.add(uri);
}

class ThrowingAppLinkSource implements AppLinkSource {
  @override
  Future<Uri?> initialLink() async =>
      throw MissingPluginException('no app_links here');

  @override
  Stream<Uri> get linkStream =>
      throw MissingPluginException('no app_links here');
}

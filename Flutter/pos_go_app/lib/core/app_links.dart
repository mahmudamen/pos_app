import 'dart:async';

import 'package:app_links/app_links.dart';
import 'package:flutter/material.dart';

/// Returns the app to the POS, its home surface.
///
/// A link is meant to open the POS. If the user was deep in a pushed
/// sub-screen (dashboard, customers, community…) and taps the link again, the
/// app must actually come back rather than leave them where they were — and
/// that pop has to happen from a navigator the app owns, since the pushed
/// screens belong to the POS widget's own navigation scope.
///
/// Returns false when there is nothing to pop, so callers can tell "landed on
/// the POS" from "was already there".
bool revealPosScreen(GlobalKey<NavigatorState> navigatorKey) {
  final navigator = navigatorKey.currentState;
  if (navigator == null) return false;
  if (!navigator.canPop()) return false;
  navigator.popUntil((route) => route.isFirst);
  return true;
}

/// Where inside the POS an App Link points.
///
/// Only destinations the app can actually reach are enumerated. A path the app
/// does not understand yet still resolves to [pos] rather than to an error
/// screen: the link was accepted by the intent filter, so opening the POS is
/// the useful answer, and [unknown] is what telemetry reports so the real
/// demand for more destinations shows up in the numbers instead of in bug
/// reports.
enum PosLinkDestination {
  pos,
  sales,
  products,
  customers,
  dashboard,
  community,
  unknown,
}

extension PosLinkDestinationWire on PosLinkDestination {
  String get wireName => name;

  static PosLinkDestination fromPath(String path) {
    // /pos/<section>; the section is the first non-empty segment below the
    // prefix. Trailing slashes and query strings must not change the outcome.
    final segments = path
        .split('/')
        .where((segment) => segment.isNotEmpty)
        .toList(growable: false);
    if (segments.length < 2) return PosLinkDestination.pos;
    switch (segments[1].toLowerCase()) {
      case 'sales':
        return PosLinkDestination.sales;
      case 'products':
      case 'catalog':
        return PosLinkDestination.products;
      case 'customers':
        return PosLinkDestination.customers;
      case 'dashboard':
      case 'insights':
        return PosLinkDestination.dashboard;
      case 'community':
        return PosLinkDestination.community;
      default:
        return PosLinkDestination.unknown;
    }
  }
}

/// A verified App Link that belongs to this app.
class PosAppLink {
  const PosAppLink({required this.uri, required this.destination});

  final Uri uri;
  final PosLinkDestination destination;

  /// The only origin the manifest claims, and the only one whose
  /// `assetlinks.json` we control. Both must match the
  /// `<intent-filter>` in `android/app/src/main/AndroidManifest.xml`; if either
  /// drifts, the link verifies but the app is never offered for it, and the
  /// failure is invisible from here.
  static const String claimedHost = 'api.xamltech.com';

  /// Path prefix claimed by the manifest. Kept deliberately narrow so the app
  /// is never offered for `/v1` API calls or the `/admin` console.
  static const String claimedPathPrefix = '/pos';

  /// Returns null for anything this app should not handle.
  ///
  /// Deliberately strict: a link is only honoured when the scheme, host and
  /// path prefix all match, because anything looser turns the app into a
  /// candidate handler for links it cannot serve.
  static PosAppLink? parse(Uri? uri) {
    if (uri == null) return null;
    if (uri.scheme != 'https') return null;
    if (uri.host.toLowerCase() != claimedHost) return null;
    final path = uri.path.toLowerCase();
    // `/positional` must not pass as `/pos`, and `/pos` itself must.
    if (path != claimedPathPrefix &&
        !path.startsWith('$claimedPathPrefix/')) {
      return null;
    }
    return PosAppLink(
      uri: uri,
      destination: PosLinkDestinationWire.fromPath(uri.path),
    );
  }

  @override
  String toString() => 'PosAppLink(${uri.toString()}, $destination)';
}

/// Source of incoming links, abstracted so tests never touch the plugin.
///
/// The plugin is a platform channel: calling it in a widget test throws a
/// MissingPluginException, and on a desktop or web target there is no App Links
/// support at all. Everything platform-specific lives behind this interface.
abstract class AppLinkSource {
  /// The link the activity was launched with, if any.
  Future<Uri?> initialLink();

  /// Links delivered while the app is already running. `singleTop` plus
  /// `app_links` means a second tap arrives here rather than restarting the
  /// activity.
  Stream<Uri> get linkStream;
}

class PluginAppLinkSource implements AppLinkSource {
  const PluginAppLinkSource();

  @override
  Future<Uri?> initialLink() => AppLinks().getInitialLink();

  @override
  Stream<Uri> get linkStream => AppLinks().uriLinkStream;
}

/// Turns a [AppLinkSource] into verified [PosAppLink]s.
class AppLinkService {
  AppLinkService({AppLinkSource? source})
      : _source = source ?? const PluginAppLinkSource();

  final AppLinkSource _source;
  final StreamController<PosAppLink> _links =
      StreamController<PosAppLink>.broadcast();
  StreamSubscription<Uri>? _subscription;
  bool _started = false;

  /// Verified links only.
  Stream<PosAppLink> get links => _links.stream;

  /// Starts delivering links, replaying the launch link first.
  ///
  /// Returns false when the platform has no App Links support, which is the
  /// normal case on web and desktop. That is not an error: the app must keep
  /// working, and the only consequence is that App Links do nothing there.
  Future<bool> start() async {
    if (_started) return true;
    _started = true;
    try {
      final initial = PosAppLink.parse(await _source.initialLink());
      if (initial != null) _links.add(initial);
      _subscription = _source.linkStream.listen(
        (uri) {
          final link = PosAppLink.parse(uri);
          if (link != null) _links.add(link);
        },
        onError: (Object _) {
          // A platform channel failure must not take the app down; the app is
          // fully usable without deep links.
        },
      );
      return true;
    } catch (_) {
      // MissingPluginException on a target without App Links support.
      return false;
    }
  }

  Future<void> dispose() async {
    await _subscription?.cancel();
    _subscription = null;
    await _links.close();
  }
}

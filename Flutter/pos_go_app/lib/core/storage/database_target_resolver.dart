/// Picks the offline-database implementation for the platform.
///
/// The web implementation imports `dart:js_interop`-based packages that cannot
/// be compiled for Android, and the mobile one calls `getDatabasesPath()` from
/// `dart:io`, so neither can be imported unconditionally.
library;

export 'database_target_io.dart'
    if (dart.library.js_interop) 'database_target_web.dart';
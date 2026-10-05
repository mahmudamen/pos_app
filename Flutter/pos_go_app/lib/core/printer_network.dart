/// Printing needs raw TCP sockets, which only exist on platforms with
/// `dart:io`. The browser gets an equivalent surface that reports the
/// capability as missing instead of failing to compile.
library;

export 'printer_network_io.dart'
    if (dart.library.js_interop) 'printer_network_web.dart';
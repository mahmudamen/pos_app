/// `SocketException` lives in `dart:io`, which the browser has no equivalent
/// of. On the web the transport failure arrives as an `http.ClientException`
/// instead, so the io-only check is compiled out there.
library;

export 'socket_error_io.dart'
    if (dart.library.js_interop) 'socket_error_web.dart';

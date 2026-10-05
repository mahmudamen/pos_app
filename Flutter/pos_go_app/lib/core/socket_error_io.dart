import 'dart:io';

/// Whether [error] is the `dart:io` socket failure a dead link produces.
bool isSocketError(Object error) => error is SocketException;

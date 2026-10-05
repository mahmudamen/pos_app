/// No `dart:io` in the browser, so no `SocketException` can be raised: a dead
/// link surfaces as an `http.ClientException` instead.
bool isSocketError(Object error) => false;

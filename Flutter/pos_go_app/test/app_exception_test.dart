import 'package:flutter_test/flutter_test.dart';

import 'package:pos_go_app/core/app_exception.dart';

void main() {
  test('NetworkException stores message', () {
    const e = NetworkException('connection refused');
    expect(e.message, 'connection refused');
    expect(e, isA<AppException>());
  });

  test('AuthException stores code and message', () {
    const e = AuthException('token expired', code: 'unauthorized');
    expect(e.message, 'token expired');
    expect(e.code, 'unauthorized');
  });

  test('StockException stores message', () {
    const e = StockException('insufficient stock');
    expect(e.message, 'insufficient stock');
    expect(e, isA<AppException>());
  });

  test('mapApiError returns AuthException for 401', () {
    final e = mapApiError(401, 'unauthorized');
    expect(e, isA<AuthException>());
    expect(e.code, 'unauthorized');
  });

  test('mapApiError returns StockException for 409 with stock message', () {
    final e = mapApiError(409, 'insufficient stock');
    expect(e, isA<StockException>());
  });

  test('mapApiError returns ServerException for 503', () {
    final e = mapApiError(503, 'database unavailable');
    expect(e, isA<ServerException>());
    expect(e.code, 'unavailable');
  });

  test('mapApiError returns ValidationException for 400', () {
    final e = mapApiError(400, 'invalid request');
    expect(e, isA<ValidationException>());
  });

  test('mapApiError returns ServerException for unknown status', () {
    final e = mapApiError(500, 'something broke');
    expect(e, isA<ServerException>());
    expect(e.code, 'unknown');
  });

  test('AppException toString returns message', () {
    const e = AppException('test message');
    expect(e.toString(), 'test message');
  });
}

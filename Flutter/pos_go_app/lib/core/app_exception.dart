import 'dart:async';

import 'package:http/http.dart' as http;

import 'socket_error.dart';

class AppException implements Exception {
  const AppException(this.message, {this.code, this.original});
  final String message;
  final String? code;
  final Object? original;

  @override
  String toString() => message;
}

class NetworkException extends AppException {
  const NetworkException(super.message, {super.original});
}

/// Whether a failure means "the request never reached the server".
///
/// This is the switch between the two modes the POS runs in. Online it is
/// ordinary error handling; when the network cuts, the sale has to be taken
/// locally and pushed later, and the only safe trigger for that is a transport
/// failure. Treating "any error that is not an ApiException" as offline is
/// what this predicate exists to stop: a StateError from a malformed payload is
/// a bug, and queueing it would replay a sale the server never accepted.
///
/// Covers [NetworkException] (thrown by ApiClient for a dead link) plus the
/// raw socket/client/timeout errors a caller can still see from an
/// unauthenticated call such as login. `SocketException` is matched through
/// [isSocketError] because it only exists where `dart:io` does; on the web the
/// same condition arrives as an `http.ClientException`.
bool isTransportFailure(Object error) =>
    error is NetworkException ||
    isSocketError(error) ||
    error is http.ClientException ||
    error is TimeoutException;

class AuthException extends AppException {
  const AuthException(super.message, {super.code, super.original});
}

class ServerException extends AppException {
  const ServerException(super.message, {required super.code, super.original});
}

class ValidationException extends AppException {
  const ValidationException(super.message, {super.original});
}

class StockException extends AppException {
  const StockException(super.message, {super.original});
}

/// A 402 the merchant can actually act on: either the plan does not grant the
/// feature (`plan_limit_exceeded`, whose 402 body names the missing
/// entitlements) or a metered window is exhausted (`ocr_window_limit_reached`).
/// Carried as a type so the UI can say "not in your plan" in the user's
/// language instead of pasting the server's English sentence.
class PlanLimitException extends AppException {
  const PlanLimitException(super.message, {required super.code, this.feature});

  final String? feature;

  bool get isMetered => code == 'ocr_window_limit_reached';
}

AppException mapApiError(int statusCode, String message, {String? code}) {
  switch (statusCode) {
    case 401:
      return AuthException(message, code: 'unauthorized');
    case 402:
      if (code == 'plan_limit_exceeded' || code == 'ocr_window_limit_reached') {
        return PlanLimitException(message, code: code);
      }
      return ServerException(message, code: code ?? 'payment_required');
    case 404:
      return ServerException(message, code: 'not_found');
    case 409:
      if (message.contains('stock')) {
        return StockException(message);
      }
      return ServerException(message, code: 'conflict');
    case 422:
    case 400:
      return ValidationException(message);
    case 503:
      return ServerException(message, code: 'unavailable');
    default:
      return ServerException(message, code: code ?? 'unknown');
  }
}

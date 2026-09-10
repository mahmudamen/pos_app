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

AppException mapApiError(int statusCode, String message) {
  switch (statusCode) {
    case 401:
      return AuthException(message, code: 'unauthorized');
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
      return ServerException(message, code: 'unknown');
  }
}

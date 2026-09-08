import '../core/soap_client.dart';
import 'package:meta/meta.dart';
import '../features/auth/auth_api.dart';
import '../features/catalog/catalog_api.dart';
import '../features/balance/balance_api.dart';
import '../features/invoice/invoice_api.dart';
import '../features/orders/orders_api.dart';
import '../features/time/time_api.dart';

@immutable
class ApiRepository {
  final SoapClient client;
  late final AuthApi auth;
  late final CatalogApi catalog;
  late final BalanceApi balance;
  late final InvoiceApi invoice;
  late final OrdersApi orders;
  late final TimeApi time;
  ApiRepository({SoapClient? client}) : client = client ?? SoapClient() {
    auth = AuthApi(this.client);
    catalog = CatalogApi(this.client);
    balance = BalanceApi(this.client);
    invoice = InvoiceApi(this.client);
    orders = OrdersApi(this.client);
    time=TimeApi(this.client);
  }
}

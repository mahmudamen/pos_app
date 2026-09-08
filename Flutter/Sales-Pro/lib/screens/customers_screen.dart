import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../state/app_provider.dart';
import '../services/notification_service.dart'; // تأكد من استيراد الخدمة هنا
import '../features/catalog/dto/customer_dto.dart';
import 'widgets/data_list_item.dart';

class CustomersScreen extends StatefulWidget {
  const CustomersScreen({super.key});

  @override
  State<CustomersScreen> createState() => _CustomersScreenState();
}

class _CustomersScreenState extends State<CustomersScreen> {
  bool _loading = true;
  String _query = '';

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) async {
      final app = context.read<AppProvider>();
      if (!app.initialized) {
        await app.initialize();
      }

      final customers = context.read<AppProvider>().customers;
      if (customers.isEmpty) {
        await app.loadCustomers(); // تحميل العملاء إن لم تكن البيانات موجودة
      }
      setState(() => _loading = false);
    });
  }

  @override
  Widget build(BuildContext context) {
    final app = context.watch<AppProvider>();
    final List<TCustomerDto> list = _query.trim().isEmpty
        ? app.customers
        : app.customers
              .where(
                (c) =>
                    c.name.contains(_query) ||
                    c.barcode.contains(_query) ||
                    c.id.toString().contains(_query),
              )
              .toList();

    return Padding(
      padding: const EdgeInsets.all(16.0),
      child: _loading
          ? const Center(child: CircularProgressIndicator())
          : Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                TextField(
                  textAlign: TextAlign.left,
                  decoration: const InputDecoration(
                    prefixIcon: Icon(Icons.search),
                    labelText: 'Search by Name / Barcode / ID',
                  ),
                  onChanged: (v) => setState(() => _query = v.trim()),
                ),
                const SizedBox(height: 12),
                Expanded(
                  child: list.isEmpty
                      ? const Center(
                          child: Text(
                            'No results found',
                            style: TextStyle(color: Colors.black54),
                          ),
                        )
                      : ListView.separated(
                          itemCount: list.length,
                          separatorBuilder: (_, __) => const Divider(height: 1),
                          itemBuilder: (context, i) {
                            final c = list[i];
                            return DataListItem(
                              title: c.name.isEmpty ? '(${c.id})' : c.name,
                              subtitle: 'Barcode: ${c.barcode}',

                              trailingBottom: c.credit.toString(),
                              onTap: () {
                                // Open customer details
                              },
                            );
                          },
                        ),
                ),
              ],
            ),
    );
  }
}

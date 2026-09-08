import 'package:sales_pro/screens/widgets/data_list_item.dart';
import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../state/app_provider.dart';
import '../services/prefs.dart';
import '../features/catalog/dto/price_dto.dart';
import '../theme/app_theme.dart';

class StoresScreen extends StatefulWidget {
  const StoresScreen({super.key});

  @override
  State<StoresScreen> createState() => _StoresScreenState();
}

class _StoresScreenState extends State<StoresScreen> {
  bool _loading = true;
  int? _storeId;
  String _query = '';

  @override
  void initState() {
    super.initState();
    _initializeScreen();
  }

  Future<void> _initializeScreen() async {
    final app = context.read<AppProvider>();

    setState(() => _loading = true);

    // Load stores if empty
    if (app.stores.isEmpty) {
      await app.loadStores();
    }

    // Get default store
    int? defId = await Prefs.getDefaultStoreIdWithFallback();
    if (defId == null) {
      final idStr = app.currentUser?.idTStore ?? '';
      defId = int.tryParse(idStr);
    }
    setState(() => _storeId = defId);

    // Load prices if there is a store
    if (defId != null) {
      await app.loadPrices(defId);
    }

    if (mounted) setState(() => _loading = false);
  }

  @override
  Widget build(BuildContext context) {
    final app = context.watch<AppProvider>();
    final items = app.prices;

    // Filter price list
    final list = _query.trim().isEmpty
        ? items
        : items
              .where(
                (p) =>
                    p.name.toLowerCase().contains(_query.toLowerCase()) ||
                    p.barcode.toLowerCase().contains(_query.toLowerCase()) ||
                    p.id.toString().contains(_query),
              )
              .toList();

    // Store name safely
    final storeName = (_storeId != null && app.stores.isNotEmpty)
        ? app.stores
              .firstWhere(
                (s) => s.id == _storeId,
                orElse: () => app.stores.first,
              )
              .name
        : '—';

    return Padding(
      padding: const EdgeInsets.all(16.0),
      child: _loading
          ? const Center(child: CircularProgressIndicator())
          : (_storeId == null)
          ? _buildNoStore(context)
          : Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                Row(
                  children: [
                    Expanded(
                      child: Text(
                        'Warehouse: $storeName (ID: ${_storeId ?? '-'} )',
                        textAlign: TextAlign.left,
                        style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                          color: AppColors.text.withValues(alpha: 0.6),
                        ),
                      ),
                    ),
                    IconButton(
                      tooltip: 'Refresh',
                      onPressed: _storeId == null
                          ? null
                          : () async {
                              setState(() => _loading = true);
                              await app.loadPrices(_storeId!);
                              if (mounted) setState(() => _loading = false);
                            },
                      icon: const Icon(Icons.refresh, color: AppColors.refresh),
                    ),
                  ],
                ),
                const SizedBox(height: 12),
                TextField(
                  textAlign: TextAlign.left,
                  decoration: const InputDecoration(
                    prefixIcon: Icon(Icons.search),
                    labelText: 'Search by Name / Barcode / Code',
                  ),
                  onChanged: (v) => setState(() => _query = v.trim()),
                ),
                const SizedBox(height: 12),
                Expanded(
                  child: list.isEmpty
                      ? Center(
                          child: Text(
                            'No matching items',
                            style: TextStyle(
                              color: AppColors.text.withValues(alpha: 0.6),
                            ),
                          ),
                        )
                      : ListView.separated(
                          itemCount: list.length,
                          separatorBuilder: (_, __) => const Divider(height: 1),
                          itemBuilder: (context, i) {
                            final TablePriceDto p = list[i];
                            return DataListItem(
                              title: p.name.isEmpty ? '(${p.id})' : p.name,
                              subtitle: 'Barcode: ${p.barcode}',
                              trailingTop: p.price.toStringAsFixed(2),
                              trailingBottom:
                                  'Stock: ${p.availableQty.toStringAsFixed(2)}',
                              trailingColor: AppColors.tabActive,
                              onTap: () {
                                // Handle item selection
                              },
                            );
                          },
                        ),
                ),
              ],
            ),
    );
  }

  Widget _buildNoStore(BuildContext context) {
    return Center(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(
            Icons.store_mall_directory,
            size: 64,
            color: AppColors.text.withValues(alpha: 0.4),
          ),
          const SizedBox(height: 12),
          Text(
            'No default warehouse selected',
            style: TextStyle(
              fontSize: 16,
              color: AppColors.text.withValues(alpha: 0.6),
            ),
          ),
          const SizedBox(height: 8),
          ElevatedButton.icon(
            onPressed: () => Navigator.pushNamed(context, '/settings'),
            icon: const Icon(Icons.settings),
            label: const Text('Select Default Warehouse'),
          ),
        ],
      ),
    );
  }
}

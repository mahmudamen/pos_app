// lib/screens/draft_invoices_screen.dart
import 'package:sales_pro/db/app_database.dart';
import 'package:sales_pro/screens/invoice_screen.dart';
import 'package:sales_pro/screens/widgets/data_list_item.dart';
import 'package:sales_pro/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:intl/intl.dart';
import 'package:provider/provider.dart';
import '../state/app_provider.dart';

class DraftInvoicesScreen extends StatelessWidget {
  const DraftInvoicesScreen({super.key});

  @override
  Widget build(BuildContext context) {
    final df = DateFormat('dd/MM/yyyy - hh:mm a', 'en');
    final app = context.read<AppProvider>();

    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 8.0, vertical: 12.0),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Expanded(
            child: StreamBuilder<List<InvoiceDraftData>>(
              stream: app.watchDraftInvoices(),
              builder: (context, snapshot) {
                if (snapshot.connectionState == ConnectionState.waiting) {
                  return const Center(child: CircularProgressIndicator());
                }

                if (snapshot.hasError) {
                  return Center(
                    child: Text(
                      'Error: ${snapshot.error}',
                      style: const TextStyle(color: Colors.red),
                    ),
                  );
                }

                final drafts = snapshot.data ?? [];

                if (drafts.isEmpty) {
                  return _buildEmptyState();
                }

                return _buildDraftsList(context, drafts, df, app);
              },
            ),
          ),
        ],
      ),
    );
  }

  // Empty state
  Widget _buildEmptyState() {
    return Center(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(Icons.note_add, size: 64, color: Colors.grey[400]),
          const SizedBox(height: 16),
          const Text(
            'No saved drafts',
            style: TextStyle(fontSize: 18, color: Colors.grey),
          ),
          const SizedBox(height: 8),
          const Text(
            'Start creating a new invoice',
            style: TextStyle(color: Colors.grey),
          ),
        ],
      ),
    );
  }

  // Drafts list
  Widget _buildDraftsList(
    BuildContext context,
    List<InvoiceDraftData> drafts,
    DateFormat df,
    AppProvider app,
  ) {
    return Scrollbar(
      thumbVisibility: true,
      child: ListView.builder(
        padding: const EdgeInsets.symmetric(horizontal: 4),
        itemCount: drafts.length,
        itemBuilder: (context, index) {
          final d = drafts[index];
          final createdAt = df.format(d.createdAt);

          return Dismissible(
            key: Key(d.id.toString()),
            direction: DismissDirection.endToStart, // Swipe from right to left
            background: Container(
              color: Colors.red,
              alignment: Alignment.centerRight,
              padding: const EdgeInsets.only(right: 20),
              child: const Icon(Icons.delete, color: Colors.white),
            ),
            confirmDismiss: (_) async {
              final confirm = await showDialog<bool>(
                context: context,
                builder: (ctx) => AlertDialog(
                  title: const Row(
                    children: [
                      Icon(Icons.warning, color: Colors.orange),
                      SizedBox(width: 8),
                      Text('Confirm Deletion'),
                    ],
                  ),
                  content: Text(
                    'Are you sure you want to delete draft "${d.customerName}"?\n\nThis cannot be undone.',
                    textAlign: TextAlign.left,
                  ),
                  actions: [
                    TextButton(
                      onPressed: () => Navigator.pop(ctx, false),
                      child: const Text('Cancel'),
                    ),
                    ElevatedButton(
                      style: ElevatedButton.styleFrom(
                        backgroundColor: Colors.red,
                      ),
                      onPressed: () => Navigator.pop(ctx, true),
                      child: const Text('Delete'),
                    ),
                  ],
                ),
              );

              if (confirm == true) {
                app.deleteInvoiceDraft(d.id);
                ScaffoldMessenger.of(
                  context,
                ).showSnackBar(const SnackBar(content: Text('Draft deleted')));
              }
              return confirm;
            },
            child: DataListItem(
              leading: null, // Can put number or icon here
              title: d.customerName,
              subtitle: df.format(d.createdAt),
              trailingTop: '${d.credit} SAR',
              trailingColor: AppColors.tabActive,
              onTap: () {
                Navigator.push(
                  context,
                  MaterialPageRoute(
                    builder: (_) => InvoiceScreen(billDraft: d),
                  ),
                );
              },
            ),
          );
        },
      ),
    );
  }

  // تأكيد الحذف
  void _confirmDelete(
    BuildContext context,
    AppProvider app,
    InvoiceDraftData draft,
  ) {
    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
        title: const Row(
          children: [
            Icon(Icons.warning, color: Colors.orange),
            SizedBox(width: 8),
            Text('Confirm Deletion'),
          ],
        ),
        content: Text(
          'Are you sure you want to delete draft "${draft.customerName}"?\n\nThis cannot be undone.',
          textAlign: TextAlign.left,
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx),
            child: const Text('Cancel'),
          ),
          ElevatedButton(
            style: ElevatedButton.styleFrom(backgroundColor: Colors.red),
            onPressed: () {
              Navigator.pop(ctx);
              app.deleteInvoiceDraft(draft.id);
              ScaffoldMessenger.of(
                context,
              ).showSnackBar(const SnackBar(content: Text('Draft deleted')));
            },
            child: const Text('Delete'),
          ),
        ],
      ),
    );
  }
}

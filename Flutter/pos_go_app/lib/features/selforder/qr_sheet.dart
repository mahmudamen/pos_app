import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:qr_flutter/qr_flutter.dart';

import '../../core/api_client.dart';
import '../../core/session_store.dart';
import '../../l10n/strings.dart';

/// Bottom sheet the cashier shows to a customer: a QR code that opens the
/// store's self-order menu in any browser. [Regenerate] re-renders the code
/// (with a fresh cache-buster so the customer always gets the newest menu).
class SelfOrderQrSheet extends StatefulWidget {
  const SelfOrderQrSheet({
    super.key,
    required this.session,
    required this.apiClient,
  });

  final Session session;
  final ApiClient apiClient;

  @override
  State<SelfOrderQrSheet> createState() => _SelfOrderQrSheetState();
}

class _SelfOrderQrSheetState extends State<SelfOrderQrSheet> {
  final TextEditingController _tableController = TextEditingController();
  int _nonce = 0;

  String get _tableName => _tableController.text.trim();

  String get _url {
    final base = widget.apiClient.baseUrl;
    final table = Uri.encodeComponent(_tableName);
    final nonce = _nonce == 0 ? '' : '&v=$_nonce';
    return '$base/selforder?tenant=${widget.session.tenantId}&table=$table$nonce';
  }

  @override
  void dispose() {
    _tableController.dispose();
    super.dispose();
  }

  Future<void> _copyUrl() async {
    final s = AppStrings.of(context);
    await Clipboard.setData(ClipboardData(text: _url));
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(content: Text('${s.selfOrderQR}: ${s.done}')),
    );
  }

  @override
  Widget build(BuildContext context) {
    final s = AppStrings.of(context);
    final theme = Theme.of(context);
    return SafeArea(
      child: SingleChildScrollView(
        padding: const EdgeInsets.fromLTRB(24, 16, 24, 24),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Text(s.selfOrderQR, style: theme.textTheme.titleLarge),
            const SizedBox(height: 4),
            Text(
              s.scanToOpenMenu,
              textAlign: TextAlign.center,
              style: theme.textTheme.bodySmall,
            ),
            const SizedBox(height: 16),
            TextField(
              controller: _tableController,
              decoration: InputDecoration(
                labelText: s.tableNameOptional,
                hintText: s.tableNameHint,
                prefixIcon: const Icon(Icons.table_restaurant_outlined),
                border: const OutlineInputBorder(),
              ),
              onChanged: (_) => setState(() {}),
            ),
            const SizedBox(height: 16),
            Container(
              padding: const EdgeInsets.all(12),
              decoration: BoxDecoration(
                color: theme.colorScheme.surfaceContainerHighest,
                borderRadius: BorderRadius.circular(16),
              ),
              child: QrImageView(
                data: _url,
                version: QrVersions.auto,
                size: 220,
                backgroundColor: Colors.white,
                eyeStyle: QrEyeStyle(
                  eyeShape: QrEyeShape.square,
                  color: theme.colorScheme.onSurface,
                ),
                dataModuleStyle: QrDataModuleStyle(
                  dataModuleShape: QrDataModuleShape.square,
                  color: theme.colorScheme.onSurface,
                ),
              ),
            ),
            const SizedBox(height: 8),
            Text(
              _url,
              textAlign: TextAlign.center,
              style: theme.textTheme.bodySmall?.copyWith(
                color: theme.colorScheme.outline,
              ),
              maxLines: 2,
              overflow: TextOverflow.ellipsis,
            ),
            const SizedBox(height: 16),
            Row(
              mainAxisAlignment: MainAxisAlignment.spaceEvenly,
              children: [
                OutlinedButton.icon(
                  onPressed: () => setState(() => _nonce++),
                  icon: const Icon(Icons.qr_code_2),
                  label: Text(s.regenerate),
                ),
                FilledButton.tonalIcon(
                  onPressed: _copyUrl,
                  icon: const Icon(Icons.copy),
                  label: Text(s.done),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }
}
import 'dart:typed_data';

import 'package:flutter/material.dart';
import 'package:image_picker/image_picker.dart';

import '../../core/api_client.dart';
import '../../core/blur_detector.dart';
import '../../core/layout.dart';
import '../../core/purchases.dart';
import '../../core/session_store.dart';
import '../../l10n/strings.dart';
import 'ocr_review_screen.dart';

/// Supplier-invoice intake: capture/upload an invoice photo, get a blur sanity
/// check before the OCR round-trip consumes a metered scan, review the parsed
/// lines, and commit the purchase (stock + cost + unit). Manager-level users
/// only — the backend enforces `inventory.adjust` for scan/apply/topup.
class PurchasesScreen extends StatefulWidget {
  const PurchasesScreen({
    super.key,
    required this.session,
    required this.apiClient,
  });

  final Session session;
  final ApiClient apiClient;

  @override
  State<PurchasesScreen> createState() => _PurchasesScreenState();
}

class _PurchasesScreenState extends State<PurchasesScreen> {
  static const _blurDetector = BlurDetector();

  bool _capturing = false;
  bool _loading = true;
  bool _loadingMeter = true;
  String? _error;
  String? _meterError;
  PurchasesPage? _page;
  OcrMeterState? _meter;
  int _pageNo = 1;
  bool _scanning = false;

  @override
  void initState() {
    super.initState();
    _loadPurchases();
    _loadMeter();
  }

  Future<void> _loadPurchases({int page = 1}) async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final result =
          await widget.apiClient.listPurchases(widget.session, page: page);
      if (!mounted) return;
      setState(() {
        _page = result;
        _pageNo = page;
        _loading = false;
      });
    } catch (e) {
      if (!mounted) return;
      setState(() {
        _error = e.toString();
        _loading = false;
      });
    }
  }

  Future<void> _loadMeter() async {
    setState(() => _loadingMeter = true);
    try {
      final meter = await widget.apiClient.ocrUsage(widget.session);
      if (!mounted) return;
      setState(() {
        _meter = meter;
        _meterError = null;
        _loadingMeter = false;
      });
    } catch (e) {
      if (!mounted) return;
      setState(() {
        _meterError = e.toString();
        _loadingMeter = false;
      });
    }
  }

  Future<void> _pickAndCapture(ImageSource source) async {
    final s = AppStrings.of(context);
    final messenger = ScaffoldMessenger.of(context);
    if (_capturing) return;
    setState(() => _capturing = true);
    try {
      final picked = await ImagePicker().pickImage(
        source: source,
        maxWidth: 1600,
        imageQuality: 90,
      );
      if (picked == null) return;
      final bytes = await picked.readAsBytes();
      if (bytes.isEmpty) {
        messenger.showSnackBar(SnackBar(content: Text(s.ocrImageEmpty)));
        return;
      }
      await _blurGate(bytes, source);
    } catch (e) {
      if (!mounted) return;
      messenger.showSnackBar(SnackBar(content: Text(e.toString())));
    } finally {
      if (mounted) setState(() => _capturing = false);
    }
  }

  /// Blur sanity gate: a blurred invoice photo OCRs poorly, and every scan
  /// spends a metered window slot, so we warn before uploading and let the
  /// merchant retake or proceed anyway.
  Future<void> _blurGate(Uint8List bytes, ImageSource source) async {
    final s = AppStrings.of(context);
    BlurAssessment assessment;
    try {
      assessment = _blurDetector.assessBytes(bytes);
    } catch (_) {
      // Decode failure: do not block the flow, let the server decide.
      await _scanAndOpenReview(bytes, source);
      return;
    }
    if (!assessment.blurred) {
      await _scanAndOpenReview(bytes, source);
      return;
    }
    if (!mounted) return;
    final proceed = await showDialog<bool>(
      context: context,
      builder: (dialogContext) => AlertDialog(
        title: Text(s.ocrBlurredTitle),
        content: Text(
          '${s.ocrBlurredBody}\n\n${s.ocrBlurredScore} ${assessment.score.toStringAsFixed(0)}',
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(dialogContext).pop(false),
            child: Text(s.retry),
          ),
          FilledButton(
            onPressed: () => Navigator.of(dialogContext).pop(true),
            child: Text(s.continueAnyway),
          ),
        ],
      ),
    );
    if (proceed == true) {
      await _scanAndOpenReview(bytes, source);
    }
  }

  Future<void> _scanAndOpenReview(Uint8List bytes, ImageSource source) async {
    final messenger = ScaffoldMessenger.of(context);
    setState(() => _scanning = true);
    try {
      final result = await widget.apiClient
          .ocrScanInvoice(widget.session, bytes, filename: 'invoice.jpg');
      if (!mounted) return;
      await Navigator.of(context).push(
        MaterialPageRoute(
          builder: (_) => OcrReviewScreen(
            session: widget.session,
            apiClient: widget.apiClient,
            scan: result,
          ),
        ),
      );
      if (mounted) {
        _loadPurchases();
        _loadMeter();
      }
    } catch (e) {
      if (!mounted) return;
      messenger.showSnackBar(SnackBar(content: Text(e.toString())));
    } finally {
      if (mounted) setState(() => _scanning = false);
    }
  }

  Future<void> _topup() async {
    final s = AppStrings.of(context);
    final messenger = ScaffoldMessenger.of(context);
    final controller = TextEditingController();
    final points = await showModalBottomSheet<int>(
      context: context,
      isScrollControlled: true,
      builder: (sheetContext) => Padding(
        padding: EdgeInsets.only(
          left: 24,
          right: 24,
          top: 24,
          bottom: MediaQuery.of(context).viewInsets.bottom + 24,
        ),
        child: SafeArea(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Text(s.ocrTopupTitle, style: Theme.of(context).textTheme.titleLarge),
              const SizedBox(height: 16),
              TextField(
                controller: controller,
                keyboardType: TextInputType.number,
                decoration: InputDecoration(labelText: s.ocrTopupPoints),
              ),
              const SizedBox(height: 16),
              FilledButton(
                onPressed: () {
                  final v = int.tryParse(controller.text.trim()) ?? 0;
                  if (v > 0) Navigator.of(sheetContext).pop(v);
                },
                child: Text(s.topUp),
              ),
            ],
          ),
        ),
      ),
    );
    controller.dispose();
    if (points == null) return;
    try {
      final balance = await widget.apiClient.ocrTopup(widget.session, points);
      if (!mounted) return;
      messenger.showSnackBar(SnackBar(
          content: Text('${s.ocrTopupDone}: $balance')));
      _loadMeter();
    } catch (e) {
      if (!mounted) return;
      messenger.showSnackBar(SnackBar(content: Text(e.toString())));
    }
  }

  @override
  Widget build(BuildContext context) {
    final s = AppStrings.of(context);
    final currency = widget.session.currencyCode;
    return Scaffold(
      appBar: AppBar(title: Text(s.purchases)),
      body: MaxWidthBox(
        child: RefreshIndicator(
          onRefresh: () async {
            await _loadPurchases();
            await _loadMeter();
          },
          child: ListView(
            padding: const EdgeInsets.all(16),
            children: [
              _MeterCard(
                meter: _meter,
                loading: _loadingMeter,
                error: _meterError,
                currency: currency,
                strings: s,
                onTopup: widget.session.isManagerLevel ? _topup : null,
                onRetry: _loadMeter,
              ),
              const SizedBox(height: 12),
              Row(
                children: [
                  Expanded(
                    child: FilledButton.icon(
                      onPressed: _capturing || _scanning
                          ? null
                          : () => _pickAndCapture(ImageSource.camera),
                      icon: _capturing
                          ? const SizedBox(
                              width: 16,
                              height: 16,
                              child: CircularProgressIndicator(strokeWidth: 2),
                            )
                          : const Icon(Icons.photo_camera_outlined),
                      label: Text(s.takeInvoicePhoto),
                    ),
                  ),
                  const SizedBox(width: 12),
                  Expanded(
                    child: OutlinedButton.icon(
                      onPressed: _capturing || _scanning
                          ? null
                          : () => _pickAndCapture(ImageSource.gallery),
                      icon: const Icon(Icons.photo_library_outlined),
                      label: Text(s.uploadInvoicePhoto),
                    ),
                  ),
                ],
              ),
              if (_scanning) ...[
                const SizedBox(height: 12),
                Center(child: Text(s.scanningInvoice)),
              ],
              const SizedBox(height: 20),
              _PurchaseListSection(
                page: _page,
                loading: _loading,
                error: _error,
                currency: currency,
                strings: s,
                onRetry: () => _loadPurchases(page: _pageNo),
                onPrev: _pageNo > 1
                    ? () => _loadPurchases(page: _pageNo - 1)
                    : null,
                onNext: (_page != null && _pageNo < _page!.totalPages)
                    ? () => _loadPurchases(page: _pageNo + 1)
                    : null,
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _MeterCard extends StatelessWidget {
  const _MeterCard({
    required this.meter,
    required this.loading,
    required this.error,
    required this.currency,
    required this.strings,
    required this.onTopup,
    required this.onRetry,
  });

  final OcrMeterState? meter;
  final bool loading;
  final String? error;
  final String currency;
  final AppStrings strings;
  final VoidCallback? onTopup;
  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    final m = AppStrings.of(context);
    if (loading) {
      return const Card(
        child: Padding(
          padding: EdgeInsets.all(24),
          child: Center(child: CircularProgressIndicator()),
        ),
      );
    }
    final meter = this.meter;
    if (error != null || meter == null) {
      return Card(
        child: Padding(
          padding: const EdgeInsets.all(16),
          child: Row(
            children: [
              Expanded(
                child: Text(
                  error ?? strings.ocrUsageUnavailable,
                  style: TextStyle(color: Theme.of(context).colorScheme.error),
                ),
              ),
              TextButton(onPressed: onRetry, child: Text(strings.retry)),
            ],
          ),
        ),
      );
    }
    final used = meter.used;
    final limits = meter.limits;
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Icon(Icons.document_scanner_outlined,
                    color: Theme.of(context).colorScheme.primary),
                const SizedBox(width: 8),
                Expanded(
                  child: Text(strings.ocrUsageTitle,
                      style: Theme.of(context).textTheme.titleMedium),
                ),
                if (onTopup != null)
                  OutlinedButton.icon(
                    onPressed: onTopup,
                    icon: const Icon(Icons.add_card_outlined, size: 16),
                    label: Text(strings.topUp),
                  ),
              ],
            ),
            const SizedBox(height: 12),
            if (meter.metered)
              Wrap(
                spacing: 16,
                runSpacing: 8,
                children: [
                  _WindowStat(strings.ocrWindowDay, used.day, limits.day, m),
                  _WindowStat(strings.ocrWindowWeek, used.week, limits.week, m),
                  _WindowStat(
                      strings.ocrWindowMonth, used.month, limits.month, m),
                ],
              )
            else
              Text(strings.ocrUnmetered),
            const SizedBox(height: 8),
            Text(
              '${strings.ocrCredits}: ${meter.creditsRemaining}',
              style: Theme.of(context).textTheme.bodyMedium,
            ),
          ],
        ),
      ),
    );
  }
}

class _WindowStat extends StatelessWidget {
  const _WindowStat(this.label, this.used, this.limit, this.strings);

  final String label;
  final int used;
  final int limit;
  final AppStrings strings;

  @override
  Widget build(BuildContext context) {
    final over = limit > 0 && used >= limit;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(label, style: Theme.of(context).textTheme.bodySmall),
        Text(
          '$used${limit > 0 ? ' / $limit' : ''}',
          style: Theme.of(context).textTheme.titleMedium?.copyWith(
                color: over
                    ? Theme.of(context).colorScheme.error
                    : Theme.of(context).colorScheme.onSurface,
                fontWeight: FontWeight.bold,
              ),
        ),
      ],
    );
  }
}

class _PurchaseListSection extends StatelessWidget {
  const _PurchaseListSection({
    required this.page,
    required this.loading,
    required this.error,
    required this.currency,
    required this.strings,
    required this.onRetry,
    required this.onPrev,
    required this.onNext,
  });

  final PurchasesPage? page;
  final bool loading;
  final String? error;
  final String currency;
  final AppStrings strings;
  final VoidCallback onRetry;
  final VoidCallback? onPrev;
  final VoidCallback? onNext;

  @override
  Widget build(BuildContext context) {
    final m = AppStrings.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(strings.purchaseHistory,
            style: Theme.of(context).textTheme.titleMedium),
        const SizedBox(height: 8),
        if (loading)
          const Center(
            child: Padding(
              padding: EdgeInsets.all(24),
              child: CircularProgressIndicator(),
            ),
          )
        else if (error != null)
          Card(
            child: Padding(
              padding: const EdgeInsets.all(16),
              child: Row(
                children: [
                  Expanded(child: Text(error!)),
                  TextButton(onPressed: onRetry, child: Text(strings.retry)),
                ],
              ),
            ),
          )
        else if (page == null || page!.purchases.isEmpty)
          Card(
            child: Padding(
              padding: const EdgeInsets.all(24),
              child: Center(child: Text(strings.noPurchasesYet)),
            ),
          )
        else
          ...page!.purchases.map(
            (p) => Card(
              margin: const EdgeInsets.only(bottom: 8),
              child: ListTile(
                leading: const CircleAvatar(
                  child: Icon(Icons.receipt_long_outlined),
                ),
                title: Text(
                  p.supplier.isNotEmpty ? p.supplier : '—',
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                ),
                subtitle: Text(
                  '${p.invoiceNo.isNotEmpty ? p.invoiceNo : '—'}\n${m.formatDate(_parseDate(p.createdAt))}',
                  maxLines: 2,
                  overflow: TextOverflow.ellipsis,
                ),
                trailing: Column(
                  mainAxisAlignment: MainAxisAlignment.center,
                  crossAxisAlignment: CrossAxisAlignment.end,
                  children: [
                    Text(m.formatMoney(p.totalMinor, currency),
                        style: const TextStyle(fontWeight: FontWeight.bold)),
                    Text(
                      '${p.itemCount} ${strings.items}',
                      style: Theme.of(context).textTheme.bodySmall,
                    ),
                  ],
                ),
              ),
            ),
          ),
        if (page != null && (onPrev != null || onNext != null))
          Padding(
            padding: const EdgeInsets.symmetric(vertical: 8),
            child: Row(
              mainAxisAlignment: MainAxisAlignment.spaceBetween,
              children: [
                TextButton(
                  onPressed: onPrev,
                  onLongPress: onPrev,
                  child: Text(strings.previous),
                ),
                Text('${page!.page} / ${page!.totalPages}'),
                TextButton(onPressed: onNext, child: Text(strings.next)),
              ],
            ),
          ),
      ],
    );
  }
}

DateTime _parseDate(String value) {
  final d = DateTime.tryParse(value.replaceFirst(' ', 'T'));
  return d ?? DateTime.fromMillisecondsSinceEpoch(0);
}
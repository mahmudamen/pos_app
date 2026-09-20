import 'package:flutter/material.dart';

import '../../core/api_client.dart';
import '../../core/layout.dart';
import '../../core/selforder.dart';
import '../../core/session_store.dart';
import '../../l10n/strings.dart';

/// Cashier queue of customer self-orders: pending orders are approved (turns
/// them into a real sale) or cancelled. Status filter chips + pagination.
class SelfOrdersScreen extends StatefulWidget {
  const SelfOrdersScreen({
    super.key,
    required this.session,
    required this.apiClient,
  });

  final Session session;
  final ApiClient apiClient;

  @override
  State<SelfOrdersScreen> createState() => _SelfOrdersScreenState();
}

class _SelfOrdersScreenState extends State<SelfOrdersScreen> {
  static const _pageSize = 50;
  String? _status;
  int _page = 1;
  int _total = 0;
  bool _loading = true;
  String? _error;
  List<SelfOrder> _orders = [];
  bool _busy = false;

  @override
  void initState() {
    super.initState();
    _load(page: 1);
  }

  Future<void> _load({required int page}) async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final result = await widget.apiClient.selfOrders(
        widget.session,
        status: _status,
        page: page,
        limit: _pageSize,
      );
      if (!mounted) return;
      setState(() {
        _orders = result.orders;
        _total = result.total;
        _page = result.page;
        _loading = false;
      });
    } catch (e) {
      if (!mounted) return;
      setState(() {
        _loading = false;
        _error = e.toString();
      });
    }
  }

  void _setStatus(String? status) {
    setState(() => _status = status);
    _load(page: 1);
  }

  Future<void> _approve(SelfOrder order) async {
    final s = AppStrings.of(context);
    final messenger = ScaffoldMessenger.of(context);
    final navigator = Navigator.of(context);
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: Text(s.approveOrder),
        content: Text('${order.reference}\n${s.approveOrderBody}'),
        actions: [
          TextButton(
            onPressed: () => navigator.pop(false),
            child: Text(s.cancel),
          ),
          FilledButton(
            onPressed: () => navigator.pop(true),
            child: Text(s.approve),
          ),
        ],
      ),
    );
    if (confirmed != true || !mounted) return;
    setState(() => _busy = true);
    try {
      await widget.apiClient.approveSelfOrder(widget.session, order.id);
      if (!mounted) return;
      messenger.showSnackBar(SnackBar(content: Text(s.orderApproved)));
      _load(page: _page);
    } catch (e) {
      if (!mounted) return;
      messenger.showSnackBar(SnackBar(content: Text(e.toString())));
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  Future<void> _cancel(SelfOrder order) async {
    final s = AppStrings.of(context);
    final messenger = ScaffoldMessenger.of(context);
    final navigator = Navigator.of(context);
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: Text(s.cancelOrder),
        content: Text('${order.reference}\n${s.cancelOrderBody}'),
        actions: [
          TextButton(
            onPressed: () => navigator.pop(false),
            child: Text(s.cancel),
          ),
          FilledButton(
            onPressed: () => navigator.pop(true),
            child: Text(s.cancelOrder),
          ),
        ],
      ),
    );
    if (confirmed != true || !mounted) return;
    setState(() => _busy = true);
    try {
      await widget.apiClient.cancelSelfOrder(widget.session, order.id);
      if (!mounted) return;
      messenger.showSnackBar(SnackBar(content: Text(s.orderCancelled)));
      _load(page: _page);
    } catch (e) {
      if (!mounted) return;
      messenger.showSnackBar(SnackBar(content: Text(e.toString())));
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final s = AppStrings.of(context);
    final totalPages = _pageSize > 0 ? (_total / _pageSize).ceil() : 1;
    return Scaffold(
      appBar: AppBar(title: Text(s.selfOrdersQueue)),
      body: Column(
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(12, 8, 12, 4),
            child: SingleChildScrollView(
              scrollDirection: Axis.horizontal,
              child: Row(
                children: [
                  ChoiceChip(
                    label: Text(s.selfOrdersQueue),
                    selected: _status == null,
                    onSelected: (_) => _setStatus(null),
                  ),
                  const SizedBox(width: 8),
                  ChoiceChip(
                    label: Text(s.pendingLabel),
                    selected: _status == 'pending',
                    onSelected: (_) => _setStatus('pending'),
                  ),
                  const SizedBox(width: 8),
                  ChoiceChip(
                    label: Text(s.approvedLabel),
                    selected: _status == 'approved',
                    onSelected: (_) => _setStatus('approved'),
                  ),
                  const SizedBox(width: 8),
                  ChoiceChip(
                    label: Text(s.cancelledLabel),
                    selected: _status == 'cancelled',
                    onSelected: (_) => _setStatus('cancelled'),
                  ),
                ],
              ),
            ),
          ),
          Expanded(child: _buildBody(context)),
          if (totalPages > 1)
            Padding(
              padding: const EdgeInsets.only(bottom: 8),
              child: Row(
                mainAxisAlignment: MainAxisAlignment.center,
                children: [
                  IconButton(
                    onPressed: _page > 1 ? () => _load(page: _page - 1) : null,
                    icon: const Icon(Icons.chevron_left),
                  ),
                  Text(s.pageLabel(_page, totalPages)),
                  IconButton(
                    onPressed: _page < totalPages
                        ? () => _load(page: _page + 1)
                        : null,
                    icon: const Icon(Icons.chevron_right),
                  ),
                ],
              ),
            ),
        ],
      ),
    );
  }

  Widget _buildBody(BuildContext context) {
    final s = AppStrings.of(context);
    if (_loading) return const Center(child: CircularProgressIndicator());
    if (_error != null) {
      return Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Text(_error!,
                style: TextStyle(color: Theme.of(context).colorScheme.error)),
            const SizedBox(height: 12),
            FilledButton(onPressed: () => _load(page: 1), child: Text(s.retry)),
          ],
        ),
      );
    }
    if (_orders.isEmpty) return Center(child: Text(s.emptyQueue));
    final canAct = widget.session.role != 'guest';
    return MaxWidthBox(
      child: RefreshIndicator(
        onRefresh: () => _load(page: 1),
        child: ListView.separated(
          physics: const AlwaysScrollableScrollPhysics(),
          itemCount: _orders.length,
          separatorBuilder: (_, __) => const Divider(height: 1),
          itemBuilder: (context, index) => _OrderTile(
            order: _orders[index],
            s: s,
            busy: _busy,
            canAct: canAct,
            onApprove: () => _approve(_orders[index]),
            onCancel: () => _cancel(_orders[index]),
          ),
        ),
      ),
    );
  }
}

class _OrderTile extends StatelessWidget {
  const _OrderTile({
    required this.order,
    required this.s,
    required this.busy,
    required this.canAct,
    required this.onApprove,
    required this.onCancel,
  });

  final SelfOrder order;
  final AppStrings s;
  final bool busy;
  final bool canAct;
  final VoidCallback onApprove;
  final VoidCallback onCancel;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final lamp = switch (order.status) {
      'pending' => theme.colorScheme.tertiaryContainer,
      'approved' => theme.colorScheme.primaryContainer,
      _ => theme.colorScheme.surfaceContainerHighest,
    };
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Expanded(
                child: Text(order.reference,
                    style: theme.textTheme.titleMedium
                        ?.copyWith(fontWeight: FontWeight.bold)),
              ),
              Container(
                padding:
                    const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
                decoration: BoxDecoration(
                  color: lamp,
                  borderRadius: BorderRadius.circular(999),
                ),
                child: Text(
                  order.isPending
                      ? s.pendingLabel
                      : order.isApproved
                          ? s.approvedLabel
                          : s.cancelledLabel,
                  style: theme.textTheme.labelSmall,
                ),
              ),
            ],
          ),
          const SizedBox(height: 2),
          Text(
            '${s.tableLabel}: ${order.tableName.isEmpty ? '—' : order.tableName}'
            ' · ${order.customerName.isEmpty ? s.anonCustomer : order.customerName}',
            style: theme.textTheme.bodySmall,
          ),
          Text(order.createdAt, style: theme.textTheme.bodySmall),
          const SizedBox(height: 4),
          for (final item in order.items)
            Padding(
              padding: const EdgeInsets.symmetric(vertical: 1),
              child: Row(
                children: [
                  Expanded(child: Text('${item.quantity}× ${item.name}')),
                  Text(s.formatMoney(item.totalMinor, order.currency)),
                ],
              ),
            ),
          const Divider(height: 8),
          Row(
            children: [
              Expanded(
                child: Text(
                  '${s.total}: ${s.formatMoney(order.totalMinor, order.currency)}',
                  style: theme.textTheme.titleSmall
                      ?.copyWith(fontWeight: FontWeight.bold),
                ),
              ),
              if (order.isPending && canAct) ...[
                IconButton(
                  onPressed: busy ? null : onCancel,
                  tooltip: s.cancelOrder,
                  icon: const Icon(Icons.close),
                ),
                FilledButton.icon(
                  onPressed: busy ? null : onApprove,
                  icon: const Icon(Icons.check),
                  label: Text(s.approve),
                ),
              ],
            ],
          ),
        ],
      ),
    );
  }
}
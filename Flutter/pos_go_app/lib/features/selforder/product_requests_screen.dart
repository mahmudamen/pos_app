import 'package:flutter/material.dart';

import '../../core/api_client.dart';
import '../../core/layout.dart';
import '../../core/selforder.dart';
import '../../core/session_store.dart';
import '../../l10n/strings.dart';

/// Product-request inbox: customers' feedback messages (what's missing from
/// the menu), each optionally carrying a web-push subscription for a
/// back-in-stock notification. Staff fulfills or closes requests.
class ProductRequestsScreen extends StatefulWidget {
  const ProductRequestsScreen({
    super.key,
    required this.session,
    required this.apiClient,
  });

  final Session session;
  final ApiClient apiClient;

  @override
  State<ProductRequestsScreen> createState() => _ProductRequestsScreenState();
}

class _ProductRequestsScreenState extends State<ProductRequestsScreen> {
  static const _pageSize = 50;
  String? _status;
  int _page = 1;
  int _total = 0;
  bool _loading = true;
  String? _error;
  List<ProductRequest> _requests = [];
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
      final result = await widget.apiClient.productRequests(
        widget.session,
        status: _status,
        page: page,
        limit: _pageSize,
      );
      if (!mounted) return;
      setState(() {
        _requests = result.requests;
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

  Future<void> _run(
      BuildContext context, String action, Future<void> Function() call) async {
    final s = AppStrings.of(context);
    final messenger = ScaffoldMessenger.of(context);
    setState(() => _busy = true);
    try {
      await call();
      if (!mounted) return;
      messenger.showSnackBar(
        SnackBar(content: Text(action == 'fulfill' ? s.requestFulfilled : s.requestClosed)),
      );
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
      appBar: AppBar(title: Text(s.productRequests)),
      body: Column(
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(12, 8, 12, 4),
            child: SingleChildScrollView(
              scrollDirection: Axis.horizontal,
              child: Row(
                children: [
                  ChoiceChip(
                    label: Text(s.openLabel),
                    selected: _status == 'open' || _status == null,
                    onSelected: (_) => _setStatus(_status == 'open' ? null : 'open'),
                  ),
                  const SizedBox(width: 8),
                  ChoiceChip(
                    label: Text(s.fulfilledLabel),
                    selected: _status == 'fulfilled',
                    onSelected: (_) => _setStatus('fulfilled'),
                  ),
                  const SizedBox(width: 8),
                  ChoiceChip(
                    label: Text(s.closedLabel),
                    selected: _status == 'closed',
                    onSelected: (_) => _setStatus('closed'),
                  ),
                  const SizedBox(width: 8),
                  ChoiceChip(
                    label: Text(s.all),
                    selected: _status == null,
                    onSelected: (_) => _setStatus(null),
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
    if (_requests.isEmpty) return Center(child: Text(s.emptyRequests));
    final canManage = widget.session.isManagerLevel;
    return MaxWidthBox(
      child: RefreshIndicator(
        onRefresh: () => _load(page: 1),
        child: ListView.separated(
          physics: const AlwaysScrollableScrollPhysics(),
          itemCount: _requests.length,
          separatorBuilder: (_, __) => const Divider(height: 1),
          itemBuilder: (context, index) {
            final request = _requests[index];
            final s = AppStrings.of(context);
            final theme = Theme.of(context);
            return ListTile(
              leading: Icon(
                request.hasWebPush ? Icons.notifications_active_outlined : Icons.feedback_outlined,
                color: request.hasWebPush
                    ? theme.colorScheme.primary
                    : theme.colorScheme.outline,
              ),
              title: Text(request.productName),
              subtitle: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  if (request.note.isNotEmpty)
                    Text('${s.requestNote}: ${request.note}'),
                  if (request.contact.isNotEmpty)
                    Text('${s.requestContact}: ${request.contact}'),
                  Text(request.createdAt),
                  if (request.hasWebPush)
                    Text(
                      s.wantsNotification,
                      style: theme.textTheme.labelSmall
                          ?.copyWith(color: theme.colorScheme.primary),
                    ),
                ],
              ),
              isThreeLine: true,
              trailing: request.isOpen && canManage
                  ? Row(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        IconButton(
                          onPressed: _busy
                              ? null
                              : () => _run(
                                    context,
                                    'fulfill',
                                    () => widget.apiClient.fulfillProductRequest(
                                        widget.session, request.id),
                                  ),
                          tooltip: s.fulfillRequest,
                          icon: const Icon(Icons.check_circle_outline),
                        ),
                        IconButton(
                          onPressed: _busy
                              ? null
                              : () => _run(
                                    context,
                                    'close',
                                    () => widget.apiClient.closeProductRequest(
                                        widget.session, request.id),
                                  ),
                          tooltip: s.closeRequest,
                          icon: const Icon(Icons.remove_circle_outline),
                        ),
                      ],
                    )
                  : Text(
                      request.isOpen
                          ? s.openLabel
                          : request.isFulfilled
                              ? s.fulfilledLabel
                              : s.closedLabel,
                      style: theme.textTheme.labelMedium,
                    ),
            );
          },
        ),
      ),
    );
  }
}
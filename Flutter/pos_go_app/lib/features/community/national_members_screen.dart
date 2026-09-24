import 'package:flutter/material.dart';

import '../../core/api_client.dart';
import '../../core/community.dart';
import '../../core/layout.dart';
import '../../core/session_store.dart';
import '../../l10n/strings.dart';
import 'national_labels.dart';

/// Directory of national community members, searchable + level-filtered.
/// When `canModerate` is true the detail view exposes moderation actions.
class NationalMembersScreen extends StatefulWidget {
  const NationalMembersScreen({
    super.key,
    required this.session,
    required this.apiClient,
    this.canModerate = false,
    this.myUserId = '',
  });

  final Session session;
  final ApiClient apiClient;
  final bool canModerate;
  final String myUserId;

  @override
  State<NationalMembersScreen> createState() => _NationalMembersScreenState();
}

class _NationalMembersScreenState extends State<NationalMembersScreen> {
  NationalMembersPage? _page;
  bool _loading = true;
  String? _error;
  int _currentPage = 1;
  String _query = '';
  String _level = '';
  static const _levels = ['', 'bronze', 'silver', 'gold', 'platinum'];

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load({int page = 1}) async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final result = await widget.apiClient.listNationalMembers(
        widget.session,
        query: _query,
        level: _level,
        page: page,
      );
      if (!mounted) return;
      setState(() {
        _page = result;
        _currentPage = page;
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

  /// Moderator quick-action from the directory: confirm, call the API, toast,
  /// reload the current page.
  Future<void> _moderate(
      BuildContext context, NationalMember member, String action) async {
    final s = AppStrings.of(context);
    final message = switch (action) {
      'promote' => s.promoteModerator,
      'demote' => s.demoteModerator,
      'suspend' => s.suspendMember,
      _ => s.restoreMember,
    };
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(s.confirmAction),
        content: Text('$message? · ${member.displayName}'),
        actions: [
          TextButton(
              onPressed: () => Navigator.pop(ctx, false),
              child: Text(s.cancel)),
          FilledButton(
              onPressed: () => Navigator.pop(ctx, true), child: Text(s.ok)),
        ],
      ),
    );
    if (confirmed != true) return;
    if (!context.mounted) return;
    try {
      await widget.apiClient.updateNationalMember(
        widget.session,
        member.userId,
        status: action == 'suspend'
            ? 'suspended'
            : action == 'restore'
                ? 'active'
                : null,
        role: action == 'promote'
            ? 'moderator'
            : action == 'demote'
                ? 'member'
                : null,
      );
      if (!context.mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(
        content: Text(s.memberUpdated),
        behavior: SnackBarBehavior.floating,
      ));
      _load(page: _currentPage);
    } catch (_) {
      if (!context.mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(
        content: Text(s.actionFailed),
        behavior: SnackBarBehavior.floating,
      ));
    }
  }

  @override
  Widget build(BuildContext context) {
    final s = AppStrings.of(context);
    return Scaffold(
      appBar: AppBar(title: Text(s.nationalDirectory)),
      body: MaxWidthBox(
        child: Column(
          children: [
            Padding(
              padding: const EdgeInsets.all(8),
              child: Row(
                children: [
                  Expanded(
                    child: TextField(
                      decoration: InputDecoration(
                        hintText: s.searchMembers,
                        prefixIcon: const Icon(Icons.search),
                        border: const OutlineInputBorder(),
                        isDense: true,
                      ),
                      onSubmitted: (value) {
                        _query = value.trim();
                        _load();
                      },
                    ),
                  ),
                  const SizedBox(width: 8),
                  PopupMenuButton<String>(
                    initialValue: _level,
                    tooltip: s.membershipLevel,
                    onSelected: (level) {
                      setState(() => _level = level);
                      _load();
                    },
                    itemBuilder: (_) => [
                      for (final level in _levels)
                        CheckedPopupMenuItem(
                          value: level,
                          checked: _level == level,
                          child: Text(level.isEmpty
                              ? s.allItems
                              : nationalLevelLabel(s, level)),
                        ),
                    ],
                    child: const Padding(
                      padding: EdgeInsets.all(8),
                      child: Icon(Icons.filter_list),
                    ),
                  ),
                ],
              ),
            ),
            Expanded(
              child: _loading
                  ? const Center(child: CircularProgressIndicator())
                  : _error != null
                      ? Center(
                          child: Column(
                            mainAxisSize: MainAxisSize.min,
                            children: [
                              Text(_error!,
                                  style: TextStyle(
                                      color:
                                          Theme.of(context).colorScheme.error)),
                              const SizedBox(height: 12),
                              FilledButton(
                                  onPressed: () => _load(page: _currentPage),
                                  child: Text(s.retry)),
                            ],
                          ),
                        )
                      : _buildList(context, s),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildList(BuildContext context, AppStrings s) {
    final members = _page?.members ?? [];
    final total = _page?.total ?? 0;
    final totalPages =
        (_page?.limit ?? 50) > 0 ? (total / (_page?.limit ?? 50)).ceil() : 1;

    return Column(
      children: [
        if (total > 0)
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 4, 16, 8),
            child: Align(
              alignment: AlignmentDirectional.centerStart,
              child: Text('${s.membersLabel}: $total',
                  style: Theme.of(context).textTheme.labelMedium?.copyWith(
                        color: Theme.of(context).colorScheme.primary,
                      )),
            ),
          ),
        Expanded(
          child: members.isEmpty
              ? Center(
                  child: Column(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      Icon(Icons.group_outlined,
                          size: 48,
                          color: Theme.of(context).colorScheme.outline),
                      const SizedBox(height: 12),
                      Text(s.noMembers),
                    ],
                  ),
                )
              : RefreshIndicator(
                  onRefresh: () => _load(page: _currentPage),
                  child: ListView.separated(
                    physics: const AlwaysScrollableScrollPhysics(),
                    itemCount: members.length,
                    separatorBuilder: (_, __) => const Divider(height: 1),
                    itemBuilder: (context, index) {
                      final member = members[index];
                      return ListTile(
                        leading: CircleAvatar(
                          child: Text(member.displayName.isNotEmpty
                              ? member.displayName[0].toUpperCase()
                              : '?'),
                        ),
                        title: Row(
                          mainAxisSize: MainAxisSize.min,
                          children: [
                            Flexible(child: Text(member.displayName)),
                            if (member.isModerator) ...[
                              const SizedBox(width: 6),
                              Icon(Icons.shield_outlined,
                                  size: 16,
                                  color: Theme.of(context).colorScheme.primary),
                            ],
                          ],
                        ),
                        subtitle: Text(nationalLevelLabel(s, member.level)),
                        trailing: Row(
                          mainAxisSize: MainAxisSize.min,
                          children: [
                            Text(
                              nationalStatusLabel(s, member.status),
                              style: member.isActive
                                  ? null
                                  : TextStyle(
                                      color:
                                          Theme.of(context).colorScheme.error),
                            ),
                            if (widget.canModerate &&
                                member.userId != widget.myUserId)
                              PopupMenuButton<String>(
                                tooltip: s.confirmAction,
                                onSelected: (action) =>
                                    _moderate(context, member, action),
                                itemBuilder: (_) => [
                                  if (member.isModerator)
                                    PopupMenuItem(
                                        value: 'demote',
                                        child: Text(s.demoteModerator))
                                  else
                                    PopupMenuItem(
                                        value: 'promote',
                                        child: Text(s.promoteModerator)),
                                  if (member.isActive)
                                    PopupMenuItem(
                                        value: 'suspend',
                                        child: Text(s.suspendMember))
                                  else
                                    PopupMenuItem(
                                        value: 'restore',
                                        child: Text(s.restoreMember)),
                                ],
                              ),
                          ],
                        ),
                        onTap: () => Navigator.of(context)
                            .push(
                          MaterialPageRoute<bool>(
                            builder: (_) => NationalMemberDetailScreen(
                              session: widget.session,
                              apiClient: widget.apiClient,
                              member: member,
                              canModerate: widget.canModerate,
                              myUserId: widget.myUserId,
                            ),
                          ),
                        )
                            .then((changed) {
                          if (changed == true) _load(page: _currentPage);
                        }),
                      );
                    },
                  ),
                ),
        ),
        if (totalPages > 1)
          Padding(
            padding: const EdgeInsets.all(8.0),
            child: Row(
              mainAxisAlignment: MainAxisAlignment.center,
              children: [
                IconButton(
                  onPressed: _currentPage > 1
                      ? () => _load(page: _currentPage - 1)
                      : null,
                  icon: const Icon(Icons.chevron_left),
                ),
                Text('${s.pageOf} $_currentPage / $totalPages'),
                IconButton(
                  onPressed: _currentPage < totalPages
                      ? () => _load(page: _currentPage + 1)
                      : null,
                  icon: const Icon(Icons.chevron_right),
                ),
              ],
            ),
          ),
      ],
    );
  }
}

class NationalMemberDetailScreen extends StatefulWidget {
  const NationalMemberDetailScreen({
    super.key,
    required this.session,
    required this.apiClient,
    required this.member,
    this.canModerate = false,
    this.myUserId = '',
  });

  final Session session;
  final ApiClient apiClient;
  final NationalMember member;
  final bool canModerate;
  final String myUserId;

  @override
  State<NationalMemberDetailScreen> createState() =>
      _NationalMemberDetailScreenState();
}

class _NationalMemberDetailScreenState
    extends State<NationalMemberDetailScreen> {
  bool _busy = false;
  String? _error;

  Future<bool> _update({String? status, String? role}) async {
    final s = AppStrings.of(context);
    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      await widget.apiClient.updateNationalMember(
        widget.session,
        widget.member.userId,
        status: status,
        role: role,
      );
      if (!mounted) return false;
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(s.memberUpdated)));
      return true;
    } catch (e) {
      if (!mounted) return false;
      final ApiException api = e is ApiException ? e : const ApiException('');
      if (api.code == 'permission_denied') {
        setState(() => _error = s.permissionDenied);
      } else {
        setState(() => _error = e.toString());
      }
      return false;
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  /// Confirmation gate for moderation actions.
  Future<bool> _confirm(String message) async {
    final s = AppStrings.of(context);
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(s.confirmAction),
        content: Text(message),
        actions: [
          TextButton(
              onPressed: () => Navigator.pop(ctx, false),
              child: Text(s.cancel)),
          FilledButton(
              onPressed: () => Navigator.pop(ctx, true), child: Text(s.ok)),
        ],
      ),
    );
    return ok == true;
  }

  @override
  Widget build(BuildContext context) {
    final s = AppStrings.of(context);
    final member = widget.member;
    final isSelf = member.userId == widget.myUserId;
    return Scaffold(
      appBar: AppBar(title: Text(member.displayName)),
      body: MaxWidthBox(
        child: ListView(
          padding: const EdgeInsets.all(16),
          children: [
            Row(
              children: [
                CircleAvatar(
                  radius: 32,
                  child: Text(member.displayName.isNotEmpty
                      ? member.displayName[0].toUpperCase()
                      : '?'),
                ),
                const SizedBox(width: 16),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(member.displayName,
                          style: Theme.of(context).textTheme.titleLarge),
                      const SizedBox(height: 4),
                      Chip(
                        visualDensity: VisualDensity.compact,
                        label: Text(nationalLevelLabel(s, member.level)),
                      ),
                    ],
                  ),
                ),
              ],
            ),
            const SizedBox(height: 16),
            Wrap(
              spacing: 12,
              runSpacing: 12,
              children: [
                _InfoChip(
                  icon: Icons.military_tech_outlined,
                  label: '${s.expertiseScore}: ${member.expertiseScore}',
                ),
                _InfoChip(
                  icon: Icons.verified_user_outlined,
                  label: nationalStatusLabel(s, member.status),
                ),
                _InfoChip(
                  icon: Icons.shield_outlined,
                  label: nationalRoleLabel(s, member.role),
                ),
                if (member.joinedVia.isNotEmpty)
                  _InfoChip(
                    icon: Icons.route_outlined,
                    label: '${s.joinedViaLabel}: ${member.joinedVia}',
                  ),
                if (member.invitedBy.isNotEmpty)
                  _InfoChip(
                    icon: Icons.person_outline,
                    label: '${s.invitedByLabel}: ${member.invitedBy}',
                  ),
                if (member.joinedAt.isNotEmpty)
                  _InfoChip(
                    icon: Icons.event_outlined,
                    label: member.joinedAt,
                  ),
              ],
            ),
            if (_error != null) ...[
              const SizedBox(height: 16),
              Text(_error!,
                  style: TextStyle(color: Theme.of(context).colorScheme.error)),
            ],
            if (widget.canModerate && !isSelf) ...[
              const Divider(height: 32),
              Text(s.nationalMembership.toUpperCase(),
                  style: Theme.of(context).textTheme.labelSmall),
              const SizedBox(height: 8),
              Wrap(
                spacing: 8,
                runSpacing: 8,
                children: [
                  if (member.isModerator)
                    OutlinedButton.icon(
                      onPressed: _busy
                          ? null
                          : () async {
                              if (!await _confirm(
                                  '${s.demoteModerator}? · ${member.displayName}')) {
                                return;
                              }
                              final ok = await _update(
                                  role: member.role == 'admin'
                                      ? 'member'
                                      : 'moderator');
                              if (ok && context.mounted) {
                                Navigator.of(context).pop(true);
                              }
                            },
                      icon: const Icon(Icons.undo),
                      label: Text(s.demoteModerator),
                    )
                  else
                    FilledButton.icon(
                      onPressed: _busy
                          ? null
                          : () async {
                              if (!await _confirm(
                                  '${s.promoteModerator}? · ${member.displayName}')) {
                                return;
                              }
                              final ok = await _update(role: 'moderator');
                              if (ok && context.mounted) {
                                Navigator.of(context).pop(true);
                              }
                            },
                      icon: const Icon(Icons.verified_outlined),
                      label: Text(s.promoteModerator),
                    ),
                  if (member.isActive)
                    OutlinedButton.icon(
                      onPressed: _busy
                          ? null
                          : () async {
                              if (!await _confirm(
                                  '${s.suspendMember}? · ${member.displayName}')) {
                                return;
                              }
                              final ok = await _update(status: 'suspended');
                              if (ok && context.mounted) {
                                Navigator.of(context).pop(true);
                              }
                            },
                      icon: const Icon(Icons.block_outlined),
                      label: Text(s.suspendMember),
                    )
                  else
                    OutlinedButton.icon(
                      onPressed: _busy
                          ? null
                          : () async {
                              if (!await _confirm(
                                  '${s.restoreMember}? · ${member.displayName}')) {
                                return;
                              }
                              final ok = await _update(status: 'active');
                              if (ok && context.mounted) {
                                Navigator.of(context).pop(true);
                              }
                            },
                      icon: const Icon(Icons.lock_open_outlined),
                      label: Text(s.restoreMember),
                    ),
                ],
              ),
            ],
          ],
        ),
      ),
    );
  }
}

class _InfoChip extends StatelessWidget {
  const _InfoChip({required this.icon, required this.label});

  final IconData icon;
  final String label;

  @override
  Widget build(BuildContext context) {
    return Chip(
      avatar: Icon(icon, size: 16),
      label: Text(label),
    );
  }
}

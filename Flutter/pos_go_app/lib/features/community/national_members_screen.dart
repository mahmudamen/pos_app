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
    final (title, body, label, destructive) = switch (action) {
      'promote' => (
          s.confirmPromote,
          s.confirmPromoteBody,
          s.promoteModerator,
          false,
        ),
      'demote' => (
          s.confirmDemote,
          s.confirmDemoteBody,
          s.demoteModerator,
          false,
        ),
      'suspend' => (
          s.confirmSuspend,
          s.confirmSuspendBody,
          s.suspendMember,
          true,
        ),
      _ => (
          s.confirmRestore,
          s.confirmRestoreBody,
          s.restoreMember,
          false,
        ),
    };
    final confirmed = await _confirmModeration(
      context,
      title: title,
      body: body,
      memberName: member.displayName,
      confirmLabel: label,
      destructive: destructive,
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

  Future<void> _openDetail(NationalMember member) async {
    final changed = await Navigator.of(context).push<bool>(
      MaterialPageRoute<bool>(
        builder: (_) => NationalMemberDetailScreen(
          session: widget.session,
          apiClient: widget.apiClient,
          member: member,
          canModerate: widget.canModerate,
          myUserId: widget.myUserId,
        ),
      ),
    );
    if (changed == true && mounted) _load(page: _currentPage);
  }

  @override
  Widget build(BuildContext context) {
    final s = AppStrings.of(context);
    final scheme = Theme.of(context).colorScheme;
    return Scaffold(
      appBar: AppBar(title: Text(s.nationalDirectory)),
      body: MaxWidthBox(
        child: Column(
          children: [
            Padding(
              padding: const EdgeInsets.fromLTRB(16, 8, 16, 12),
              child: Row(
                children: [
                  Expanded(
                    child: TextField(
                      decoration: InputDecoration(
                        hintText: s.searchMembers,
                        prefixIcon: const Icon(Icons.search, size: 20),
                        filled: true,
                        fillColor: scheme.surfaceContainerHigh,
                        isDense: true,
                        contentPadding: const EdgeInsets.symmetric(
                            horizontal: 16, vertical: 14),
                        border: OutlineInputBorder(
                          borderRadius: BorderRadius.circular(26),
                          borderSide: BorderSide.none,
                        ),
                        enabledBorder: OutlineInputBorder(
                          borderRadius: BorderRadius.circular(26),
                          borderSide: BorderSide.none,
                        ),
                        focusedBorder: OutlineInputBorder(
                          borderRadius: BorderRadius.circular(26),
                          borderSide:
                              BorderSide(color: scheme.outline, width: 1.2),
                        ),
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
                    child: Container(
                      width: 46,
                      height: 46,
                      alignment: Alignment.center,
                      decoration: BoxDecoration(
                        color: _level.isEmpty
                            ? scheme.surfaceContainerHigh
                            : scheme.primaryContainer,
                        shape: BoxShape.circle,
                      ),
                      child: Icon(
                        Icons.filter_list,
                        size: 22,
                        color: _level.isEmpty
                            ? scheme.onSurfaceVariant
                            : scheme.onPrimaryContainer,
                      ),
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
                                  style:
                                      TextStyle(color: scheme.error)),
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
    final limit = _page?.limit ?? 50;
    final totalPages = limit > 0 ? (total / limit).ceil() : 1;

    return Column(
      children: [
        Expanded(
          child: members.isEmpty
              ? _EmptyMembers(
                  hasFilter: _query.isNotEmpty || _level.isNotEmpty)
              : RefreshIndicator(
                  onRefresh: () => _load(page: _currentPage),
                  child: _communityCard(
                    context,
                    margin: const EdgeInsets.fromLTRB(12, 0, 12, 8),
                    child: Column(
                      children: [
                        Padding(
                          padding: const EdgeInsets.fromLTRB(16, 14, 16, 14),
                          child: Row(
                            children: [
                              Icon(
                                Icons.groups_outlined,
                                size: 15,
                                color:
                                    Theme.of(context).colorScheme.onSurfaceVariant,
                              ),
                              const SizedBox(width: 7),
                              Text(
                                '${s.membersLabel} · $total',
                                style: Theme.of(context)
                                    .textTheme
                                    .labelMedium
                                    ?.copyWith(
                                      color: Theme.of(context)
                                          .colorScheme
                                          .onSurfaceVariant,
                                      fontWeight: FontWeight.w600,
                                      letterSpacing:
                                          s.isArabic ? 0 : 0.6,
                                    ),
                              ),
                            ],
                          ),
                        ),
                        const Divider(height: 1),
                        Expanded(
                          child: ListView.separated(
                            physics: const AlwaysScrollableScrollPhysics(),
                            itemCount: members.length,
                            separatorBuilder: (_, __) =>
                                const Divider(height: 1),
                            itemBuilder: (context, index) {
                              final member = members[index];
                              return _MemberRow(
                                member: member,
                                strings: s,
                                canModerate: widget.canModerate,
                                isSelf: member.userId == widget.myUserId,
                                onOpen: () => _openDetail(member),
                                onModerate: (action) =>
                                    _moderate(context, member, action),
                              );
                            },
                          ),
                        ),
                      ],
                    ),
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

class _MemberRow extends StatelessWidget {
  const _MemberRow({
    required this.member,
    required this.strings,
    required this.canModerate,
    required this.isSelf,
    required this.onOpen,
    required this.onModerate,
  });

  final NationalMember member;
  final AppStrings strings;
  final bool canModerate;
  final bool isSelf;
  final VoidCallback onOpen;
  final void Function(String action) onModerate;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final scheme = theme.colorScheme;
    final s = strings;
    final showRole = member.role == 'moderator' || member.role == 'admin';
    return InkWell(
      onTap: onOpen,
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
        child: Row(
          children: [
            _MonogramAvatar(name: member.displayName, radius: 22),
            const SizedBox(width: 12),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Row(
                    children: [
                      Flexible(
                        child: Text(
                          member.displayName,
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                          style: theme.textTheme.titleMedium
                              ?.copyWith(fontWeight: FontWeight.w600),
                        ),
                      ),
                      if (showRole) ...[
                        const SizedBox(width: 8),
                        _roleBadge(s, theme, member.role),
                      ],
                    ],
                  ),
                  const SizedBox(height: 6),
                  Wrap(
                    spacing: 6,
                    runSpacing: 4,
                    crossAxisAlignment: WrapCrossAlignment.center,
                    children: [
                      _levelBadge(s, theme, member.level),
                      if (!member.isActive)
                        _statusBadge(s, theme, member.status),
                    ],
                  ),
                ],
              ),
            ),
            if (canModerate && !isSelf)
              PopupMenuButton<String>(
                tooltip: s.moderation,
                icon: const Icon(Icons.more_vert),
                iconColor: scheme.onSurfaceVariant,
                onSelected: onModerate,
                itemBuilder: (_) => [
                  if (member.isModerator)
                    _menuItem(theme,
                        label: s.demoteModerator,
                        icon: Icons.undo,
                        value: 'demote')
                  else
                    _menuItem(theme,
                        label: s.promoteModerator,
                        icon: Icons.verified_outlined,
                        value: 'promote'),
                  if (member.isActive)
                    _menuItem(theme,
                        label: s.suspendMember,
                        icon: Icons.block_outlined,
                        value: 'suspend',
                        destructive: true)
                  else
                    _menuItem(theme,
                        label: s.restoreMember,
                        icon: Icons.lock_open_outlined,
                        value: 'restore'),
                ],
              ),
          ],
        ),
      ),
    );
  }
}

class _MonogramAvatar extends StatelessWidget {
  const _MonogramAvatar({required this.name, this.radius = 20});

  final String name;
  final double radius;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    final tones = <(Color, Color)>[
      (scheme.primaryContainer, scheme.onPrimaryContainer),
      (scheme.secondaryContainer, scheme.onSecondaryContainer),
      (scheme.tertiaryContainer, scheme.onTertiaryContainer),
      (scheme.surfaceContainerHighest, scheme.onSurfaceVariant),
    ];
    var hash = 0;
    for (final unit in name.codeUnits) {
      hash = (hash + unit) & 0x7fffffff;
    }
    final (bg, fg) = tones[hash % tones.length];
    return CircleAvatar(
      radius: radius,
      backgroundColor: bg,
      foregroundColor: fg,
      child: Text(
        name.isNotEmpty ? name[0].toUpperCase() : '?',
        style: TextStyle(
          fontSize: radius * 0.8,
          fontWeight: FontWeight.w600,
        ),
      ),
    );
  }
}

class _Pill extends StatelessWidget {
  const _Pill({
    required this.label,
    required this.icon,
    required this.background,
    required this.foreground,
    this.outlined = false,
  });

  final String label;
  final IconData icon;
  final Color background;
  final Color foreground;
  final bool outlined;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
      decoration: BoxDecoration(
        color: outlined ? null : background,
        borderRadius: BorderRadius.circular(999),
        border: outlined
            ? Border.all(color: scheme.outlineVariant, width: 1)
            : null,
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(icon, size: 13, color: foreground),
          const SizedBox(width: 5),
          Text(
            label,
            style: Theme.of(context).textTheme.labelSmall?.copyWith(
                  color: foreground,
                  fontWeight: FontWeight.w600,
                ),
          ),
        ],
      ),
    );
  }
}

_Pill _levelBadge(AppStrings s, ThemeData theme, String level) {
  final c = theme.colorScheme;
  final (bg, fg, icon, outlined) = switch (level) {
    'platinum' => (c.primaryContainer, c.onPrimaryContainer, Icons.diamond, false),
    'gold' => (
        c.secondaryContainer,
        c.onSecondaryContainer,
        Icons.workspace_premium_outlined,
        false,
      ),
    'silver' => (
        c.surfaceContainerHighest,
        c.onSurfaceVariant,
        Icons.emoji_events_outlined,
        true,
      ),
    _ => (
        c.tertiaryContainer,
        c.onTertiaryContainer,
        Icons.military_tech_outlined,
        false,
      ),
  };
  return _Pill(
    label: nationalLevelLabel(s, level),
    icon: icon,
    background: bg,
    foreground: fg,
    outlined: outlined,
  );
}

_Pill _statusBadge(AppStrings s, ThemeData theme, String status) {
  final c = theme.colorScheme;
  final (bg, fg, icon, outlined) = switch (status) {
    'active' => (
        c.primaryContainer,
        c.onPrimaryContainer,
        Icons.check_circle_outline,
        false,
      ),
    'suspended' => (
        c.errorContainer,
        c.onErrorContainer,
        Icons.block_outlined,
        false,
      ),
    _ => (
        c.surfaceContainerHighest,
        c.onSurfaceVariant,
        Icons.person_off_outlined,
        true,
      ),
  };
  return _Pill(
    label: nationalStatusLabel(s, status),
    icon: icon,
    background: bg,
    foreground: fg,
    outlined: outlined,
  );
}

_Pill _roleBadge(AppStrings s, ThemeData theme, String role) {
  final c = theme.colorScheme;
  final (bg, fg, icon, outlined) = switch (role) {
    'admin' => (
        c.secondary,
        c.onSecondary,
        Icons.admin_panel_settings_outlined,
        false,
      ),
    'moderator' => (c.primary, c.onPrimary, Icons.shield_outlined, false),
    _ => (
        c.surfaceContainerHighest,
        c.onSurfaceVariant,
        Icons.person_outline,
        true,
      ),
  };
  return _Pill(
    label: nationalRoleLabel(s, role),
    icon: icon,
    background: bg,
    foreground: fg,
    outlined: outlined,
  );
}

PopupMenuItem<String> _menuItem(
  ThemeData theme, {
  required String label,
  required IconData icon,
  required String value,
  bool destructive = false,
}) {
  final color =
      destructive ? theme.colorScheme.error : theme.colorScheme.onSurface;
  return PopupMenuItem<String>(
    value: value,
    child: Row(
      children: [
        Icon(icon, size: 18, color: color),
        const SizedBox(width: 12),
        Expanded(
          child: Text(
            label,
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
            style: TextStyle(color: color),
          ),
        ),
      ],
    ),
  );
}

Card _communityCard(
  BuildContext context, {
  required Widget child,
  EdgeInsetsGeometry margin = EdgeInsets.zero,
}) {
  final scheme = Theme.of(context).colorScheme;
  return Card(
    margin: margin,
    elevation: 0,
    clipBehavior: Clip.antiAlias,
    shape: RoundedRectangleBorder(
      borderRadius: BorderRadius.circular(16),
      side: BorderSide(color: scheme.outlineVariant, width: 1),
    ),
    child: child,
  );
}

class _EmptyMembers extends StatelessWidget {
  const _EmptyMembers({required this.hasFilter});

  final bool hasFilter;

  @override
  Widget build(BuildContext context) {
    final s = AppStrings.of(context);
    final theme = Theme.of(context);
    final scheme = theme.colorScheme;
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Container(
              width: 96,
              height: 96,
              decoration: BoxDecoration(
                color: scheme.surfaceContainerHigh,
                shape: BoxShape.circle,
              ),
              child: Icon(
                hasFilter ? Icons.search_off : Icons.group_outlined,
                size: 40,
                color: hasFilter ? scheme.onSurfaceVariant : scheme.primary,
              ),
            ),
            const SizedBox(height: 16),
            Text(
              hasFilter ? s.noMembersFound : s.noMembers,
              textAlign: TextAlign.center,
              style: theme.textTheme.titleMedium
                  ?.copyWith(fontWeight: FontWeight.w600),
            ),
            const SizedBox(height: 8),
            Text(
              hasFilter ? s.noResultsHint : s.noMembersHint,
              textAlign: TextAlign.center,
              style: theme.textTheme.bodyMedium
                  ?.copyWith(color: scheme.onSurfaceVariant),
            ),
          ],
        ),
      ),
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
  Future<bool> _confirmAction(
    String title,
    String body,
    String label, {
    bool destructive = false,
  }) {
    return _confirmModeration(
      context,
      title: title,
      body: body,
      memberName: widget.member.displayName,
      confirmLabel: label,
      destructive: destructive,
    );
  }

  @override
  Widget build(BuildContext context) {
    final s = AppStrings.of(context);
    final theme = Theme.of(context);
    final scheme = theme.colorScheme;
    final member = widget.member;
    final isSelf = member.userId == widget.myUserId;
    final invited = member.invitedByName.isNotEmpty
        ? member.invitedByName
        : member.invitedBy;

    return Scaffold(
      appBar: AppBar(title: Text(member.displayName)),
      body: MaxWidthBox(
        child: ListView(
          padding: const EdgeInsets.all(16),
          children: [
            _communityCard(
              context,
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: Row(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    _MonogramAvatar(name: member.displayName, radius: 36),
                    const SizedBox(width: 16),
                    Expanded(
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text(
                            member.displayName,
                            maxLines: 1,
                            overflow: TextOverflow.ellipsis,
                            style: theme.textTheme.titleLarge
                                ?.copyWith(fontWeight: FontWeight.w600),
                          ),
                          const SizedBox(height: 10),
                          Wrap(
                            spacing: 6,
                            runSpacing: 6,
                            crossAxisAlignment: WrapCrossAlignment.center,
                            children: [
                              _levelBadge(s, theme, member.level),
                              _statusBadge(s, theme, member.status),
                              _roleBadge(s, theme, member.role),
                            ],
                          ),
                          if (member.joinedAt.isNotEmpty) ...[
                            const SizedBox(height: 10),
                            Row(
                              mainAxisSize: MainAxisSize.min,
                              children: [
                                Icon(
                                  Icons.event_outlined,
                                  size: 14,
                                  color: scheme.onSurfaceVariant,
                                ),
                                const SizedBox(width: 6),
                                Flexible(
                                  child: Text(
                                    '${s.joinedOn} ${member.joinedAt}',
                                    maxLines: 1,
                                    overflow: TextOverflow.ellipsis,
                                    style: theme.textTheme.bodySmall
                                        ?.copyWith(
                                            color: scheme.onSurfaceVariant),
                                  ),
                                ),
                              ],
                            ),
                          ],
                        ],
                      ),
                    ),
                  ],
                ),
              ),
            ),
            const SizedBox(height: 12),
            _communityCard(
              context,
              child: Column(
                children: [
                  _MetaRow(
                    icon: Icons.military_tech_outlined,
                    label: s.expertiseScore,
                    value: '${member.expertiseScore}',
                  ),
                  if (member.joinedVia.isNotEmpty) ...[
                    const Divider(height: 1, indent: 46),
                    _MetaRow(
                      icon: Icons.route_outlined,
                      label: s.joinedViaLabel,
                      value: member.joinedVia,
                    ),
                  ],
                  if (invited.isNotEmpty) ...[
                    const Divider(height: 1, indent: 46),
                    _MetaRow(
                      icon: Icons.person_outline,
                      label: s.invitedByLabel,
                      value: invited,
                    ),
                  ],
                ],
              ),
            ),
            if (_error != null) ...[
              const SizedBox(height: 16),
              Text(_error!,
                  textAlign: TextAlign.center,
                  style: TextStyle(color: scheme.error)),
            ],
            if (widget.canModerate && !isSelf) ...[
              const SizedBox(height: 24),
              Text(
                s.nationalMembership.toUpperCase(),
                style: theme.textTheme.labelSmall?.copyWith(
                  fontWeight: FontWeight.w600,
                  letterSpacing: s.isArabic ? 0 : 1.4,
                  color: scheme.onSurfaceVariant,
                ),
              ),
              const SizedBox(height: 10),
              _communityCard(
                context,
                child: Padding(
                  padding: const EdgeInsets.all(16),
                  child: Wrap(
                    spacing: 8,
                    runSpacing: 8,
                    children: [
                      if (member.isModerator)
                        OutlinedButton.icon(
                          onPressed: _busy
                              ? null
                              : () async {
                                  if (!await _confirmAction(s.confirmDemote,
                                      s.confirmDemoteBody, s.demoteModerator)) {
                                    return;
                                  }
                                  final ok = await _update(role: 'member');
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
                                  if (!await _confirmAction(s.confirmPromote,
                                      s.confirmPromoteBody, s.promoteModerator)) {
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
                                  if (!await _confirmAction(s.confirmSuspend,
                                      s.confirmSuspendBody, s.suspendMember,
                                      destructive: true)) {
                                    return;
                                  }
                                  final ok =
                                      await _update(status: 'suspended');
                                  if (ok && context.mounted) {
                                    Navigator.of(context).pop(true);
                                  }
                                },
                          style: OutlinedButton.styleFrom(
                            foregroundColor: scheme.error,
                            side: BorderSide(color: scheme.error),
                          ),
                          icon: const Icon(Icons.block_outlined),
                          label: Text(s.suspendMember),
                        )
                      else
                        OutlinedButton.icon(
                          onPressed: _busy
                              ? null
                              : () async {
                                  if (!await _confirmAction(s.confirmRestore,
                                      s.confirmRestoreBody, s.restoreMember)) {
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
                ),
              ),
            ],
          ],
        ),
      ),
    );
  }
}

class _MetaRow extends StatelessWidget {
  const _MetaRow({
    required this.icon,
    required this.label,
    required this.value,
  });

  final IconData icon;
  final String label;
  final String value;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final scheme = theme.colorScheme;
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 13),
      child: Row(
        children: [
          Icon(icon, size: 18, color: scheme.onSurfaceVariant),
          const SizedBox(width: 12),
          Expanded(
            child: Text(
              label,
              style: theme.textTheme.bodyMedium
                  ?.copyWith(color: scheme.onSurfaceVariant),
            ),
          ),
          const SizedBox(width: 12),
          Flexible(
            child: Text(
              value,
              textAlign: TextAlign.end,
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
              style: theme.textTheme.titleSmall
                  ?.copyWith(fontWeight: FontWeight.w600),
            ),
          ),
        ],
      ),
    );
  }
}

/// Shared confirmation gate for moderator actions. Destructive actions get an
/// error-styled confirm button.
Future<bool> _confirmModeration(
  BuildContext context, {
  required String title,
  required String body,
  required String memberName,
  required String confirmLabel,
  bool destructive = false,
}) {
  return showDialog<bool>(
    context: context,
    builder: (ctx) {
      final theme = Theme.of(ctx);
      final text = AppStrings.of(ctx);
      return AlertDialog(
        title: Text(title),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(memberName, style: theme.textTheme.titleMedium),
            const SizedBox(height: 8),
            Text(body),
          ],
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx, false),
            child: Text(text.cancel),
          ),
          FilledButton(
            style: destructive
                ? FilledButton.styleFrom(
                    backgroundColor: theme.colorScheme.error,
                    foregroundColor: theme.colorScheme.onError,
                  )
                : null,
            onPressed: () => Navigator.pop(ctx, true),
            child: Text(confirmLabel),
          ),
        ],
      );
    },
  ).then((value) => value ?? false);
}

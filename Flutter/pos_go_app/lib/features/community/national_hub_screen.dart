import 'package:flutter/material.dart';

import '../../core/api_client.dart';
import '../../core/community.dart';
import '../../core/layout.dart';
import '../../core/session_store.dart';
import '../../l10n/strings.dart';
import 'national_invitations_screen.dart';
import 'national_labels.dart';
import 'national_members_screen.dart';

/// Landing screen for the Egypt national community. Shows the caller's own
/// membership or lets them join with an invite code, then links to the
/// national invitations and member directory.
class NationalHubScreen extends StatefulWidget {
  const NationalHubScreen({
    super.key,
    required this.session,
    required this.apiClient,
  });

  final Session session;
  final ApiClient apiClient;

  @override
  State<NationalHubScreen> createState() => _NationalHubScreenState();
}

class _NationalHubScreenState extends State<NationalHubScreen> {
  NationalMember? _me;
  bool _loading = true;
  String? _error;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final me = await widget.apiClient.nationalMe(widget.session);
      if (!mounted) return;
      setState(() {
        _me = me;
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

  Future<void> _promptJoin() async {
    final s = AppStrings.of(context);
    final member = await showDialog<NationalMember>(
      context: context,
      builder: (_) => _JoinCodeDialog(
        session: widget.session,
        apiClient: widget.apiClient,
      ),
    );
    if (member == null) return;
    await _load();
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(content: Text(s.joined)),
    );
  }

  @override
  Widget build(BuildContext context) {
    final s = AppStrings.of(context);
    return Scaffold(
      appBar: AppBar(title: Text(s.nationalCommunity)),
      body: MaxWidthBox(
        child: _buildBody(context, s),
      ),
    );
  }

  Widget _buildBody(BuildContext context, AppStrings s) {
    if (_loading) {
      return const Center(child: CircularProgressIndicator());
    }
    if (_error != null) {
      return Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Text(_error!,
                style: TextStyle(color: Theme.of(context).colorScheme.error)),
            const SizedBox(height: 12),
            FilledButton(onPressed: _load, child: Text(s.retry)),
          ],
        ),
      );
    }
    final me = _me;
    if (me == null) {
      return _buildNotMember(context, s);
    }
    return _buildMember(context, s, me);
  }

  Widget _buildNotMember(BuildContext context, AppStrings s) {
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        const _NationalBadge(),
        const SizedBox(height: 16),
        Text(s.notYetMember, style: Theme.of(context).textTheme.titleLarge),
        const SizedBox(height: 8),
        Text(s.notYetMemberBody),
        const SizedBox(height: 16),
        FilledButton.icon(
          onPressed: _promptJoin,
          icon: const Icon(Icons.group_add_outlined),
          label: Text(s.joinNow),
        ),
      ],
    );
  }

  Widget _buildMember(BuildContext context, AppStrings s, NationalMember me) {
    final memberCard = Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Row(
          children: [
            CircleAvatar(
              radius: 24,
              child: Text(me.displayName.isNotEmpty
                  ? me.displayName[0].toUpperCase()
                  : '?'),
            ),
            const SizedBox(width: 16),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(me.displayName,
                      style: Theme.of(context).textTheme.titleLarge),
                  const SizedBox(height: 4),
                  Wrap(
                    spacing: 8,
                    runSpacing: 4,
                    crossAxisAlignment: WrapCrossAlignment.center,
                    children: [
                      Chip(
                        visualDensity: VisualDensity.compact,
                        avatar:
                            const Icon(Icons.military_tech_outlined, size: 16),
                        label: Text(nationalLevelLabel(s, me.level)),
                      ),
                      if (!me.isActive)
                        Chip(
                          visualDensity: VisualDensity.compact,
                          avatar: const Icon(Icons.block, size: 16),
                          label: Text(nationalStatusLabel(s, me.status)),
                        ),
                    ],
                  ),
                  const SizedBox(height: 4),
                  Text(
                    '${s.expertiseScore}: ${me.expertiseScore}',
                    style: Theme.of(context).textTheme.bodySmall,
                  ),
                  if (me.joinedAt.isNotEmpty)
                    Text('${s.joinedAs} ${me.joinedAt}',
                        style: Theme.of(context).textTheme.bodySmall),
                ],
              ),
            ),
          ],
        ),
      ),
    );

    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        memberCard,
        const SizedBox(height: 16),
        _HubTile(
          icon: Icons.mail_outline,
          title: s.invitePeers,
          subtitle: s.invitePeersHint,
          onTap: () => Navigator.of(context).push(
            MaterialPageRoute(
              builder: (_) => NationalInvitationsScreen(
                session: widget.session,
                apiClient: widget.apiClient,
              ),
            ),
          ),
        ),
        const SizedBox(height: 12),
        _HubTile(
          icon: Icons.groups_outlined,
          title: s.nationalDirectory,
          subtitle: s.nationalDirectoryHint,
          onTap: () => Navigator.of(context).push(
            MaterialPageRoute(
              builder: (_) => NationalMembersScreen(
                session: widget.session,
                apiClient: widget.apiClient,
                canModerate: me.isModerator,
                myUserId: me.userId,
              ),
            ),
          ),
        ),
      ],
    );
  }
}

class _JoinCodeDialog extends StatefulWidget {
  const _JoinCodeDialog({
    required this.session,
    required this.apiClient,
  });

  final Session session;
  final ApiClient apiClient;

  @override
  State<_JoinCodeDialog> createState() => _JoinCodeDialogState();
}

class _JoinCodeDialogState extends State<_JoinCodeDialog> {
  // Owned here so the controller lives as long as the dialog's exit animation,
  // which would otherwise read a disposed controller (framework crash).
  final _controller = TextEditingController();
  bool _busy = false;
  String? _error;

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final s = AppStrings.of(context);
    return AlertDialog(
      title: Text(s.joinCodeDialog),
      content: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          TextField(
            controller: _controller,
            autofocus: true,
            textCapitalization: TextCapitalization.characters,
            decoration: InputDecoration(
              hintText: s.joinCode,
              prefixIcon: const Icon(Icons.password),
              border: const OutlineInputBorder(),
            ),
            onSubmitted: (_) => _submit(),
          ),
          if (_error != null) ...[
            const SizedBox(height: 8),
            Text(_error!,
                style: TextStyle(color: Theme.of(context).colorScheme.error)),
          ],
        ],
      ),
      actions: [
        TextButton(
          onPressed: _busy ? null : () => Navigator.of(context).pop(false),
          child: Text(s.cancel),
        ),
        FilledButton(
          onPressed: _busy ? null : _submit,
          child: _busy
              ? const SizedBox(
                  width: 18,
                  height: 18,
                  child: CircularProgressIndicator(strokeWidth: 2),
                )
              : Text(s.joinNow),
        ),
      ],
    );
  }

  Future<void> _submit() async {
    final s = AppStrings.of(context);
    final code = _controller.text.trim();
    if (code.isEmpty) {
      setState(() => _error = s.joinCode);
      return;
    }
    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      final member = await widget.apiClient.nationalJoin(widget.session, code);
      if (!mounted) return;
      Navigator.of(context).pop(member);
    } catch (e) {
      if (!mounted) return;
      setState(() {
        _busy = false;
        _error = _joinErrorMessage(e, s);
      });
    }
  }

  String _joinErrorMessage(Object e, AppStrings s) {
    final ApiException api = e is ApiException ? e : const ApiException('');
    return switch (api.code) {
      'invalid_code' => s.invalidCodeError,
      'expired' => s.expiredCodeError,
      'already_member' => s.alreadyMemberError,
      'egypt_only' => s.egyptOnlyError,
      _ => e.toString(),
    };
  }
}

class _NationalBadge extends StatelessWidget {
  const _NationalBadge();

  @override
  Widget build(BuildContext context) {
    return CircleAvatar(
      radius: 36,
      backgroundColor: Theme.of(context).colorScheme.primaryContainer,
      child: Icon(
        Icons.public,
        size: 36,
        color: Theme.of(context).colorScheme.onPrimaryContainer,
      ),
    );
  }
}

class _HubTile extends StatelessWidget {
  const _HubTile({
    required this.icon,
    required this.title,
    required this.subtitle,
    required this.onTap,
  });

  final IconData icon;
  final String title;
  final String subtitle;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    return Card(
      clipBehavior: Clip.antiAlias,
      child: ListTile(
        contentPadding: const EdgeInsets.all(16),
        leading: CircleAvatar(radius: 24, child: Icon(icon)),
        title: Text(title, style: Theme.of(context).textTheme.titleLarge),
        subtitle: subtitle.isEmpty ? null : Text(subtitle),
        trailing: const Icon(Icons.chevron_right),
        onTap: onTap,
      ),
    );
  }
}

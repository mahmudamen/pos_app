import 'package:flutter/material.dart';

import '../../core/api_client.dart';
import '../../core/community.dart';
import '../../core/layout.dart';
import '../../core/session_store.dart';
import '../../l10n/strings.dart';
import 'national_labels.dart';

/// Lists the caller's national community invitations, with create + revoke.
class NationalInvitationsScreen extends StatefulWidget {
  const NationalInvitationsScreen({
    super.key,
    required this.session,
    required this.apiClient,
  });

  final Session session;
  final ApiClient apiClient;

  @override
  State<NationalInvitationsScreen> createState() =>
      _NationalInvitationsScreenState();
}

class _NationalInvitationsScreenState extends State<NationalInvitationsScreen> {
  List<NationalInvitation>? _invitations;
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
      final invitations =
          await widget.apiClient.listNationalInvitations(widget.session);
      if (!mounted) return;
      setState(() {
        _invitations = invitations;
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

  Future<void> _create() async {
    final created = await showDialog<NationalInvitation>(
      context: context,
      builder: (_) => _CreateInvitationDialog(
        session: widget.session,
        apiClient: widget.apiClient,
      ),
    );
    if (created == null || !mounted) return;
    await _load();
    if (!mounted) return;
    await showDialog<void>(
      context: context,
      builder: (_) => _CodeRevealDialog(invitation: created),
    );
  }

  Future<void> _revoke(NationalInvitation invitation) async {
    final s = AppStrings.of(context);
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (dialogContext) => AlertDialog(
        title: Text(s.confirmRevoke),
        content: Text(s.confirmRevokeBody),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(dialogContext).pop(false),
            child: Text(s.cancel),
          ),
          FilledButton(
            onPressed: () => Navigator.of(dialogContext).pop(true),
            child: Text(s.revokeInvitation),
          ),
        ],
      ),
    );
    if (confirmed != true || !mounted) return;
    try {
      await widget.apiClient
          .revokeNationalInvitation(widget.session, invitation.id);
      if (!mounted) return;
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(s.invitationRevoked)));
      await _load();
    } catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(e.toString())));
    }
  }

  @override
  Widget build(BuildContext context) {
    final s = AppStrings.of(context);
    return Scaffold(
      appBar: AppBar(
        title: Text(s.invitations),
        actions: [
          IconButton(
            onPressed: _loading ? null : _create,
            tooltip: s.newInvitation,
            icon: const Icon(Icons.add),
          ),
        ],
      ),
      body: MaxWidthBox(
        child: _loading
            ? const Center(child: CircularProgressIndicator())
            : _error != null
                ? Center(
                    child: Column(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        Text(_error!,
                            style: TextStyle(
                                color: Theme.of(context).colorScheme.error)),
                        const SizedBox(height: 12),
                        FilledButton(onPressed: _load, child: Text(s.retry)),
                      ],
                    ),
                  )
                : _buildList(context, s),
      ),
    );
  }

  Widget _buildList(BuildContext context, AppStrings s) {
    final invitations = _invitations ?? [];
    if (invitations.isEmpty) {
      return Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            const Icon(Icons.mail_outline, size: 48),
            const SizedBox(height: 12),
            Text(s.noInvitations),
          ],
        ),
      );
    }
    return RefreshIndicator(
      onRefresh: _load,
      child: ListView.separated(
        physics: const AlwaysScrollableScrollPhysics(),
        itemCount: invitations.length,
        separatorBuilder: (_, __) => const Divider(height: 1),
        itemBuilder: (context, index) {
          final invitation = invitations[index];
          return _InvitationTile(
            invitation: invitation,
            onRevoke: invitation.isActive ? () => _revoke(invitation) : null,
          );
        },
      ),
    );
  }
}

class _InvitationTile extends StatelessWidget {
  const _InvitationTile({required this.invitation, this.onRevoke});

  final NationalInvitation invitation;
  final VoidCallback? onRevoke;

  @override
  Widget build(BuildContext context) {
    final s = AppStrings.of(context);
    final recipient =
        invitation.email.isNotEmpty ? invitation.email : s.inviteNoteEmpty;
    final subtitle = [
      invitation.note.isNotEmpty ? invitation.note : null,
      '${s.usesLabel}: ${invitation.usedCount}/${invitation.maxUses}',
      if (invitation.expiresAt.isNotEmpty)
        '${s.expiresLabel}: ${invitation.expiresAt}',
    ].whereType<String>().join(' · ');
    return ListTile(
      leading: Icon(
        invitation.isActive ? Icons.mail_outline : Icons.mail_lock,
        color:
            invitation.isActive ? Theme.of(context).colorScheme.primary : null,
      ),
      title: Row(
        children: [
          Flexible(child: Text(recipient)),
          const SizedBox(width: 8),
          Chip(
            visualDensity: VisualDensity.compact,
            label: Text(nationalStatusLabel(s, invitation.status)),
          ),
        ],
      ),
      subtitle: subtitle.isEmpty ? null : Text(subtitle),
      trailing: onRevoke == null
          ? null
          : IconButton(
              onPressed: onRevoke,
              tooltip: s.revokeInvitation,
              icon: const Icon(Icons.delete_outline),
            ),
    );
  }
}

class _CreateInvitationDialog extends StatefulWidget {
  const _CreateInvitationDialog({
    required this.session,
    required this.apiClient,
  });

  final Session session;
  final ApiClient apiClient;

  @override
  State<_CreateInvitationDialog> createState() =>
      _CreateInvitationDialogState();
}

class _CreateInvitationDialogState extends State<_CreateInvitationDialog> {
  final _email = TextEditingController();
  final _note = TextEditingController();
  int _maxUses = 1;
  bool _busy = false;
  String? _error;

  @override
  void dispose() {
    _email.dispose();
    _note.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final s = AppStrings.of(context);
    return AlertDialog(
      title: Text(s.createInvitation),
      content: SingleChildScrollView(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            TextField(
              controller: _email,
              keyboardType: TextInputType.emailAddress,
              decoration: InputDecoration(
                labelText: s.recipientEmail,
                prefixIcon: const Icon(Icons.email_outlined),
                border: const OutlineInputBorder(),
              ),
            ),
            const SizedBox(height: 12),
            TextField(
              controller: _note,
              decoration: InputDecoration(
                labelText: s.inviteNote,
                prefixIcon: const Icon(Icons.notes_outlined),
                border: const OutlineInputBorder(),
              ),
            ),
            const SizedBox(height: 12),
            DropdownButtonFormField<int>(
              initialValue: _maxUses,
              decoration: InputDecoration(
                labelText: s.maxUsesLabel,
                border: const OutlineInputBorder(),
              ),
              items: [1, 2, 3, 5, 10]
                  .map((u) => DropdownMenuItem(value: u, child: Text('$u')))
                  .toList(),
              onChanged: (v) => setState(() => _maxUses = v ?? 1),
            ),
            if (_error != null) ...[
              const SizedBox(height: 8),
              Text(_error!,
                  style: TextStyle(color: Theme.of(context).colorScheme.error)),
            ],
          ],
        ),
      ),
      actions: [
        TextButton(
          onPressed: _busy ? null : () => Navigator.of(context).pop(),
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
              : Text(s.createInvitation),
        ),
      ],
    );
  }

  Future<void> _submit() async {
    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      final invitation = await widget.apiClient.createNationalInvitation(
        widget.session,
        email: _email.text.trim(),
        note: _note.text.trim(),
        maxUses: _maxUses,
      );
      if (!mounted) return;
      Navigator.of(context).pop(invitation);
    } catch (e) {
      if (!mounted) return;
      setState(() {
        _busy = false;
        _error = e.toString();
      });
    }
  }
}

class _CodeRevealDialog extends StatelessWidget {
  const _CodeRevealDialog({required this.invitation});

  final NationalInvitation invitation;

  @override
  Widget build(BuildContext context) {
    final s = AppStrings.of(context);
    return AlertDialog(
      icon: const Icon(Icons.key, size: 32),
      title: Text(s.invitationCodeTitle),
      content: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          SelectableText(
            invitation.codeDisplay,
            style: Theme.of(context).textTheme.headlineSmall?.copyWith(
                  letterSpacing: 1.5,
                ),
            textAlign: TextAlign.center,
          ),
          const SizedBox(height: 8),
          Text(s.codeShareHint,
              textAlign: TextAlign.center,
              style: Theme.of(context).textTheme.bodySmall),
        ],
      ),
      actions: [
        FilledButton(
          onPressed: () => Navigator.of(context).pop(),
          child: Text(s.ok),
        ),
      ],
    );
  }
}

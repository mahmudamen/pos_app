import 'package:flutter/material.dart';

import '../../core/api_client.dart';
import '../../core/community.dart';
import '../../core/layout.dart';
import '../../core/session_store.dart';
import '../../l10n/strings.dart';

const _manageRoles = ['owner', 'manager', 'saas_admin'];

const _iconChoices = [
  'military_tech',
  'workspace_premium',
  'emoji_events',
  'star',
  'favorite',
];

/// Maps a backend badge icon name to a Material icon (fallback trophy).
IconData badgeIconData(String name) => switch (name) {
      'star' => Icons.star,
      'workspace_premium' => Icons.workspace_premium,
      'favorite' => Icons.favorite,
      'military_tech' => Icons.military_tech,
      _ => Icons.emoji_events,
    };

class BadgesScreen extends StatefulWidget {
  const BadgesScreen({
    super.key,
    required this.session,
    required this.apiClient,
  });

  final Session session;
  final ApiClient apiClient;

  @override
  State<BadgesScreen> createState() => _BadgesScreenState();
}

class _BadgesScreenState extends State<BadgesScreen> {
  List<BadgeDefinition> _badges = const [];
  bool _loading = true;
  String? _error;
  String _query = '';

  final _formCode = TextEditingController();
  final _formName = TextEditingController();
  final _formDescription = TextEditingController();
  String _formIcon = 'military_tech';

  final _editName = TextEditingController();
  final _editDescription = TextEditingController();
  String _editIcon = 'military_tech';
  bool _editActive = true;
  BadgeDefinition? _editing;

  bool get _canWrite => _manageRoles.contains(widget.session.role);

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
      final result = await widget.apiClient.listBadges(
        widget.session,
        query: _query,
        includeInactive: _canWrite,
      );
      if (!mounted) return;
      setState(() {
        _badges = result;
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

  String _messageFor(Object error) {
    final s = AppStrings.of(context);
    if (error is ApiException) {
      if (error.code == 'code_taken') return s.codeTaken;
    }
    return error.toString();
  }

  Future<void> _createBadge(BuildContext sheetContext) async {
    final s = AppStrings.of(context);
    final code = _formCode.text.trim();
    final name = _formName.text.trim();
    if (code.isEmpty || name.isEmpty) {
      ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('${s.badgeCode} / ${s.badgeName}: ${s.required}')));
      return;
    }
    try {
      await widget.apiClient.createBadge(
        widget.session,
        code: code,
        name: name,
        description: _formDescription.text.trim(),
        icon: _formIcon,
      );
      _formCode.clear();
      _formName.clear();
      _formDescription.clear();
      if (!sheetContext.mounted) return;
      Navigator.of(sheetContext).pop();
      if (!mounted) return;
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(s.badgeSaved)));
      await _load();
    } catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(_messageFor(e))));
    }
  }

  Future<void> _saveEdit(BuildContext dialogContext) async {
    final s = AppStrings.of(context);
    final badge = _editing;
    if (badge == null) return;
    final name = _editName.text.trim();
    if (name.isEmpty) {
      ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('${s.badgeName}: ${s.required}')));
      return;
    }
    try {
      await widget.apiClient.updateBadge(
        widget.session,
        badge.id,
        name: name,
        description: _editDescription.text.trim(),
        icon: _editIcon,
        isActive: _editActive,
      );
      if (!dialogContext.mounted) return;
      Navigator.of(dialogContext).pop();
      if (!mounted) return;
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(s.badgeSaved)));
      await _load();
    } catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(_messageFor(e))));
    }
  }

  void _showCreateForm() {
    final s = AppStrings.of(context);
    showModalBottomSheet<void>(
      context: context,
      isScrollControlled: true,
      builder: (sheetContext) => Padding(
        padding: EdgeInsets.only(
          bottom: MediaQuery.of(sheetContext).viewInsets.bottom,
        ),
        child: SafeArea(
          child: SingleChildScrollView(
            padding: const EdgeInsets.all(16),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                Text(s.addBadge,
                    style: Theme.of(sheetContext).textTheme.titleLarge),
                const SizedBox(height: 12),
                TextField(
                  controller: _formCode,
                  autofocus: true,
                  decoration: InputDecoration(
                      labelText: s.badgeCode,
                      hintText: s.badgeCodeHint,
                      border: const OutlineInputBorder()),
                ),
                const SizedBox(height: 12),
                TextField(
                  controller: _formName,
                  decoration: InputDecoration(
                      labelText: s.badgeName,
                      border: const OutlineInputBorder()),
                ),
                const SizedBox(height: 12),
                TextField(
                  controller: _formDescription,
                  maxLines: 2,
                  decoration: InputDecoration(
                      labelText: s.descriptionLabel,
                      border: const OutlineInputBorder()),
                ),
                const SizedBox(height: 12),
                DropdownButtonFormField<String>(
                  initialValue: _formIcon,
                  decoration: InputDecoration(
                      labelText: s.communityBadges,
                      border: const OutlineInputBorder()),
                  items: _iconChoices
                      .map((name) => DropdownMenuItem(
                          value: name, child: Text(name)))
                      .toList(),
                  onChanged: (value) =>
                      setState(() => _formIcon = value ?? 'military_tech'),
                ),
                const SizedBox(height: 16),
                Row(
                  mainAxisAlignment: MainAxisAlignment.end,
                  children: [
                    TextButton(
                        onPressed: () => Navigator.of(sheetContext).pop(),
                        child: Text(s.cancel)),
                    const SizedBox(width: 8),
                    FilledButton(
                      onPressed: () => _createBadge(sheetContext),
                      child: Text(s.save),
                    ),
                  ],
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }

  void _showEditForm(BadgeDefinition badge) {
    final s = AppStrings.of(context);
    _editing = badge;
    _editName.text = badge.name;
    _editDescription.text = badge.description;
    _editIcon = badge.icon;
    _editActive = badge.isActive;
    showDialog<void>(
      context: context,
      builder: (dialogContext) => StatefulBuilder(
        builder: (dialogContext, setDialogState) => AlertDialog(
          title: Text(badge.name),
          content: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              TextField(
                controller: _editName,
                decoration: InputDecoration(
                    labelText: s.badgeName,
                    border: const OutlineInputBorder()),
              ),
              const SizedBox(height: 12),
              TextField(
                controller: _editDescription,
                maxLines: 2,
                decoration: InputDecoration(
                    labelText: s.descriptionLabel,
                    border: const OutlineInputBorder()),
              ),
              const SizedBox(height: 12),
              DropdownButtonFormField<String>(
                initialValue: _editIcon,
                decoration: InputDecoration(
                    labelText: s.communityBadges,
                    border: const OutlineInputBorder()),
                items: _iconChoices
                    .map((name) =>
                        DropdownMenuItem(value: name, child: Text(name)))
                    .toList(),
                onChanged: (value) => setDialogState(
                    () => _editIcon = value ?? 'military_tech'),
              ),
              SwitchListTile(
                contentPadding: EdgeInsets.zero,
                title: Text(s.badgeActive),
                value: _editActive,
                onChanged: (v) => setDialogState(() => _editActive = v),
              ),
            ],
          ),
          actions: [
            TextButton(
                onPressed: () => Navigator.of(dialogContext).pop(),
                child: Text(s.cancel)),
            FilledButton(
              onPressed: () => _saveEdit(dialogContext),
              child: Text(s.save),
            ),
          ],
        ),
      ),
    );
  }

  @override
  void dispose() {
    _formCode.dispose();
    _formName.dispose();
    _formDescription.dispose();
    _editName.dispose();
    _editDescription.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final s = AppStrings.of(context);
    return Scaffold(
      appBar: AppBar(title: Text(s.communityBadges)),
      floatingActionButton: _canWrite
          ? FloatingActionButton(
              onPressed: _showCreateForm,
              tooltip: s.addBadge,
              child: const Icon(Icons.add),
            )
          : null,
      body: MaxWidthBox(
        child: Column(
          children: [
            Padding(
              padding: const EdgeInsets.all(8),
              child: TextField(
                decoration: InputDecoration(
                  hintText: s.communityBadges,
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
                              FilledButton(onPressed: _load, child: Text(s.retry)),
                            ],
                          ),
                        )
                      : _badges.isEmpty
                          ? Center(child: Text(s.noBadges))
                          : ListView.separated(
                              itemCount: _badges.length,
                              separatorBuilder: (_, __) =>
                                  const Divider(height: 1),
                              itemBuilder: (context, index) {
                                final badge = _badges[index];
                                return ListTile(
                                  leading: Icon(badgeIconData(badge.icon)),
                                  title: Row(
                                    mainAxisSize: MainAxisSize.min,
                                    children: [
                                      Flexible(child: Text(badge.name)),
                                      if (!badge.isActive) ...[
                                        const SizedBox(width: 6),
                                        _pill(s.badgeInactive,
                                            Theme.of(context).colorScheme.error),
                                      ],
                                    ],
                                  ),
                                  subtitle: Text(
                                    [
                                      badge.code,
                                      if (badge.description.isNotEmpty)
                                        badge.description,
                                    ].join(' · '),
                                    maxLines: 2,
                                    overflow: TextOverflow.ellipsis,
                                  ),
                                  trailing: Text(
                                      '${badge.earnedCount} ${s.earnedLabel}'),
                                  onTap: _canWrite
                                      ? () => _showEditForm(badge)
                                      : null,
                                );
                              },
                            ),
            ),
          ],
        ),
      ),
    );
  }
}

Widget _pill(String label, Color color) => Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.15),
        borderRadius: BorderRadius.circular(999),
      ),
      child: Text(label, style: TextStyle(fontSize: 11, color: color)),
    );

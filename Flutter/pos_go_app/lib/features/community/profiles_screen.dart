import 'package:flutter/material.dart';

import '../../core/api_client.dart';
import '../../core/community.dart';
import '../../core/layout.dart';
import '../../core/session_store.dart';
import '../../l10n/strings.dart';
import 'badges_screen.dart';

const _manageRoles = ['owner', 'manager', 'saas_admin'];

class ProfilesScreen extends StatefulWidget {
  const ProfilesScreen({
    super.key,
    required this.session,
    required this.apiClient,
  });

  final Session session;
  final ApiClient apiClient;

  @override
  State<ProfilesScreen> createState() => _ProfilesScreenState();
}

class _ProfilesScreenState extends State<ProfilesScreen> {
  ProfilesPage? _page;
  bool _loading = true;
  String? _error;
  int _currentPage = 1;
  String _query = '';
  bool _chiefsOnly = false;

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
      final result = await widget.apiClient.listProfiles(
        widget.session,
        query: _query,
        role: _chiefsOnly ? 'chief' : '',
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

  @override
  Widget build(BuildContext context) {
    final s = AppStrings.of(context);
    return Scaffold(
      appBar: AppBar(title: Text(s.communityProfiles)),
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
                        hintText: s.searchProfiles,
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
                  FilterChip(
                    label: Text(s.chiefLabel),
                    selected: _chiefsOnly,
                    onSelected: (v) {
                      setState(() => _chiefsOnly = v);
                      _load();
                    },
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
                      : _buildList(context),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildList(BuildContext context) {
    final s = AppStrings.of(context);
    final profiles = _page?.profiles ?? [];
    final total = _page?.total ?? 0;
    final totalPages = (_page?.limit ?? 50) > 0
        ? (total / (_page?.limit ?? 50)).ceil()
        : 1;

    return Column(
      children: [
        Expanded(
          child: profiles.isEmpty
              ? Center(child: Text(s.noProfiles))
              : ListView.separated(
                  itemCount: profiles.length,
                  separatorBuilder: (_, __) => const Divider(height: 1),
                  itemBuilder: (context, index) {
                    final profile = profiles[index];
                    final subtitle = [
                      if (profile.headline.isNotEmpty) profile.headline,
                      if (profile.location.isNotEmpty) profile.location,
                    ].join(' · ');
                    return ListTile(
                      leading: CircleAvatar(
                        child: profile.avatarUrl.isNotEmpty
                            ? null
                            : const Icon(Icons.person),
                      ),
                      title: Row(
                        mainAxisSize: MainAxisSize.min,
                        children: [
                          Flexible(child: Text(profile.displayName)),
                          if (profile.isChief) ...[
                            const SizedBox(width: 6),
                            Icon(Icons.verified,
                                size: 16,
                                color: Theme.of(context).colorScheme.primary),
                          ],
                        ],
                      ),
                      subtitle: subtitle.isEmpty ? null : Text(subtitle),
                      trailing: profile.skills.isNotEmpty
                          ? Text('${s.skills}: ${profile.skills.length}')
                          : null,
                      onTap: () => Navigator.of(context).push(
                        MaterialPageRoute(
                          builder: (_) => ProfileDetailScreen(
                            session: widget.session,
                            apiClient: widget.apiClient,
                            profile: profile,
                          ),
                        ),
                      ),
                    );
                  },
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

class ProfileDetailScreen extends StatefulWidget {
  const ProfileDetailScreen({
    super.key,
    required this.session,
    required this.apiClient,
    required this.profile,
  });

  final Session session;
  final ApiClient apiClient;
  final StaffProfile profile;

  @override
  State<ProfileDetailScreen> createState() => _ProfileDetailScreenState();
}

class _ProfileDetailScreenState extends State<ProfileDetailScreen> {
  StaffProfile? _profile;
  bool _loading = true;
  List<UserBadge> _badges = const [];
  List<EndorsementGroup> _endorsements = const [];

  bool get _canManage => _manageRoles.contains(widget.session.role);

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    try {
      final detail = await widget.apiClient.getProfile(
          widget.session, widget.profile.userId);
      if (!mounted) return;
      setState(() {
        _profile = detail;
        _loading = false;
      });
    } catch (e) {
      if (!mounted) return;
      setState(() {
        _profile = widget.profile;
        _loading = false;
      });
    }
    // Badge/endorsement enrichment is best-effort: a failure here must not
    // hide the profile itself.
    try {
      final badges = await widget.apiClient
          .listProfileBadges(widget.session, widget.profile.userId);
      if (!mounted) return;
      setState(() => _badges = badges);
    } catch (_) {}
    try {
      final groups = await widget.apiClient
          .listProfileEndorsements(widget.session, widget.profile.userId);
      if (!mounted) return;
      setState(() => _endorsements = groups);
    } catch (_) {}
  }

  Future<void> _revokeBadge(UserBadge badge) async {
    final s = AppStrings.of(context);
    try {
      await widget.apiClient.revokeProfileBadge(
          widget.session, widget.profile.userId, badge.badgeId);
      if (!mounted) return;
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(s.badgeRevoked)));
      await _load();
    } catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(e.toString())));
    }
  }

  Future<void> _awardBadge(String badgeId, String note) async {
    final s = AppStrings.of(context);
    try {
      await widget.apiClient.awardProfileBadge(
        widget.session,
        widget.profile.userId,
        badgeId: badgeId,
        note: note,
      );
      if (!mounted) return;
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(s.badgeAwarded)));
      await _load();
    } catch (e) {
      if (!mounted) return;
      final message = e is ApiException && e.code == 'already_awarded'
          ? s.badgeAlreadyAwarded
          : e.toString();
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(message)));
    }
  }

  Future<void> _showAwardDialog() async {
    final s = AppStrings.of(context);
    List<BadgeDefinition> badges;
    try {
      badges = await widget.apiClient.listBadges(widget.session);
    } catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(e.toString())));
      return;
    }
    if (!mounted) return;
    if (badges.isEmpty) {
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(s.noBadges)));
      return;
    }
    await showDialog<void>(
      context: context,
      builder: (dialogContext) => _AwardDialog(
        badges: badges,
        onAward: (badgeId, note) {
          Navigator.of(dialogContext).pop();
          _awardBadge(badgeId, note);
        },
      ),
    );
  }

  Future<void> _endorse(String skill, String note) async {
    final s = AppStrings.of(context);
    try {
      await widget.apiClient.endorsePeer(
        widget.session,
        receiverId: widget.profile.userId,
        skill: skill,
        note: note,
      );
      if (!mounted) return;
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(s.endorsementSaved)));
      await _load();
    } catch (e) {
      if (!mounted) return;
      final message = switch (e) {
        ApiException(code: 'already_endorsed') => s.alreadyEndorsed,
        ApiException(code: 'cannot_endorse_self') => s.cannotEndorseSelf,
        _ => e.toString(),
      };
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(message)));
    }
  }

  Future<void> _showEndorseDialog() async {
    final s = AppStrings.of(context);
    final skills = (_profile ?? widget.profile)
        .skills
        .where((skill) =>
            !_endorsements.any((g) => g.skill == skill && g.endorsedByMe))
        .toList();
    if (skills.isEmpty) {
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(s.endorseSkill)));
      return;
    }
    await showDialog<void>(
      context: context,
      builder: (dialogContext) => _EndorseDialog(
        skills: skills,
        onEndorse: (skill, note) {
          Navigator.of(dialogContext).pop();
          _endorse(skill, note);
        },
      ),
    );
  }

  Future<void> _withdraw(EndorsementGroup group) async {
    final s = AppStrings.of(context);
    try {
      await widget.apiClient
          .withdrawEndorsement(widget.session, group.myEndorsementId);
      if (!mounted) return;
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(s.endorsementWithdrawn)));
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
    final profile = _profile ?? widget.profile;
    return Scaffold(
      appBar: AppBar(title: Text(profile.displayName)),
      body: MaxWidthBox(
        child: _loading
            ? const Center(child: CircularProgressIndicator())
            : ListView(
                padding: const EdgeInsets.all(16),
                children: [
                  Row(
                    children: [
                      CircleAvatar(
                        radius: 32,
                        child: profile.avatarUrl.isNotEmpty
                            ? null
                            : Text(profile.displayName.isNotEmpty
                                ? profile.displayName[0].toUpperCase()
                                : '?'),
                      ),
                      const SizedBox(width: 16),
                      Expanded(
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Text(profile.displayName,
                                style: Theme.of(context)
                                    .textTheme
                                    .titleLarge),
                            if (profile.headline.isNotEmpty)
                              Text(profile.headline),
                            if (profile.isChief)
                              Row(
                                children: [
                                  Icon(Icons.verified,
                                      size: 16,
                                      color: Theme.of(context)
                                          .colorScheme
                                          .primary),
                                  const SizedBox(width: 4),
                                  Text(s.chiefLabel),
                                ],
                              ),
                          ],
                        ),
                      ),
                    ],
                  ),
                  if (profile.bio.isNotEmpty) ...[
                    const SizedBox(height: 16),
                    Text(profile.bio),
                  ],
                  if (profile.location.isNotEmpty ||
                      profile.yearsExperience > 0) ...[
                    const SizedBox(height: 16),
                    Wrap(
                      spacing: 12,
                      children: [
                        if (profile.location.isNotEmpty)
                          _InfoChip(
                            icon: Icons.place_outlined,
                            label: profile.location,
                          ),
                        if (profile.yearsExperience > 0)
                          _InfoChip(
                            icon: Icons.work_outline,
                            label:
                                '${profile.yearsExperience} ${s.yearsShort}',
                          ),
                      ],
                    ),
                  ],
                  if (profile.skills.isNotEmpty) ...[
                    const SizedBox(height: 16),
                    Text(s.skills,
                        style: Theme.of(context).textTheme.titleMedium),
                    const SizedBox(height: 8),
                    Wrap(
                      spacing: 8,
                      runSpacing: 8,
                      children: profile.skills
                          .map((skill) => Chip(label: Text(skill)))
                          .toList(),
                    ),
                  ],
                  if (profile.memberships.isNotEmpty) ...[
                    const SizedBox(height: 16),
                    Text(s.membersLabel,
                        style: Theme.of(context).textTheme.titleMedium),
                    const SizedBox(height: 8),
                    ...profile.memberships.map(
                      (m) => ListTile(
                        contentPadding: EdgeInsets.zero,
                        leading: const Icon(Icons.apartment_outlined),
                        title: Text(m.title.isNotEmpty ? m.title : m.role),
                        subtitle:
                            Text(m.roleLabel(s)),
                      ),
                    ),
                  ],
                  if (_badges.isNotEmpty || _canManage) ...[
                    const SizedBox(height: 16),
                    Row(
                      children: [
                        Expanded(
                          child: Text(s.communityBadges,
                              style: Theme.of(context).textTheme.titleMedium),
                        ),
                        if (_canManage)
                          IconButton(
                            tooltip: s.awardBadge,
                            icon: const Icon(Icons.add_circle_outline),
                            onPressed: _showAwardDialog,
                          ),
                      ],
                    ),
                    const SizedBox(height: 4),
                    if (_badges.isEmpty)
                      Text(s.noBadges,
                          style: Theme.of(context).textTheme.bodySmall)
                    else
                      ..._badges.map(
                        (badge) => ListTile(
                          contentPadding: EdgeInsets.zero,
                          leading: Icon(badgeIconData(badge.icon)),
                          title: Text(badge.name),
                          subtitle: Text([
                            if (badge.note.isNotEmpty) badge.note,
                            if (badge.awardedByName.isNotEmpty)
                              badge.awardedByName,
                          ].join(' · ')),
                          trailing: _canManage
                              ? IconButton(
                                  tooltip: s.badgeRevoked,
                                  icon: const Icon(Icons.close),
                                  onPressed: () => _revokeBadge(badge),
                                )
                              : null,
                        ),
                      ),
                  ],
                  if (_endorsements.isNotEmpty ||
                      (profile.skills.isNotEmpty &&
                          profile.userId != widget.session.userId)) ...[
                    const SizedBox(height: 16),
                    Row(
                      children: [
                        Expanded(
                          child: Text(s.endorsements,
                              style: Theme.of(context).textTheme.titleMedium),
                        ),
                        if (profile.userId != widget.session.userId)
                          OutlinedButton.icon(
                            onPressed: _showEndorseDialog,
                            icon: const Icon(Icons.thumb_up_alt_outlined),
                            label: Text(s.endorseSkill),
                          ),
                      ],
                    ),
                    const SizedBox(height: 4),
                    if (_endorsements.isEmpty)
                      Text(s.noEndorsements,
                          style: Theme.of(context).textTheme.bodySmall)
                    else
                      ..._endorsements.map(
                        (group) => ListTile(
                          contentPadding: EdgeInsets.zero,
                          leading:
                              const Icon(Icons.thumb_up_alt_outlined),
                          title: Text(group.skill),
                          subtitle: group.givers.isEmpty
                              ? null
                              : Text(group.givers
                                  .map((g) => g.giverName)
                                  .join(', ')),
                          trailing: group.endorsedByMe
                              ? TextButton.icon(
                                  onPressed: () => _withdraw(group),
                                  icon: const Icon(Icons.undo, size: 16),
                                  label: Text(s.withdrawEndorsement),
                                )
                              : Chip(label: Text('${group.count}')),
                        ),
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

extension CompanyMemberRoleLabel on CompanyMember {
  String roleLabel(AppStrings s) => switch (role) {
        'owner' => s.roleOwner,
        'manager' => s.roleManager,
        _ => s.roleStaff,
      };
}

/// Manager dialog: pick one of the tenant's active badges to award.
class _AwardDialog extends StatefulWidget {
  const _AwardDialog({required this.badges, required this.onAward});

  final List<BadgeDefinition> badges;
  final void Function(String badgeId, String note) onAward;

  @override
  State<_AwardDialog> createState() => _AwardDialogState();
}

class _AwardDialogState extends State<_AwardDialog> {
  late String _badgeId = widget.badges.first.id;
  final _note = TextEditingController();

  @override
  void dispose() {
    _note.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final s = AppStrings.of(context);
    return AlertDialog(
      title: Text(s.awardBadge),
      content: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          DropdownButtonFormField<String>(
            initialValue: _badgeId,
            decoration: InputDecoration(
              labelText: s.communityBadges,
              border: const OutlineInputBorder(),
            ),
            items: widget.badges
                .map((b) => DropdownMenuItem(
                      value: b.id,
                      child: Text('${b.name} (${b.code})'),
                    ))
                .toList(),
            onChanged: (value) =>
                setState(() => _badgeId = value ?? widget.badges.first.id),
          ),
          const SizedBox(height: 12),
          TextField(
            controller: _note,
            maxLength: 500,
            decoration: InputDecoration(
              labelText: s.badgeNote,
              border: const OutlineInputBorder(),
            ),
          ),
        ],
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.of(context).pop(),
          child: Text(s.cancel),
        ),
        FilledButton(
          onPressed: () => widget.onAward(_badgeId, _note.text.trim()),
          child: Text(s.awardBadge),
        ),
      ],
    );
  }
}

/// Peer dialog: endorse one of this profile's skills.
class _EndorseDialog extends StatefulWidget {
  const _EndorseDialog({required this.skills, required this.onEndorse});

  final List<String> skills;
  final void Function(String skill, String note) onEndorse;

  @override
  State<_EndorseDialog> createState() => _EndorseDialogState();
}

class _EndorseDialogState extends State<_EndorseDialog> {
  late String _skill = widget.skills.first;
  final _note = TextEditingController();

  @override
  void dispose() {
    _note.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final s = AppStrings.of(context);
    return AlertDialog(
      title: Text(s.endorseSkill),
      content: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          DropdownButtonFormField<String>(
            initialValue: _skill,
            decoration: InputDecoration(
              labelText: s.endorseSkillHint,
              border: const OutlineInputBorder(),
            ),
            items: widget.skills
                .map((skill) =>
                    DropdownMenuItem(value: skill, child: Text(skill)))
                .toList(),
            onChanged: (value) =>
                setState(() => _skill = value ?? widget.skills.first),
          ),
          const SizedBox(height: 12),
          TextField(
            controller: _note,
            maxLength: 500,
            decoration: InputDecoration(
              labelText: s.badgeNote,
              border: const OutlineInputBorder(),
            ),
          ),
        ],
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.of(context).pop(),
          child: Text(s.cancel),
        ),
        FilledButton(
          onPressed: () => widget.onEndorse(_skill, _note.text.trim()),
          child: Text(s.endorseAdd),
        ),
      ],
    );
  }
}
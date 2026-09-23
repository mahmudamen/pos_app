import 'package:flutter/material.dart';

import '../../core/api_client.dart';
import '../../core/community.dart';
import '../../core/layout.dart';
import '../../core/session_store.dart';
import '../../l10n/strings.dart';

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
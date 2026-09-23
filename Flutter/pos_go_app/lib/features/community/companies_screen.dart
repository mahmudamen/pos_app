import 'package:flutter/material.dart';

import '../../core/api_client.dart';
import '../../core/community.dart';
import '../../core/layout.dart';
import '../../core/session_store.dart';
import '../../l10n/strings.dart';

class CompaniesScreen extends StatefulWidget {
  const CompaniesScreen({
    super.key,
    required this.session,
    required this.apiClient,
  });

  final Session session;
  final ApiClient apiClient;

  @override
  State<CompaniesScreen> createState() => _CompaniesScreenState();
}

class _CompaniesScreenState extends State<CompaniesScreen> {
  CompaniesPage? _page;
  bool _loading = true;
  String? _error;
  int _currentPage = 1;
  String _query = '';

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
      final result = await widget.apiClient.listCompanies(
        widget.session,
        query: _query,
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

  Future<void> _createCompany(BuildContext sheetContext) async {
    final s = AppStrings.of(context);
    try {
      await widget.apiClient.createCompany(
        widget.session,
        name: _formName.text.trim(),
        description: _formDescription.text.trim(),
        industry: _formIndustry.text.trim(),
        website: _formWebsite.text.trim(),
        city: _formCity.text.trim(),
      );
      _formName.clear();
      _formDescription.clear();
      _formIndustry.clear();
      _formWebsite.clear();
      _formCity.clear();
      if (!sheetContext.mounted) return;
      Navigator.of(sheetContext).pop();
      if (!mounted) return;
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(s.companySaved)));
      await _load();
    } catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(e.toString())));
    }
  }

  final _formName = TextEditingController();
  final _formDescription = TextEditingController();
  final _formIndustry = TextEditingController();
  final _formWebsite = TextEditingController();
  final _formCity = TextEditingController();

  void _showForm() {
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
                Text(s.addCompany,
                    style: Theme.of(sheetContext).textTheme.titleLarge),
                const SizedBox(height: 12),
                TextField(
                  controller: _formName,
                  autofocus: true,
                  decoration: InputDecoration(
                      labelText: s.companyName,
                      border: const OutlineInputBorder()),
                ),
                const SizedBox(height: 12),
                TextField(
                  controller: _formIndustry,
                  decoration: InputDecoration(
                      labelText: s.companyIndustry,
                      border: const OutlineInputBorder()),
                ),
                const SizedBox(height: 12),
                TextField(
                  controller: _formDescription,
                  maxLines: 3,
                  decoration: InputDecoration(
                      labelText: s.companyDescription,
                      border: const OutlineInputBorder()),
                ),
                const SizedBox(height: 12),
                TextField(
                  controller: _formWebsite,
                  decoration: InputDecoration(
                      labelText: s.companyWebsite,
                      border: const OutlineInputBorder()),
                ),
                const SizedBox(height: 12),
                TextField(
                  controller: _formCity,
                  decoration: InputDecoration(
                      labelText: s.companyCity,
                      border: const OutlineInputBorder()),
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
                      onPressed: () => _createCompany(sheetContext),
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

  @override
  void dispose() {
    _formName.dispose();
    _formDescription.dispose();
    _formIndustry.dispose();
    _formWebsite.dispose();
    _formCity.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final s = AppStrings.of(context);
    return Scaffold(
      appBar: AppBar(
        title: Text(s.communityCompanies),
        actions: [
          IconButton(
            icon: const Icon(Icons.add_business_outlined),
            tooltip: s.addCompany,
            onPressed: _showForm,
          ),
        ],
      ),
      body: MaxWidthBox(
        child: Column(
          children: [
            Padding(
              padding: const EdgeInsets.all(8),
              child: TextField(
                decoration: InputDecoration(
                  hintText: s.searchCompanies,
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
    final companies = _page?.companies ?? [];
    final total = _page?.total ?? 0;
    final totalPages = (_page?.limit ?? 50) > 0
        ? (total / (_page?.limit ?? 50)).ceil()
        : 1;

    return Column(
      children: [
        Expanded(
          child: companies.isEmpty
              ? Center(child: Text(s.noCompanies))
              : ListView.separated(
                  itemCount: companies.length,
                  separatorBuilder: (_, __) => const Divider(height: 1),
                  itemBuilder: (context, index) {
                    final company = companies[index];
                    final subtitle = [
                      if (company.industry.isNotEmpty) company.industry,
                      if (company.city.isNotEmpty) company.city,
                      if (company.membersCount > 0)
                        '${company.membersCount} ${s.membersLabel}',
                    ].join(' · ');
                    return ListTile(
                      leading: CircleAvatar(
                        child: company.logoUrl.isNotEmpty
                            ? null
                            : Text(company.name.isNotEmpty
                                ? company.name[0].toUpperCase()
                                : '?'),
                      ),
                      title: Text(company.name),
                      subtitle: subtitle.isEmpty ? null : Text(subtitle),
                      trailing: const Icon(Icons.chevron_right),
                      onTap: () => Navigator.of(context).push(
                        MaterialPageRoute(
                          builder: (_) => CompanyDetailScreen(
                            session: widget.session,
                            apiClient: widget.apiClient,
                            company: company,
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

class CompanyDetailScreen extends StatefulWidget {
  const CompanyDetailScreen({
    super.key,
    required this.session,
    required this.apiClient,
    required this.company,
  });

  final Session session;
  final ApiClient apiClient;
  final Company company;

  @override
  State<CompanyDetailScreen> createState() => _CompanyDetailScreenState();
}

class _CompanyDetailScreenState extends State<CompanyDetailScreen> {
  Company? _company;
  bool _loading = true;
  bool _adding = false;
  final _memberUserId = TextEditingController();
  final _memberRole = TextEditingController();
  final _memberTitle = TextEditingController();

  @override
  void initState() {
    super.initState();
    _load();
  }

  @override
  void dispose() {
    _memberUserId.dispose();
    _memberRole.dispose();
    _memberTitle.dispose();
    super.dispose();
  }

  Future<void> _load() async {
    try {
      final detail = await widget.apiClient.getCompany(
          widget.session, widget.company.id);
      if (!mounted) return;
      setState(() {
        _company = detail;
        _loading = false;
      });
    } catch (e) {
      if (!mounted) return;
      setState(() {
        _company = widget.company;
        _loading = false;
      });
    }
  }

  Future<void> _addMember(BuildContext sheetContext) async {
    final s = AppStrings.of(context);
    final userId = _memberUserId.text.trim();
    if (userId.isEmpty) return;
    setState(() => _adding = true);
    try {
      await widget.apiClient.addCompanyMember(
        widget.session,
        _company?.id ?? widget.company.id,
        userId: userId,
        role: _memberRole.text.trim().isEmpty
            ? 'staff'
            : _memberRole.text.trim().toLowerCase(),
        title: _memberTitle.text.trim(),
      );
      _memberUserId.clear();
      _memberRole.clear();
      _memberTitle.clear();
      if (!sheetContext.mounted) return;
      Navigator.of(sheetContext).pop();
      if (!mounted) return;
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(s.memberAdded)));
      await _load();
    } catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(e.toString())));
    } finally {
      if (mounted) setState(() => _adding = false);
    }
  }

  void _showAddMember() {
    final s = AppStrings.of(context);
    showModalBottomSheet<void>(
      context: context,
      isScrollControlled: true,
      builder: (sheetContext) => Padding(
        padding: EdgeInsets.only(
          bottom: MediaQuery.of(sheetContext).viewInsets.bottom,
        ),
        child: SafeArea(
          child: Padding(
            padding: const EdgeInsets.all(16),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                Text(s.addMember,
                    style: Theme.of(sheetContext).textTheme.titleLarge),
                const SizedBox(height: 12),
                TextField(
                  controller: _memberUserId,
                  autofocus: true,
                  decoration: InputDecoration(
                      labelText: s.userIdLabel,
                      border: const OutlineInputBorder()),
                ),
                const SizedBox(height: 12),
                TextField(
                  controller: _memberRole,
                  decoration: InputDecoration(
                      labelText: s.memberRole,
                      hintText: s.roleStaff,
                      border: const OutlineInputBorder()),
                ),
                const SizedBox(height: 12),
                TextField(
                  controller: _memberTitle,
                  decoration: InputDecoration(
                      labelText: s.memberTitle,
                      border: const OutlineInputBorder()),
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
                      onPressed:
                          _adding ? null : () => _addMember(sheetContext),
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

  Future<void> _removeMember(CompanyMember member) async {
    final s = AppStrings.of(context);
    try {
      await widget.apiClient.removeCompanyMember(
          widget.session, _company?.id ?? widget.company.id, member.userId);
      if (!mounted) return;
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(s.memberRemoved)));
      await _load();
    } catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(e.toString())));
    }
  }

  Future<void> _deleteCompany() async {
    final s = AppStrings.of(context);
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (dialogContext) => AlertDialog(
        title: Text(s.confirmDeleteCompany),
        content: Text(s.confirmDeleteCompanyBody),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(dialogContext).pop(false),
            child: Text(s.cancel),
          ),
          FilledButton(
            onPressed: () => Navigator.of(dialogContext).pop(true),
            child: Text(s.deleteItem),
          ),
        ],
      ),
    );
    if (confirmed != true || !mounted) return;
    try {
      await widget.apiClient.deleteCompany(
          widget.session, _company?.id ?? widget.company.id);
      if (!mounted) return;
      Navigator.of(context).pop();
    } catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(e.toString())));
    }
  }

  @override
  Widget build(BuildContext context) {
    final s = AppStrings.of(context);
    final company = _company ?? widget.company;
    return Scaffold(
      appBar: AppBar(
        title: Text(company.name),
        actions: [
          IconButton(
            icon: const Icon(Icons.person_add_alt_1),
            tooltip: s.addMember,
            onPressed: _showAddMember,
          ),
          IconButton(
            icon: const Icon(Icons.delete_outline),
            tooltip: s.confirmDeleteCompany,
            onPressed: _deleteCompany,
          ),
        ],
      ),
      body: MaxWidthBox(
        child: _loading
            ? const Center(child: CircularProgressIndicator())
            : ListView(
                padding: const EdgeInsets.all(16),
                children: [
                  if (company.description.isNotEmpty) ...[
                    Text(company.description),
                    const SizedBox(height: 12),
                  ],
                  if (company.industry.isNotEmpty ||
                      company.city.isNotEmpty ||
                      company.website.isNotEmpty)
                    Wrap(
                      spacing: 12,
                      runSpacing: 8,
                      children: [
                        if (company.industry.isNotEmpty)
                          Chip(label: Text(company.industry)),
                        if (company.city.isNotEmpty)
                          Chip(label: Text(company.city)),
                        if (company.website.isNotEmpty)
                          Chip(label: Text(company.website)),
                      ],
                    ),
                  const SizedBox(height: 16),
                  Text('${s.membersLabel} (${company.members.length})',
                      style: Theme.of(context).textTheme.titleMedium),
                  const SizedBox(height: 8),
                  if (company.members.isEmpty)
                    Text(s.noMembersYet)
                  else
                    ...company.members.map(
                      (member) => ListTile(
                        contentPadding: EdgeInsets.zero,
                        leading: const Icon(Icons.person_outline),
                        title: Text(member.displayName),
                        subtitle: Text([
                          if (member.title.isNotEmpty) member.title,
                          _roleLabel(s, member.role),
                        ]
                            .join(' · ')),
                        trailing: IconButton(
                          icon: const Icon(Icons.person_remove_outlined),
                          tooltip: s.removeMember,
                          onPressed: () => _removeMember(member),
                        ),
                      ),
                    ),
                ],
              ),
      ),
    );
  }

  static String _roleLabel(AppStrings s, String role) => switch (role) {
        'owner' => s.roleOwner,
        'manager' => s.roleManager,
        _ => s.roleStaff,
      };
}
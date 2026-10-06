import 'package:flutter/material.dart';

import '../../core/api_client.dart';
import '../../core/community.dart';
import '../../core/layout.dart';
import '../../core/session_store.dart';
import '../../l10n/strings.dart';

const _manageRoles = ['owner', 'manager', 'saas_admin'];

String _typeLabel(String type, AppStrings s) => switch (type) {
      'part_time' => s.typePartTime,
      'contract' => s.typeContract,
      'freelance' => s.typeFreelance,
      _ => s.typeFullTime,
    };

String _statusLabel(String status, AppStrings s) => switch (status) {
      'under_review' => s.statusUnderReview,
      'accepted' => s.statusAccepted,
      'rejected' => s.statusRejected,
      _ => s.statusApplied,
    };

class JobsScreen extends StatefulWidget {
  const JobsScreen({
    super.key,
    required this.session,
    required this.apiClient,
  });

  final Session session;
  final ApiClient apiClient;

  @override
  State<JobsScreen> createState() => _JobsScreenState();
}

class _JobsScreenState extends State<JobsScreen> {
  JobsPage? _page;
  bool _loading = true;
  String? _error;
  int _currentPage = 1;
  String _query = '';
  String _type = '';
  bool _mineOnly = false;

  final _formTitle = TextEditingController();
  final _formDescription = TextEditingController();
  final _formLocation = TextEditingController();
  final _formSalary = TextEditingController();
  final _formSkills = TextEditingController();
  String _formType = 'full_time';

  bool get _canWrite => _manageRoles.contains(widget.session.role);

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
      final result = await widget.apiClient.listJobs(
        widget.session,
        query: _query,
        type: _type,
        mine: _mineOnly,
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

  Future<void> _createJob(BuildContext sheetContext) async {
    final s = AppStrings.of(context);
    final title = _formTitle.text.trim();
    if (title.isEmpty) {
      ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('${s.jobTitle}: ${s.required}')));
      return;
    }
    try {
      final salaryMajor = int.tryParse(_formSalary.text.trim()) ?? 0;
      final skills = _formSkills.text
          .split(',')
          .map((e) => e.trim())
          .where((e) => e.isNotEmpty)
          .toList();
      await widget.apiClient.createJob(
        widget.session,
        title: title,
        description: _formDescription.text.trim(),
        employmentType: _formType,
        location: _formLocation.text.trim(),
        salaryMinor: salaryMajor * 100,
        salaryCurrency: widget.session.currencyCode,
        skillTags: skills,
      );
      _formTitle.clear();
      _formDescription.clear();
      _formLocation.clear();
      _formSalary.clear();
      _formSkills.clear();
      if (!sheetContext.mounted) return;
      Navigator.of(sheetContext).pop();
      if (!mounted) return;
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(s.jobSaved)));
      await _load();
    } catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(e.toString())));
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
                Text(s.addJob,
                    style: Theme.of(sheetContext).textTheme.titleLarge),
                const SizedBox(height: 12),
                TextField(
                  controller: _formTitle,
                  autofocus: true,
                  decoration: InputDecoration(
                      labelText: s.jobTitle,
                      border: const OutlineInputBorder()),
                ),
                const SizedBox(height: 12),
                DropdownButtonFormField<String>(
                  initialValue: _formType,
                  decoration: InputDecoration(
                      labelText: s.jobType,
                      border: const OutlineInputBorder()),
                  items: [
                    DropdownMenuItem(
                        value: 'full_time', child: Text(s.typeFullTime)),
                    DropdownMenuItem(
                        value: 'part_time', child: Text(s.typePartTime)),
                    DropdownMenuItem(
                        value: 'contract', child: Text(s.typeContract)),
                    DropdownMenuItem(
                        value: 'freelance', child: Text(s.typeFreelance)),
                  ],
                  onChanged: (value) =>
                      setState(() => _formType = value ?? 'full_time'),
                ),
                const SizedBox(height: 12),
                TextField(
                  controller: _formDescription,
                  maxLines: 3,
                  decoration: InputDecoration(
                      labelText: s.jobDescription,
                      border: const OutlineInputBorder()),
                ),
                const SizedBox(height: 12),
                TextField(
                  controller: _formLocation,
                  decoration: InputDecoration(
                      labelText: s.jobLocation,
                      border: const OutlineInputBorder()),
                ),
                const SizedBox(height: 12),
                TextField(
                  controller: _formSalary,
                  keyboardType: TextInputType.number,
                  decoration: InputDecoration(
                      labelText: s.jobSalary,
                      helperText: widget.session.currencyCode,
                      border: const OutlineInputBorder()),
                ),
                const SizedBox(height: 12),
                TextField(
                  controller: _formSkills,
                  decoration: InputDecoration(
                      labelText: s.jobSkillsHint,
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
                      onPressed: () => _createJob(sheetContext),
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
    _formTitle.dispose();
    _formDescription.dispose();
    _formLocation.dispose();
    _formSalary.dispose();
    _formSkills.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final s = AppStrings.of(context);
    return Scaffold(
      appBar: AppBar(title: Text(s.communityJobs)),
      floatingActionButton: _canWrite
          ? FloatingActionButton(
              onPressed: _showCreateForm,
              tooltip: s.addJob,
              child: const Icon(Icons.add),
            )
          : null,
      body: MaxWidthBox(
        child: Column(
          children: [
            Padding(
              padding: const EdgeInsets.all(8),
              child: Column(
                children: [
                  TextField(
                    decoration: InputDecoration(
                      hintText: s.searchJobs,
                      prefixIcon: const Icon(Icons.search),
                      border: const OutlineInputBorder(),
                      isDense: true,
                    ),
                    onSubmitted: (value) {
                      _query = value.trim();
                      _load();
                    },
                  ),
                  const SizedBox(height: 8),
                  Row(
                    children: [
                      Expanded(
                        child: DropdownButtonFormField<String>(
                          initialValue: _type.isEmpty ? null : _type,
                          isExpanded: true,
                          hint: Text(s.jobType),
                          decoration: const InputDecoration(
                            border: OutlineInputBorder(),
                            isDense: true,
                          ),
                          items: [
                            DropdownMenuItem<String>(
                                value: '', child: Text(s.jobType)),
                            DropdownMenuItem<String>(
                                value: 'full_time',
                                child: Text(s.typeFullTime)),
                            DropdownMenuItem<String>(
                                value: 'part_time',
                                child: Text(s.typePartTime)),
                            DropdownMenuItem<String>(
                                value: 'contract',
                                child: Text(s.typeContract)),
                            DropdownMenuItem<String>(
                                value: 'freelance',
                                child: Text(s.typeFreelance)),
                          ],
                          onChanged: (value) {
                            _type = value ?? '';
                            _load();
                          },
                        ),
                      ),
                      const SizedBox(width: 8),
                      FilterChip(
                        label: Text(s.myApplications),
                        selected: _mineOnly,
                        onSelected: (v) {
                          setState(() => _mineOnly = v);
                          _load();
                        },
                      ),
                    ],
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
    final jobs = _page?.jobs ?? [];
    final total = _page?.total ?? 0;
    final limit = _page?.limit ?? 50;
    final totalPages = limit > 0 ? (total / limit).ceil() : 1;

    return Column(
      children: [
        Expanded(
          child: jobs.isEmpty
              ? Center(child: Text(s.noJobs))
              : ListView.separated(
                  itemCount: jobs.length,
                  separatorBuilder: (_, __) => const Divider(height: 1),
                  itemBuilder: (context, index) {
                    final job = jobs[index];
                    final parts = [
                      if (job.companyName.isNotEmpty) job.companyName,
                      if (job.location.isNotEmpty) job.location,
                      _typeLabel(job.employmentType, s),
                    ];
                    return ListTile(
                      leading: const Icon(Icons.work_outline),
                      title: Row(
                        mainAxisSize: MainAxisSize.min,
                        children: [
                          Flexible(child: Text(job.title)),
                          if (job.applied) ...[
                            const SizedBox(width: 6),
                            Icon(Icons.check_circle,
                                size: 16,
                                color: Theme.of(context).colorScheme.primary),
                          ],
                        ],
                      ),
                      subtitle: Text(parts.join(' · ')),
                      trailing: job.salaryMinor > 0
                          ? Text(
                              s.formatMoney(job.salaryMinor, job.salaryCurrency),
                              textDirection: TextDirection.ltr,
                            )
                          : null,
                      onTap: () => Navigator.of(context).push(
                        MaterialPageRoute(
                          builder: (_) => JobDetailScreen(
                            session: widget.session,
                            apiClient: widget.apiClient,
                            job: job,
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

class JobDetailScreen extends StatefulWidget {
  const JobDetailScreen({
    super.key,
    required this.session,
    required this.apiClient,
    required this.job,
  });

  final Session session;
  final ApiClient apiClient;
  final JobOffer job;

  @override
  State<JobDetailScreen> createState() => _JobDetailScreenState();
}

class _JobDetailScreenState extends State<JobDetailScreen> {
  JobOffer? _job;
  bool _loading = true;
  late bool _applied;
  final _coverNote = TextEditingController();

  bool get _canManage => _manageRoles.contains(widget.session.role);

  @override
  void initState() {
    super.initState();
    _applied = widget.job.applied;
    _load();
  }

  Future<void> _load() async {
    try {
      final detail =
          await widget.apiClient.getJob(widget.session, widget.job.id);
      if (!mounted) return;
      setState(() {
        _job = detail;
        _applied = detail.applied;
        _loading = false;
      });
    } catch (_) {
      if (!mounted) return;
      setState(() {
        _job = widget.job;
        _loading = false;
      });
    }
  }

  Future<void> _apply() async {
    final s = AppStrings.of(context);
    try {
      await widget.apiClient.applyToJob(
        widget.session,
        widget.job.id,
        coverNote: _coverNote.text.trim(),
      );
      _coverNote.clear();
      if (!mounted) return;
      setState(() => _applied = true);
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(s.applicationSent)));
    } on ApiException catch (api) {
      if (!mounted) return;
      if (api.code == 'already_applied') {
        setState(() => _applied = true);
        ScaffoldMessenger.of(context)
            .showSnackBar(SnackBar(content: Text(s.appliedAlready)));
        return;
      }
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(api.toString())));
    } catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(e.toString())));
    }
  }

  void _showApplySheet() {
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
                Text('${s.applyForJob} — ${widget.job.title}',
                    style: Theme.of(sheetContext).textTheme.titleLarge),
                const SizedBox(height: 12),
                TextField(
                  controller: _coverNote,
                  autofocus: true,
                  maxLines: 4,
                  maxLength: 2000,
                  decoration: InputDecoration(
                      labelText: s.coverNote,
                      hintText: s.coverNoteHint,
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
                      onPressed: () {
                        Navigator.of(sheetContext).pop();
                        _apply();
                      },
                      child: Text(s.submitApplication),
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
    _coverNote.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final s = AppStrings.of(context);
    final job = _job ?? widget.job;
    return Scaffold(
      appBar: AppBar(title: Text(job.title)),
      body: MaxWidthBox(
        child: _loading
            ? const Center(child: CircularProgressIndicator())
            : ListView(
                padding: const EdgeInsets.all(16),
                children: [
                  if (job.companyName.isNotEmpty)
                    Text(job.companyName,
                        style: Theme.of(context).textTheme.titleMedium),
                  const SizedBox(height: 12),
                  Wrap(
                    spacing: 8,
                    runSpacing: 8,
                    children: [
                      Chip(
                        avatar: const Icon(Icons.schedule, size: 16),
                        label: Text(_typeLabel(job.employmentType, s)),
                      ),
                      if (job.location.isNotEmpty)
                        Chip(
                          avatar: const Icon(Icons.place_outlined, size: 16),
                          label: Text(job.location),
                        ),
                      if (job.salaryMinor > 0)
                        Chip(
                          avatar: const Icon(Icons.payments_outlined,
                              size: 16),
                          label: Text(s.formatMoney(
                              job.salaryMinor, job.salaryCurrency)),
                        ),
                    ],
                  ),
                  if (job.description.isNotEmpty) ...[
                    const SizedBox(height: 16),
                    Text(job.description),
                  ],
                  if (job.skillTags.isNotEmpty) ...[
                    const SizedBox(height: 16),
                    Text(s.skills,
                        style: Theme.of(context).textTheme.titleMedium),
                    const SizedBox(height: 8),
                    Wrap(
                      spacing: 8,
                      runSpacing: 8,
                      children:
                          job.skillTags.map((t) => Chip(label: Text(t))).toList(),
                    ),
                  ],
                  if (job.closesAt.isNotEmpty) ...[
                    const SizedBox(height: 16),
                    Text('${s.closesAtLabel}: ${job.closesAt}',
                        textDirection: TextDirection.ltr),
                  ],
                  if (_canManage && job.applications > 0) ...[
                    const SizedBox(height: 16),
                    Text('${s.applicants}: ${job.applications}'),
                  ],
                  const SizedBox(height: 24),
                  Row(
                    children: [
                      Expanded(
                        child: _applied
                            ? OutlinedButton.icon(
                                onPressed: null,
                                icon: const Icon(Icons.check_circle_outline),
                                label: Text(s.appliedAlready),
                              )
                            : FilledButton.icon(
                                onPressed: _showApplySheet,
                                icon: const Icon(Icons.send_outlined),
                                label: Text(s.applyForJob),
                              ),
                      ),
                      if (_canManage) ...[
                        const SizedBox(width: 8),
                        OutlinedButton(
                          onPressed: () => Navigator.of(context).push(
                            MaterialPageRoute(
                              builder: (_) => JobApplicantsScreen(
                                session: widget.session,
                                apiClient: widget.apiClient,
                                job: job,
                              ),
                            ),
                          ),
                          child: Text(s.viewApplicants),
                        ),
                      ],
                    ],
                  ),
                ],
              ),
      ),
    );
  }
}

class JobApplicantsScreen extends StatefulWidget {
  const JobApplicantsScreen({
    super.key,
    required this.session,
    required this.apiClient,
    required this.job,
  });

  final Session session;
  final ApiClient apiClient;
  final JobOffer job;

  @override
  State<JobApplicantsScreen> createState() => _JobApplicantsScreenState();
}

class _JobApplicantsScreenState extends State<JobApplicantsScreen> {
  List<JobApplication> _applications = const [];
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
      final result = await widget.apiClient
          .listJobApplications(widget.session, widget.job.id);
      if (!mounted) return;
      setState(() {
        _applications = result.applications;
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

  Future<void> _setStatus(JobApplication app, String status) async {
    final s = AppStrings.of(context);
    try {
      await widget.apiClient
          .updateApplicationStatus(widget.session, app.id, status);
      if (!mounted) return;
      await _load();
    } catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(e.toString())));
      return;
    }
    if (!mounted) return;
    ScaffoldMessenger.of(context)
        .showSnackBar(SnackBar(content: Text(_statusLabel(status, s))));
  }

  @override
  Widget build(BuildContext context) {
    final s = AppStrings.of(context);
    return Scaffold(
      appBar: AppBar(title: Text('${s.applicants} — ${widget.job.title}')),
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
                        FilledButton(
                            onPressed: _load, child: Text(s.retry)),
                      ],
                    ),
                  )
                : _applications.isEmpty
                    ? Center(child: Text(s.noApplicants))
                    : ListView.separated(
                        itemCount: _applications.length,
                        separatorBuilder: (_, __) =>
                            const Divider(height: 1),
                        itemBuilder: (context, index) {
                          final app = _applications[index];
                          return ListTile(
                            leading: CircleAvatar(
                              child: app.displayName.isNotEmpty
                                  ? Text(app.displayName[0].toUpperCase())
                                  : const Icon(Icons.person),
                            ),
                            title: Row(
                              mainAxisSize: MainAxisSize.min,
                              children: [
                                Flexible(child: Text(app.displayName)),
                                const SizedBox(width: 8),
                                _StatusChip(status: app.status),
                              ],
                            ),
                            subtitle: Text(
                              [
                                if (app.headline.isNotEmpty) app.headline,
                                if (app.coverNote.isNotEmpty) app.coverNote,
                              ].join('\n'),
                              maxLines: 3,
                              overflow: TextOverflow.ellipsis,
                            ),
                            trailing: PopupMenuButton<String>(
                              tooltip: s.applicants,
                              onSelected: (value) => _setStatus(app, value),
                              itemBuilder: (menuContext) => [
                                PopupMenuItem(
                                    value: 'applied',
                                    child: Text(s.statusApplied)),
                                PopupMenuItem(
                                    value: 'under_review',
                                    child: Text(s.statusUnderReview)),
                                PopupMenuItem(
                                    value: 'accepted',
                                    child: Text(s.statusAccepted)),
                                PopupMenuItem(
                                    value: 'rejected',
                                    child: Text(s.statusRejected)),
                              ],
                            ),
                          );
                        },
                      ),
      ),
    );
  }
}

class _StatusChip extends StatelessWidget {
  const _StatusChip({required this.status});

  final String status;

  @override
  Widget build(BuildContext context) {
    final s = AppStrings.of(context);
    final scheme = Theme.of(context).colorScheme;
    final (label, color) = switch (status) {
      'accepted' => (s.statusAccepted, scheme.primary),
      'rejected' => (s.statusRejected, scheme.error),
      'under_review' => (s.statusUnderReview, scheme.tertiary),
      _ => (s.statusApplied, scheme.surfaceContainerHighest),
    };
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.15),
        borderRadius: BorderRadius.circular(999),
      ),
      child: Text(label,
          style: TextStyle(
              fontSize: 11, color: status == 'applied' ? null : color)),
    );
  }
}

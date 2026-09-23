import 'package:flutter/material.dart';

import '../../core/api_client.dart';
import '../../core/layout.dart';
import '../../core/session_store.dart';
import '../../l10n/strings.dart';

/// Edit (upsert) the signed-in user's own community profile.
class ProfileEditScreen extends StatefulWidget {
  const ProfileEditScreen({
    super.key,
    required this.session,
    required this.apiClient,
  });

  final Session session;
  final ApiClient apiClient;

  @override
  State<ProfileEditScreen> createState() => _ProfileEditScreenState();
}

class _ProfileEditScreenState extends State<ProfileEditScreen> {
  final TextEditingController _headline = TextEditingController();
  final TextEditingController _bio = TextEditingController();
  final TextEditingController _location = TextEditingController();
  final TextEditingController _years = TextEditingController();
  final TextEditingController _skills = TextEditingController();
  bool _isChief = false;
  bool _loading = true;
  bool _saving = false;
  String? _error;

  @override
  void initState() {
    super.initState();
    _load();
  }

  @override
  void dispose() {
    _headline.dispose();
    _bio.dispose();
    _location.dispose();
    _years.dispose();
    _skills.dispose();
    super.dispose();
  }

  Future<void> _load() async {
    try {
      final profile = await widget.apiClient.myProfile(widget.session);
      if (!mounted) return;
      setState(() {
        _headline.text = profile.headline;
        _bio.text = profile.bio;
        _location.text = profile.location;
        _years.text = profile.yearsExperience > 0
            ? '${profile.yearsExperience}'
            : '';
        _skills.text = profile.skills.join(', ');
        _isChief = profile.isChief;
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

  Future<void> _save() async {
    final s = AppStrings.of(context);
    setState(() => _saving = true);
    try {
      final skills = _skills.text
          .split(',')
          .map((x) => x.trim())
          .where((x) => x.isNotEmpty)
          .toList();
      await widget.apiClient.updateMyProfile(
        widget.session,
        headline: _headline.text.trim(),
        bio: _bio.text.trim(),
        location: _location.text.trim(),
        yearsExperience: int.tryParse(_years.text.trim()) ?? 0,
        isChief: _isChief,
        skills: skills,
      );
      if (!mounted) return;
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(s.profileSaved)));
      Navigator.of(context).pop();
    } catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(e.toString())));
    } finally {
      if (mounted) setState(() => _saving = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final s = AppStrings.of(context);
    return Scaffold(
      appBar: AppBar(
        title: Text(s.myProfile),
        actions: [
          TextButton(
            onPressed: _loading || _saving ? null : _save,
            child: Text(s.save),
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
                        FilledButton(
                            onPressed: _load, child: Text(s.retry)),
                      ],
                    ),
                  )
                : ListView(
                    padding: const EdgeInsets.all(16),
                    children: [
                      TextField(
                        controller: _headline,
                        decoration: InputDecoration(
                          labelText: s.headline,
                          hintText: s.headline,
                          border: const OutlineInputBorder(),
                        ),
                      ),
                      const SizedBox(height: 12),
                      TextField(
                        controller: _bio,
                        maxLines: 3,
                        decoration: InputDecoration(
                          labelText: s.bio,
                          border: const OutlineInputBorder(),
                        ),
                      ),
                      const SizedBox(height: 12),
                      TextField(
                        controller: _location,
                        decoration: InputDecoration(
                          labelText: s.companyCity,
                          border: const OutlineInputBorder(),
                        ),
                      ),
                      const SizedBox(height: 12),
                      TextField(
                        controller: _years,
                        keyboardType: TextInputType.number,
                        decoration: InputDecoration(
                          labelText: s.yearsExperience,
                          border: const OutlineInputBorder(),
                        ),
                      ),
                      const SizedBox(height: 12),
                      TextField(
                        controller: _skills,
                        decoration: InputDecoration(
                          labelText: s.skills,
                          hintText: s.skillsHint,
                          border: const OutlineInputBorder(),
                        ),
                      ),
                      const SizedBox(height: 12),
                      SwitchListTile(
                        title: Text(s.chiefLabel),
                        value: _isChief,
                        onChanged: (v) => setState(() => _isChief = v),
                        contentPadding: EdgeInsets.zero,
                      ),
                      const SizedBox(height: 24),
                      FilledButton.icon(
                        onPressed: _saving ? null : _save,
                        icon: const Icon(Icons.check),
                        label: Text(s.save),
                      ),
                    ],
                  ),
      ),
    );
  }
}
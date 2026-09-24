import 'package:flutter/material.dart';

import '../../core/api_client.dart';
import '../../core/layout.dart';
import '../../core/session_store.dart';
import '../../l10n/strings.dart';
import 'companies_screen.dart';
import 'national_hub_screen.dart';
import 'profile_edit_screen.dart';
import 'profiles_screen.dart';

class CommunityHubScreen extends StatelessWidget {
  const CommunityHubScreen({
    super.key,
    required this.session,
    required this.apiClient,
  });

  final Session session;
  final ApiClient apiClient;

  @override
  Widget build(BuildContext context) {
    final s = AppStrings.of(context);
    return Scaffold(
      appBar: AppBar(
        title: Text(s.community),
        actions: [
          IconButton(
            onPressed: () => Navigator.of(context).push(
              MaterialPageRoute(
                builder: (_) => ProfileEditScreen(
                  session: session,
                  apiClient: apiClient,
                ),
              ),
            ),
            tooltip: s.myProfile,
            icon: const Icon(Icons.badge_outlined),
          ),
        ],
      ),
      body: MaxWidthBox(
        child: Padding(
          padding: const EdgeInsets.all(16),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              _HubTile(
                icon: Icons.public,
                title: s.nationalCommunity,
                subtitle: s.nationalMembership,
                onTap: () => Navigator.of(context).push(
                  MaterialPageRoute(
                    builder: (_) => NationalHubScreen(
                      session: session,
                      apiClient: apiClient,
                    ),
                  ),
                ),
              ),
              const SizedBox(height: 12),
              _HubTile(
                icon: Icons.groups_outlined,
                title: s.communityProfiles,
                subtitle: s.searchProfiles,
                onTap: () => Navigator.of(context).push(
                  MaterialPageRoute(
                    builder: (_) => ProfilesScreen(
                      session: session,
                      apiClient: apiClient,
                    ),
                  ),
                ),
              ),
              const SizedBox(height: 12),
              _HubTile(
                icon: Icons.apartment_outlined,
                title: s.communityCompanies,
                subtitle: s.companyCity,
                onTap: () => Navigator.of(context).push(
                  MaterialPageRoute(
                    builder: (_) => CompaniesScreen(
                      session: session,
                      apiClient: apiClient,
                    ),
                  ),
                ),
              ),
            ],
          ),
        ),
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

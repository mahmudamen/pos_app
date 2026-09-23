import 'package:flutter/foundation.dart';

/// Staff professional profile (community slice 1). Mirrors the Go
/// `StaffProfile` wire shape plus `memberships` (only on the detail endpoint).
@immutable
class StaffProfile {
  const StaffProfile({
    required this.userId,
    required this.displayName,
    required this.email,
    this.headline = '',
    this.bio = '',
    this.avatarUrl = '',
    this.location = '',
    this.yearsExperience = 0,
    this.isChief = false,
    this.skills = const [],
    this.resume = const {},
    this.hasProfile = false,
    this.memberships = const [],
  });

  factory StaffProfile.fromJson(Map<String, dynamic> json) => StaffProfile(
        userId: json['user_id'] as String? ?? '',
        displayName: json['display_name'] as String? ?? '',
        email: json['email'] as String? ?? '',
        headline: json['headline'] as String? ?? '',
        bio: json['bio'] as String? ?? '',
        avatarUrl: json['avatar_url'] as String? ?? '',
        location: json['location'] as String? ?? '',
        yearsExperience: _toInt(json['years_experience']),
        isChief: json['is_chief'] as bool? ?? false,
        skills: (json['skills'] as List<dynamic>? ?? [])
            .map((e) => e as String)
            .toList(),
        resume: (json['resume'] as Map<String, dynamic>?) ?? const {},
        hasProfile: json['has_profile'] as bool? ?? false,
        memberships: (json['memberships'] as List<dynamic>? ?? [])
            .map((e) => CompanyMember.fromJson(e as Map<String, dynamic>))
            .toList(),
      );

  final String userId;
  final String displayName;
  final String email;
  final String headline;
  final String bio;
  final String avatarUrl;
  final String location;
  final int yearsExperience;
  final bool isChief;
  final List<String> skills;
  final Map<String, dynamic> resume;
  final bool hasProfile;
  final List<CompanyMember> memberships;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is StaffProfile &&
          runtimeType == other.runtimeType &&
          userId == other.userId &&
          displayName == other.displayName &&
          email == other.email &&
          headline == other.headline &&
          bio == other.bio &&
          avatarUrl == other.avatarUrl &&
          location == other.location &&
          yearsExperience == other.yearsExperience &&
          isChief == other.isChief &&
          listEquals(skills, other.skills) &&
          mapEquals(resume, other.resume) &&
          hasProfile == other.hasProfile &&
          listEquals(memberships, other.memberships);

  @override
  int get hashCode => Object.hash(
      userId,
      displayName,
      email,
      headline,
      bio,
      avatarUrl,
      location,
      yearsExperience,
      isChief,
      Object.hashAll(skills),
      Object.hashAll(resume.entries),
      hasProfile,
      Object.hashAll(memberships));
}

/// One membership of a staff member in a company.
@immutable
class CompanyMember {
  const CompanyMember({
    required this.id,
    required this.companyId,
    required this.userId,
    required this.displayName,
    required this.role,
    required this.title,
    required this.createdAt,
  });

  factory CompanyMember.fromJson(Map<String, dynamic> json) => CompanyMember(
        id: json['id'] as String? ?? '',
        companyId: json['company_id'] as String? ?? '',
        userId: json['user_id'] as String? ?? '',
        displayName: json['display_name'] as String? ?? '',
        role: json['role'] as String? ?? '',
        title: json['title'] as String? ?? '',
        createdAt: json['created_at'] as String? ?? '',
      );

  final String id;
  final String companyId;
  final String userId;
  final String displayName;
  final String role;
  final String title;
  final String createdAt;

  /// The manager-of-record role; owner/manager holders can manage members.
  bool get canManage => role == 'owner' || role == 'manager';

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is CompanyMember &&
          runtimeType == other.runtimeType &&
          id == other.id &&
          companyId == other.companyId &&
          userId == other.userId &&
          displayName == other.displayName &&
          role == other.role &&
          title == other.title;

  @override
  int get hashCode => Object.hash(id, companyId, userId, displayName, role, title);
}

/// Employer (store's companies), mirroring the Go `Company` wire shape.
@immutable
class Company {
  const Company({
    required this.id,
    required this.name,
    this.slug = '',
    this.description = '',
    this.industry = '',
    this.website = '',
    this.logoUrl = '',
    this.city = '',
    this.isActive = true,
    this.createdAt = '',
    this.membersCount = 0,
    this.members = const [],
  });

  factory Company.fromJson(Map<String, dynamic> json) => Company(
        id: json['id'] as String? ?? '',
        name: json['name'] as String? ?? '',
        slug: json['slug'] as String? ?? '',
        description: json['description'] as String? ?? '',
        industry: json['industry'] as String? ?? '',
        website: json['website'] as String? ?? '',
        logoUrl: json['logo_url'] as String? ?? '',
        city: json['city'] as String? ?? '',
        isActive: json['is_active'] as bool? ?? true,
        createdAt: json['created_at'] as String? ?? '',
        membersCount: _toInt(json['members_count']),
        members: (json['members'] as List<dynamic>? ?? [])
            .map((e) => CompanyMember.fromJson(e as Map<String, dynamic>))
            .toList(),
      );

  final String id;
  final String name;
  final String slug;
  final String description;
  final String industry;
  final String website;
  final String logoUrl;
  final String city;
  final bool isActive;
  final String createdAt;
  final int membersCount;
  final List<CompanyMember> members;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is Company &&
          runtimeType == other.runtimeType &&
          id == other.id &&
          name == other.name &&
          slug == other.slug &&
          description == other.description &&
          industry == other.industry &&
          website == other.website &&
          logoUrl == other.logoUrl &&
          city == other.city &&
          isActive == other.isActive &&
          membersCount == other.membersCount &&
          listEquals(members, other.members);

  @override
  int get hashCode => Object.hash(id, name, slug, description, industry,
      website, logoUrl, city, isActive, membersCount, Object.hashAll(members));
}

class ProfilesPage {
  const ProfilesPage({
    required this.profiles,
    required this.total,
    required this.page,
    required this.limit,
  });

  final List<StaffProfile> profiles;
  final int total;
  final int page;
  final int limit;
}

class CompaniesPage {
  const CompaniesPage({
    required this.companies,
    required this.total,
    required this.page,
    required this.limit,
  });

  final List<Company> companies;
  final int total;
  final int page;
  final int limit;
}

int _toInt(dynamic v) {
  if (v is int) return v;
  if (v is String) return int.tryParse(v) ?? 0;
  return 0;
}
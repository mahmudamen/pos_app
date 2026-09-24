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
  int get hashCode =>
      Object.hash(id, companyId, userId, displayName, role, title);
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

/// A member of the Egypt national community (slice A). Also the shape of
/// `GET /v1/community/national/me`. Mirrors the Go `Member` wire struct.
@immutable
class NationalMember {
  const NationalMember({
    required this.userId,
    required this.displayName,
    this.originTenantId = '',
    this.joinedVia = '',
    this.invitedBy = '',
    this.invitedByName = '',
    this.role = 'member',
    this.status = 'active',
    this.level = 'bronze',
    this.expertiseScore = 0,
    this.joinedAt = '',
  });

  factory NationalMember.fromJson(Map<String, dynamic> json) => NationalMember(
        userId: json['user_id'] as String? ?? '',
        displayName: json['display_name'] as String? ?? '',
        originTenantId: json['origin_tenant_id'] as String? ?? '',
        joinedVia: json['joined_via'] as String? ?? '',
        invitedBy: json['invited_by'] as String? ?? '',
        invitedByName: json['invited_by_name'] as String? ?? '',
        role: json['role'] as String? ?? 'member',
        status: json['status'] as String? ?? 'active',
        level: json['level'] as String? ?? 'bronze',
        expertiseScore: _toInt(json['expertise_score']),
        joinedAt: json['joined_at'] as String? ?? '',
      );

  final String userId;
  final String displayName;
  final String originTenantId;
  final String joinedVia;
  final String invitedBy;
  final String invitedByName;
  final String role;
  final String status;
  final String level;
  final int expertiseScore;
  final String joinedAt;

  bool get isActive => status == 'active';

  /// Holds the national `community.moderate` permission dynamically.
  bool get isModerator => role == 'moderator' || role == 'admin';

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is NationalMember &&
          runtimeType == other.runtimeType &&
          userId == other.userId &&
          displayName == other.displayName &&
          originTenantId == other.originTenantId &&
          joinedVia == other.joinedVia &&
          invitedBy == other.invitedBy &&
          invitedByName == other.invitedByName &&
          role == other.role &&
          status == other.status &&
          level == other.level &&
          expertiseScore == other.expertiseScore &&
          joinedAt == other.joinedAt;

  @override
  int get hashCode => Object.hash(
      userId,
      displayName,
      originTenantId,
      joinedVia,
      invitedBy,
      invitedByName,
      role,
      status,
      level,
      expertiseScore,
      joinedAt);
}

/// One national community invitation (slice A). `code` is only present on the
/// row returned by `POST /v1/community/national/invitations` (the code is
/// hashed at rest; the plaintext is never stored).
@immutable
class NationalInvitation {
  const NationalInvitation({
    required this.id,
    this.code = '',
    this.inviterId = '',
    this.email = '',
    this.note = '',
    this.maxUses = 1,
    this.usedCount = 0,
    this.status = 'active',
    this.expiresAt = '',
    this.createdAt = '',
  });

  factory NationalInvitation.fromJson(Map<String, dynamic> json) =>
      NationalInvitation(
        id: json['id'] as String? ?? '',
        code: json['code'] as String? ?? '',
        inviterId: json['inviter_id'] as String? ?? '',
        email: json['email'] as String? ?? '',
        note: json['note'] as String? ?? '',
        maxUses: _toInt(json['max_uses']),
        usedCount: _toInt(json['used_count']),
        status: json['status'] as String? ?? 'active',
        expiresAt: json['expires_at'] as String? ?? '',
        createdAt: json['created_at'] as String? ?? '',
      );

  final String id;
  final String code;
  final String inviterId;
  final String email;
  final String note;
  final int maxUses;
  final int usedCount;
  final String status;
  final String expiresAt;
  final String createdAt;

  bool get isActive => status == 'active';

  /// `EG`-prefixed display form of the code; the canonical stored form has the
  /// prefix stripped.
  String get codeDisplay => code.startsWith('EG-') ? code : 'EG-$code';

  int get remainingUses => isActive ? (maxUses - usedCount) : 0;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is NationalInvitation &&
          runtimeType == other.runtimeType &&
          id == other.id &&
          code == other.code &&
          inviterId == other.inviterId &&
          email == other.email &&
          note == other.note &&
          maxUses == other.maxUses &&
          usedCount == other.usedCount &&
          status == other.status &&
          expiresAt == other.expiresAt &&
          createdAt == other.createdAt;

  @override
  int get hashCode => Object.hash(id, code, inviterId, email, note, maxUses,
      usedCount, status, expiresAt, createdAt);
}

class NationalMembersPage {
  const NationalMembersPage({
    required this.members,
    required this.total,
    required this.page,
    required this.limit,
  });

  final List<NationalMember> members;
  final int total;
  final int page;
  final int limit;
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

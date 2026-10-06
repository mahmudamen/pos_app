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

/// A job posting (community slice 2), mirroring the Go `JobOffer` wire shape.
@immutable
class JobOffer {
  const JobOffer({
    required this.id,
    required this.title,
    this.companyId = '',
    this.companyName = '',
    this.description = '',
    this.employmentType = 'full_time',
    this.location = '',
    this.salaryMinor = 0,
    this.salaryCurrency = 'EGP',
    this.skillTags = const [],
    this.isActive = true,
    this.closesAt = '',
    this.createdBy = '',
    this.createdAt = '',
    this.updatedAt = '',
    this.applied = false,
    this.applications = 0,
  });

  factory JobOffer.fromJson(Map<String, dynamic> json) => JobOffer(
        id: json['id'] as String? ?? '',
        title: json['title'] as String? ?? '',
        companyId: json['company_id'] as String? ?? '',
        companyName: json['company_name'] as String? ?? '',
        description: json['description'] as String? ?? '',
        employmentType: json['employment_type'] as String? ?? 'full_time',
        location: json['location'] as String? ?? '',
        salaryMinor: _toInt(json['salary_minor']),
        salaryCurrency: json['salary_currency'] as String? ?? 'EGP',
        skillTags: (json['skill_tags'] as List<dynamic>? ?? [])
            .map((e) => e.toString())
            .toList(),
        isActive: json['is_active'] as bool? ?? true,
        closesAt: json['closes_at'] as String? ?? '',
        createdBy: json['created_by'] as String? ?? '',
        createdAt: json['created_at'] as String? ?? '',
        updatedAt: json['updated_at'] as String? ?? '',
        applied: json['applied'] as bool? ?? false,
        applications: _toInt(json['applications']),
      );

  final String id;
  final String title;
  final String companyId;
  final String companyName;
  final String description;
  final String employmentType;
  final String location;
  final int salaryMinor;
  final String salaryCurrency;
  final List<String> skillTags;
  final bool isActive;
  final String closesAt;
  final String createdBy;
  final String createdAt;
  final String updatedAt;
  final bool applied;
  final int applications;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is JobOffer &&
          runtimeType == other.runtimeType &&
          id == other.id &&
          title == other.title &&
          companyId == other.companyId &&
          companyName == other.companyName &&
          description == other.description &&
          employmentType == other.employmentType &&
          location == other.location &&
          salaryMinor == other.salaryMinor &&
          salaryCurrency == other.salaryCurrency &&
          listEquals(skillTags, other.skillTags) &&
          isActive == other.isActive &&
          closesAt == other.closesAt &&
          createdBy == other.createdBy &&
          createdAt == other.createdAt &&
          updatedAt == other.updatedAt &&
          applied == other.applied &&
          applications == other.applications;

  @override
  int get hashCode => Object.hash(
      id,
      title,
      companyId,
      companyName,
      description,
      employmentType,
      location,
      salaryMinor,
      salaryCurrency,
      isActive,
      closesAt,
      createdBy,
      createdAt,
      updatedAt,
      applied,
      applications,
      Object.hashAll(skillTags));
}

/// One application to a job (community slice 2), mirroring the Go
/// `JobApplication` wire struct.
@immutable
class JobApplication {
  const JobApplication({
    required this.id,
    required this.jobOfferId,
    required this.applicantId,
    this.jobTitle = '',
    this.displayName = '',
    this.headline = '',
    this.coverNote = '',
    this.resume = const {},
    this.status = 'applied',
    this.createdAt = '',
  });

  factory JobApplication.fromJson(Map<String, dynamic> json) => JobApplication(
        id: json['id'] as String? ?? '',
        jobOfferId: json['job_offer_id'] as String? ?? '',
        jobTitle: json['job_title'] as String? ?? '',
        applicantId: json['applicant_id'] as String? ?? '',
        displayName: json['display_name'] as String? ?? '',
        headline: json['headline'] as String? ?? '',
        coverNote: json['cover_note'] as String? ?? '',
        resume: (json['resume'] as Map<String, dynamic>?) ?? const {},
        status: json['status'] as String? ?? 'applied',
        createdAt: json['created_at'] as String? ?? '',
      );

  final String id;
  final String jobOfferId;
  final String jobTitle;
  final String applicantId;
  final String displayName;
  final String headline;
  final String coverNote;
  final Map<String, dynamic> resume;
  final String status;
  final String createdAt;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is JobApplication &&
          runtimeType == other.runtimeType &&
          id == other.id &&
          jobOfferId == other.jobOfferId &&
          jobTitle == other.jobTitle &&
          applicantId == other.applicantId &&
          displayName == other.displayName &&
          headline == other.headline &&
          coverNote == other.coverNote &&
          status == other.status &&
          createdAt == other.createdAt;

  @override
  int get hashCode => Object.hash(id, jobOfferId, jobTitle, applicantId,
      displayName, headline, coverNote, status, createdAt);
}

class JobsPage {
  const JobsPage({
    required this.jobs,
    required this.total,
    required this.page,
    required this.limit,
  });

  final List<JobOffer> jobs;
  final int total;
  final int page;
  final int limit;
}

class JobApplicationsPage {
  const JobApplicationsPage({
    required this.applications,
    required this.appliedTotal,
  });

  final List<JobApplication> applications;
  final int appliedTotal;
}

/// A tenant badge in the catalog (community slice 3), mirroring the Go
/// `BadgeDefinition` wire struct.
@immutable
class BadgeDefinition {
  const BadgeDefinition({
    required this.id,
    required this.code,
    required this.name,
    this.description = '',
    this.icon = 'military_tech',
    this.criteria = const {},
    this.isActive = true,
    this.createdAt = '',
    this.earnedCount = 0,
  });

  factory BadgeDefinition.fromJson(Map<String, dynamic> json) =>
      BadgeDefinition(
        id: json['id'] as String? ?? '',
        code: json['code'] as String? ?? '',
        name: json['name'] as String? ?? '',
        description: json['description'] as String? ?? '',
        icon: json['icon'] as String? ?? 'military_tech',
        criteria: (json['criteria'] as Map<String, dynamic>?) ?? const {},
        isActive: json['is_active'] as bool? ?? true,
        createdAt: json['created_at'] as String? ?? '',
        earnedCount: _toInt(json['earned_count']),
      );

  final String id;
  final String code;
  final String name;
  final String description;
  final String icon;
  final Map<String, dynamic> criteria;
  final bool isActive;
  final String createdAt;
  final int earnedCount;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is BadgeDefinition &&
          runtimeType == other.runtimeType &&
          id == other.id &&
          code == other.code &&
          name == other.name &&
          description == other.description &&
          icon == other.icon &&
          isActive == other.isActive &&
          createdAt == other.createdAt &&
          earnedCount == other.earnedCount;

  @override
  int get hashCode => Object.hash(
      id, code, name, description, icon, isActive, createdAt, earnedCount);
}

/// One badge awarded to one user (community slice 3), mirroring the Go
/// `UserBadge` wire struct.
@immutable
class UserBadge {
  const UserBadge({
    required this.badgeId,
    required this.code,
    required this.name,
    this.description = '',
    this.icon = 'military_tech',
    this.note = '',
    this.awardedAt = '',
    this.awardedByName = '',
  });

  factory UserBadge.fromJson(Map<String, dynamic> json) => UserBadge(
        badgeId: json['badge_id'] as String? ?? '',
        code: json['code'] as String? ?? '',
        name: json['name'] as String? ?? '',
        description: json['description'] as String? ?? '',
        icon: json['icon'] as String? ?? 'military_tech',
        note: json['note'] as String? ?? '',
        awardedAt: json['awarded_at'] as String? ?? '',
        awardedByName: json['awarded_by_name'] as String? ?? '',
      );

  final String badgeId;
  final String code;
  final String name;
  final String description;
  final String icon;
  final String note;
  final String awardedAt;
  final String awardedByName;
}

/// One peer inside an endorsement group (community slice 3).
@immutable
class EndorsementGiver {
  const EndorsementGiver({
    required this.id,
    required this.giverId,
    required this.giverName,
    this.note = '',
    this.createdAt = '',
  });

  factory EndorsementGiver.fromJson(Map<String, dynamic> json) =>
      EndorsementGiver(
        id: json['id'] as String? ?? '',
        giverId: json['giver_id'] as String? ?? '',
        giverName: json['giver_name'] as String? ?? '',
        note: json['note'] as String? ?? '',
        createdAt: json['created_at'] as String? ?? '',
      );

  final String id;
  final String giverId;
  final String giverName;
  final String note;
  final String createdAt;
}

/// One skill's endorsement tally for a receiver (community slice 3),
/// mirroring the Go `EndorsementGroup` wire struct.
@immutable
class EndorsementGroup {
  const EndorsementGroup({
    required this.skill,
    required this.count,
    this.givers = const [],
    this.myEndorsementId = '',
  });

  factory EndorsementGroup.fromJson(Map<String, dynamic> json) =>
      EndorsementGroup(
        skill: json['skill'] as String? ?? '',
        count: _toInt(json['count']),
        givers: (json['givers'] as List<dynamic>? ?? [])
            .map((e) => EndorsementGiver.fromJson(e as Map<String, dynamic>))
            .toList(),
        myEndorsementId: json['my_endorsement_id'] as String? ?? '',
      );

  final String skill;
  final int count;
  final List<EndorsementGiver> givers;

  /// Non-empty when the viewer already endorsed this skill (their own
  /// endorsement's id — used to offer a withdraw action).
  final String myEndorsementId;

  bool get endorsedByMe => myEndorsementId.isNotEmpty;
}

/// A single endorsement as returned from create (community slice 3).
@immutable
class Endorsement {
  const Endorsement({
    required this.id,
    required this.skill,
    required this.receiverId,
    this.note = '',
    this.giverId = '',
    this.giverName = '',
    this.createdAt = '',
  });

  factory Endorsement.fromJson(Map<String, dynamic> json) => Endorsement(
        id: json['id'] as String? ?? '',
        skill: json['skill'] as String? ?? '',
        note: json['note'] as String? ?? '',
        giverId: json['giver_id'] as String? ?? '',
        giverName: json['giver_name'] as String? ?? '',
        receiverId: json['receiver_id'] as String? ?? '',
        createdAt: json['created_at'] as String? ?? '',
      );

  final String id;
  final String skill;
  final String note;
  final String giverId;
  final String giverName;
  final String receiverId;
  final String createdAt;
}

/// One board (category) in the staff forum.
class ForumCategory {
  const ForumCategory({
    required this.id,
    required this.name,
    this.slug = '',
    this.description = '',
    this.sortOrder = 0,
    this.isActive = true,
    this.postCount = 0,
    this.createdAt = '',
  });

  final String id;
  final String name;
  final String slug;
  final String description;
  final int sortOrder;
  final bool isActive;
  final int postCount;
  final String createdAt;

  factory ForumCategory.fromJson(Map<String, dynamic> json) => ForumCategory(
        id: json['id'] as String? ?? '',
        name: json['name'] as String? ?? '',
        slug: json['slug'] as String? ?? '',
        description: json['description'] as String? ?? '',
        sortOrder: _toInt(json['sort_order']),
        isActive: json['is_active'] as bool? ?? true,
        postCount: _toInt(json['post_count']),
        createdAt: json['created_at'] as String? ?? '',
      );
}

/// One thread in the staff forum, with the list-view aggregates the client
/// renders inline (counts, viewer state, moderation flags).
class ForumPost {
  const ForumPost({
    required this.id,
    required this.categoryId,
    this.categoryName = '',
    this.authorId = '',
    this.authorName = '',
    required this.title,
    this.body = '',
    this.isPinned = false,
    this.isClosed = false,
    this.commentCount = 0,
    this.likeCount = 0,
    this.likedByMe = false,
    this.isAuthor = false,
    this.createdAt = '',
    this.updatedAt = '',
  });

  final String id;
  final String categoryId;
  final String categoryName;
  final String authorId;
  final String authorName;
  final String title;
  final String body;
  final bool isPinned;
  final bool isClosed;
  final int commentCount;
  final int likeCount;
  final bool likedByMe;
  final bool isAuthor;
  final String createdAt;
  final String updatedAt;

  ForumPost copyWith({
    bool? isPinned,
    bool? isClosed,
    bool? likedByMe,
    int? likeCount,
    int? commentCount,
    String? title,
    String? body,
  }) =>
      ForumPost(
        id: id,
        categoryId: categoryId,
        categoryName: categoryName,
        authorId: authorId,
        authorName: authorName,
        title: title ?? this.title,
        body: body ?? this.body,
        isPinned: isPinned ?? this.isPinned,
        isClosed: isClosed ?? this.isClosed,
        commentCount: commentCount ?? this.commentCount,
        likeCount: likeCount ?? this.likeCount,
        likedByMe: likedByMe ?? this.likedByMe,
        isAuthor: isAuthor,
        createdAt: createdAt,
        updatedAt: updatedAt,
      );

  factory ForumPost.fromJson(Map<String, dynamic> json) => ForumPost(
        id: json['id'] as String? ?? '',
        categoryId: json['category_id'] as String? ?? '',
        categoryName: json['category_name'] as String? ?? '',
        authorId: json['author_id'] as String? ?? '',
        authorName: json['author_name'] as String? ?? '',
        title: json['title'] as String? ?? '',
        body: json['body'] as String? ?? '',
        isPinned: json['is_pinned'] as bool? ?? false,
        isClosed: json['is_closed'] as bool? ?? false,
        commentCount: _toInt(json['comment_count']),
        likeCount: _toInt(json['like_count']),
        likedByMe: json['liked_by_me'] as bool? ?? false,
        isAuthor: json['is_author'] as bool? ?? false,
        createdAt: json['created_at'] as String? ?? '',
        updatedAt: json['updated_at'] as String? ?? '',
      );
}

/// One reply under a thread.
class ForumComment {
  const ForumComment({
    required this.id,
    required this.postId,
    this.authorId = '',
    this.authorName = '',
    required this.body,
    this.createdAt = '',
  });

  final String id;
  final String postId;
  final String authorId;
  final String authorName;
  final String body;
  final String createdAt;

  factory ForumComment.fromJson(Map<String, dynamic> json) => ForumComment(
        id: json['id'] as String? ?? '',
        postId: json['post_id'] as String? ?? '',
        authorId: json['author_id'] as String? ?? '',
        authorName: json['author_name'] as String? ?? '',
        body: json['body'] as String? ?? '',
        createdAt: json['created_at'] as String? ?? '',
      );
}

/// Thread detail: the post plus its comment thread (the backend returns
/// `comments` alongside `data`, not nested inside it).
class ForumPostDetail {
  const ForumPostDetail({required this.post, required this.comments});

  final ForumPost post;
  final List<ForumComment> comments;
}

/// Paged board listing.
class ForumCategoriesPage {
  const ForumCategoriesPage({required this.categories, required this.total});

  final List<ForumCategory> categories;
  final int total;
}

/// Paged thread listing with the usual meta bounds.
class ForumPostsPage {
  const ForumPostsPage({
    required this.posts,
    required this.total,
    required this.page,
    required this.limit,
  });

  final List<ForumPost> posts;
  final int total;
  final int page;
  final int limit;
}

/// Result of the idempotent like toggle.
class ForumLikeResult {
  const ForumLikeResult({required this.liked, required this.likeCount});

  final bool liked;
  final int likeCount;
}

int _toInt(dynamic v) {
  if (v is int) return v;
  if (v is String) return int.tryParse(v) ?? 0;
  return 0;
}

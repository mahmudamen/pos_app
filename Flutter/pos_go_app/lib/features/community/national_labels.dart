import '../../l10n/strings.dart';

String nationalLevelLabel(AppStrings s, String level) => switch (level) {
      'platinum' => s.levelPlatinum,
      'gold' => s.levelGold,
      'silver' => s.levelSilver,
      _ => s.levelBronze,
    };

String nationalStatusLabel(AppStrings s, String status) => switch (status) {
      'suspended' => s.statusSuspended,
      'left' => s.statusLeft,
      _ => s.statusActive,
    };

String nationalRoleLabel(AppStrings s, String role) => switch (role) {
      'admin' => s.roleAdmin,
      'moderator' => s.roleModerator,
      _ => s.roleMember,
    };

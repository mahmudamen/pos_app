import 'dart:io';

import 'package:flutter_test/flutter_test.dart';
import 'package:pos_go_app/core/telemetry.dart';

/// Guards the one piece of state that is duplicated in this project.
///
/// The app version lives in two places by necessity: `pubspec.yaml`, which
/// Gradle reads to stamp the manifest, and `Telemetry.buildVersion`, which is a
/// `const` string sent as `app_version` on every client event. Dart has no
/// build-time way to read pubspec, so one of the two has to be hand-maintained.
///
/// It has already drifted: the repo docs recorded `0.1.0+1` while pubspec was at
/// `1.0.0+3`, and a stale `buildVersion` sends wrong data to
/// `POST /v1/client/events`, where it is what you would use to tell build 3 from
/// build 4 in a bug report. So it is worth a test rather than a comment.
void main() {
  test('Telemetry.buildVersion matches pubspec version', () {
    final pubspec = File('pubspec.yaml');
    expect(pubspec.existsSync(), isTrue, reason: 'run from the package root');

    final match = RegExp(
      r'^version:\s*(\S+)\s*$',
      multiLine: true,
    ).firstMatch(pubspec.readAsStringSync());
    expect(match, isNotNull, reason: 'no version: line in pubspec.yaml');

    expect(
      Telemetry.buildVersion,
      match!.group(1),
      reason:
          'Telemetry.buildVersion is out of sync with pubspec.yaml. Bump both, '
          'or every client_event will report the wrong build.',
    );
  });
}

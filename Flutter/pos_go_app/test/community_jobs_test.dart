import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:pos_go_app/core/api_client.dart';
import 'package:pos_go_app/core/community.dart';
import 'package:pos_go_app/core/session_store.dart';
import 'package:pos_go_app/features/community/community_hub_screen.dart';
import 'package:pos_go_app/features/community/jobs_screen.dart';
import 'package:pos_go_app/l10n/strings.dart';

const _manager = Session(
  accessToken: 'access-token',
  refreshToken: 'refresh-token',
  userId: 'user-manager',
  displayName: 'Store Manager',
  tenantId: 'tenant-1',
  deviceId: 'device-1',
  role: 'manager',
  currencyCode: 'EGP',
);

const _cashier = Session(
  accessToken: 'access-token',
  refreshToken: 'refresh-token',
  userId: 'user-cashier',
  displayName: 'Cashier One',
  tenantId: 'tenant-1',
  deviceId: 'device-1',
  role: 'cashier',
  currencyCode: 'EGP',
);

const _headChefJob = JobOffer(
  id: 'job-1',
  title: 'Head Chef',
  companyId: 'company-1',
  companyName: 'Bakery X',
  description: 'Grill master needed',
  employmentType: 'full_time',
  location: 'Cairo',
  salaryMinor: 6500000,
  salaryCurrency: 'EGP',
  skillTags: ['grill', 'sous'],
  applications: 2,
);

const _lineCookJob = JobOffer(
  id: 'job-2',
  title: 'Line Cook',
  employmentType: 'part_time',
  applied: true,
);

class _FakeJobsApi extends ApiClient {
  String? lastQuery;
  String lastType = '';
  bool lastMine = false;
  String? lastCoverNote;
  String? lastStatus;

  @override
  Future<JobsPage> listJobs(
    Session session, {
    String query = '',
    String companyId = '',
    String type = '',
    bool mine = false,
    int page = 1,
    int limit = 50,
  }) async {
    lastQuery = query;
    lastType = type;
    lastMine = mine;
    return JobsPage(
      jobs: const [_headChefJob, _lineCookJob],
      total: 2,
      page: page,
      limit: limit,
    );
  }

  @override
  Future<JobOffer> getJob(Session session, String jobId) async =>
      jobId == _lineCookJob.id ? _lineCookJob : _headChefJob;

  @override
  Future<JobApplication> applyToJob(
    Session session,
    String jobId, {
    String coverNote = '',
  }) async {
    lastCoverNote = coverNote;
    return JobApplication(
      id: 'app-9',
      jobOfferId: jobId,
      applicantId: 'user-manager',
      coverNote: coverNote,
      status: 'applied',
    );
  }

  @override
  Future<JobApplicationsPage> listJobApplications(
      Session session, String jobId) async {
    return const JobApplicationsPage(
      applications: [
        JobApplication(
          id: 'app-1',
          jobOfferId: 'job-1',
          applicantId: 'user-2',
          displayName: 'Omar Cook',
          headline: 'Line Cook',
          coverNote: 'Hire me please',
          status: 'under_review',
        ),
      ],
      appliedTotal: 1,
    );
  }

  @override
  Future<void> updateApplicationStatus(
      Session session, String applicationId, String status) async {
    lastStatus = status;
  }
}

Widget _wrap(Widget child) {
  return MaterialApp(
    locale: const Locale('en'),
    supportedLocales: AppStrings.supportedLocales,
    localizationsDelegates: const [
      AppStrings.delegate,
      GlobalMaterialLocalizations.delegate,
      GlobalWidgetsLocalizations.delegate,
      GlobalCupertinoLocalizations.delegate,
    ],
    home: child,
  );
}

void main() {
  testWidgets('hub shows the jobs entry and lists jobs', (tester) async {
    await tester.pumpWidget(_wrap(CommunityHubScreen(
        session: _manager, apiClient: _FakeJobsApi())));
    await tester.pumpAndSettle();

    expect(find.text('Jobs'), findsOneWidget);
    expect(find.text('Open roles and applications'), findsOneWidget);

    await tester.tap(find.text('Jobs'));
    await tester.pumpAndSettle();

    expect(find.byType(JobsScreen), findsOneWidget);
    expect(find.text('Head Chef'), findsOneWidget);
    expect(find.text('Line Cook'), findsOneWidget);
    expect(find.text('Bakery X · Cairo · Full-time'), findsOneWidget);
    expect(find.text('E£65000.00'), findsOneWidget);
    expect(find.byIcon(Icons.check_circle), findsOneWidget);
  });

  testWidgets('search and my-applications filter reach the API',
      (tester) async {
    final api = _FakeJobsApi();
    await tester.pumpWidget(_wrap(JobsScreen(session: _manager, apiClient: api)));
    await tester.pumpAndSettle();

    await tester.enterText(find.byType(TextField).first, 'chef');
    await tester.testTextInput.receiveAction(TextInputAction.done);
    await tester.pumpAndSettle();
    expect(api.lastQuery, 'chef');

    await tester.tap(find.text('My applications'));
    await tester.pumpAndSettle();
    expect(api.lastMine, isTrue);
  });

  testWidgets('apply flow sends the cover note and flips to applied',
      (tester) async {
    final api = _FakeJobsApi();
    await tester.pumpWidget(
        _wrap(JobDetailScreen(session: _manager, apiClient: api, job: _headChefJob)));
    await tester.pumpAndSettle();

    expect(find.widgetWithText(FilledButton, 'Apply'), findsOneWidget);

    await tester.tap(find.widgetWithText(FilledButton, 'Apply'));
    await tester.pumpAndSettle();

    await tester.enterText(find.byType(TextField).first, 'I can grill');
    await tester.tap(find.widgetWithText(FilledButton, 'Submit application'));
    await tester.pumpAndSettle();

    expect(api.lastCoverNote, 'I can grill');
    expect(find.widgetWithText(OutlinedButton, 'Already applied'),
        findsOneWidget);
    expect(find.text('Application sent'), findsOneWidget);
    expect(find.text('View applicants'), findsOneWidget);

    await tester.pump(const Duration(seconds: 5));
    await tester.pumpAndSettle();
  });

  testWidgets('an applied job shows the already-applied state',
      (tester) async {
    final api = _FakeJobsApi();
    await tester.pumpWidget(
        _wrap(JobDetailScreen(session: _manager, apiClient: api, job: _lineCookJob)));
    await tester.pumpAndSettle();

    expect(
        find.widgetWithText(OutlinedButton, 'Already applied'), findsOneWidget);
    expect(find.widgetWithText(FilledButton, 'Apply'), findsNothing);
  });

  testWidgets('manager changes an application status from the applicants screen',
      (tester) async {
    final api = _FakeJobsApi();
    await tester.pumpWidget(_wrap(JobDetailScreen(
        session: _manager, apiClient: api, job: _headChefJob)));
    await tester.pumpAndSettle();

    await tester.tap(find.widgetWithText(OutlinedButton, 'View applicants'));
    await tester.pumpAndSettle();

    expect(find.byType(JobApplicantsScreen), findsOneWidget);
    expect(find.text('Omar Cook'), findsOneWidget);
    expect(find.textContaining('Hire me please'), findsOneWidget);
    expect(find.text('Under review'), findsOneWidget);

    await tester.tap(find.byType(PopupMenuButton<String>));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Accepted'));
    await tester.pumpAndSettle();

    expect(api.lastStatus, 'accepted');

    await tester.pump(const Duration(seconds: 5));
    await tester.pumpAndSettle();
  });

  testWidgets('cashier does not see the post-a-job action', (tester) async {
    await tester.pumpWidget(
        _wrap(JobsScreen(session: _cashier, apiClient: _FakeJobsApi())));
    await tester.pumpAndSettle();

    expect(find.byType(FloatingActionButton), findsNothing);
    expect(find.text('Head Chef'), findsOneWidget);
  });

  testWidgets('manager sees the post-a-job action', (tester) async {
    await tester.pumpWidget(
        _wrap(JobsScreen(session: _manager, apiClient: _FakeJobsApi())));
    await tester.pumpAndSettle();

    expect(find.byType(FloatingActionButton), findsOneWidget);
  });
}

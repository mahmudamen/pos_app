import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:pos_go_app/core/api_client.dart';
import 'package:pos_go_app/core/community.dart';
import 'package:pos_go_app/core/session_store.dart';
import 'package:pos_go_app/features/community/community_hub_screen.dart';
import 'package:pos_go_app/features/community/forum_screen.dart';
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

const _board = ForumCategory(
  id: 'cat-1',
  name: 'Kitchen Talk',
  slug: 'kitchen-talk',
  postCount: 2,
);

const _oldBoard = ForumCategory(
  id: 'cat-2',
  name: 'Old Board',
  isActive: false,
);

const _thread = ForumPost(
  id: 'post-1',
  categoryId: 'cat-1',
  categoryName: 'Kitchen Talk',
  authorId: 'user-cashier',
  authorName: 'Cashier One',
  title: 'Grill station tips',
  body: 'Preheat before service.',
  commentCount: 1,
);

const _pinnedThread = ForumPost(
  id: 'post-2',
  categoryId: 'cat-1',
  categoryName: 'Kitchen Talk',
  authorId: 'user-manager',
  authorName: 'Store Manager',
  title: 'Shift handover',
  isPinned: true,
  isClosed: true,
);

const _comment = ForumComment(
  id: 'c-1',
  postId: 'post-1',
  authorId: 'user-manager',
  authorName: 'Store Manager',
  body: 'Agreed.',
);

class _FakeForumApi extends ApiClient {
  String lastSort = 'latest';
  String lastCategoryId = '';
  bool lastIncludeInactive = false;
  String? lastCreatedCategoryId;
  String? lastCreatedTitle;
  String? lastCreatedCommentBody;
  String? lastCreatedBoardName;
  String? lastToggledPostId;
  bool? lastPinned;
  bool? lastClosed;

  @override
  Future<ForumCategoriesPage> listForumCategories(
    Session session, {
    bool includeInactive = false,
  }) async {
    lastIncludeInactive = includeInactive;
    return const ForumCategoriesPage(
      categories: [_board, _oldBoard],
      total: 2,
    );
  }

  @override
  Future<ForumPostsPage> listForumPosts(
    Session session, {
    String query = '',
    String categoryId = '',
    String sort = 'latest',
    int page = 1,
    int limit = 50,
  }) async {
    lastCategoryId = categoryId;
    lastSort = sort;
    return ForumPostsPage(
      posts: const [_thread, _pinnedThread],
      total: 2,
      page: page,
      limit: limit,
    );
  }

  @override
  Future<ForumPost> createForumPost(
    Session session, {
    required String categoryId,
    required String title,
    String body = '',
  }) async {
    lastCreatedCategoryId = categoryId;
    lastCreatedTitle = title;
    return _thread;
  }

  @override
  Future<ForumPostDetail> getForumPost(Session session, String postId) async =>
      const ForumPostDetail(post: _thread, comments: [_comment]);

  @override
  Future<ForumLikeResult> toggleForumLike(
      Session session, String postId) async {
    lastToggledPostId = postId;
    return const ForumLikeResult(liked: true, likeCount: 4);
  }

  @override
  Future<ForumComment> createForumComment(
    Session session,
    String postId, {
    required String body,
  }) async {
    lastCreatedCommentBody = body;
    return ForumComment(
      id: 'c-new',
      postId: postId,
      body: body,
      authorName: 'Store Manager',
    );
  }

  @override
  Future<ForumCategory> createForumCategory(
    Session session, {
    required String name,
    String slug = '',
    String description = '',
  }) async {
    lastCreatedBoardName = name;
    return _board;
  }

  @override
  Future<ForumPost> updateForumPost(
    Session session,
    String postId, {
    String? title,
    String? body,
    bool? isPinned,
    bool? isClosed,
  }) async {
    lastPinned = isPinned;
    lastClosed = isClosed;
    return _thread.copyWith(
      isPinned: isPinned,
      isClosed: isClosed,
    );
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

/// Lets an already-scheduled SnackBar reach the screen. Advancing 5s here
/// would run its 4s lifetime out and dismiss it before the assertion.
Future<void> _flushSnackbar(WidgetTester tester) async {
  await tester.pump();
  await tester.pump(const Duration(milliseconds: 300));
  await tester.pumpAndSettle();
}

void main() {
  testWidgets('hub shows the forum entry and opens the thread list',
      (tester) async {
    await tester.pumpWidget(_wrap(CommunityHubScreen(
        session: _manager, apiClient: _FakeForumApi())));
    await tester.pumpAndSettle();

    await tester.scrollUntilVisible(
      find.text('Forum'),
      200,
      scrollable: find.byType(Scrollable).first,
    );
    expect(find.text('Forum'), findsOneWidget);
    expect(
        find.text('Team discussions and announcements'), findsOneWidget);

    await tester.tap(find.text('Forum'));
    await tester.pumpAndSettle();

    expect(find.byType(ForumScreen), findsOneWidget);
    expect(find.text('Grill station tips'), findsOneWidget);
    expect(find.text('Shift handover'), findsOneWidget);
    expect(find.widgetWithText(ChoiceChip, 'Kitchen Talk'), findsOneWidget);

    await tester.tap(find.text('Grill station tips'));
    await tester.pumpAndSettle();

    expect(find.byType(ForumThreadScreen), findsOneWidget);
    expect(find.text('Preheat before service.'), findsOneWidget);
    expect(find.text('Agreed.'), findsOneWidget);
    expect(find.textContaining('Cashier One'), findsWidgets);
  });

  testWidgets('sort and board chips reach the API', (tester) async {
    final api = _FakeForumApi();
    await tester.pumpWidget(_wrap(ForumScreen(session: _manager, apiClient: api)));
    await tester.pumpAndSettle();

    expect(api.lastSort, 'latest');
    expect(api.lastCategoryId, '');

    await tester.tap(find.widgetWithText(ChoiceChip, 'Top'));
    await tester.pumpAndSettle();
    expect(api.lastSort, 'top');

    await tester.tap(find.widgetWithText(ChoiceChip, 'Pinned'));
    await tester.pumpAndSettle();
    expect(api.lastSort, 'pinned');

    await tester.tap(find.widgetWithText(ChoiceChip, 'Kitchen Talk'));
    await tester.pumpAndSettle();
    expect(api.lastCategoryId, 'cat-1');

    await tester.enterText(find.byType(TextField).first, 'grill');
    await tester.testTextInput.receiveAction(TextInputAction.done);
    await tester.pumpAndSettle();
    expect(api.lastSort, 'pinned');
    expect(api.lastCategoryId, 'cat-1');
  });

  testWidgets('new thread sheet posts and opens the thread',
      (tester) async {
    final api = _FakeForumApi();
    await tester.pumpWidget(_wrap(ForumScreen(session: _manager, apiClient: api)));
    await tester.pumpAndSettle();

    await tester.tap(find.byTooltip('New thread'));
    await tester.pumpAndSettle();

    await tester
        .tap(find.widgetWithText(DropdownButtonFormField<String>, 'Boards'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Kitchen Talk').last);
    await tester.pumpAndSettle();

    await tester.enterText(
        find.widgetWithText(TextField, 'Thread title'), 'Fresh ideas');
    await tester.enterText(
        find.widgetWithText(TextField, 'Message'), 'Let us brainstorm.');
    await tester.tap(find.widgetWithText(FilledButton, 'Post'));
    await _flushSnackbar(tester);

    expect(api.lastCreatedCategoryId, 'cat-1');
    expect(api.lastCreatedTitle, 'Fresh ideas');
    expect(find.text('Thread posted'), findsOneWidget);
    expect(find.byType(ForumThreadScreen), findsOneWidget);
    expect(find.text('Preheat before service.'), findsOneWidget);
  });

  testWidgets('thread detail toggles likes and posts a comment',
      (tester) async {
    final api = _FakeForumApi();
    await tester.pumpWidget(
        _wrap(ForumThreadScreen(session: _manager, apiClient: api, post: _thread)));
    await tester.pumpAndSettle();

    expect(find.text('Like · 0'), findsOneWidget);

    await tester.tap(find.text('Like · 0'));
    await _flushSnackbar(tester);
    expect(api.lastToggledPostId, 'post-1');
    expect(find.text('Liked · 4'), findsOneWidget);

    await tester.enterText(
        find.widgetWithText(TextField, 'Add a comment...'), 'Noted.');
    await tester.tap(find.byTooltip('Post'));
    await _flushSnackbar(tester);

    expect(api.lastCreatedCommentBody, 'Noted.');
    expect(find.text('Comment added'), findsOneWidget);
    expect(find.text('Noted.'), findsOneWidget);
    expect(find.text('Agreed.'), findsOneWidget);
  });

  testWidgets('boards screen creates a board', (tester) async {
    final api = _FakeForumApi();
    await tester
        .pumpWidget(_wrap(ForumBoardsScreen(session: _manager, apiClient: api)));
    await tester.pumpAndSettle();

    expect(api.lastIncludeInactive, isTrue);
    expect(find.text('Kitchen Talk'), findsOneWidget);
    expect(find.text('Old Board'), findsOneWidget);

    await tester.tap(find.byTooltip('Add board'));
    await tester.pumpAndSettle();
    await tester.enterText(
        find.widgetWithText(TextField, 'Board name'), 'Bar Corner');
    await tester.tap(find.widgetWithText(FilledButton, 'Save'));
    await _flushSnackbar(tester);

    expect(api.lastCreatedBoardName, 'Bar Corner');
    expect(find.text('Board saved'), findsOneWidget);
  });

  testWidgets('cashier sees the new-thread action but not board management',
      (tester) async {
    final api = _FakeForumApi();
    await tester.pumpWidget(_wrap(ForumScreen(session: _cashier, apiClient: api)));
    await tester.pumpAndSettle();

    expect(find.byTooltip('Boards'), findsNothing);
    expect(find.byTooltip('New thread'), findsOneWidget);
  });
}

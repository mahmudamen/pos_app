import 'package:flutter/material.dart';

import '../../core/api_client.dart';
import '../../core/community.dart';
import '../../core/layout.dart';
import '../../core/session_store.dart';
import '../../l10n/strings.dart';

const _manageRoles = ['owner', 'manager', 'saas_admin'];

String _shortDate(String iso) =>
    iso.length >= 10 ? iso.substring(0, 10) : iso;

class ForumScreen extends StatefulWidget {
  const ForumScreen({
    super.key,
    required this.session,
    required this.apiClient,
  });

  final Session session;
  final ApiClient apiClient;

  @override
  State<ForumScreen> createState() => _ForumScreenState();
}

class _ForumScreenState extends State<ForumScreen> {
  ForumPostsPage? _page;
  List<ForumCategory> _categories = const [];
  bool _loading = true;
  String? _error;
  int _currentPage = 1;
  String _query = '';
  String _sort = 'latest';
  String _categoryId = '';

  final _formTitle = TextEditingController();
  final _formBody = TextEditingController();
  String _formCategory = '';

  bool get _canManage => _manageRoles.contains(widget.session.role);

  @override
  void initState() {
    super.initState();
    _load();
    _loadCategories();
  }

  Future<void> _load({int page = 1}) async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final result = await widget.apiClient.listForumPosts(
        widget.session,
        query: _query,
        categoryId: _categoryId,
        sort: _sort,
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

  Future<void> _loadCategories() async {
    try {
      final result =
          await widget.apiClient.listForumCategories(widget.session);
      if (!mounted) return;
      setState(() => _categories = result.categories);
    } catch (_) {
      // Boards are filter sugar; the thread list still works without them.
    }
  }

  Future<void> _createThread(BuildContext sheetContext) async {
    final s = AppStrings.of(context);
    final title = _formTitle.text.trim();
    if (_formCategory.isEmpty) {
      ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('${s.forumBoards}: ${s.required}')));
      return;
    }
    if (title.isEmpty) {
      ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('${s.threadTitle}: ${s.required}')));
      return;
    }
    try {
      final post = await widget.apiClient.createForumPost(
        widget.session,
        categoryId: _formCategory,
        title: title,
        body: _formBody.text.trim(),
      );
      _formTitle.clear();
      _formBody.clear();
      if (!sheetContext.mounted) return;
      Navigator.of(sheetContext).pop();
      if (!mounted) return;
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(s.threadPosted)));
      await _load();
      if (!mounted) return;
      Navigator.of(context).push(
        MaterialPageRoute(
          builder: (_) => ForumThreadScreen(
            session: widget.session,
            apiClient: widget.apiClient,
            post: post,
          ),
        ),
      );
    } catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(e.toString())));
    }
  }

  void _showCreateForm() {
    final s = AppStrings.of(context);
    _formCategory = _categoryId;
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
                Text(s.newThread,
                    style: Theme.of(sheetContext).textTheme.titleLarge),
                const SizedBox(height: 12),
                DropdownButtonFormField<String>(
                  initialValue: _formCategory.isEmpty ? null : _formCategory,
                  decoration: InputDecoration(
                      labelText: s.forumBoards,
                      border: const OutlineInputBorder()),
                  items: [
                    for (final category in _categories)
                      DropdownMenuItem(
                          value: category.id, child: Text(category.name)),
                  ],
                  onChanged: (value) => _formCategory = value ?? '',
                ),
                const SizedBox(height: 12),
                TextField(
                  controller: _formTitle,
                  autofocus: true,
                  decoration: InputDecoration(
                      labelText: s.threadTitle,
                      border: const OutlineInputBorder()),
                ),
                const SizedBox(height: 12),
                TextField(
                  controller: _formBody,
                  maxLines: 4,
                  decoration: InputDecoration(
                      labelText: s.threadBody,
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
                      onPressed: () => _createThread(sheetContext),
                      child: Text(s.postThread),
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
    _formBody.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final s = AppStrings.of(context);
    return Scaffold(
      appBar: AppBar(
        title: Text(s.communityForum),
        actions: [
          if (_canManage)
            IconButton(
              onPressed: () => Navigator.of(context).push(
                MaterialPageRoute(
                  builder: (_) => ForumBoardsScreen(
                    session: widget.session,
                    apiClient: widget.apiClient,
                  ),
                ),
              ),
              tooltip: s.forumBoards,
              icon: const Icon(Icons.dashboard_customize_outlined),
            ),
        ],
      ),
      floatingActionButton: FloatingActionButton(
        onPressed: _showCreateForm,
        tooltip: s.newThread,
        child: const Icon(Icons.add),
      ),
      body: MaxWidthBox(
        child: Column(
          children: [
            Padding(
              padding: const EdgeInsets.all(8),
              child: Column(
                children: [
                  TextField(
                    decoration: InputDecoration(
                      hintText: s.searchForum,
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
                      for (final sort in const [
                        ('latest', 0),
                        ('top', 1),
                        ('pinned', 2),
                      ])
                        Padding(
                          padding: const EdgeInsetsDirectional.only(end: 6),
                          child: ChoiceChip(
                            label: Text(switch (sort.$1) {
                              'top' => s.sortTop,
                              'pinned' => s.sortPinned,
                              _ => s.sortLatest,
                            }),
                            selected: _sort == sort.$1,
                            onSelected: (_) {
                              setState(() => _sort = sort.$1);
                              _load();
                            },
                          ),
                        ),
                    ],
                  ),
                  if (_categories.isNotEmpty)
                    SizedBox(
                      height: 40,
                      child: ListView(
                        scrollDirection: Axis.horizontal,
                        children: [
                          Padding(
                            padding: const EdgeInsetsDirectional.only(end: 6),
                            child: ChoiceChip(
                              label: Text(s.allBoards),
                              selected: _categoryId.isEmpty,
                              onSelected: (_) {
                                setState(() => _categoryId = '');
                                _load();
                              },
                            ),
                          ),
                          for (final category in _categories)
                            Padding(
                              padding:
                                  const EdgeInsetsDirectional.only(end: 6),
                              child: ChoiceChip(
                                label: Text(category.name),
                                selected: _categoryId == category.id,
                                onSelected: (_) {
                                  setState(() => _categoryId = category.id);
                                  _load();
                                },
                              ),
                            ),
                        ],
                      ),
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
    final posts = _page?.posts ?? const [];
    final total = _page?.total ?? 0;
    final limit = _page?.limit ?? 50;
    final totalPages = limit > 0 ? (total / limit).ceil() : 1;

    return Column(
      children: [
        Expanded(
          child: posts.isEmpty
              ? Center(child: Text(s.noThreads))
              : ListView.separated(
                  itemCount: posts.length,
                  separatorBuilder: (_, __) => const Divider(height: 1),
                  itemBuilder: (context, index) {
                    final post = posts[index];
                    final meta = [
                      if (post.authorName.isNotEmpty) post.authorName,
                      if (post.categoryName.isNotEmpty) post.categoryName,
                      _shortDate(post.createdAt),
                    ];
                    return ListTile(
                      leading: Icon(post.isPinned
                          ? Icons.push_pin
                          : Icons.forum_outlined),
                      title: Row(
                        mainAxisSize: MainAxisSize.min,
                        children: [
                          Flexible(child: Text(post.title)),
                          if (post.isClosed) ...[
                            const SizedBox(width: 6),
                            Chip(
                              label: Text(s.closedLabel),
                              visualDensity: VisualDensity.compact,
                              materialTapTargetSize:
                                  MaterialTapTargetSize.shrinkWrap,
                            ),
                          ],
                        ],
                      ),
                      subtitle: Text(meta.join(' · ')),
                      trailing: Row(
                        mainAxisSize: MainAxisSize.min,
                        children: [
                          const Icon(Icons.chat_bubble_outline, size: 16),
                          const SizedBox(width: 4),
                          Text('${post.commentCount}'),
                          const SizedBox(width: 10),
                          const Icon(Icons.thumb_up_alt_outlined, size: 16),
                          const SizedBox(width: 4),
                          Text('${post.likeCount}'),
                        ],
                      ),
                      onTap: () => Navigator.of(context).push(
                        MaterialPageRoute(
                          builder: (_) => ForumThreadScreen(
                            session: widget.session,
                            apiClient: widget.apiClient,
                            post: post,
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

class ForumThreadScreen extends StatefulWidget {
  const ForumThreadScreen({
    super.key,
    required this.session,
    required this.apiClient,
    required this.post,
  });

  final Session session;
  final ApiClient apiClient;
  final ForumPost post;

  @override
  State<ForumThreadScreen> createState() => _ForumThreadScreenState();
}

class _ForumThreadScreenState extends State<ForumThreadScreen> {
  ForumPost? _post;
  List<ForumComment> _comments = const [];
  bool _loading = true;
  String? _error;
  final _commentController = TextEditingController();

  bool get _canManage => _manageRoles.contains(widget.session.role);

  ForumPost get _view => _post ?? widget.post;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    try {
      final detail =
          await widget.apiClient.getForumPost(widget.session, widget.post.id);
      if (!mounted) return;
      setState(() {
        _post = detail.post;
        _comments = detail.comments;
        _loading = false;
        _error = null;
      });
    } catch (e) {
      if (!mounted) return;
      setState(() {
        _error = e.toString();
        _loading = false;
      });
    }
  }

  Future<void> _toggleLike() async {
    try {
      final result =
          await widget.apiClient.toggleForumLike(widget.session, _view.id);
      if (!mounted) return;
      setState(() {
        _post = _view.copyWith(likedByMe: result.liked, likeCount: result.likeCount);
      });
    } catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(e.toString())));
    }
  }

  Future<void> _sendComment() async {
    final s = AppStrings.of(context);
    final body = _commentController.text.trim();
    if (body.isEmpty) return;
    try {
      final comment = await widget.apiClient.createForumComment(
        widget.session,
        _view.id,
        body: body,
      );
      _commentController.clear();
      if (!mounted) return;
      setState(() {
        _comments = [..._comments, comment];
        _post = _view.copyWith(commentCount: _view.commentCount + 1);
      });
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(s.commentPosted)));
    } catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(e.toString())));
    }
  }

  Future<void> _setFlag({bool? isPinned, bool? isClosed}) async {
    try {
      final updated = await widget.apiClient.updateForumPost(
        widget.session,
        _view.id,
        isPinned: isPinned,
        isClosed: isClosed,
      );
      if (!mounted) return;
      setState(() => _post = updated);
    } catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(e.toString())));
    }
  }

  Future<void> _delete() async {
    final s = AppStrings.of(context);
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (dialogContext) => AlertDialog(
        title: Text(s.deleteThread),
        content: Text(s.deleteThreadConfirm),
        actions: [
          TextButton(
              onPressed: () => Navigator.of(dialogContext).pop(false),
              child: Text(s.cancel)),
          FilledButton(
            onPressed: () => Navigator.of(dialogContext).pop(true),
            child: Text(s.deleteItem),
          ),
        ],
      ),
    );
    if (confirmed != true || !mounted) return;
    try {
      await widget.apiClient.deleteForumPost(widget.session, _view.id);
      if (!mounted) return;
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(s.threadDeleted)));
      Navigator.of(context).pop();
    } catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(e.toString())));
    }
  }

  @override
  void dispose() {
    _commentController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final s = AppStrings.of(context);
    final post = _view;
    final canDelete = post.isAuthor || _canManage;
    return Scaffold(
      appBar: AppBar(
        title: Text(post.title, overflow: TextOverflow.ellipsis),
        actions: [
          if (_canManage)
            PopupMenuButton<String>(
              onSelected: (value) {
                switch (value) {
                  case 'pin':
                    _setFlag(isPinned: !post.isPinned);
                  case 'close':
                    _setFlag(isClosed: !post.isClosed);
                  case 'delete':
                    _delete();
                }
              },
              itemBuilder: (menuContext) => [
                PopupMenuItem(
                  value: 'pin',
                  child: Text(post.isPinned ? s.unpinThread : s.pinThread),
                ),
                PopupMenuItem(
                  value: 'close',
                  child: Text(post.isClosed ? s.openThread : s.closeThread),
                ),
                if (canDelete)
                  PopupMenuItem(value: 'delete', child: Text(s.deleteThread)),
              ],
            )
          else if (canDelete)
            IconButton(
              onPressed: _delete,
              tooltip: s.deleteThread,
              icon: const Icon(Icons.delete_outline),
            ),
        ],
      ),
      body: _loading
          ? const Center(child: CircularProgressIndicator())
          : _error != null
              ? Center(
                  child: Padding(
                    padding: const EdgeInsets.all(16),
                    child: Text(_error!,
                        textAlign: TextAlign.center,
                        style:
                            TextStyle(color: Theme.of(context).colorScheme.error)),
                  ),
                )
              : MaxWidthBox(
                  child: Column(
                    children: [
                      Expanded(
                        child: ListView(
                          padding: const EdgeInsets.all(16),
                          children: [
                            Row(
                              children: [
                                if (post.isPinned) ...[
                                  Chip(
                                    label: Text(s.pinnedLabel),
                                    avatar: const Icon(Icons.push_pin, size: 16),
                                  ),
                                  const SizedBox(width: 8),
                                ],
                                if (post.isClosed) ...[
                                  Chip(
                                    label: Text(s.closedLabel),
                                    avatar:
                                        const Icon(Icons.lock_outline, size: 16),
                                  ),
                                ],
                              ],
                            ),
                            const SizedBox(height: 4),
                            Text(
                              [
                                if (post.authorName.isNotEmpty)
                                  post.authorName,
                                if (post.categoryName.isNotEmpty)
                                  post.categoryName,
                                _shortDate(post.createdAt),
                              ].join(' · '),
                              style: Theme.of(context).textTheme.bodySmall,
                            ),
                            const SizedBox(height: 12),
                            Text(post.body,
                                style: Theme.of(context).textTheme.bodyLarge),
                            const SizedBox(height: 12),
                            Align(
                              alignment: AlignmentDirectional.centerStart,
                              child: FilledButton.tonalIcon(
                                onPressed: _toggleLike,
                                icon: Icon(post.likedByMe
                                    ? Icons.thumb_up
                                    : Icons.thumb_up_outlined),
                                label: Text(post.likedByMe
                                    ? '${s.likedAction} · ${post.likeCount}'
                                    : '${s.likeAction} · ${post.likeCount}'),
                              ),
                            ),
                            const SizedBox(height: 24),
                            Text(s.commentsLabel,
                                style: Theme.of(context).textTheme.titleMedium),
                            const SizedBox(height: 8),
                            if (_comments.isEmpty)
                              Text(s.noComments,
                                  style: Theme.of(context).textTheme.bodySmall),
                            for (final comment in _comments)
                              Card(
                                margin: const EdgeInsets.only(bottom: 8),
                                child: Padding(
                                  padding: const EdgeInsets.all(12),
                                  child: Column(
                                    crossAxisAlignment:
                                        CrossAxisAlignment.start,
                                    children: [
                                      Text(
                                        [
                                          if (comment.authorName.isNotEmpty)
                                            comment.authorName,
                                          _shortDate(comment.createdAt),
                                        ].join(' · '),
                                        style: Theme.of(context)
                                            .textTheme
                                            .bodySmall,
                                      ),
                                      const SizedBox(height: 4),
                                      Text(comment.body),
                                    ],
                                  ),
                                ),
                              ),
                            const SizedBox(height: 72),
                          ],
                        ),
                      ),
                      SafeArea(
                        top: false,
                        child: Padding(
                          padding: const EdgeInsets.fromLTRB(16, 8, 16, 8),
                          child: Row(
                            children: [
                              Expanded(
                                child: TextField(
                                  controller: _commentController,
                                  decoration: InputDecoration(
                                    hintText: s.addComment,
                                    border: const OutlineInputBorder(),
                                    isDense: true,
                                  ),
                                  onSubmitted: (_) => _sendComment(),
                                ),
                              ),
                              const SizedBox(width: 8),
                              IconButton.filled(
                                onPressed: _sendComment,
                                tooltip: s.postThread,
                                icon: const Icon(Icons.send),
                              ),
                            ],
                          ),
                        ),
                      ),
                    ],
                  ),
                ),
    );
  }
}

class ForumBoardsScreen extends StatefulWidget {
  const ForumBoardsScreen({
    super.key,
    required this.session,
    required this.apiClient,
  });

  final Session session;
  final ApiClient apiClient;

  @override
  State<ForumBoardsScreen> createState() => _ForumBoardsScreenState();
}

class _ForumBoardsScreenState extends State<ForumBoardsScreen> {
  List<ForumCategory> _categories = const [];
  bool _loading = true;
  String? _error;
  final _formName = TextEditingController();
  final _formDescription = TextEditingController();

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
          .listForumCategories(widget.session, includeInactive: true);
      if (!mounted) return;
      setState(() {
        _categories = result.categories;
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

  Future<void> _createBoard(BuildContext sheetContext) async {
    final s = AppStrings.of(context);
    final name = _formName.text.trim();
    if (name.isEmpty) {
      ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('${s.boardName}: ${s.required}')));
      return;
    }
    try {
      await widget.apiClient.createForumCategory(
        widget.session,
        name: name,
        description: _formDescription.text.trim(),
      );
      _formName.clear();
      _formDescription.clear();
      if (!sheetContext.mounted) return;
      Navigator.of(sheetContext).pop();
      if (!mounted) return;
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(s.boardSaved)));
      await _load();
    } catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(e.toString())));
    }
  }

  Future<void> _renameBoard(ForumCategory category) async {
    final s = AppStrings.of(context);
    final controller = TextEditingController(text: category.name);
    final name = await showDialog<String>(
      context: context,
      builder: (dialogContext) => AlertDialog(
        title: Text(s.forumBoards),
        content: TextField(
          controller: controller,
          autofocus: true,
          decoration: InputDecoration(
              labelText: s.boardName, border: const OutlineInputBorder()),
        ),
        actions: [
          TextButton(
              onPressed: () => Navigator.of(dialogContext).pop(),
              child: Text(s.cancel)),
          FilledButton(
            onPressed: () =>
                Navigator.of(dialogContext).pop(controller.text.trim()),
            child: Text(s.save),
          ),
        ],
      ),
    );
    controller.dispose();
    if (name == null || name.isEmpty || !mounted) return;
    try {
      await widget.apiClient
          .updateForumCategory(widget.session, category.id, name: name);
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
                Text(s.addBoard,
                    style: Theme.of(sheetContext).textTheme.titleLarge),
                const SizedBox(height: 12),
                TextField(
                  controller: _formName,
                  autofocus: true,
                  decoration: InputDecoration(
                      labelText: s.boardName,
                      border: const OutlineInputBorder()),
                ),
                const SizedBox(height: 12),
                TextField(
                  controller: _formDescription,
                  maxLines: 2,
                  decoration: InputDecoration(
                      labelText: s.threadBody,
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
                      onPressed: () => _createBoard(sheetContext),
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
    _formName.dispose();
    _formDescription.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final s = AppStrings.of(context);
    return Scaffold(
      appBar: AppBar(title: Text(s.forumBoards)),
      floatingActionButton: FloatingActionButton(
        onPressed: _showCreateForm,
        tooltip: s.addBoard,
        child: const Icon(Icons.add),
      ),
      body: MaxWidthBox(
        child: _loading
            ? const Center(child: CircularProgressIndicator())
            : _error != null
                ? Center(
                    child: Padding(
                      padding: const EdgeInsets.all(16),
                      child: Text(_error!,
                          textAlign: TextAlign.center,
                          style: TextStyle(
                              color: Theme.of(context).colorScheme.error)),
                    ),
                  )
                : ListView.separated(
                    itemCount: _categories.length,
                    separatorBuilder: (_, __) => const Divider(height: 1),
                    itemBuilder: (context, index) {
                      final category = _categories[index];
                      return ListTile(
                        leading: const Icon(Icons.dashboard_outlined),
                        title: Text(category.name),
                        subtitle: Text([
                          '${category.postCount}',
                          if (!category.isActive) s.boardInactive,
                        ].join(' · ')),
                        trailing: Row(
                          mainAxisSize: MainAxisSize.min,
                          children: [
                            IconButton(
                              onPressed: () => _renameBoard(category),
                              tooltip: s.save,
                              icon: const Icon(Icons.edit_outlined),
                            ),
                            Switch(
                              value: category.isActive,
                              onChanged: (value) async {
                                final messenger =
                                    ScaffoldMessenger.of(context);
                                try {
                                  await widget.apiClient.updateForumCategory(
                                    widget.session,
                                    category.id,
                                    isActive: value,
                                  );
                                  await _load();
                                } catch (e) {
                                  messenger.showSnackBar(
                                      SnackBar(content: Text(e.toString())));
                                }
                              },
                            ),
                          ],
                        ),
                      );
                    },
                  ),
      ),
    );
  }
}

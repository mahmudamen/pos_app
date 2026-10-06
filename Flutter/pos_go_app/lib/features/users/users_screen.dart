import 'package:flutter/material.dart';

import '../../core/api_client.dart';
import '../../core/layout.dart';
import '../../core/security.dart';
import '../../core/session_store.dart';
import '../../core/users.dart';
import '../../l10n/strings.dart';

/// Tenant staff directory: lists `GET /v1/users` and creates users through
/// the existing `POST /v1/users` (role hierarchy + plan cap are enforced
/// server-side, so the errors surface as snackbar/inline messages here).
class UsersScreen extends StatefulWidget {
  const UsersScreen({
    super.key,
    required this.session,
    required this.apiClient,
  });

  final Session session;
  final ApiClient apiClient;

  @override
  State<UsersScreen> createState() => _UsersScreenState();
}

class _UsersScreenState extends State<UsersScreen> {
  TenantUsersPage? _page;
  bool _loading = true;
  bool _adding = false;
  String? _error;
  int _currentPage = 1;

  final TextEditingController _email = TextEditingController();
  final TextEditingController _displayName = TextEditingController();
  final TextEditingController _password = TextEditingController();
  String _role = 'cashier';
  String _accountType = 'standard';

  @override
  void initState() {
    super.initState();
    _loadUsers();
  }

  @override
  void dispose() {
    _email.dispose();
    _displayName.dispose();
    _password.dispose();
    super.dispose();
  }

  Future<void> _loadUsers({int page = 1}) async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final result = await widget.apiClient.listUsers(widget.session, page: page);
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

  /// Owners (and platform admins) may create any role; a manager may only
  /// create cashiers — the same hierarchy the backend enforces.
  bool get _canCreateAnyRole =>
      widget.session.isOwner || widget.session.isPlatformAdmin;

  bool get _formValid {
    final email = _email.text.trim();
    return email.contains('@') &&
        _displayName.text.trim().isNotEmpty &&
        _password.text.length >= 8;
  }

  Future<void> _addUser(void Function(VoidCallback) setSheetState) async {
    final s = AppStrings.of(context);
    if (!_formValid || _adding) return;
    setState(() => _adding = true);
    setSheetState(() {});
    try {
      await widget.apiClient.createUser(
        widget.session,
        email: _email.text.trim(),
        displayName: _displayName.text.trim(),
        password: _password.text,
        role: _role,
        accountType: _accountType,
      );
      _email.clear();
      _displayName.clear();
      _password.clear();
      if (!mounted) return;
      Navigator.of(context).pop();
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(s.userCreated)),
      );
      await _loadUsers();
    } on ApiException catch (e) {
      if (!mounted) return;
      setSheetState(() {});
      final message = switch (e.code) {
        'user_conflict' => s.emailTaken,
        'plan_limit_exceeded' => s.planLimitReached,
        _ => e.message,
      };
      setSheetState(() => _sheetError = message);
    } catch (e) {
      if (!mounted) return;
      setSheetState(() => _sheetError = e.toString());
    } finally {
      if (mounted) {
        setState(() => _adding = false);
        setSheetState(() {});
      }
    }
  }

  String? _sheetError;

  void _showAddSheet() {
    final s = AppStrings.of(context);
    _sheetError = null;
    _role = _canCreateAnyRole ? 'manager' : 'cashier';
    _accountType = 'standard';
    showModalBottomSheet<void>(
      context: context,
      isScrollControlled: true,
      builder: (sheetContext) => StatefulBuilder(
        builder: (sheetContext, setSheetState) => Padding(
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
                  Text(s.addUser,
                      style: Theme.of(sheetContext).textTheme.titleLarge),
                  const SizedBox(height: 12),
                  TextField(
                    controller: _email,
                    autofocus: true,
                    keyboardType: TextInputType.emailAddress,
                    onChanged: (_) => setSheetState(() {}),
                    decoration: InputDecoration(
                      labelText: s.email,
                      border: const OutlineInputBorder(),
                    ),
                  ),
                  const SizedBox(height: 12),
                  TextField(
                    controller: _displayName,
                    onChanged: (_) => setSheetState(() {}),
                    decoration: InputDecoration(
                      labelText: s.displayName,
                      border: const OutlineInputBorder(),
                    ),
                  ),
                  const SizedBox(height: 12),
                  TextField(
                    controller: _password,
                    obscureText: true,
                    onChanged: (_) => setSheetState(() {}),
                    decoration: InputDecoration(
                      labelText: s.password,
                      helperText: s.passwordHint,
                      border: const OutlineInputBorder(),
                    ),
                  ),
                  const SizedBox(height: 12),
                  DropdownButtonFormField<String>(
                    initialValue: _role,
                    decoration: InputDecoration(
                      labelText: s.userRole,
                      border: const OutlineInputBorder(),
                    ),
                    items: [
                      if (_canCreateAnyRole)
                        DropdownMenuItem(
                            value: 'owner', child: Text(s.roleOwner)),
                      if (_canCreateAnyRole)
                        DropdownMenuItem(
                            value: 'manager', child: Text(s.roleManager)),
                      DropdownMenuItem(
                          value: 'cashier', child: Text(s.roleCashier)),
                    ],
                    onChanged: (v) =>
                        setSheetState(() => _role = v ?? _role),
                  ),
                  const SizedBox(height: 12),
                  DropdownButtonFormField<String>(
                    initialValue: _accountType,
                    decoration: InputDecoration(
                      labelText: s.accountType,
                      border: const OutlineInputBorder(),
                    ),
                    items: const [
                      DropdownMenuItem(value: 'standard', child: Text('standard')),
                      DropdownMenuItem(value: 'demo', child: Text('demo')),
                      DropdownMenuItem(value: 'guest', child: Text('guest')),
                    ],
                    onChanged: (v) =>
                        setSheetState(() => _accountType = v ?? _accountType),
                  ),
                  if (_sheetError != null) ...[
                    const SizedBox(height: 12),
                    Text(
                      _sheetError!,
                      style: TextStyle(
                          color: Theme.of(sheetContext).colorScheme.error),
                    ),
                  ],
                  const SizedBox(height: 16),
                  Row(
                    mainAxisAlignment: MainAxisAlignment.end,
                    children: [
                      TextButton(
                          onPressed: () => Navigator.of(sheetContext).pop(),
                          child: Text(s.cancel)),
                      const SizedBox(width: 8),
                      FilledButton(
                        onPressed: _formValid && !_adding
                            ? () => _addUser(setSheetState)
                            : null,
                        child: _adding
                            ? const SizedBox(
                                width: 18,
                                height: 18,
                                child:
                                    CircularProgressIndicator(strokeWidth: 2),
                              )
                            : Text(s.save),
                      ),
                    ],
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }

  /// The same hierarchy the backend enforces on `PUT /users/:id/security`:
  /// owners and platform admins may configure anyone, a manager only
  /// cashiers. Everyone else has no edit rights at all.
  bool _canEditSecurity(TenantUser user) {
    if (widget.session.isOwner || widget.session.isPlatformAdmin) return true;
    return widget.session.isManager && user.role == 'cashier';
  }

  /// Seeds the editor from the RAW row when the viewer can see it, otherwise
  /// from the resolved permissions; a still-unknown level falls back to the
  /// role default (mirroring the backend's `DefaultLevelForRole`).
  UserSecurityProfile _seedProfile(TenantUser user) {
    final profile = user.security ??
        UserSecurityProfile.fromPermissions(
            user.permissions ?? UserPermissions.unknown);
    if (profile.accessLevel != PosAccessLevel.unknown) return profile;
    return profile.copyWith(
      accessLevel: switch (user.role) {
        'owner' || 'saas_admin' => PosAccessLevel.admin,
        'manager' => PosAccessLevel.manager,
        _ => PosAccessLevel.cashier,
      },
    );
  }

  void _showSecuritySheet(TenantUser user) {
    final s = AppStrings.of(context);
    var state = _seedProfile(user);
    var custom = state.useCustomPermissions;
    var saving = false;
    var showMaxError = false;
    String? error;
    // No TextEditingController: the sheet's route keeps rebuilding through its
    // exit animation, and a controller disposed at pop-start trips the
    // "used after being disposed" framework assert. TextFormField owns (and
    // disposes) its own controller, so the text lives in this local instead.
    var capText = state.maxDiscountPct?.toString() ?? '';
    showModalBottomSheet<void>(
      context: context,
      isScrollControlled: true,
      builder: (sheetContext) => StatefulBuilder(
        builder: (sheetContext, setSheetState) {
          Future<void> save() async {
            final text = capText.trim();
            int? cap;
            if (text.isNotEmpty) {
              final parsed = int.tryParse(text);
              if (parsed == null || parsed < 0 || parsed > 100) {
                setSheetState(() {
                  showMaxError = true;
                  error = null;
                });
                return;
              }
              cap = parsed;
            }
            final toSave = state.copyWith(
              maxDiscountPct: cap,
              clearMaxDiscountPct: cap == null,
              useCustomPermissions: custom,
            );
            setSheetState(() {
              saving = true;
              error = null;
              showMaxError = false;
            });
            try {
              await widget.apiClient.updateUserSecurity(
                widget.session,
                userId: user.id,
                security: toSave,
              );
              if (!mounted) return;
              Navigator.of(context).pop();
              ScaffoldMessenger.of(context).showSnackBar(
                SnackBar(content: Text(s.securitySaved)),
              );
              await _loadUsers(page: _currentPage);
            } on ApiException catch (e) {
              if (!mounted) return;
              setSheetState(() {
                saving = false;
                error = e.message;
              });
            } catch (e) {
              if (!mounted) return;
              setSheetState(() {
                saving = false;
                error = e.toString();
              });
            }
          }

          Widget perm(String label, bool value, void Function(bool) assign) =>
              SwitchListTile(
                contentPadding: EdgeInsets.zero,
                title: Text(label),
                value: value,
                onChanged: custom && !saving
                    ? (v) => setSheetState(() => assign(v))
                    : null,
              );

          return Padding(
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
                    Text(s.securityProfile,
                        style: Theme.of(sheetContext).textTheme.titleLarge),
                    const SizedBox(height: 4),
                    Text(user.email,
                        style: Theme.of(sheetContext).textTheme.bodySmall),
                    const SizedBox(height: 12),
                    DropdownButtonFormField<PosAccessLevel>(
                      initialValue: state.accessLevel,
                      decoration: InputDecoration(
                        labelText: s.accessLevel,
                        border: const OutlineInputBorder(),
                      ),
                      items: [
                        DropdownMenuItem(
                            value: PosAccessLevel.none,
                            child: Text(s.accessLevelNone)),
                        DropdownMenuItem(
                            value: PosAccessLevel.cashier,
                            child: Text(s.accessLevelCashier)),
                        DropdownMenuItem(
                            value: PosAccessLevel.advanced,
                            child: Text(s.accessLevelAdvanced)),
                        DropdownMenuItem(
                            value: PosAccessLevel.manager,
                            child: Text(s.accessLevelManagerLevel)),
                        DropdownMenuItem(
                            value: PosAccessLevel.admin,
                            child: Text(s.accessLevelAdmin)),
                      ],
                      onChanged: saving
                          ? null
                          : (v) => setSheetState(() => state =
                              state.copyWith(accessLevel: v ?? state.accessLevel)),
                    ),
                    const SizedBox(height: 12),
                    TextFormField(
                      initialValue: capText,
                      keyboardType: TextInputType.number,
                      enabled: !saving,
                      onChanged: (v) {
                        capText = v;
                        if (showMaxError) {
                          setSheetState(() => showMaxError = false);
                        }
                      },
                      decoration: InputDecoration(
                        labelText: s.maxDiscountPctLabel,
                        helperText: s.maxDiscountLevelDefault,
                        errorText:
                            showMaxError ? s.maxDiscountRangeError : null,
                        border: const OutlineInputBorder(),
                      ),
                    ),
                    SwitchListTile(
                      contentPadding: EdgeInsets.zero,
                      title: Text(s.customPermissions),
                      subtitle: Text(s.customPermissionsHint),
                      value: custom,
                      onChanged: saving
                          ? null
                          : (v) => setSheetState(() => custom = v),
                    ),
                    if (custom) ...[
                      perm(s.permDeleteOrder, state.canDeleteOrder,
                          (v) => state = state.copyWith(canDeleteOrder: v)),
                      perm(s.permDeleteLine, state.canDeleteLine,
                          (v) => state = state.copyWith(canDeleteLine: v)),
                      perm(s.permChangeQty, state.canChangeQty,
                          (v) => state = state.copyWith(canChangeQty: v)),
                      perm(s.permNegativeQty, state.canNegativeQty,
                          (v) => state = state.copyWith(canNegativeQty: v)),
                      perm(s.permPriceChange, state.canPriceChange,
                          (v) => state = state.copyWith(canPriceChange: v)),
                      perm(s.permDiscount, state.canDiscount,
                          (v) => state = state.copyWith(canDiscount: v)),
                      perm(s.permOpenSession, state.canOpenSession,
                          (v) => state = state.copyWith(canOpenSession: v)),
                      perm(s.permCloseSession, state.canCloseSession,
                          (v) => state = state.copyWith(canCloseSession: v)),
                      perm(s.permPaymentModification,
                          state.canPaymentModification,
                          (v) => state =
                              state.copyWith(canPaymentModification: v)),
                      perm(s.permRefund, state.canRefund,
                          (v) => state = state.copyWith(canRefund: v)),
                      perm(s.permNegativeStock, state.canNegativeStock,
                          (v) => state = state.copyWith(canNegativeStock: v)),
                    ],
                    if (error != null) ...[
                      const SizedBox(height: 8),
                      Text(error!,
                          style:
                              TextStyle(color: Theme.of(sheetContext).colorScheme.error)),
                    ],
                    const SizedBox(height: 16),
                    Row(
                      mainAxisAlignment: MainAxisAlignment.end,
                      children: [
                        TextButton(
                            onPressed: () =>
                                Navigator.of(sheetContext).pop(),
                            child: Text(s.cancel)),
                        const SizedBox(width: 8),
                        FilledButton(
                          onPressed: saving ? null : save,
                          child: saving
                              ? const SizedBox(
                                  width: 18,
                                  height: 18,
                                  child: CircularProgressIndicator(
                                      strokeWidth: 2),
                                )
                              : Text(s.save),
                        ),
                      ],
                    ),
                  ],
                ),
              ),
            ),
          );
        },
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final s = AppStrings.of(context);
    return Scaffold(
      appBar: AppBar(
        title: Text(s.users),
        actions: [
          IconButton(
            icon: const Icon(Icons.person_add),
            tooltip: s.addUser,
            onPressed: _showAddSheet,
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
                            onPressed: () =>
                                _loadUsers(page: _currentPage),
                            child: Text(s.retry)),
                      ],
                    ),
                  )
                : _buildList(context),
      ),
    );
  }

  Widget _buildList(BuildContext context) {
    final s = AppStrings.of(context);
    final users = _page?.users ?? [];
    final total = _page?.total ?? 0;
    final limit = _page?.limit ?? 50;
    final totalPages = limit > 0 ? (total / limit).ceil() : 1;

    return Column(
      children: [
        Expanded(
          child: users.isEmpty
              ? Center(child: Text(s.noUsers))
              : ListView.separated(
                  itemCount: users.length,
                  separatorBuilder: (_, __) => const Divider(height: 1),
                  itemBuilder: (context, index) {
                    final user = users[index];
                    return ListTile(
                      leading: CircleAvatar(
                        child: Icon(
                            user.isActive ? Icons.person : Icons.person_off),
                      ),
                      title: Text(user.displayName),
                      subtitle: Text(user.email),
                      trailing: Row(
                        mainAxisSize: MainAxisSize.min,
                        children: [
                          Column(
                            mainAxisAlignment: MainAxisAlignment.center,
                            crossAxisAlignment: CrossAxisAlignment.end,
                            children: [
                              Text(
                                _roleLabel(s, user.role),
                                style: Theme.of(context)
                                    .textTheme
                                    .bodySmall
                                    ?.copyWith(fontWeight: FontWeight.w600),
                              ),
                              if (!user.isActive) ...[
                                const SizedBox(height: 2),
                                Text(
                                  s.inactiveLabel,
                                  style: Theme.of(context)
                                      .textTheme
                                      .bodySmall
                                      ?.copyWith(
                                          color: Theme.of(context)
                                              .colorScheme
                                              .error),
                                ),
                              ],
                            ],
                          ),
                          if (_canEditSecurity(user)) ...[
                            const SizedBox(width: 4),
                            IconButton(
                              icon: const Icon(Icons.security),
                              tooltip: s.securityProfile,
                              onPressed: () => _showSecuritySheet(user),
                            ),
                          ],
                        ],
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
                      ? () => _loadUsers(page: _currentPage - 1)
                      : null,
                  icon: const Icon(Icons.chevron_left),
                ),
                Text('${s.pageOf} $_currentPage / $totalPages'),
                IconButton(
                  onPressed: _currentPage < totalPages
                      ? () => _loadUsers(page: _currentPage + 1)
                      : null,
                  icon: const Icon(Icons.chevron_right),
                ),
              ],
            ),
          ),
      ],
    );
  }

  String _roleLabel(AppStrings s, String role) => switch (role) {
        'owner' => s.roleOwner,
        'manager' => s.roleManager,
        'cashier' => s.roleCashier,
        _ => role,
      };
}

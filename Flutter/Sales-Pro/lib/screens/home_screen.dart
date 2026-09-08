import 'package:sales_pro/screens/draft_invoices_screen.dart';
import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../state/app_provider.dart';
import 'orders_screen.dart';
import 'stores_screen.dart';
import 'customers_screen.dart';
import 'settings_screen.dart';
import '../theme/app_theme.dart';
import '../core/localization/app_localizations.dart';

class HomeScreen extends StatefulWidget {
  const HomeScreen({super.key});

  @override
  State<HomeScreen> createState() => _HomeScreenState();
}

class _HomeScreenState extends State<HomeScreen> {
  int _currentIndex = 0;

  final List<Widget> _screens = [
    OrdersScreen(),
    DraftInvoicesScreen(),
    StoresScreen(),
    CustomersScreen(),
    // SettingsScreen(),
  ];

  @override
  Widget build(BuildContext context) {
    final loc = AppLocalizations.of(context);
    final titles = [loc.orders, loc.savedDrafts, loc.warehouse, loc.customers];

    return Scaffold(
      backgroundColor: AppColors.background,

      appBar: AppBar(
        centerTitle: true,
        title: AnimatedSwitcher(
          duration: const Duration(milliseconds: 250),
          transitionBuilder: (child, anim) =>
              FadeTransition(opacity: anim, child: child),
          child: Text(
            titles[_currentIndex],
            key: ValueKey(titles[_currentIndex]),
            style: Theme.of(context).textTheme.titleLarge?.copyWith(
              color: Colors.white,
              fontWeight: FontWeight.bold,
            ),
          ),
        ),
        actions: [
          IconButton(
            icon: const Icon(Icons.settings),
            onPressed: () {
              final sync = context.read<AppProvider>().syncService;
              // Sync status handled silently (debug info removed for production)
              if (sync != null) {
                Navigator.push(
                  context,

                  MaterialPageRoute(builder: (_) => SettingsScreen()),
                );
              }
            },
          ),
        ],
      ),

      extendBody: true,

      floatingActionButton: FloatingActionButton(
        onPressed: () => Navigator.pushNamed(context, '/invoice'),
        tooltip: loc.newInvoice,
        backgroundColor: AppColors.fabBg,
        foregroundColor: AppColors.fabIcon,
        shape: const CircleBorder(),
        child: const Icon(Icons.receipt_long),
      ),
      floatingActionButtonLocation: FloatingActionButtonLocation.centerDocked,

      bottomNavigationBar: BottomAppBar(
        shape: const CircularNotchedRectangle(),
        notchMargin: 6,
        color: AppColors.bottomBar,
        elevation: 6,
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 8.0),
          child: Row(
            mainAxisAlignment: MainAxisAlignment.spaceAround,
            children: [
              _buildTabIcon(Icons.list, 0),
              _buildTabIcon(Icons.drafts, 1),
              const SizedBox(width: 48),
              _buildTabIcon(Icons.store, 2),

              _buildTabIcon(Icons.group, 3),
            ],
          ),
        ),
      ),

      body: SafeArea(
        child: AnimatedSwitcher(
          duration: const Duration(milliseconds: 300),
          child: _screens[_currentIndex],
        ),
      ),
    );
  }

  Widget _buildTabIcon(IconData icon, int index) {
    final loc = AppLocalizations.of(context);
    final titles = [loc.orders, loc.savedDrafts, loc.warehouse, loc.customers];
    final bool isActive = _currentIndex == index;
    return InkWell(
      onTap: () {
        setState(() {
          _currentIndex = index;
        });
      },
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(
            icon,
            color: isActive ? Colors.white : AppColors.card.withOpacity(0.7),
          ),
          const SizedBox(height: 2),
          Text(
            titles[index],
            style: TextStyle(
              fontSize: 10,
              color: isActive ? Colors.white : AppColors.card.withOpacity(0.7),
            ),
          ),
        ],
      ),
    );
  }
}

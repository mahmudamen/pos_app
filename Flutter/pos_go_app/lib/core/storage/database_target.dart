import 'package:sqflite/sqflite.dart';

/// Where the offline database lives for the current platform.
class DatabaseTarget {
  const DatabaseTarget({required this.factory, required this.path});

  final DatabaseFactory factory;
  final String path;
}
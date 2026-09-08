#!/usr/bin/env dart
// ignore_for_file: avoid_print

/// Tool to help manage localization strings
///
/// Usage:
///   dart run tool/localization_helper.dart check    # Check for missing keys
///   dart run tool/localization_helper.dart unused   # Find unused keys
///   dart run tool/localization_helper.dart sync     # Sync keys between languages

import 'dart:io';
import 'dart:convert';

void main(List<String> arguments) async {
  if (arguments.isEmpty) {
    _printUsage();
    return;
  }

  final command = arguments[0];

  switch (command) {
    case 'check':
      await _checkMissingKeys();
      break;
    case 'unused':
      await _findUnusedKeys();
      break;
    case 'sync':
      await _syncKeys();
      break;
    case 'stats':
      await _showStats();
      break;
    default:
      print('Unknown command: $command');
      _printUsage();
  }
}

void _printUsage() {
  print('''
Localization Helper Tool

Usage:
  dart run tool/localization_helper.dart <command>

Commands:
  check   - Check for missing translation keys
  unused  - Find unused translation keys in code
  sync    - Sync keys between ar.json and en.json
  stats   - Show translation statistics

Examples:
  dart run tool/localization_helper.dart check
  dart run tool/localization_helper.dart sync
''');
}

Future<void> _checkMissingKeys() async {
  print('🔍 Checking for missing translation keys...\n');

  final arFile = File('assets/lang/ar.json');
  final enFile = File('assets/lang/en.json');

  if (!arFile.existsSync() || !enFile.existsSync()) {
    print('❌ Translation files not found!');
    return;
  }

  final arContent =
      json.decode(await arFile.readAsString()) as Map<String, dynamic>;
  final enContent =
      json.decode(await enFile.readAsString()) as Map<String, dynamic>;

  final arKeys = arContent.keys.toSet();
  final enKeys = enContent.keys.toSet();

  final missingInAr = enKeys.difference(arKeys);
  final missingInEn = arKeys.difference(enKeys);

  if (missingInAr.isEmpty && missingInEn.isEmpty) {
    print('✅ All keys are synchronized!');
  } else {
    if (missingInAr.isNotEmpty) {
      print('⚠️  Missing in ar.json (${missingInAr.length} keys):');
      for (final key in missingInAr) {
        print('   - $key');
      }
      print('');
    }

    if (missingInEn.isNotEmpty) {
      print('⚠️  Missing in en.json (${missingInEn.length} keys):');
      for (final key in missingInEn) {
        print('   - $key');
      }
      print('');
    }
  }
}

Future<void> _findUnusedKeys() async {
  print('🔍 Finding unused translation keys...\n');

  final arFile = File('assets/lang/ar.json');
  if (!arFile.existsSync()) {
    print('❌ Translation file not found!');
    return;
  }

  final arContent =
      json.decode(await arFile.readAsString()) as Map<String, dynamic>;
  final allKeys = arContent.keys.toSet();

  // Search for key usage in Dart files
  final libDir = Directory('lib');
  final dartFiles = libDir
      .listSync(recursive: true)
      .whereType<File>()
      .where((f) => f.path.endsWith('.dart'));

  final usedKeys = <String>{};

  for (final file in dartFiles) {
    final content = await file.readAsString();
    for (final key in allKeys) {
      if (content.contains(key)) {
        usedKeys.add(key);
      }
    }
  }

  final unusedKeys = allKeys.difference(usedKeys);

  if (unusedKeys.isEmpty) {
    print('✅ All translation keys are being used!');
  } else {
    print('⚠️  Unused keys (${unusedKeys.length}):');
    for (final key in unusedKeys) {
      print('   - $key');
    }
    print(
      '\n💡 Consider removing these keys or checking if they should be used.',
    );
  }
}

Future<void> _syncKeys() async {
  print('🔄 Synchronizing translation keys...\n');

  final arFile = File('assets/lang/ar.json');
  final enFile = File('assets/lang/en.json');

  if (!arFile.existsSync() || !enFile.existsSync()) {
    print('❌ Translation files not found!');
    return;
  }

  final arContent =
      json.decode(await arFile.readAsString()) as Map<String, dynamic>;
  final enContent =
      json.decode(await enFile.readAsString()) as Map<String, dynamic>;

  final arKeys = arContent.keys.toSet();
  final enKeys = enContent.keys.toSet();

  var modified = false;

  // Add missing keys to ar.json
  for (final key in enKeys.difference(arKeys)) {
    arContent[key] = '[TODO: Translate] ${enContent[key]}';
    print('➕ Added to ar.json: $key');
    modified = true;
  }

  // Add missing keys to en.json
  for (final key in arKeys.difference(enKeys)) {
    enContent[key] = '[TODO: Translate] ${arContent[key]}';
    print('➕ Added to en.json: $key');
    modified = true;
  }

  if (modified) {
    // Sort keys alphabetically
    final sortedAr = Map.fromEntries(
      arContent.entries.toList()..sort((a, b) => a.key.compareTo(b.key)),
    );
    final sortedEn = Map.fromEntries(
      enContent.entries.toList()..sort((a, b) => a.key.compareTo(b.key)),
    );

    // Write back with pretty formatting
    await arFile.writeAsString(
      const JsonEncoder.withIndent('  ').convert(sortedAr) + '\n',
    );
    await enFile.writeAsString(
      const JsonEncoder.withIndent('  ').convert(sortedEn) + '\n',
    );

    print('\n✅ Keys synchronized successfully!');
    print('⚠️  Please translate the [TODO: Translate] entries.');
  } else {
    print('✅ All keys are already synchronized!');
  }
}

Future<void> _showStats() async {
  print('📊 Translation Statistics\n');

  final arFile = File('assets/lang/ar.json');
  final enFile = File('assets/lang/en.json');

  if (!arFile.existsSync() || !enFile.existsSync()) {
    print('❌ Translation files not found!');
    return;
  }

  final arContent =
      json.decode(await arFile.readAsString()) as Map<String, dynamic>;
  final enContent =
      json.decode(await enFile.readAsString()) as Map<String, dynamic>;

  print('Arabic (ar.json):');
  print('  Total keys: ${arContent.length}');
  print('  File size: ${await arFile.length()} bytes');

  print('\nEnglish (en.json):');
  print('  Total keys: ${enContent.length}');
  print('  File size: ${await enFile.length()} bytes');

  final commonKeys = arContent.keys.toSet().intersection(
    enContent.keys.toSet(),
  );
  print('\nCommon keys: ${commonKeys.length}');

  final syncPercentage = (commonKeys.length / arContent.length * 100)
      .toStringAsFixed(1);
  print('Synchronization: $syncPercentage%');

  // Check for TODO markers
  final arTodos = arContent.values
      .where((v) => v.toString().contains('[TODO'))
      .length;
  final enTodos = enContent.values
      .where((v) => v.toString().contains('[TODO'))
      .length;

  if (arTodos > 0 || enTodos > 0) {
    print('\n⚠️  Pending translations:');
    if (arTodos > 0) print('  ar.json: $arTodos');
    if (enTodos > 0) print('  en.json: $enTodos');
  } else {
    print('\n✅ All translations are complete!');
  }
}

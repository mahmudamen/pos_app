import 'dart:io';
import 'package:crypto/crypto.dart';
import 'dart:convert';

void main() {
  print("========================================");
  print("   SOLID POS - MASTER KEY GENERATOR     ");
  print("========================================");
  print("");
  
  stdout.write("Enter Client's REQUEST CODE: ");
  final requestCode = stdin.readLineSync()?.trim().toUpperCase();

  if (requestCode == null || requestCode.isEmpty) {
    print("❌ Error: Request Code cannot be empty!");
    return;
  }

  if (requestCode.length != 16) {
    print("⚠️ Warning: Request Code is usually 16 characters (yours is ${requestCode.length})");
  }

  // Same logic as in HardwareInfo.generateActivationKey
  const salt = "SOLID_POS_MASTER_KEY_2026";
  final bytes = utf8.encode(requestCode + salt);
  final hash = sha256.convert(bytes);
  
  final fullHash = hash.toString().toUpperCase();
  final key = "${fullHash.substring(4, 8)}-${fullHash.substring(12, 16)}-${fullHash.substring(20, 24)}-${fullHash.substring(28, 32)}";

  print("");
  print("✅ ACTIVATION KEY GENERATED:");
  print("----------------------------------------");
  print("   $key");
  print("----------------------------------------");
  print("");
  print("Press ENTER to exit...");
  stdin.readLineSync();
}

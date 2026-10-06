import 'package:flutter/widgets.dart';

/// App-wide holder of GlobalKeys that tour steps point at. The POS screen
/// registers a key per [TourTarget] the moment its widget builds, so a tour
/// can always resolve the exact pixel rect of the element it wants to
/// highlight.
class TourTargetRegistry {
  TourTargetRegistry();

  static TourTargetRegistry instance = TourTargetRegistry();

  final Map<String, Object> _keys = {};

  void register(String id, GlobalKey key) => _keys[id] = key;

  void unregister(String id) => _keys.remove(id);

  GlobalKey? keyFor(String id) => _keys[id] as GlobalKey?;

  bool has(String id) => _keys.containsKey(id);

  void clear() => _keys.clear();

  /// Test seam: a fresh registry so parallel tests never see each other's keys.
  static void reset() => instance = TourTargetRegistry();
}
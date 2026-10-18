import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:uuid/uuid.dart';

import '../../app/providers.dart';

part 'installation_id.g.dart';

const _prefsKey = 'installation_id';

/// A random id for this app install (UUIDv7), created on first use and kept
/// until the app is uninstalled. Sent with the session and device calls; the
/// backend rate-limits sessions per installation (D-059).
@Riverpod(keepAlive: true)
String installationId(Ref ref) {
  final prefs = ref.watch(sharedPreferencesProvider);
  final stored = prefs.getString(_prefsKey);
  if (stored != null && Uuid.isValidUUID(fromString: stored)) {
    return stored;
  }
  final id = const Uuid().v7();
  // Fire and forget: SharedPreferences updates its cache synchronously.
  prefs.setString(_prefsKey, id).ignore();
  return id;
}

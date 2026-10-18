import 'package:riverpod_annotation/riverpod_annotation.dart';

import '../data/profile_providers.dart';
import '../domain/me.dart';

part 'profile_controllers.g.dart';

/// `GET /v1/me` for the profile screen; reloaded every time it opens.
@riverpod
Future<Me> me(Ref ref) => ref.watch(profileRepositoryProvider).me();

/// Privacy and app settings, saved on every change (roadmap 1.8.6).
@riverpod
class SettingsController extends _$SettingsController {
  @override
  Future<UserSettings> build() =>
      ref.watch(profileRepositoryProvider).settings();

  /// Shows [next] at once and saves it; on failure the previous settings
  /// come back and the error is rethrown for the screen to show.
  Future<void> change(UserSettings next) async {
    final previous = state.value;
    state = AsyncData(next);
    try {
      state = AsyncData(
        await ref.read(profileRepositoryProvider).updateSettings(next),
      );
    } on Object {
      if (previous != null && ref.mounted) {
        state = AsyncData(previous);
      }
      rethrow;
    }
  }
}

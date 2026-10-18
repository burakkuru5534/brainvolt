import 'package:riverpod_annotation/riverpod_annotation.dart';

import '../../../core/network/api_client.dart';
import '../domain/profile_repository.dart';
import 'api_profile_repository.dart';

part 'profile_providers.g.dart';

/// The profile endpoints. Tests override it with a fake.
@Riverpod(keepAlive: true)
ProfileRepository profileRepository(Ref ref) =>
    ApiProfileRepository(ref.watch(apiClientProvider));

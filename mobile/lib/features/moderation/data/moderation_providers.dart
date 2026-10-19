import 'package:riverpod_annotation/riverpod_annotation.dart';

import '../../../core/network/api_client.dart';
import '../domain/moderation_repository.dart';
import 'api_moderation_repository.dart';

part 'moderation_providers.g.dart';

/// The moderation endpoints. Tests override it with a fake.
@Riverpod(keepAlive: true)
ModerationRepository moderationRepository(Ref ref) =>
    ApiModerationRepository(ref.watch(apiClientProvider));

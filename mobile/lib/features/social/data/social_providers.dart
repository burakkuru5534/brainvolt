import 'package:riverpod_annotation/riverpod_annotation.dart';

import '../../../core/network/api_client.dart';
import '../domain/social_repository.dart';
import 'api_social_repository.dart';

part 'social_providers.g.dart';

/// The social endpoints. Tests override it with a fake.
@Riverpod(keepAlive: true)
SocialRepository socialRepository(Ref ref) =>
    ApiSocialRepository(ref.watch(apiClientProvider));

import 'package:riverpod_annotation/riverpod_annotation.dart';

import '../../../core/network/api_client.dart';
import '../domain/creator_repository.dart';
import 'api_creator_repository.dart';

part 'creator_providers.g.dart';

/// The creator endpoints. Tests override it with a fake.
@Riverpod(keepAlive: true)
CreatorRepository creatorRepository(Ref ref) =>
    ApiCreatorRepository(ref.watch(apiClientProvider));

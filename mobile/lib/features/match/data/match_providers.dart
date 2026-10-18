import 'package:riverpod_annotation/riverpod_annotation.dart';

import '../../../core/network/api_client.dart';
import '../domain/match_repository.dart';
import 'api_match_repository.dart';

part 'match_providers.g.dart';

/// The game and matchmaking endpoints. Tests override it with a fake.
@Riverpod(keepAlive: true)
MatchRepository matchRepository(Ref ref) =>
    ApiMatchRepository(ref.watch(apiClientProvider));

import 'package:riverpod_annotation/riverpod_annotation.dart';

import '../../../core/network/api_client.dart';
import '../domain/tournament_repository.dart';
import 'api_tournament_repository.dart';

part 'tournament_providers.g.dart';

/// The tournament endpoints. Tests override it.
@Riverpod(keepAlive: true)
TournamentRepository tournamentRepository(Ref ref) =>
    ApiTournamentRepository(ref.watch(apiClientProvider));

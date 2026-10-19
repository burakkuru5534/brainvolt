import 'package:riverpod_annotation/riverpod_annotation.dart';

import '../../../core/network/api_client.dart';
import '../domain/team_repository.dart';
import 'api_team_repository.dart';

part 'team_providers.g.dart';

/// The team endpoints. Tests override it with a fake.
@Riverpod(keepAlive: true)
TeamRepository teamRepository(Ref ref) =>
    ApiTeamRepository(ref.watch(apiClientProvider));

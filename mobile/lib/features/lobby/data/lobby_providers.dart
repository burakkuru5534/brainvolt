import 'package:riverpod_annotation/riverpod_annotation.dart';

import '../../../core/network/api_client.dart';
import '../domain/lobby_repository.dart';
import 'api_lobby_repository.dart';

part 'lobby_providers.g.dart';

/// The lobby endpoints. Tests override it with a fake.
@Riverpod(keepAlive: true)
LobbyRepository lobbyRepository(Ref ref) =>
    ApiLobbyRepository(ref.watch(apiClientProvider));

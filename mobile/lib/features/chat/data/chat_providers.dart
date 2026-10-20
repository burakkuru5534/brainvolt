import 'package:riverpod_annotation/riverpod_annotation.dart';

import '../../../core/network/api_client.dart';
import '../domain/chat_repository.dart';
import 'api_chat_repository.dart';

part 'chat_providers.g.dart';

/// The chat endpoints. Tests override it.
@Riverpod(keepAlive: true)
ChatRepository chatRepository(Ref ref) =>
    ApiChatRepository(ref.watch(apiClientProvider));

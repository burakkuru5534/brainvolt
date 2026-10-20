import '../../../core/network/api_client.dart';
import '../domain/chat_repository.dart';

class ApiChatRepository implements ChatRepository {
  const ApiChatRepository(this.api);

  final ApiClient api;

  @override
  Future<ChatChannel> open({String? teamId, String? coupleId}) async =>
      ChatChannel.fromJson(
        (await api.postJson(
          '/v1/chat/channels',
          body: {'team_id': ?teamId, 'couple_id': ?coupleId},
        ))!,
      );

  @override
  Future<List<ChatMessage>> history(String channelId, {String? before}) async {
    final j = await api.getJson(
      Uri(
        path: '/v1/chat/channels/$channelId/messages',
        queryParameters: {'cursor': ?before},
      ).toString(),
    );
    return [
      for (final m in (j['messages'] as List? ?? const []))
        ChatMessage.fromJson((m as Map).cast<String, dynamic>()),
    ];
  }

  @override
  Future<void> read(String channelId, String lastId) => api.postJson(
    '/v1/chat/channels/$channelId/read',
    body: {'last_read_id': lastId},
  );

  @override
  Future<void> remove(String messageId) =>
      api.delete('/v1/chat/messages/$messageId');
}

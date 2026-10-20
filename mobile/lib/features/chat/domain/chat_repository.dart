/// Team and couple chat (schema-platform §2, D-088). Messages are sent
/// over the WebSocket (`chat.send`); history and housekeeping use REST.
library;

import '../../social/domain/social_repository.dart';

/// The preset quick phrases the server accepts.
const quickPhrases = [
  'good_luck',
  'well_played',
  'nice',
  'thanks',
  'oops',
  'lets_go',
  'gg',
  'rematch',
];

class ChatChannel {
  const ChatChannel({
    required this.id,
    required this.members,
    required this.unread,
  });

  factory ChatChannel.fromJson(Map<String, dynamic> j) => ChatChannel(
    id: j['id'] as String,
    members: [
      for (final m in (j['members'] as List? ?? const []))
        UserCard.fromJson((m as Map).cast<String, dynamic>()),
    ],
    unread: (j['unread'] as num?)?.toInt() ?? 0,
  );

  final String id;
  final List<UserCard> members;
  final int unread;
}

class ChatMessage {
  const ChatMessage({
    required this.id,
    required this.channelId,
    required this.sender,
    required this.createdAt,
    this.body,
    this.quickId,
    this.removed = false,
  });

  factory ChatMessage.fromJson(Map<String, dynamic> j) => ChatMessage(
    id: j['id'] as String,
    channelId: j['channel_id'] as String? ?? '',
    sender: UserCard.fromJson(
      (j['sender'] as Map? ?? const {}).cast<String, dynamic>(),
    ),
    body: j['body'] as String?,
    quickId: j['quick_id'] as String?,
    createdAt: DateTime.parse(j['created_at'] as String).toLocal(),
    removed: j['removed'] == true,
  );

  final String id;
  final String channelId;
  final UserCard sender;
  final String? body;
  final String? quickId;
  final DateTime createdAt;
  final bool removed;

  ChatMessage asRemoved() => ChatMessage(
    id: id,
    channelId: channelId,
    sender: sender,
    createdAt: createdAt,
    removed: true,
  );
}

abstract interface class ChatRepository {
  /// The channel of one of the caller's teams or of their couple.
  Future<ChatChannel> open({String? teamId, String? coupleId});

  /// Messages newest first; [before] is a message id.
  Future<List<ChatMessage>> history(String channelId, {String? before});

  Future<void> read(String channelId, String lastId);

  Future<void> remove(String messageId);
}

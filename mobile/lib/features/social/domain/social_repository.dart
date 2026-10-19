/// Friends, friend requests and blocks (rest.md §7, schema-social §2).
library;

/// A user's public card.
class UserCard {
  const UserCard({
    required this.userId,
    required this.displayName,
    this.username = '',
    this.avatarId = '',
  });

  factory UserCard.fromJson(Map<String, dynamic> j) => UserCard(
    userId: j['user_id'] as String,
    username: j['username'] as String? ?? '',
    displayName: j['display_name'] as String? ?? '',
    avatarId: j['avatar_id'] as String? ?? '',
  );

  final String userId;
  final String username;
  final String displayName;
  final String avatarId;
}

/// A friend; [online] is null when they hide their status.
class Friend {
  const Friend(this.card, {this.online});

  final UserCard card;
  final bool? online;
}

/// A pending friend request with the other user's card.
class FriendRequest {
  const FriendRequest({
    required this.id,
    required this.other,
    required this.incoming,
  });

  final String id;
  final UserCard other;
  final bool incoming;
}

/// Friend request actions.
enum RequestAction { accept, decline, cancel }

abstract interface class SocialRepository {
  Future<List<Friend>> friends();

  Future<List<FriendRequest>> requests({required bool incoming});

  /// Sends a request by user id or username; true when it made a
  /// friendship at once (the other user had asked too).
  Future<bool> sendRequest({
    String? userId,
    String? username,
    String via = 'search',
  });

  Future<void> respond(String requestId, RequestAction action);

  Future<void> unfriend(String userId);

  Future<void> block(String userId);
}

/// Teams and couple profiles (rest.md §8, schema-social §4).
library;

import '../../social/domain/social_repository.dart';

/// The built-in team emblems (D-069) and the emoji the app draws for them.
const emblemEmoji = <String, String>{
  'bolt': '⚡',
  'flame': '🔥',
  'shield': '🛡️',
  'star': '⭐',
  'crown': '👑',
  'rocket': '🚀',
  'brain': '🧠',
  'owl': '🦉',
  'lion': '🦁',
  'wave': '🌊',
};

class TeamMember {
  const TeamMember(this.card, {required this.captain});
  final UserCard card;
  final bool captain;
}

class Team {
  const Team({
    required this.id,
    required this.name,
    required this.tag,
    required this.emblemId,
    required this.captainUserId,
    required this.members,
  });

  final String id;
  final String name;
  final String tag;
  final String emblemId;
  final String captainUserId;
  final List<TeamMember> members;
}

/// A pending invite to a team.
class TeamInvite {
  const TeamInvite({
    required this.id,
    required this.teamName,
    required this.teamTag,
    required this.emblemId,
    required this.invitedBy,
  });

  final String id;
  final String teamName;
  final String teamTag;
  final String emblemId;
  final UserCard invitedBy;
}

/// A couple profile; [status] is pending or active for the caller's own.
class Couple {
  const Couple({
    required this.id,
    required this.status,
    required this.requestedBy,
    required this.partners,
    this.name,
  });

  final String id;
  final String status;
  final String requestedBy;
  final List<UserCard> partners;
  final String? name;
}

/// Membership actions on a team.
enum TeamAction { leave, disband }

abstract interface class TeamRepository {
  Future<List<Team>> myTeams();

  Future<Team> team(String id);

  Future<Team> createTeam({
    required String name,
    required String tag,
    required String emblemId,
  });

  Future<void> invite(String teamId, String userId);

  Future<List<TeamInvite>> invites();

  Future<void> respondInvite(String inviteId, {required bool accept});

  Future<void> teamAction(String teamId, TeamAction action);

  Future<void> kick(String teamId, String userId);

  Future<void> makeCaptain(String teamId, String userId);

  /// The caller's pending or active couple, if any.
  Future<Couple?> myCouple();

  Future<Couple> requestCouple(String partnerUserId);

  /// accept / decline / dissolve.
  Future<Couple> respondCouple(String coupleId, String action);
}

/// Tournaments (rest.md §14, D-090): list, detail with the bracket,
/// registration, check-in, showing up for a bracket match, admin create.
library;

/// A registered user, couple or team.
class TournamentEntry {
  const TournamentEntry({
    required this.id,
    required this.entityKind,
    required this.entityId,
    required this.name,
    required this.image,
    required this.lineup,
    required this.status,
    this.seed,
    this.finalPlace,
  });

  factory TournamentEntry.fromJson(Map<String, dynamic> j) => TournamentEntry(
    id: j['id'] as String,
    entityKind: j['entity_kind'] as String? ?? 'user',
    entityId: j['entity_id'] as String? ?? '',
    name: j['name'] as String? ?? '',
    image: j['image'] as String? ?? '',
    lineup: [for (final u in (j['lineup'] as List? ?? const [])) u as String],
    status: j['status'] as String? ?? 'registered',
    seed: (j['seed'] as num?)?.toInt(),
    finalPlace: (j['final_place'] as num?)?.toInt(),
  );

  final String id;
  final String entityKind;
  final String entityId;
  final String name;

  /// Avatar id (users) or emblem id (teams, couples).
  final String image;
  final List<String> lineup;

  /// registered / checked_in / active / eliminated / withdrawn / winner.
  final String status;
  final int? seed;
  final int? finalPlace;
}

/// A bracket match.
class BracketSlot {
  const BracketSlot({
    required this.round,
    required this.slot,
    required this.status,
    this.entryA,
    this.entryB,
    this.winner,
    this.matchId,
    this.readyUntil,
    this.present = const [],
  });

  factory BracketSlot.fromJson(Map<String, dynamic> j) => BracketSlot(
    round: (j['round_no'] as num).toInt(),
    slot: (j['slot_no'] as num).toInt(),
    status: j['status'] as String? ?? 'pending',
    entryA: j['entry_a_id'] as String?,
    entryB: j['entry_b_id'] as String?,
    winner: j['winner_entry_id'] as String?,
    matchId: j['match_id'] as String?,
    readyUntil: DateTime.tryParse(j['ready_until'] as String? ?? '')?.toLocal(),
    present: [for (final u in (j['present'] as List? ?? const [])) u as String],
  );

  final int round;
  final int slot;

  /// pending / ready / in_progress / done / walkover.
  final String status;
  final String? entryA;
  final String? entryB;
  final String? winner;
  final String? matchId;
  final DateTime? readyUntil;
  final List<String> present;
}

/// A tournament as the list and the detail show it.
class Tournament {
  const Tournament({
    required this.id,
    required this.name,
    required this.status,
    required this.entityKind,
    required this.format,
    required this.matchType,
    required this.minEntrants,
    required this.maxEntrants,
    required this.entrants,
    required this.registrationOpensAt,
    required this.registrationClosesAt,
    required this.checkInOpensAt,
    required this.startsAt,
    required this.readyWindowS,
    this.description,
    this.rewards = const [],
    this.participationXp = 0,
    this.myEntry,
    this.kind = 'official',
    this.isOwner = false,
  });

  factory Tournament.fromJson(Map<String, dynamic> j) {
    DateTime at(String k) => DateTime.parse(j[k] as String).toLocal();
    final cfg = (j['match_config'] as Map?)?.cast<String, dynamic>() ?? {};
    final rw = (j['rewards'] as Map?)?.cast<String, dynamic>() ?? {};
    final mine = (j['my_entry'] as Map?)?.cast<String, dynamic>();
    return Tournament(
      id: j['id'] as String,
      name: j['name'] as String? ?? '',
      description: j['description'] as String?,
      status: j['status'] as String,
      entityKind: j['entity_kind'] as String? ?? 'user',
      format: cfg['format'] as String? ?? '1v1',
      matchType: cfg['match_type'] as String? ?? 'quick_duel',
      minEntrants: (j['min_entrants'] as num?)?.toInt() ?? 2,
      maxEntrants: (j['max_entrants'] as num?)?.toInt() ?? 2,
      entrants: (j['entrants'] as num?)?.toInt() ?? 0,
      registrationOpensAt: at('registration_opens_at'),
      registrationClosesAt: at('registration_closes_at'),
      checkInOpensAt: at('check_in_opens_at'),
      startsAt: at('starts_at'),
      readyWindowS: (j['round_ready_window_s'] as num?)?.toInt() ?? 300,
      rewards: [
        for (final p in (rw['places'] as List? ?? const []))
          (((p as Map)['place'] as num).toInt(), (p['xp'] as num).toInt()),
      ],
      participationXp: (rw['participation_xp'] as num?)?.toInt() ?? 0,
      myEntry: mine == null ? null : TournamentEntry.fromJson(mine),
      kind: j['kind'] as String? ?? 'official',
      isOwner: j['is_owner'] == true,
    );
  }

  /// official / friends (D-109).
  final String kind;

  /// The caller set up this friend tournament.
  final bool isOwner;

  bool get friends => kind == 'friends';

  final String id;
  final String name;
  final String? description;

  /// draft / registration / running / finished / cancelled.
  final String status;

  /// user / couple / team.
  final String entityKind;
  final String format;
  final String matchType;
  final int minEntrants;
  final int maxEntrants;
  final int entrants;
  final DateTime registrationOpensAt;
  final DateTime registrationClosesAt;
  final DateTime checkInOpensAt;
  final DateTime startsAt;
  final int readyWindowS;

  /// (place, xp).
  final List<(int, int)> rewards;
  final int participationXp;
  final TournamentEntry? myEntry;
}

/// GET /v1/tournaments/{id}.
class TournamentDetail {
  const TournamentDetail({
    required this.tournament,
    required this.rounds,
    required this.entries,
    required this.bracket,
    this.myMatch,
  });

  factory TournamentDetail.fromJson(Map<String, dynamic> j) {
    final mm = (j['my_match'] as Map?)?.cast<String, dynamic>();
    return TournamentDetail(
      tournament: Tournament.fromJson(j),
      rounds: (j['rounds'] as num?)?.toInt() ?? 0,
      entries: [
        for (final e in (j['entries'] as List? ?? const []))
          TournamentEntry.fromJson((e as Map).cast<String, dynamic>()),
      ],
      bracket: [
        for (final s in (j['bracket'] as List? ?? const []))
          BracketSlot.fromJson((s as Map).cast<String, dynamic>()),
      ],
      myMatch: mm == null ? null : BracketSlot.fromJson(mm),
    );
  }

  final Tournament tournament;
  final int rounds;
  final List<TournamentEntry> entries;
  final List<BracketSlot> bracket;
  final BracketSlot? myMatch;

  TournamentEntry? entry(String? id) =>
      entries.where((e) => e.id == id).firstOrNull;
}

/// What an admin fills in to create a tournament.
class TournamentDraft {
  const TournamentDraft({
    required this.nameTr,
    required this.entityKind,
    required this.format,
    required this.matchType,
    required this.maxEntrants,
    required this.startsAt,
    required this.readyWindowS,
    required this.rewards,
    this.nameEn,
    this.description,
    this.participationXp = 20,
    this.publish = true,
  });

  final String nameTr;
  final String? nameEn;
  final String? description;
  final String entityKind;
  final String format;
  final String matchType;
  final int maxEntrants;
  final DateTime startsAt;
  final int readyWindowS;

  /// XP for places 1, 2, 3.
  final List<int> rewards;
  final int participationXp;
  final bool publish;
}

abstract interface class TournamentRepository {
  Future<List<Tournament>> list();

  Future<TournamentDetail> get(String id);

  /// Team tournaments need [teamId] and the [lineup].
  Future<void> register(String id, {String? teamId, List<String>? lineup});

  Future<void> withdraw(String id);

  Future<void> checkIn(String id);

  /// Shows up for the current bracket match; the match id once both sides
  /// are present.
  Future<String?> play(String id);

  /// Admin: returns the new tournament's id.
  Future<String> create(TournamentDraft d);

  /// A friend tournament (D-109); returns its id.
  Future<String> createFriends({
    required String name,
    required String matchType,
    required int maxEntrants,
    required DateTime startsAt,
  });

  /// The owner cancels a friend tournament; the fee comes back.
  Future<void> cancelFriends(String id);
}

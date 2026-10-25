import 'dart:async';
import 'dart:convert';

import 'package:brainvolt/app/config.dart';
import 'package:brainvolt/app/providers.dart';
import 'package:brainvolt/core/ads/ad_gateway.dart';
import 'package:brainvolt/core/network/api_exception.dart';
import 'package:brainvolt/core/network/token_source.dart';
import 'package:brainvolt/core/purchases/purchase_gateway.dart';
import 'package:brainvolt/core/realtime/realtime_client.dart';
import 'package:brainvolt/core/realtime/realtime_providers.dart';
import 'package:brainvolt/core/share/share_service.dart';
import 'package:brainvolt/core/widgets/cosmetics.dart';
import 'package:brainvolt/features/ads/domain/ads_repository.dart';
import 'package:brainvolt/features/ads/presentation/ads.dart';
import 'package:brainvolt/features/auth/data/auth_providers.dart';
import 'package:brainvolt/features/auth/domain/auth_repository.dart';
import 'package:brainvolt/features/auth/domain/session.dart';
import 'package:brainvolt/features/auth/domain/session_repository.dart';
import 'package:brainvolt/features/chat/data/chat_providers.dart';
import 'package:brainvolt/features/chat/domain/chat_repository.dart';
import 'package:brainvolt/features/content/data/content_providers.dart';
import 'package:brainvolt/features/content/domain/content_repository.dart';
import 'package:brainvolt/features/content/domain/credit.dart';
import 'package:brainvolt/features/creator/data/creator_providers.dart';
import 'package:brainvolt/features/creator/domain/creator_repository.dart';
import 'package:brainvolt/features/daily/data/daily_providers.dart';
import 'package:brainvolt/features/daily/domain/daily_repository.dart';
import 'package:brainvolt/features/lobby/data/lobby_providers.dart';
import 'package:brainvolt/features/lobby/domain/lobby_repository.dart';
import 'package:brainvolt/features/match/data/match_providers.dart';
import 'package:brainvolt/features/match/domain/match_repository.dart';
import 'package:brainvolt/features/moderation/data/moderation_providers.dart';
import 'package:brainvolt/features/moderation/domain/moderation_repository.dart';
import 'package:brainvolt/features/notifications/data/notification_providers.dart';
import 'package:brainvolt/features/notifications/domain/notification_repository.dart';
import 'package:brainvolt/features/pass/domain/season_pass.dart';
import 'package:brainvolt/features/pass/presentation/season_pass_screen.dart';
import 'package:brainvolt/features/profile/data/profile_providers.dart';
import 'package:brainvolt/features/profile/domain/me.dart';
import 'package:brainvolt/features/profile/domain/profile_repository.dart';
import 'package:brainvolt/features/progress/data/progress_providers.dart';
import 'package:brainvolt/features/progress/domain/progress_repository.dart';
import 'package:brainvolt/features/social/data/social_providers.dart';
import 'package:brainvolt/features/social/domain/social_repository.dart';
import 'package:brainvolt/features/store/domain/store_repository.dart';
import 'package:brainvolt/features/store/presentation/plus_screen.dart';
import 'package:brainvolt/features/team/data/team_providers.dart';
import 'package:brainvolt/features/team/domain/team_repository.dart';
import 'package:brainvolt/features/tournament/data/tournament_providers.dart';
import 'package:brainvolt/features/tournament/domain/tournament_repository.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';

/// Firebase Auth in memory. Every call is appended to [calls].
class FakeAuthRepository implements AuthRepository {
  FakeAuthRepository({this._user});

  AuthUser? _user;
  final calls = <String>[];
  int _nextUid = 1;

  /// Accounts that already own a credential: linking it conflicts.
  final existing = <AuthMethod, AuthUser>{};

  /// Thrown by the next [obtainCredential].
  AuthFailure? obtainFailure;

  /// Thrown by the next [signIn].
  AuthFailure? signInFailure;

  /// Email is verified after the next [reload].
  bool verifyOnReload = false;

  @override
  AuthUser? get currentUser => _user;

  @override
  Future<AuthUser> signInAnonymously() async {
    calls.add('signInAnonymously');
    return _user = AuthUser(uid: 'guest-${_nextUid++}', isAnonymous: true);
  }

  @override
  Future<ProviderCredential> obtainCredential(
    AuthMethod method, {
    String? email,
    String? password,
  }) async {
    calls.add('obtain:${method.name}');
    final failure = obtainFailure;
    if (failure != null) {
      obtainFailure = null;
      throw failure;
    }
    return ProviderCredential(method, '${method.name}:$email');
  }

  @override
  Future<AuthUser> link(ProviderCredential credential) async {
    calls.add('link:${credential.method.name}');
    if (existing.containsKey(credential.method)) {
      throw AuthFailure(
        AuthFailureKind.credentialAlreadyInUse,
        conflict: credential,
      );
    }
    final email = credential.method == AuthMethod.email
        ? (credential.raw as String).split(':').last
        : '${credential.method.name}@example.com';
    return _user = AuthUser(
      uid: _user!.uid,
      isAnonymous: false,
      email: email,
      emailVerified: credential.method != AuthMethod.email,
      providers: {credential.method},
    );
  }

  @override
  Future<AuthUser> signIn(ProviderCredential credential) async {
    calls.add('signIn:${credential.method.name}');
    final failure = signInFailure;
    if (failure != null) {
      signInFailure = null;
      throw failure;
    }
    return _user = existing[credential.method]!;
  }

  @override
  Future<void> sendEmailVerification() async =>
      calls.add('sendEmailVerification');

  @override
  Future<AuthUser?> reload() async {
    calls.add('reload');
    final u = _user;
    if (u != null && verifyOnReload) {
      _user = AuthUser(
        uid: u.uid,
        isAnonymous: false,
        email: u.email,
        emailVerified: true,
        providers: u.providers,
      );
    }
    return _user;
  }

  /// The next [idToken] fails as if Firebase forgot the user.
  bool loseSession = false;

  @override
  Future<String?> idToken({bool forceRefresh = false}) async {
    calls.add(forceRefresh ? 'idToken:refresh' : 'idToken');
    if (loseSession) {
      loseSession = false;
      throw const AuthFailure(AuthFailureKind.sessionLost);
    }
    final u = _user;
    return u == null ? null : 'token-of-${u.uid}';
  }

  @override
  Future<void> signOut() async {
    calls.add('signOut');
    _user = null;
  }
}

/// The identity endpoints in memory: the session mirrors the fake auth user.
class FakeSessionRepository implements SessionRepository {
  FakeSessionRepository(this.auth);

  final FakeAuthRepository auth;
  final calls = <String>[];

  /// Thrown by the next [sync].
  Object? syncError;
  DateTime? deletionRequestedAt;
  AccountStatus status = AccountStatus.active;

  /// Staff roles of the signed-in user.
  List<String> roles = const [];

  @override
  Future<Session> sync() async {
    calls.add('sync');
    final error = syncError;
    if (error != null) {
      syncError = null;
      throw error;
    }
    final u = auth.currentUser!;
    final tier = u.isAnonymous
        ? Tier.guest
        : u.emailVerified
        ? Tier.full
        : Tier.unverified;
    return testSession(
      uid: u.uid,
      tier: tier,
      status: status,
      deletionRequestedAt: deletionRequestedAt,
      roles: roles,
    );
  }

  @override
  Future<void> registerDevice({String? fcmToken}) async =>
      calls.add('registerDevice');

  @override
  Future<void> unregisterDevice() async => calls.add('unregisterDevice');

  @override
  Future<void> abandonGuest({required String guestToken}) async =>
      calls.add('abandonGuest:$guestToken');

  @override
  Future<DateTime> requestDeletion() async {
    calls.add('requestDeletion');
    deletionRequestedAt = DateTime.utc(2026, 10, 18);
    return deletionRequestedAt!.add(deletionGracePeriod);
  }

  @override
  Future<void> cancelDeletion() async {
    calls.add('cancelDeletion');
    deletionRequestedAt = null;
  }
}

/// A session as the backend would answer it.
Session testSession({
  String uid = 'guest-1',
  Tier tier = Tier.guest,
  AccountStatus status = AccountStatus.active,
  DateTime? deletionRequestedAt,
  String displayName = 'Misafir 1234',
  List<String> roles = const [],
}) => Session(
  user: SessionUser(
    id: 'user-$uid',
    roles: roles,
    deletionRequestedAt: deletionRequestedAt,
  ),
  profile: ProfileSummary(
    username: 'oyuncu12345',
    displayName: displayName,
    avatarId: 'fox',
    locale: 'tr',
  ),
  tier: tier,
  status: status,
  flags: SessionFlags(
    created: false,
    restricted: tier != Tier.full,
    emailVerificationRequired: tier == Tier.unverified,
    deletionPending: deletionRequestedAt != null,
    suspended: status == AccountStatus.suspended,
  ),
);

/// The profile endpoints in memory.
class FakeProfileRepository implements ProfileRepository {
  FakeProfileRepository({this.tier = Tier.guest});

  Tier tier;
  final calls = <String>[];

  /// Thrown by the next [updateSettings].
  Object? settingsError;

  OwnProfile profile = const OwnProfile(
    username: 'oyuncu12345',
    displayName: 'Misafir 1234',
    avatarId: 'fox',
    locale: 'tr',
  );

  UserSettings userSettings = const UserSettings(
    challengesFrom: ChallengesFrom.everyone,
    showOnlineStatus: true,
    allowFriendRequests: true,
    soundEnabled: true,
    musicEnabled: true,
    hapticsEnabled: true,
  );

  @override
  Future<Me> me() async {
    calls.add('me');
    return Me(
      user: MeUser(id: 'user-1', tier: tier),
      profile: profile,
      settings: userSettings,
    );
  }

  @override
  Future<OwnProfile> updateProfile({
    String? displayName,
    String? username,
    String? avatarId,
    String? locale,
  }) async {
    calls.add(
      'updateProfile:${{'display_name': ?displayName, 'username': ?username, 'avatar_id': ?avatarId, 'locale': ?locale}}',
    );
    return profile = profile.copyWith(
      displayName: displayName ?? profile.displayName,
      username: username ?? profile.username,
      avatarId: avatarId ?? profile.avatarId,
      locale: locale ?? profile.locale,
    );
  }

  @override
  Future<UserSettings> settings() async => userSettings;

  @override
  Future<UserSettings> updateSettings(UserSettings settings) async {
    calls.add('updateSettings');
    final error = settingsError;
    if (error != null) {
      settingsError = null;
      throw error;
    }
    return userSettings = settings;
  }
}

/// The content endpoints in memory.
class FakeContentRepository implements ContentRepository {
  List<Credit> sources = const [
    Credit(
      name: 'BrainVolt',
      licenseCode: 'LicenseRef-BrainVolt',
      attributionText: 'BrainVolt team',
      shareAlike: false,
    ),
    Credit(
      name: 'Open Trivia Database',
      licenseCode: 'CC-BY-SA-4.0',
      attributionText: 'Open Trivia Database (opentdb.com), CC BY-SA 4.0',
      shareAlike: true,
      url: 'https://opentdb.com',
    ),
  ];

  @override
  Future<List<Credit>> credits() async => sources;

  @override
  Future<List<CategoryInfo>> categories() async => const [
    CategoryInfo('cat-history', 'Tarih', iconKey: 'history'),
    CategoryInfo('cat-science', 'Bilim', iconKey: 'science'),
  ];
}

/// The game and matchmaking endpoints in memory.
class FakeMatchRepository implements MatchRepository {
  final calls = <String>[];

  /// What enqueue answers; a practice ticket carries a match id.
  QueueTicket ticket = const QueueTicket(id: 't1', state: 'searching');
  String? current;
  MatchResult? matchResult;

  @override
  Future<QueueTicket> enqueue({
    bool practice = false,
    QueueMode mode = const QueueMode(),
  }) async {
    calls.add(
      practice
          ? 'enqueue:practice'
          : mode.isDefault
          ? 'enqueue'
          : 'enqueue:${mode.matchType}:${mode.format}:${mode.categoryMode}:'
                '${mode.categoryId}:${mode.partyUserIds.join(',')}:${mode.teamId}',
    );
    return ticket;
  }

  @override
  Future<void> leaveQueue() async => calls.add('leaveQueue');

  @override
  Future<String?> currentMatch() async => current;

  @override
  Future<MatchResult> result(String matchId) async => matchResult!;
}

/// Friends and requests in memory; [failure] is thrown by every call.
class FakeSocialRepository implements SocialRepository {
  final calls = <String>[];
  List<Friend> friendList = [];
  List<FriendRequest> incoming = [];
  List<FriendRequest> outgoing = [];
  Object? failure;

  void _check() {
    if (failure != null) {
      throw failure!;
    }
  }

  @override
  Future<List<Friend>> friends() async {
    _check();
    return friendList;
  }

  @override
  Future<List<FriendRequest>> requests({required bool incoming}) async {
    _check();
    return incoming ? this.incoming : outgoing;
  }

  @override
  Future<bool> sendRequest({
    String? userId,
    String? username,
    String via = 'search',
  }) async {
    _check();
    calls.add('send:${userId ?? username}:$via');
    return false;
  }

  /// Profiles by user id; unknown ids get a public stranger.
  final profiles = <String, PlayerProfile>{};

  @override
  Future<PlayerProfile> profile(String userId) async {
    _check();
    return profiles[userId] ??
        PlayerProfile(
          card: UserCard(
            userId: userId,
            displayName: 'Deniz',
            username: 'deniz',
            avatarId: 'fox',
          ),
          countryCode: 'TR',
          stats: const PlayerStats(
            level: 4,
            matches: 20,
            wins: 12,
            badges: 3,
            streak: 2,
          ),
        );
  }

  @override
  Future<void> respond(String requestId, RequestAction action) async =>
      calls.add('${action.name}:$requestId');

  @override
  Future<void> unfriend(String userId) async => calls.add('unfriend:$userId');

  @override
  Future<void> block(String userId) async => calls.add('block:$userId');
}

/// Codes and challenges in memory.
class FakeLobbyRepository implements LobbyRepository {
  final calls = <String>[];
  List<Challenge> incoming = [];
  List<Challenge> outgoing = [];

  /// What resolving any code gives.
  ResolvedCode resolved = const ResolvedCode(kind: 'friend');

  /// What responding to a challenge gives.
  Challenge? responded;

  @override
  Future<FriendCode> friendCode() async => FriendCode('ABC234', DateTime(2030));

  @override
  Future<ResolvedCode> resolveCode(String code) async {
    calls.add('resolve:$code');
    return resolved;
  }

  @override
  Future<Challenge> challenge(String targetUserId, MatchType type) async {
    calls.add('challenge:$targetUserId:${type.wire}');
    return Challenge(
      id: 'c-new',
      status: 'pending',
      challenger: const UserCard(userId: 'me', displayName: 'Ben'),
      target: UserCard(userId: targetUserId, displayName: ''),
      matchType: type.wire,
      expiresAt: DateTime(2030),
    );
  }

  @override
  Future<List<Challenge>> challenges({required bool incoming}) async =>
      incoming ? this.incoming : outgoing;

  @override
  Future<Challenge> respondChallenge(String id, ChallengeAction action) async {
    calls.add('${action.name}:$id');
    return responded!;
  }

  /// What creating or joining a room gives.
  Room room1 = const Room(
    id: 'r1',
    status: 'open',
    version: 1,
    hostUserId: 'user-guest-1',
    code: 'ROOM23',
    matchType: 'quick_duel',
    capacity: 2,
    members: [],
  );

  @override
  Future<Room> createRoom(
    MatchType type, {
    String? categoryMode,
    String format = '1v1',
    String? teamMode,
    String? creatorQuizId,
    List<String> categoryIds = const [],
  }) async {
    calls.add(
      creatorQuizId != null
          ? 'createRoom:${type.wire}:quiz:$creatorQuizId'
          : categoryIds.isNotEmpty
          ? 'createRoom:${type.wire}:topic:${categoryIds.join(',')}'
          : format == '1v1'
          ? 'createRoom:${type.wire}'
          : 'createRoom:${type.wire}:$format:$teamMode',
    );
    return room1;
  }

  @override
  Future<Room> updateRoom(
    String id,
    RoomSettings settings, {
    String? creatorQuizId,
    int? version,
  }) async {
    calls.add(
      'updateRoom:$id:${settings.matchType.wire}:${settings.format}'
      ':${settings.teamMode}:${settings.topic ?? 'all'}:v$version',
    );
    return room1;
  }

  @override
  Future<Room> joinRoom(String code) async {
    calls.add('joinRoom:$code');
    return room1;
  }

  @override
  Future<Room> room(String id) async => room1;

  @override
  Future<void> leaveRoom(String id) async => calls.add('leaveRoom:$id');
}

/// Teams and couples in memory.
class FakeTeamRepository implements TeamRepository {
  final calls = <String>[];
  List<Team> teams = [];
  List<TeamInvite> pendingInvites = [];
  Couple? couple;

  @override
  Future<List<Team>> myTeams() async => teams;

  @override
  Future<Team> team(String id) async => teams.firstWhere((t) => t.id == id);

  @override
  Future<Team> createTeam({
    required String name,
    required String tag,
    required String emblemId,
  }) async {
    calls.add('create:$name:$tag:$emblemId');
    final t = Team(
      id: 't${teams.length + 1}',
      name: name,
      tag: tag,
      emblemId: emblemId,
      captainUserId: 'user-guest-1',
      members: const [],
    );
    teams = [...teams, t];
    return t;
  }

  @override
  Future<void> invite(String teamId, String userId) async =>
      calls.add('invite:$teamId:$userId');

  @override
  Future<List<TeamInvite>> invites() async => pendingInvites;

  @override
  Future<void> respondInvite(String inviteId, {required bool accept}) async {
    calls.add('${accept ? 'accept' : 'decline'}:$inviteId');
    pendingInvites = [];
  }

  @override
  Future<void> teamAction(String teamId, TeamAction action) async =>
      calls.add('${action.name}:$teamId');

  @override
  Future<void> kick(String teamId, String userId) async =>
      calls.add('kick:$teamId:$userId');

  @override
  Future<void> makeCaptain(String teamId, String userId) async =>
      calls.add('captain:$teamId:$userId');

  @override
  Future<Couple?> myCouple() async => couple;

  @override
  Future<Couple> requestCouple(String partnerUserId) async {
    calls.add('couple:$partnerUserId');
    return couple = Couple(
      id: 'cp1',
      status: 'pending',
      requestedBy: 'user-guest-1',
      partners: [
        const UserCard(userId: 'user-guest-1', displayName: 'Ben'),
        UserCard(userId: partnerUserId, displayName: 'Ali'),
      ],
    );
  }

  @override
  Future<Couple> respondCouple(String coupleId, String action) async {
    calls.add('$action:$coupleId');
    return couple!;
  }
}

/// A WebSocket whose server side the test drives.
class FakeProgressRepository implements ProgressRepository {
  Progress current = const Progress(
    xpTotal: 120,
    level: 2,
    levelXp: 20,
    levelSize: 125,
    streak: 3,
    longestStreak: 3,
    playedToday: true,
    badges: 2,
  );
  Standing myStanding = const Standing(
    seasonPoints: 130,
    league: 'bronze',
    division: 2,
    matches: 5,
    wins: 4,
    nextAt: 200,
  );
  QuestList questList = const QuestList(daily: [], weekly: [], rerollsLeft: 1);
  List<BadgeInfo> catalogue = const [];
  final boards = <String, Board>{};
  final boardCalls = <String>[];
  final claimed = <String>[];
  final rerolled = <String>[];
  MatchAwards matchAwards = const MatchAwards();

  @override
  Future<Progress> progress() async => current;

  @override
  Future<Standing> standing(String kind, String id) async => myStanding;

  @override
  Future<QuestList> quests() async => questList;

  @override
  Future<int> claim(String questId) async {
    claimed.add(questId);
    return 50;
  }

  @override
  Future<QuestList> reroll(String questId) async {
    rerolled.add(questId);
    return questList;
  }

  @override
  Future<List<BadgeInfo>> badges() async => catalogue;

  @override
  Future<Board> leaderboard({
    required BoardPeriod period,
    required BoardEntity entity,
    required BoardScope scope,
    String? categoryId,
    String? cursor,
  }) async {
    final key = '${period.wire}/${entity.name}/${scope.name}';
    boardCalls.add(key);
    return boards[key] ?? const Board(entries: []);
  }

  @override
  Future<MatchAwards> awards(String matchId) async => matchAwards;
}

class FakeDailyRepository implements DailyRepository {
  DailyToday todayValue = const DailyToday(
    day: '2026-10-20',
    available: true,
    practiceAllowed: false,
  );
  DailyBoard boardValue = const DailyBoard(entries: []);
  final started = <String>[];

  @override
  Future<DailyToday> today() async => todayValue;

  @override
  Future<DailyRun> start(String kind) async {
    started.add(kind);
    return DailyRun(id: 'run-$kind', kind: kind, status: 'in_progress');
  }

  @override
  Future<DailyBoard> board(String day, String scope) async => boardValue;

  DailyAnswers answersValue = const DailyAnswers(
    day: '2026-10-19',
    questions: [
      DailyAnswerReview(
        position: 1,
        prompt: 'Türkiye’nin başkenti neresidir?',
        category: 'Coğrafya',
        options: [('o1', 'Ankara'), ('o2', 'İstanbul'), ('o3', 'İzmir')],
        correctOptionId: 'o1',
        explanation: 'Ankara 13 Ekim 1923’ten beri başkenttir.',
        yourOptionId: 'o2',
      ),
      DailyAnswerReview(
        position: 2,
        prompt: 'Güneş sistemindeki en büyük gezegen hangisidir?',
        category: 'Bilim',
        options: [('p1', 'Jüpiter'), ('p2', 'Satürn'), ('p3', 'Mars')],
        correctOptionId: 'p1',
        yourOptionId: 'p1',
        yourCorrect: true,
      ),
    ],
  );

  @override
  Future<DailyAnswers> answers({String? day}) async => answersValue;
}

class FakeModerationRepository implements ModerationRepository {
  final reports = <String>[];
  final appeals = <String>[];
  List<Sanction> sanctions = const [];

  @override
  Future<void> reportQuestion({
    required String questionId,
    required String reason,
    String? matchId,
    String? comment,
  }) async => reports.add('$questionId:$reason:$matchId:${comment ?? ''}');

  @override
  Future<List<Sanction>> mySanctions() async => sanctions;

  @override
  Future<void> appeal(String sanctionId, String text) async =>
      appeals.add('$sanctionId:$text');
}

class FakeCreatorRepository implements CreatorRepository {
  final calls = <String>[];
  List<Quiz> quizzes = [];
  List<Quiz> published = [];

  /// The last draft sent.
  QuizDraft? saved;

  @override
  Future<List<Quiz>> mine() async => quizzes;

  @override
  Future<List<Quiz>> discover({String? locale}) async => published;

  @override
  Future<Quiz> quiz(String id) async =>
      [...quizzes, ...published].firstWhere((q) => q.id == id);

  @override
  Future<Quiz> create(QuizDraft draft) async {
    saved = draft;
    calls.add('create:${draft.title}:${draft.questions.length}');
    final q = Quiz(
      id: 'quiz-${quizzes.length + 1}',
      title: draft.title,
      locale: draft.locale,
      answerWindowMs: draft.answerWindowMs,
      questionCount: draft.questions.length,
      status: 'draft',
      version: 1,
      categoryId: draft.categoryId,
      questions: draft.questions,
    );
    quizzes = [...quizzes, q];
    return q;
  }

  @override
  Future<Quiz> update(String id, int version, QuizDraft draft) async {
    saved = draft;
    calls.add('update:$id:$version:${draft.questions.length}');
    return quiz(id);
  }

  @override
  Future<void> delete(String id) async {
    calls.add('delete:$id');
    quizzes = quizzes.where((q) => q.id != id).toList();
  }

  @override
  Future<Quiz> submit(String id) async {
    calls.add('submit:$id');
    return quiz(id);
  }

  @override
  Future<void> like(String id, {required bool like}) async =>
      calls.add('like:$id:$like');
}

class FakeNotificationRepository implements NotificationRepository {
  final calls = <String>[];
  List<InboxItem> items = [];
  NotificationPrefs prefs = const NotificationPrefs(
    categories: [
      CategoryPref(category: 'friend', push: true, inApp: true),
      CategoryPref(
        category: 'moderation',
        push: true,
        inApp: true,
        locked: true,
      ),
    ],
    quiet: QuietHours(enabled: false, start: '23:00', end: '08:00'),
  );

  @override
  Future<InboxPage> inbox({String? cursor}) async =>
      InboxPage(items: items, unread: items.where((i) => !i.read).length);

  @override
  Future<void> markRead({List<String> ids = const [], bool all = false}) async {
    calls.add(all ? 'read:all' : 'read:${ids.join(',')}');
    items = [
      for (final i in items)
        all || ids.contains(i.id)
            ? InboxItem(
                id: i.id,
                category: i.category,
                kind: i.kind,
                title: i.title,
                body: i.body,
                deepLink: i.deepLink,
                createdAt: i.createdAt,
                read: true,
              )
            : i,
    ];
  }

  @override
  Future<NotificationPrefs> preferences() async => prefs;

  @override
  Future<NotificationPrefs> update({
    List<CategoryPref>? categories,
    QuietHours? quiet,
  }) async {
    calls.add(
      'update:${categories?.map((c) => '${c.category}/${c.push}/${c.inApp}').join(',')}'
      ':${quiet?.enabled}',
    );
    prefs = NotificationPrefs(
      categories: [
        for (final c in prefs.categories)
          categories?.where((n) => n.category == c.category).firstOrNull ?? c,
      ],
      quiet: quiet ?? prefs.quiet,
    );
    return prefs;
  }
}

/// Tournaments in memory: [details] by id; [played] is what play returns.
class FakeTournamentRepository implements TournamentRepository {
  final calls = <String>[];
  List<Tournament> tournaments = [];
  Map<String, TournamentDetail> details = {};
  String? played;

  @override
  Future<List<Tournament>> list() async => tournaments;

  @override
  Future<TournamentDetail> get(String id) async => details[id]!;

  @override
  Future<void> register(
    String id, {
    String? teamId,
    List<String>? lineup,
  }) async => calls.add(
    'register:$id${teamId == null ? '' : ':$teamId:${lineup!.join(',')}'}',
  );

  @override
  Future<void> withdraw(String id) async => calls.add('withdraw:$id');

  @override
  Future<void> checkIn(String id) async => calls.add('check-in:$id');

  @override
  Future<String?> play(String id) async {
    calls.add('play:$id');
    return played;
  }

  @override
  Future<String> create(TournamentDraft d) async {
    calls.add(
      'create:${d.nameTr}:${d.entityKind}:${d.format}:${d.maxEntrants}',
    );
    return 'new-t';
  }
}

class FakeChatRepository implements ChatRepository {
  final calls = <String>[];
  List<ChatMessage> messages = [];

  @override
  Future<ChatChannel> open({String? teamId, String? coupleId}) async {
    calls.add('open:${teamId ?? coupleId}');
    return const ChatChannel(
      id: 'ch1',
      members: [
        UserCard(userId: 'user-guest-1', displayName: 'Ben'),
        UserCard(userId: 'u-ali', displayName: 'Ali'),
      ],
      unread: 0,
    );
  }

  @override
  Future<List<ChatMessage>> history(String channelId, {String? before}) async =>
      messages.reversed.toList();

  @override
  Future<void> read(String channelId, String lastId) async =>
      calls.add('read:$lastId');

  @override
  Future<void> remove(String messageId) async => calls.add('remove:$messageId');
}

class FakeShareService implements ShareService {
  final shared = <String>[];

  @override
  Future<void> shareImage(GlobalKey boundary, String text) async =>
      shared.add(text);
}

class FakeSocket implements SocketConnection {
  final _in = StreamController<dynamic>();
  final sent = <Map<String, dynamic>>[];

  @override
  int? closeCode;

  @override
  Stream<dynamic> get stream => _in.stream;

  @override
  void add(String data) => sent.add(jsonDecode(data) as Map<String, dynamic>);

  @override
  Future<void> close([int? code, String? reason]) async {
    closeCode ??= code;
    await _in.close();
  }

  /// Sends a frame from the server.
  void serverSends(Map<String, dynamic> frame) => _in.add(jsonEncode(frame));

  /// Closes from the server side with [code].
  Future<void> serverCloses(int code) async {
    closeCode = code;
    await _in.close();
  }

  /// The last client message of [type].
  Map<String, dynamic> lastSent(String type) =>
      sent.lastWhere((m) => m['t'] == type);

  /// Acknowledges the last client message of [type].
  void ack(String type) =>
      serverSends({'t': 'sys.ack', 're': lastSent(type)['id']});
}

/// Everything a test app needs, with fakes behind the repositories.

/// Ads (D-103): a gateway whose rewarded ads are always watched to the end.
class FakeAdGateway implements AdGateway {
  AdShowResult rewardedResult = AdShowResult.earned;
  final rewarded = <String>[];
  var interstitials = 0;

  @override
  Future<void> init() async {}

  @override
  Future<AdShowResult> showRewarded({
    required String userId,
    required String customData,
  }) async {
    rewarded.add(customData);
    return rewardedResult;
  }

  @override
  Future<bool> showInterstitial() async {
    interstitials++;
    return true;
  }
}

class FakeAdsRepository implements AdsRepository {
  AdsConfig configValue = const AdsConfig();
  final calls = <String>[];

  /// Plus: rewards come back granted (D-104).
  var grantOnReserve = false;
  var _n = 0;

  @override
  Future<AdsConfig> config() async => configValue;

  @override
  Future<AdReward> reserve(String placement, {String? ref}) async {
    calls.add('reserve:$placement:${ref ?? ''}');
    _n++;
    return AdReward(
      id: 'ad-$_n',
      placement: placement,
      status: grantOnReserve ? 'granted' : 'pending',
      xp: placement == AdPlacement.bonusXp ? 35 : 0,
      streakDays: placement == AdPlacement.streakRepair ? 4 : 0,
    );
  }

  @override
  Future<AdReward> get(String id) async =>
      AdReward(id: id, placement: '', status: 'granted');

  @override
  Future<AdReward> devGrant(String id) async {
    calls.add('grant:$id');
    return AdReward(id: id, placement: '', status: 'granted');
  }
}

/// The Season Pass (D-108): tier 6 of 40, free track only; claims give
/// what is reached.
class FakeSeasonPassRepository implements SeasonPassRepository {
  FakeSeasonPassRepository(this.store);

  final FakeStoreRepository store;
  final calls = <String>[];
  var premium = false;
  final claimed = <String>{};

  static const _free = {5: PassReward(volt: 50), 10: PassReward(shield: true)};
  static PassReward? _premium(int t) => switch (t) {
    1 => const PassReward(cosmetic: 'frame_s1_storm'),
    10 => const PassReward(cosmetic: 'avatar_stormcloud'),
    5 || 15 || 25 || 35 => const PassReward(shield: true),
    40 => const PassReward(cosmetic: 'victory_s1_thunder', volt: 150),
    _ when t.isEven => const PassReward(volt: 60),
    _ => null,
  };

  /// The tier reached.
  var tier = 6;

  SeasonPass get value {
    var claimable = 0;
    final tiers = [
      for (var t = 1; t <= 40; t++)
        PassTier(
          tier: t,
          free: _free[t],
          premium: _premium(t),
          freeClaimed: claimed.contains('free:$t'),
          premiumClaimed: claimed.contains('premium:$t'),
        ),
    ];
    for (final t in tiers.take(tier)) {
      if (t.free != null && !t.freeClaimed) claimable++;
      if (premium && t.premium != null && !t.premiumClaimed) claimable++;
    }
    return SeasonPass(
      seasonNumber: 1,
      seasonName: 'Fırtına Sezonu',
      endsAt: DateTime.now().add(const Duration(days: 30, hours: 1)),
      xp: 3200,
      tier: tier,
      tierXp: 200,
      premium: premium,
      claimable: claimable,
      tiers: tiers,
    );
  }

  @override
  Future<SeasonPass> pass() async => value;

  @override
  Future<SeasonPass> buyPremium() async {
    calls.add('premium');
    premium = true;
    await store.buy(StoreItemCode.questReroll); // a stand-in charge
    return value;
  }

  @override
  Future<PassClaim> claim() async {
    calls.add('claim');
    var volt = 0;
    var shields = 0;
    final cosmetics = <String>[];
    for (final t in value.tiers.take(value.tier)) {
      for (final (track, r, done) in [
        ('free', t.free, t.freeClaimed),
        ('premium', premium ? t.premium : null, t.premiumClaimed),
      ]) {
        if (r == null || done) continue;
        claimed.add('$track:${t.tier}');
        volt += r.volt;
        if (r.shield) shields++;
        if (r.cosmetic.isNotEmpty) cosmetics.add(r.cosmetic);
      }
    }
    return PassClaim(
      pass: value,
      volt: volt,
      shields: shields,
      cosmetics: cosmetics,
    );
  }
}

/// BrainVolt Plus (D-104): a store whose purchases always go through.
class FakeStoreRepository implements StoreRepository {
  Entitlements value = const Entitlements();

  /// Volt (D-105).
  Wallet walletValue = Wallet(
    balance: 240,
    items: const [
      StoreItem(code: StoreItemCode.streakShield, price: 100),
      StoreItem(code: StoreItemCode.questReroll, price: 30),
    ],
    history: [
      VoltEntry(
        amount: 25,
        reason: 'level_up',
        ref: '3',
        createdAt: DateTime(2026, 10, 24, 18, 5),
      ),
      VoltEntry(
        amount: 10,
        reason: 'quest',
        createdAt: DateTime(2026, 10, 24, 17, 40),
      ),
      VoltEntry(
        amount: -100,
        reason: 'item',
        ref: StoreItemCode.streakShield,
        createdAt: DateTime(2026, 10, 23, 21, 12),
      ),
    ],
    adVoltLeft: 3,
    packs: const [
      VoltPack(
        productId: 'volt_starter',
        volt: 250,
        cosmetic: 'frame_starter',
        once: true,
      ),
      VoltPack(productId: 'volt_100', volt: 100),
      VoltPack(productId: 'volt_350', volt: 350, bonus: 50),
      VoltPack(productId: 'volt_800', volt: 800),
      VoltPack(productId: 'volt_2000', volt: 2000),
    ],
  );
  final bought = <String>[];

  /// Set to refuse purchases with this error code.
  String? buyError;

  @override
  Future<Entitlements> entitlements() async => value;

  @override
  Future<Wallet> wallet() async => walletValue;

  /// Cosmetics (D-106).
  MyCosmetics cosmeticsValue = MyCosmetics(
    items: [
      for (final (code, kind, price) in const [
        ('frame_bronze', 'frame', 200),
        ('frame_neon', 'frame', 300),
        ('frame_flame', 'frame', 400),
        ('frame_crown', 'frame', 600),
        ('frame_plus', 'frame', 0),
        ('avatar_octopus', 'avatar', 150),
        ('avatar_dragon', 'avatar', 250),
        ('name_gold', 'name_color', 300),
        ('name_rainbow', 'name_color', 0),
        ('victory_confetti', 'victory', 250),
        ('theme_sunset', 'match_theme', 400),
      ])
        CosmeticItem(
          code: code,
          kind: kind,
          price: price,
          plusOnly: price == 0,
          owned: code == 'frame_bronze',
        ),
    ],
  );
  final equipCalls = <String>[];

  @override
  Future<MyCosmetics> cosmetics() async => cosmeticsValue;

  @override
  Future<int> buyCosmetic(String code) async {
    if (buyError case final code?) {
      throw ApiException(code: code, status: 409);
    }
    bought.add(code);
    final c = cosmeticsValue;
    final item = c.items.firstWhere((i) => i.code == code);
    cosmeticsValue = MyCosmetics(
      items: [
        for (final i in c.items)
          i.code == code
              ? CosmeticItem(
                  code: i.code,
                  kind: i.kind,
                  price: i.price,
                  owned: true,
                )
              : i,
      ],
      equipped: c.equipped,
      plus: c.plus,
    );
    final w = walletValue;
    walletValue = Wallet(
      balance: w.balance - item.price,
      items: w.items,
      history: w.history,
      adVoltLeft: w.adVoltLeft,
      adVolt: w.adVolt,
      packs: w.packs,
    );
    return walletValue.balance;
  }

  @override
  Future<MyCosmetics> equip(String kind, String? code) async {
    equipCalls.add('$kind:${code ?? '-'}');
    final c = cosmeticsValue;
    final equipped = {...c.equipped};
    if (code == null) {
      equipped.remove(kind);
    } else {
      equipped[kind] = code;
    }
    cosmeticsValue = MyCosmetics(
      items: [
        for (final i in c.items)
          i.kind == kind
              ? CosmeticItem(
                  code: i.code,
                  kind: i.kind,
                  price: i.price,
                  plusOnly: i.plusOnly,
                  owned: i.owned,
                  equipped: i.code == code,
                )
              : i,
      ],
      equipped: equipped,
      plus: c.plus,
    );
    return cosmeticsValue;
  }

  @override
  Future<int> buy(String item) async {
    if (buyError case final code?) {
      throw ApiException(code: code, status: 409);
    }
    bought.add(item);
    final w = walletValue;
    walletValue = Wallet(
      balance: w.balance - w.priceOf(item),
      items: w.items,
      history: w.history,
      adVoltLeft: w.adVoltLeft,
      adVolt: w.adVolt,
      packs: w.packs,
    );
    return walletValue.balance;
  }
}

class FakePurchaseGateway implements PurchaseGateway {
  FakePurchaseGateway(this.store);

  final FakeStoreRepository store;
  @override
  bool available = true;
  BuyResult result = BuyResult.purchased;
  final bought = <String>[];
  var restored = 0;

  @override
  Future<List<PlusOption>> plusOptions(String userId) async => const [
    PlusOption(id: 'monthly', priceString: '₺49,99', trialDays: 7),
    PlusOption(id: 'annual', priceString: '₺399,99'),
  ];

  @override
  Future<BuyResult> buy(String userId, PlusOption option) async {
    bought.add(option.id);
    if (result == BuyResult.purchased) {
      store.value = Entitlements(plus: true, plusUntil: DateTime(2030, 1, 15));
    }
    return result;
  }

  /// Volt packs (D-107): a bought pack adds its Volt to the fake wallet.
  final boughtVolt = <String>[];

  @override
  Future<List<StoreOffer>> voltOffers(String userId) async => const [
    StoreOffer(id: 'volt_starter', priceString: '₺29,99'),
    StoreOffer(id: 'volt_100', priceString: '₺29,99'),
    StoreOffer(id: 'volt_350', priceString: '₺89,99'),
    StoreOffer(id: 'volt_800', priceString: '₺179,99'),
    StoreOffer(id: 'volt_2000', priceString: '₺399,99'),
  ];

  @override
  Future<BuyResult> buyVolt(String userId, StoreOffer offer) async {
    boughtVolt.add(offer.id);
    if (result == BuyResult.purchased) {
      final w = store.walletValue;
      final pack = w.packs.firstWhere((p) => p.productId == offer.id);
      store.walletValue = Wallet(
        balance: w.balance + pack.volt,
        items: w.items,
        history: w.history,
        adVoltLeft: w.adVoltLeft,
        adVolt: w.adVolt,
        packs: [
          for (final p in w.packs)
            if (!p.once || p != pack) p,
        ],
      );
    }
    return result;
  }

  @override
  Future<bool> restore(String userId) async {
    restored++;
    return store.value.plus;
  }
}

class TestApp {
  TestApp._(
    this.container,
    this.prefs,
    this.auth,
    this.sessions,
    this.profiles,
    this.matches,
    this.sockets,
    this.social,
    this.lobby,
    this.teams,
    this.progress,
    this.daily,
    this.moderation,
    this.creator,
    this.notifications,
    this.share,
    this.chat,
    this.tournaments,
  );

  /// [ageConfirmed] skips the age gate; [user] is an already signed-in user.
  static Future<TestApp> create({
    bool ageConfirmed = true,
    AuthUser? user,
    Map<String, Object> prefs = const {},
  }) async {
    cosmeticAnimations = false; // endless animations never settle
    SharedPreferences.setMockInitialValues({
      'age_confirmed': ?(ageConfirmed ? true : null),
      ...prefs,
    });
    final p = await SharedPreferences.getInstance();
    final auth = FakeAuthRepository(user: user);
    final sessions = FakeSessionRepository(auth);
    final profiles = FakeProfileRepository();
    final matches = FakeMatchRepository();
    final social = FakeSocialRepository();
    final lobby = FakeLobbyRepository();
    final teams = FakeTeamRepository();
    final progress = FakeProgressRepository();
    final daily = FakeDailyRepository();
    final moderation = FakeModerationRepository();
    final creator = FakeCreatorRepository();
    final notifications = FakeNotificationRepository();
    final share = FakeShareService();
    final chat = FakeChatRepository();
    final tournaments = FakeTournamentRepository();
    final ads = FakeAdsRepository();
    final store = FakeStoreRepository();
    final seasonPass = FakeSeasonPassRepository(store);
    final purchases = FakePurchaseGateway(store);
    final adGateway = FakeAdGateway();
    final sockets = <FakeSocket>[];
    final container = ProviderContainer(
      retry: noRetry,
      overrides: [
        appConfigProvider.overrideWithValue(
          AppConfig.parse(
            flavor: 'dev',
            apiBaseUrl: 'http://10.0.2.2:8080',
            wsUrl: 'ws://10.0.2.2:8080/v1/ws',
          ),
        ),
        sharedPreferencesProvider.overrideWithValue(p),
        appVersionProvider.overrideWithValue('1.0.0+1'),
        authRepositoryProvider.overrideWithValue(auth),
        sessionRepositoryProvider.overrideWithValue(sessions),
        profileRepositoryProvider.overrideWithValue(profiles),
        contentRepositoryProvider.overrideWithValue(FakeContentRepository()),
        matchRepositoryProvider.overrideWithValue(matches),
        socialRepositoryProvider.overrideWithValue(social),
        lobbyRepositoryProvider.overrideWithValue(lobby),
        teamRepositoryProvider.overrideWithValue(teams),
        progressRepositoryProvider.overrideWithValue(progress),
        dailyRepositoryProvider.overrideWithValue(daily),
        moderationRepositoryProvider.overrideWithValue(moderation),
        creatorRepositoryProvider.overrideWithValue(creator),
        notificationRepositoryProvider.overrideWithValue(notifications),
        shareServiceProvider.overrideWithValue(share),
        chatRepositoryProvider.overrideWithValue(chat),
        tournamentRepositoryProvider.overrideWithValue(tournaments),
        adsRepositoryProvider.overrideWithValue(ads),
        storeRepositoryProvider.overrideWithValue(store),
        seasonPassRepositoryProvider.overrideWithValue(seasonPass),
        purchaseGatewayProvider.overrideWithValue(purchases),
        plusPollEveryProvider.overrideWithValue(Duration.zero),
        adGatewayProvider.overrideWithValue(adGateway),
        rewardedAdRunnerProvider.overrideWith(
          (ref) => RewardedAdRunner(ref, pollEvery: Duration.zero),
        ),
        realtimeClientProvider.overrideWith((ref) {
          final client = RealtimeClient(
            url: Uri.parse('ws://test/v1/ws'),
            tokens: const _StaticTokens(),
            installationId: 'test-installation',
            connector: (url, headers) async {
              final s = FakeSocket();
              sockets.add(s);
              return s;
            },
          );
          ref.onDispose(client.dispose);
          return client;
        }),
      ],
    );
    addTearDown(container.dispose);
    return TestApp._(
        container,
        p,
        auth,
        sessions,
        profiles,
        matches,
        sockets,
        social,
        lobby,
        teams,
        progress,
        daily,
        moderation,
        creator,
        notifications,
        share,
        chat,
        tournaments,
      )
      ..ads = ads
      ..adGateway = adGateway
      ..store = store
      ..seasonPass = seasonPass
      ..purchases = purchases;
  }

  late final FakeAdsRepository ads;
  late final FakeAdGateway adGateway;
  late final FakeStoreRepository store;
  late final FakeSeasonPassRepository seasonPass;
  late final FakePurchaseGateway purchases;

  final ProviderContainer container;
  final SharedPreferences prefs;
  final FakeMatchRepository matches;
  final FakeSocialRepository social;
  final FakeLobbyRepository lobby;
  final FakeTeamRepository teams;
  final FakeProgressRepository progress;
  final FakeDailyRepository daily;
  final FakeModerationRepository moderation;
  final FakeCreatorRepository creator;
  final FakeNotificationRepository notifications;
  final FakeShareService share;
  final FakeChatRepository chat;
  final FakeTournamentRepository tournaments;

  /// Every WebSocket the app opened, newest last.
  final List<FakeSocket> sockets;
  final FakeAuthRepository auth;
  final FakeSessionRepository sessions;
  final FakeProfileRepository profiles;
}

class _StaticTokens implements TokenSource {
  const _StaticTokens();

  @override
  Future<String?> idToken({bool forceRefresh = false}) async => 'test-token';
}

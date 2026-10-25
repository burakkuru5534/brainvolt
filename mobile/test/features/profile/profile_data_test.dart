import 'package:brainvolt/core/network/api_client.dart';
import 'package:brainvolt/core/network/token_source.dart';
import 'package:brainvolt/features/auth/domain/session.dart';
import 'package:brainvolt/features/profile/data/api_profile_repository.dart';
import 'package:brainvolt/features/profile/domain/me.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../core/network/fake_adapter.dart';

const _settings = {
  'challenges_from': 'friends',
  'show_online_status': true,
  'allow_friend_requests': false,
  'sound_enabled': true,
  'music_enabled': false,
  'haptics_enabled': true,
};

const _profile = {
  'username': 'oyuncu12345',
  'display_name': 'Misafir 1234',
  'avatar_id': 'fox',
  'locale': 'tr',
  'country_code': null,
  'username_change_available_at': '2026-11-01T00:00:00Z',
  'country_change_available_at': null,
};

void main() {
  late FakeAdapter adapter;

  ApiProfileRepository repo() => ApiProfileRepository(
    ApiClient(
      buildDio(
        baseUrl: Uri.parse('http://api.test'),
        appVersion: '1.0.0+1',
        platform: 'ios',
        locale: () => 'tr',
        tokens: const SignedOutTokenSource(),
        adapter: adapter,
      ),
    ),
  );

  setUp(() {
    adapter = FakeAdapter(
      (o) => switch (o.path) {
        '/v1/me' => const FakeResponse(200, {
          'user': {'id': 'u1', 'tier': 'full', 'status': 'active'},
          'profile': _profile,
          'settings': _settings,
        }),
        '/v1/me/profile' => const FakeResponse(200, _profile),
        _ => const FakeResponse(200, _settings),
      },
    );
  });

  test('GET /v1/me', () async {
    final me = await repo().me();
    expect(me.user.tier, Tier.full);
    expect(me.profile.usernameChangeAvailableAt, DateTime.utc(2026, 11));
    expect(me.settings.challengesFrom, ChallengesFrom.friends);
    expect(me.settings.allowFriendRequests, isFalse);
  });

  test('profile PATCH sends only the changed fields', () async {
    await repo().updateProfile(displayName: 'Ayşe', avatarId: 'owl');
    final r = adapter.requests.single;
    expect(r.method, 'PATCH');
    expect(r.data, {'display_name': 'Ayşe', 'avatar_id': 'owl'});
  });

  test('settings PATCH sends snake_case fields', () async {
    final s = await repo().settings();
    await repo().updateSettings(
      s.copyWith(challengesFrom: ChallengesFrom.nobody),
    );
    expect(adapter.requests.last.data, {
      ..._settings,
      'challenges_from': 'nobody',
      'profile_private': false,
    });
  });
}

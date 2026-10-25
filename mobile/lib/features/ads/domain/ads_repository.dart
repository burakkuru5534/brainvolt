/// Rewarded and interstitial ads (D-103, rest.md §11).
library;

/// Rewarded-ad placements.
abstract final class AdPlacement {
  static const bonusXp = 'bonus_xp';
  static const questReroll = 'quest_reroll';
  static const streakRepair = 'streak_repair';

  /// The store's Volt ad (D-105).
  static const volt = 'volt';
}

/// GET /v1/ads/config.
class AdsConfig {
  const AdsConfig({
    this.interstitialEnabled = false,
    this.interstitialEvery = 2,
    this.dailyLimit = 5,
    this.usedToday = 0,
    this.bonusXpLeft = 0,
    this.questRerollsLeft = 0,
    this.streakRepairAvailable = false,
    this.streakRepairDays = 0,
  });

  factory AdsConfig.fromJson(Map<String, dynamic> j) {
    final i = (j['interstitial'] as Map?)?.cast<String, dynamic>() ?? {};
    final r = (j['rewarded'] as Map?)?.cast<String, dynamic>() ?? {};
    final s = (r['streak_repair'] as Map?)?.cast<String, dynamic>() ?? {};
    int n(Object? v, [int d = 0]) => (v as num?)?.toInt() ?? d;
    return AdsConfig(
      interstitialEnabled: i['enabled'] == true,
      interstitialEvery: n(i['every_matches'], 2),
      dailyLimit: n(r['daily_limit'], 5),
      usedToday: n(r['used_today']),
      bonusXpLeft: n(r['bonus_xp_left']),
      questRerollsLeft: n(r['quest_rerolls_left']),
      streakRepairAvailable: s['available'] == true,
      streakRepairDays: n(s['days']),
    );
  }

  final bool interstitialEnabled;
  final int interstitialEvery;
  final int dailyLimit;
  final int usedToday;
  final int bonusXpLeft;
  final int questRerollsLeft;
  final bool streakRepairAvailable;
  final int streakRepairDays;
}

/// A reserved rewarded ad (POST /v1/ads/rewards).
class AdReward {
  const AdReward({
    required this.id,
    required this.placement,
    required this.status,
    this.xp = 0,
    this.streakDays = 0,
  });

  factory AdReward.fromJson(Map<String, dynamic> j) {
    final r = (j['reward'] as Map?)?.cast<String, dynamic>() ?? {};
    return AdReward(
      id: j['id'] as String,
      placement: j['placement'] as String? ?? '',
      status: j['status'] as String? ?? 'pending',
      xp: (r['xp'] as num?)?.toInt() ?? 0,
      streakDays: (r['streak_days'] as num?)?.toInt() ?? 0,
    );
  }

  final String id;
  final String placement;

  /// pending until AdMob's verified callback, then granted.
  final String status;
  final int xp;
  final int streakDays;

  bool get granted => status == 'granted';
}

abstract interface class AdsRepository {
  Future<AdsConfig> config();

  /// [ref] is the match of a bonus_xp reward.
  Future<AdReward> reserve(String placement, {String? ref});

  Future<AdReward> get(String id);

  /// Development only: grants without AdMob (test ads cannot reach a
  /// local server).
  Future<AdReward> devGrant(String id);
}

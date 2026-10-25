/// The Season Pass (D-108, rest.md §10).
library;

int _int(Object? v) => (v as num?)?.toInt() ?? 0;

/// One tier's reward on a track.
class PassReward {
  const PassReward({this.volt = 0, this.shield = false, this.cosmetic = ''});

  factory PassReward.fromJson(Map<String, dynamic> j) => PassReward(
    volt: _int(j['volt']),
    shield: j['shield'] == true,
    cosmetic: j['cosmetic'] as String? ?? '',
  );

  final int volt;
  final bool shield;
  final String cosmetic;
}

/// A tier with both tracks.
class PassTier {
  const PassTier({
    required this.tier,
    this.free,
    this.premium,
    this.freeClaimed = false,
    this.premiumClaimed = false,
  });

  factory PassTier.fromJson(Map<String, dynamic> j) => PassTier(
    tier: _int(j['tier']),
    free: j['free'] is Map
        ? PassReward.fromJson((j['free'] as Map).cast<String, dynamic>())
        : null,
    premium: j['premium'] is Map
        ? PassReward.fromJson((j['premium'] as Map).cast<String, dynamic>())
        : null,
    freeClaimed: j['free_claimed'] == true,
    premiumClaimed: j['premium_claimed'] == true,
  );

  final int tier;
  final PassReward? free;
  final PassReward? premium;
  final bool freeClaimed;
  final bool premiumClaimed;
}

/// GET /v1/me/season-pass.
class SeasonPass {
  const SeasonPass({
    required this.seasonNumber,
    required this.seasonName,
    required this.endsAt,
    this.xp = 0,
    this.tier = 0,
    this.tierXp = 0,
    this.tierSize = 500,
    this.premium = false,
    this.price = 900,
    this.claimable = 0,
    this.tiers = const [],
  });

  factory SeasonPass.fromJson(Map<String, dynamic> j) {
    final s = (j['season'] as Map?)?.cast<String, dynamic>() ?? const {};
    return SeasonPass(
      seasonNumber: _int(s['number']),
      seasonName: s['name'] as String? ?? '',
      endsAt:
          DateTime.tryParse(s['ends_at'] as String? ?? '')?.toLocal() ??
          DateTime.now(),
      xp: _int(j['xp']),
      tier: _int(j['tier']),
      tierXp: _int(j['tier_xp']),
      tierSize: _int(j['tier_size']),
      premium: j['premium'] == true,
      price: _int(j['price']),
      claimable: _int(j['claimable']),
      tiers: [
        for (final t in (j['tiers'] as List?) ?? const [])
          PassTier.fromJson((t as Map).cast<String, dynamic>()),
      ],
    );
  }

  final int seasonNumber;
  final String seasonName;
  final DateTime endsAt;
  final int xp;
  final int tier;
  final int tierXp;
  final int tierSize;
  final bool premium;
  final int price;
  final int claimable;
  final List<PassTier> tiers;

  int get maxTier => tiers.isEmpty ? 40 : tiers.length;
}

/// What a claim gave.
class PassClaim {
  const PassClaim({
    required this.pass,
    this.volt = 0,
    this.shields = 0,
    this.cosmetics = const [],
  });

  factory PassClaim.fromJson(Map<String, dynamic> j) => PassClaim(
    pass: SeasonPass.fromJson((j['pass'] as Map).cast<String, dynamic>()),
    volt: _int(j['volt']),
    shields: _int(j['shields']),
    cosmetics: [
      for (final c in (j['cosmetics'] as List?) ?? const []) c as String,
    ],
  );

  final SeasonPass pass;
  final int volt;
  final int shields;
  final List<String> cosmetics;
}

abstract interface class SeasonPassRepository {
  Future<SeasonPass> pass();

  /// Opens the premium track for Volt.
  Future<SeasonPass> buyPremium();

  Future<PassClaim> claim();
}

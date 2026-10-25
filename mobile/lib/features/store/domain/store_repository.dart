/// BrainVolt Plus and other entitlements (D-104), Volt and the store
/// (D-105; rest.md §15).
library;

int _int(Object? v) => (v as num?)?.toInt() ?? 0;

/// Store items bought with Volt.
abstract final class StoreItemCode {
  static const streakShield = 'streak_shield';
  static const questReroll = 'quest_reroll';
}

/// A store item and its price in Volt.
class StoreItem {
  const StoreItem({required this.code, required this.price});

  factory StoreItem.fromJson(Map<String, dynamic> j) =>
      StoreItem(code: j['code'] as String? ?? '', price: _int(j['price']));

  final String code;
  final int price;
}

/// One movement of the Volt ledger.
class VoltEntry {
  const VoltEntry({
    required this.amount,
    required this.reason,
    required this.createdAt,
    this.ref = '',
  });

  factory VoltEntry.fromJson(Map<String, dynamic> j) => VoltEntry(
    amount: _int(j['amount']),
    reason: j['reason'] as String? ?? '',
    ref: j['ref'] as String? ?? '',
    createdAt:
        DateTime.tryParse(j['created_at'] as String? ?? '')?.toLocal() ??
        DateTime.now(),
  );

  final int amount;

  /// quest, daily_challenge, level_up, tournament, season, ad,
  /// plus_monthly, plus_refund, item, admin.
  final String reason;
  final String ref;
  final DateTime createdAt;
}

/// A Volt pack for sale (D-107); its price comes from the store.
class VoltPack {
  const VoltPack({
    required this.productId,
    required this.volt,
    this.bonus = 0,
    this.cosmetic = '',
    this.once = false,
  });

  factory VoltPack.fromJson(Map<String, dynamic> j) => VoltPack(
    productId: j['product_id'] as String? ?? '',
    volt: _int(j['volt']),
    bonus: _int(j['bonus']),
    cosmetic: j['cosmetic'] as String? ?? '',
    once: j['once'] == true,
  );

  final String productId;
  final int volt;
  final int bonus;
  final String cosmetic;
  final bool once;
}

/// GET /v1/me/wallet.
class Wallet {
  const Wallet({
    this.balance = 0,
    this.items = const [],
    this.history = const [],
    this.adVoltLeft = 0,
    this.adVolt = 5,
    this.packs = const [],
  });

  factory Wallet.fromJson(Map<String, dynamic> j) => Wallet(
    balance: _int(j['balance']),
    items: [
      for (final i in (j['items'] as List?) ?? const [])
        StoreItem.fromJson((i as Map).cast<String, dynamic>()),
    ],
    history: [
      for (final h in (j['history'] as List?) ?? const [])
        VoltEntry.fromJson((h as Map).cast<String, dynamic>()),
    ],
    adVoltLeft: _int(j['ad_volt_left']),
    adVolt: _int(j['ad_volt']),
    packs: [
      for (final p in (j['packs'] as List?) ?? const [])
        VoltPack.fromJson((p as Map).cast<String, dynamic>()),
    ],
  );

  final int balance;
  final List<StoreItem> items;
  final List<VoltEntry> history;

  /// Store ads left today and the Volt each gives.
  final int adVoltLeft;
  final int adVolt;

  /// Volt packs for sale (a bought once-only pack is left out).
  final List<VoltPack> packs;

  int priceOf(String code) =>
      items.where((i) => i.code == code).firstOrNull?.price ?? 0;
}

/// GET /v1/me/entitlements.
class Entitlements {
  const Entitlements({this.plus = false, this.plusUntil});

  factory Entitlements.fromJson(Map<String, dynamic> j) => Entitlements(
    plus: j['plus'] == true,
    plusUntil: DateTime.tryParse(j['plus_until'] as String? ?? '')?.toLocal(),
  );

  final bool plus;

  /// End of the current paid period (renews automatically unless
  /// cancelled); null when Plus has no end.
  final DateTime? plusUntil;
}

/// Cosmetic kinds (D-106).
abstract final class CosmeticKind {
  static const frame = 'frame';
  static const avatar = 'avatar';
  static const nameColor = 'name_color';
  static const victory = 'victory';
  static const matchTheme = 'match_theme';

  static const all = [frame, avatar, nameColor, victory, matchTheme];
}

/// A cosmetic of the catalogue for the caller.
class CosmeticItem {
  const CosmeticItem({
    required this.code,
    required this.kind,
    this.price = 0,
    this.plusOnly = false,
    this.packOnly = false,
    this.owned = false,
    this.equipped = false,
  });

  factory CosmeticItem.fromJson(Map<String, dynamic> j) => CosmeticItem(
    code: j['code'] as String? ?? '',
    kind: j['kind'] as String? ?? '',
    price: _int(j['price']),
    plusOnly: j['plus_only'] == true,
    packOnly: j['pack_only'] == true,
    owned: j['owned'] == true,
    equipped: j['equipped'] == true,
  );

  final String code;
  final String kind;
  final int price;
  final bool plusOnly;

  /// Only in a Volt pack (the starter frame, D-107).
  final bool packOnly;
  final bool owned;
  final bool equipped;

  /// The profile avatar id of an avatar cosmetic (avatar_dragon → dragon).
  String get avatarId => code.startsWith('avatar_') ? code.substring(7) : '';
}

/// GET /v1/store/cosmetics.
class MyCosmetics {
  const MyCosmetics({
    this.items = const [],
    this.equipped = const {},
    this.plus = false,
  });

  factory MyCosmetics.fromJson(Map<String, dynamic> j) => MyCosmetics(
    items: [
      for (final i in (j['items'] as List?) ?? const [])
        CosmeticItem.fromJson((i as Map).cast<String, dynamic>()),
    ],
    equipped: {
      for (final e
          in ((j['equipped'] as Map?) ?? const {})
              .cast<String, dynamic>()
              .entries)
        e.key: e.value as String,
    },
    plus: j['plus'] == true,
  );

  final List<CosmeticItem> items;

  /// The worn code per kind (frame, name_color, victory, match_theme).
  final Map<String, String> equipped;
  final bool plus;

  String worn(String kind) => equipped[kind] ?? '';

  List<CosmeticItem> ofKind(String kind) =>
      items.where((i) => i.kind == kind).toList();

  /// Avatar ids of the special avatars the player owns.
  List<String> get ownedAvatars => [
    for (final i in items)
      if (i.kind == CosmeticKind.avatar && i.owned) i.avatarId,
  ];
}

abstract interface class StoreRepository {
  Future<Entitlements> entitlements();

  Future<Wallet> wallet();

  /// Buys a store item; returns the new balance.
  Future<int> buy(String item);

  Future<MyCosmetics> cosmetics();

  /// Buys a cosmetic once; returns the new balance.
  Future<int> buyCosmetic(String code);

  /// Wears [code] in [kind]; null takes the kind off.
  Future<MyCosmetics> equip(String kind, String? code);
}

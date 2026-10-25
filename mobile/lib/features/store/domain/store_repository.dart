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

/// GET /v1/me/wallet.
class Wallet {
  const Wallet({
    this.balance = 0,
    this.items = const [],
    this.history = const [],
    this.adVoltLeft = 0,
    this.adVolt = 5,
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
  );

  final int balance;
  final List<StoreItem> items;
  final List<VoltEntry> history;

  /// Store ads left today and the Volt each gives.
  final int adVoltLeft;
  final int adVolt;

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

abstract interface class StoreRepository {
  Future<Entitlements> entitlements();

  Future<Wallet> wallet();

  /// Buys a store item; returns the new balance.
  Future<int> buy(String item);
}

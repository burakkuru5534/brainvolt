/// BrainVolt Plus and other entitlements (D-104, rest.md §15).
library;

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
}

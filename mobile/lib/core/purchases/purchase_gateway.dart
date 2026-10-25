import 'dart:io' show Platform;

import 'package:flutter/services.dart';
import 'package:logging/logging.dart';
import 'package:purchases_flutter/purchases_flutter.dart';

final _log = Logger('purchases');

/// A BrainVolt Plus plan as the store sells it (D-104).
class PlusOption {
  const PlusOption({
    required this.id,
    required this.priceString,
    this.trialDays = 0,
    this.handle,
  });

  /// monthly / annual.
  final String id;

  /// The store's localized price, e.g. "₺49,99".
  final String priceString;

  /// Free trial length; 0 = none (or already used).
  final int trialDays;

  /// The store package behind the option (gateway-private).
  final Object? handle;
}

/// How a purchase ended.
enum BuyResult { purchased, cancelled, pending, failed }

/// The store, through RevenueCat. Purchases are tied to the BrainVolt
/// user id (RevenueCat app user id), so the server's webhook can grant
/// the plus entitlement to the right account.
abstract interface class PurchaseGateway {
  /// False where purchases are not set up (tests, iOS until D-057).
  bool get available;

  Future<List<PlusOption>> plusOptions(String userId);

  Future<BuyResult> buy(String userId, PlusOption option);

  /// Restores store purchases to [userId]; true when Plus is active.
  Future<bool> restore(String userId);
}

/// No store.
class NoPurchaseGateway implements PurchaseGateway {
  const NoPurchaseGateway();

  @override
  bool get available => false;

  @override
  Future<List<PlusOption>> plusOptions(String userId) async => const [];

  @override
  Future<BuyResult> buy(String userId, PlusOption option) async =>
      BuyResult.failed;

  @override
  Future<bool> restore(String userId) async => false;
}

/// [PurchaseGateway] over purchases_flutter (Android; RevenueCat project
/// BrainVolt, entitlement `plus`, offering `default`).
class RevenueCatGateway implements PurchaseGateway {
  RevenueCatGateway({required this.androidApiKey});

  /// The public Android SDK key (`goog_…`); not a secret.
  final String androidApiKey;
  String? _user;

  @override
  bool get available => Platform.isAndroid;

  Future<void> _ensure(String userId) async {
    if (_user == userId) {
      return;
    }
    if (_user == null) {
      await Purchases.configure(
        PurchasesConfiguration(androidApiKey)..appUserID = userId,
      );
    } else {
      await Purchases.logIn(userId);
    }
    _user = userId;
  }

  static int _days(Period? p) {
    if (p == null) {
      return 0;
    }
    return switch (p.unit) {
      PeriodUnit.day => p.value,
      PeriodUnit.week => p.value * 7,
      PeriodUnit.month => p.value * 30,
      PeriodUnit.year => p.value * 365,
      _ => 0,
    };
  }

  @override
  Future<List<PlusOption>> plusOptions(String userId) async {
    await _ensure(userId);
    final offering = (await Purchases.getOfferings()).current;
    if (offering == null) {
      return const [];
    }
    return [
      for (final (id, pkg) in [
        ('monthly', offering.monthly),
        ('annual', offering.annual),
      ])
        if (pkg != null)
          PlusOption(
            id: id,
            priceString: pkg.storeProduct.priceString,
            trialDays: _days(
              pkg.storeProduct.defaultOption?.freePhase?.billingPeriod,
            ),
            handle: pkg,
          ),
    ];
  }

  @override
  Future<BuyResult> buy(String userId, PlusOption option) async {
    await _ensure(userId);
    final pkg = option.handle;
    if (pkg is! Package) {
      return BuyResult.failed;
    }
    try {
      final r = await Purchases.purchase(PurchaseParams.package(pkg));
      return r.customerInfo.entitlements.active.containsKey('plus')
          ? BuyResult.purchased
          : BuyResult.pending;
    } on PlatformException catch (e) {
      final code = PurchasesErrorHelper.getErrorCode(e);
      if (code == PurchasesErrorCode.purchaseCancelledError) {
        return BuyResult.cancelled;
      }
      if (code == PurchasesErrorCode.paymentPendingError) {
        return BuyResult.pending;
      }
      _log.warning('purchase failed: $code ${e.message}');
      return BuyResult.failed;
    }
  }

  @override
  Future<bool> restore(String userId) async {
    await _ensure(userId);
    final info = await Purchases.restorePurchases();
    return info.entitlements.active.containsKey('plus');
  }
}

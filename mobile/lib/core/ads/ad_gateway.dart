import 'dart:async';
import 'dart:io' show Platform;

import 'package:flutter/foundation.dart';
import 'package:google_mobile_ads/google_mobile_ads.dart';
import 'package:logging/logging.dart';

import '../../app/config.dart';

final _log = Logger('ads');

/// How a rewarded ad ended.
enum AdShowResult {
  /// Watched to the end; AdMob will call our server (SSV).
  earned,

  /// Closed before the reward.
  dismissed,

  /// No ad to show (no fill, no network, ads not ready).
  unavailable,
}

/// The ad network behind the app (D-103). Tests and unsupported platforms
/// use [NoAdGateway].
abstract interface class AdGateway {
  /// Asks for consent where required (EEA), then starts the SDK.
  Future<void> init();

  /// Loads and shows a rewarded ad; [userId] and [customData] go to
  /// AdMob's server-side verification callback.
  Future<AdShowResult> showRewarded({
    required String userId,
    required String customData,
  });

  /// Shows an interstitial if one is loaded; completes when it is closed.
  Future<bool> showInterstitial();
}

/// No ads: nothing loads, nothing shows.
class NoAdGateway implements AdGateway {
  const NoAdGateway();

  @override
  Future<void> init() async {}

  @override
  Future<AdShowResult> showRewarded({
    required String userId,
    required String customData,
  }) async => AdShowResult.unavailable;

  @override
  Future<bool> showInterstitial() async => false;
}

/// AdMob unit ids. Real ids only in Android release builds of the prod
/// flavor; everything else gets Google's test ids, so development never
/// produces real (invalid) traffic.
class AdUnits {
  const AdUnits({required this.rewarded, required this.interstitial});

  /// The BrainVolt units in AdMob (Android).
  static const android = AdUnits(
    rewarded: 'ca-app-pub-9463680089100140/7982840839',
    interstitial: 'ca-app-pub-9463680089100140/7005508928',
  );

  /// Google's public test units.
  static const androidTest = AdUnits(
    rewarded: 'ca-app-pub-3940256099942544/5224354917',
    interstitial: 'ca-app-pub-3940256099942544/1033173712',
  );
  static const iosTest = AdUnits(
    rewarded: 'ca-app-pub-3940256099942544/1712485313',
    interstitial: 'ca-app-pub-3940256099942544/4411468910',
  );

  /// The units for this build. iOS has no BrainVolt units yet (D-057).
  static AdUnits forBuild(Flavor flavor) {
    if (Platform.isIOS) {
      return iosTest;
    }
    return flavor == Flavor.prod && kReleaseMode ? android : androidTest;
  }

  final String rewarded;
  final String interstitial;
}

/// [AdGateway] over google_mobile_ads with Google's consent SDK (UMP).
class GoogleAdGateway implements AdGateway {
  GoogleAdGateway(this.units);

  final AdUnits units;
  final _ready = Completer<bool>();
  InterstitialAd? _interstitial;
  var _loadingInterstitial = false;

  @override
  Future<void> init() async {
    if (_ready.isCompleted) {
      return;
    }
    try {
      await _consent();
      if (!await ConsentInformation.instance.canRequestAds()) {
        _ready.complete(false);
        return;
      }
      await MobileAds.instance.updateRequestConfiguration(
        // 13+ app (D-020): teen content rating at most.
        RequestConfiguration(maxAdContentRating: MaxAdContentRating.t),
      );
      await MobileAds.instance.initialize();
      _ready.complete(true);
      _loadInterstitial();
    } on Object catch (e) {
      _log.warning('ads init failed: $e');
      if (!_ready.isCompleted) {
        _ready.complete(false);
      }
    }
  }

  Future<void> _consent() {
    final done = Completer<void>();
    ConsentInformation.instance.requestConsentInfoUpdate(
      ConsentRequestParameters(),
      () async {
        await ConsentForm.loadAndShowConsentFormIfRequired((error) {
          if (error != null) {
            _log.info('consent form: ${error.message}');
          }
          if (!done.isCompleted) {
            done.complete();
          }
        });
      },
      (error) {
        _log.info('consent info: ${error.message}');
        if (!done.isCompleted) {
          done.complete();
        }
      },
    );
    return done.future.timeout(const Duration(seconds: 20), onTimeout: () {});
  }

  Future<bool> get _ok => _ready.future.timeout(
    const Duration(seconds: 10),
    onTimeout: () => false,
  );

  void _loadInterstitial() {
    if (_interstitial != null || _loadingInterstitial) {
      return;
    }
    _loadingInterstitial = true;
    unawaited(
      InterstitialAd.load(
        adUnitId: units.interstitial,
        request: const AdRequest(),
        adLoadCallback: InterstitialAdLoadCallback(
          onAdLoaded: (ad) {
            _loadingInterstitial = false;
            _interstitial = ad;
          },
          onAdFailedToLoad: (error) {
            _loadingInterstitial = false;
            _log.info('interstitial not loaded: ${error.message}');
          },
        ),
      ),
    );
  }

  @override
  Future<bool> showInterstitial() async {
    if (!await _ok) {
      return false;
    }
    final ad = _interstitial;
    if (ad == null) {
      _loadInterstitial();
      return false;
    }
    _interstitial = null;
    final closed = Completer<bool>();
    ad.fullScreenContentCallback = FullScreenContentCallback(
      onAdDismissedFullScreenContent: (ad) {
        unawaited(ad.dispose());
        closed.complete(true);
      },
      onAdFailedToShowFullScreenContent: (ad, error) {
        unawaited(ad.dispose());
        closed.complete(false);
      },
    );
    await ad.show();
    final shown = await closed.future;
    _loadInterstitial();
    return shown;
  }

  @override
  Future<AdShowResult> showRewarded({
    required String userId,
    required String customData,
  }) async {
    if (!await _ok) {
      return AdShowResult.unavailable;
    }
    final loaded = Completer<RewardedAd?>();
    await RewardedAd.load(
      adUnitId: units.rewarded,
      request: const AdRequest(),
      rewardedAdLoadCallback: RewardedAdLoadCallback(
        onAdLoaded: loaded.complete,
        onAdFailedToLoad: (error) {
          _log.info('rewarded not loaded: ${error.message}');
          loaded.complete(null);
        },
      ),
    );
    final ad = await loaded.future.timeout(
      const Duration(seconds: 15),
      onTimeout: () => null,
    );
    if (ad == null) {
      return AdShowResult.unavailable;
    }
    await ad.setServerSideOptions(
      ServerSideVerificationOptions(userId: userId, customData: customData),
    );
    var earned = false;
    final closed = Completer<void>();
    ad.fullScreenContentCallback = FullScreenContentCallback(
      onAdDismissedFullScreenContent: (ad) {
        unawaited(ad.dispose());
        closed.complete();
      },
      onAdFailedToShowFullScreenContent: (ad, error) {
        unawaited(ad.dispose());
        closed.complete();
      },
    );
    await ad.show(onUserEarnedReward: (_, _) => earned = true);
    await closed.future;
    return earned ? AdShowResult.earned : AdShowResult.dismissed;
  }
}

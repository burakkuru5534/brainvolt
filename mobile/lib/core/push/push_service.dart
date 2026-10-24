import 'dart:async';

import 'package:firebase_messaging/firebase_messaging.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:logging/logging.dart';

final _log = Logger('push');

/// Push messages on this device (FCM, D-086).
abstract interface class PushService {
  /// Asks for permission once and returns this installation's FCM send
  /// target (its Firebase Installation ID), or null when push is denied or
  /// unavailable.
  Future<String?> token();

  /// New send targets after FCM registers the app again.
  Stream<String> get tokenRefresh;

  /// App routes of notifications the user tapped while the app ran.
  Stream<String> get opened;

  /// The route of the notification that launched the app, if any.
  Future<String?> initialRoute();
}

/// No push (tests, platforms without Firebase).
class NoPushService implements PushService {
  const NoPushService();

  @override
  Future<String?> token() async => null;

  @override
  Stream<String> get tokenRefresh => const Stream.empty();

  @override
  Stream<String> get opened => const Stream.empty();

  @override
  Future<String?> initialRoute() async => null;
}

/// FCM through firebase_messaging; the server sends the route as the
/// `deep_link` data field.
class FirebasePushService implements PushService {
  FirebasePushService([FirebaseMessaging? messaging])
    : _customMessaging = messaging;

  final FirebaseMessaging? _customMessaging;

  FirebaseMessaging get _fm => _customMessaging ?? FirebaseMessaging.instance;

  @override
  Future<String?> token() async {
    try {
      final s = await _fm.requestPermission();
      if (s.authorizationStatus == AuthorizationStatus.denied) {
        return null;
      }
      await _fm.setAutoInitEnabled(true);
      final id = _fm.onRegistered.first.timeout(const Duration(seconds: 10));
      await _fm.register();
      return await id;
    } on Object catch (e) {
      // No Google Play services, simulator without APNs, GMS proxy failure, etc.
      _log.warning('no push token', e);
      return null;
    }
  }

  @override
  Stream<String> get tokenRefresh {
    try {
      return _fm.onRegistered.handleError((Object e) {
        _log.warning('tokenRefresh stream error', e);
      });
    } on Object catch (e) {
      _log.warning('tokenRefresh setup failed', e);
      return const Stream.empty();
    }
  }

  @override
  Stream<String> get opened {
    try {
      return FirebaseMessaging.onMessageOpenedApp
          .map((m) => m.data['deep_link'] as String? ?? '')
          .where((r) => r.startsWith('/'))
          .handleError((Object e) {
            _log.warning('opened push stream error', e);
          });
    } on Object catch (e) {
      _log.warning('opened push stream setup failed', e);
      return const Stream.empty();
    }
  }

  @override
  Future<String?> initialRoute() async {
    try {
      final m = await _fm.getInitialMessage();
      final r = m?.data['deep_link'] as String?;
      return r != null && r.startsWith('/') ? r : null;
    } on Object catch (e) {
      _log.warning('initialRoute failed', e);
      return null;
    }
  }
}

/// The device's push service; main installs FCM.
final pushServiceProvider = Provider<PushService>(
  (ref) => const NoPushService(),
);

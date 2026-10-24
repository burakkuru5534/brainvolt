import 'package:flutter/widgets.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'realtime_providers.dart';

/// Whether the app is in the foreground (resumed or briefly inactive).
class AppForeground extends Notifier<bool> {
  @override
  bool build() => true;

  // ignore: use_setters_to_change_properties
  void set(bool value) => state = value;
}

final appForegroundProvider = NotifierProvider<AppForeground, bool>(
  AppForeground.new,
);

/// Closes the WebSocket as soon as the app leaves the foreground and lets
/// it reconnect on return (D-097): an open socket keeps the server billing.
/// `inactive` (a system dialog, the app switcher) still counts as
/// foreground; `hidden`, `paused` and `detached` do not.
class RealtimeLifecycle extends ConsumerStatefulWidget {
  const RealtimeLifecycle({required this.child, super.key});

  final Widget child;

  @override
  ConsumerState<RealtimeLifecycle> createState() => _RealtimeLifecycleState();
}

class _RealtimeLifecycleState extends ConsumerState<RealtimeLifecycle> {
  late final AppLifecycleListener _listener;

  @override
  void initState() {
    super.initState();
    _listener = AppLifecycleListener(onStateChange: _onState);
  }

  void _onState(AppLifecycleState s) {
    final foreground =
        s == AppLifecycleState.resumed || s == AppLifecycleState.inactive;
    ref.read(realtimeClientProvider).setForeground(foreground: foreground);
    ref.read(appForegroundProvider.notifier).set(foreground);
  }

  @override
  void dispose() {
    _listener.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => widget.child;
}

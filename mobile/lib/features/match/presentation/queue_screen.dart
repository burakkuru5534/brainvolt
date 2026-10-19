import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../app/router.dart';
import '../../../core/l10n/l10n.dart';
import '../domain/match_repository.dart';
import 'match_type_name.dart';
import 'play_controllers.dart';
import 'play_error_text.dart';

/// Looking for an opponent (or starting a practice match), then straight
/// to the match screen.
class QueueScreen extends ConsumerStatefulWidget {
  const QueueScreen({
    this.practice = false,
    this.mode = const QueueMode(),
    super.key,
  });

  final bool practice;
  final QueueMode mode;

  @override
  ConsumerState<QueueScreen> createState() => _QueueScreenState();
}

class _QueueScreenState extends ConsumerState<QueueScreen> {
  Timer? _tick;

  @override
  void initState() {
    super.initState();
    _tick = Timer.periodic(const Duration(seconds: 1), (_) => setState(() {}));
    unawaited(Future.microtask(_search));
  }

  Future<void> _search() => ref
      .read(queueControllerProvider.notifier)
      .search(practice: widget.practice, mode: widget.mode);

  @override
  void dispose() {
    _tick?.cancel();
    super.dispose();
  }

  Future<void> _cancel() async {
    await ref
        .read(queueControllerProvider.notifier)
        .cancel()
        .catchError((_) {});
    if (mounted) {
      context.go(Routes.home);
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    ref.listen(queueControllerProvider, (_, next) {
      if (next is QueueFound) {
        context.go(Routes.match(next.matchId));
      }
    });
    final state = ref.watch(queueControllerProvider);
    return PopScope(
      canPop: false,
      onPopInvokedWithResult: (didPop, _) {
        if (!didPop) {
          unawaited(_cancel());
        }
      },
      child: Scaffold(
        appBar: AppBar(title: Text(matchTypeName(l10n, widget.mode.matchType))),
        body: Center(
          child: Padding(
            padding: const EdgeInsets.all(24),
            child: switch (state) {
              QueueFailed(:final error) => Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  Text(playErrorText(l10n, error), textAlign: TextAlign.center),
                  const SizedBox(height: 16),
                  FilledButton(onPressed: _search, child: Text(l10n.retry)),
                  TextButton(
                    onPressed: () => context.go(Routes.home),
                    child: Text(l10n.backHome),
                  ),
                ],
              ),
              _ => Column(
                key: const Key('queue-searching'),
                mainAxisSize: MainAxisSize.min,
                children: [
                  const CircularProgressIndicator(),
                  const SizedBox(height: 24),
                  Text(
                    l10n.searchingTitle,
                    style: Theme.of(context).textTheme.titleLarge,
                  ),
                  if (state case QueueSearching(:final since)) ...[
                    const SizedBox(height: 8),
                    Text(
                      l10n.searchingElapsed(
                        DateTime.now().difference(since).inSeconds,
                      ),
                    ),
                  ],
                  const SizedBox(height: 24),
                  OutlinedButton(
                    key: const Key('queue-cancel'),
                    onPressed: _cancel,
                    child: Text(l10n.searchingCancel),
                  ),
                ],
              ),
            },
          ),
        ),
      ),
    );
  }
}

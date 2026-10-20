import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../app/router.dart';
import '../../../core/l10n/l10n.dart';
import '../../../core/realtime/realtime_providers.dart';
import '../../../core/widgets/bv.dart';
import '../../lobby/data/lobby_providers.dart';
import '../../lobby/domain/lobby_repository.dart';
import '../domain/match_repository.dart';
import 'mode_picker_screen.dart';
import 'play_error_text.dart';
import 'result_screen.dart';

/// The pending revenge request against this player for [matchId], if any.
final incomingRevengeProvider = FutureProvider.autoDispose
    .family<Challenge?, String>((ref, matchId) async {
      final incoming = await ref
          .watch(lobbyRepositoryProvider)
          .challenges(incoming: true);
      return incoming
          .where(
            (c) =>
                c.kind == 'revenge' &&
                c.parentMatchId == matchId &&
                c.status == 'pending',
          )
          .firstOrNull;
    });

/// Revenge on the match-end screen (D-045, D-089): the losing side sends
/// it from here, the winners see it arrive and answer it here, and both
/// go straight to the new match once it is accepted.
class RevengePanel extends ConsumerStatefulWidget {
  const RevengePanel({required this.matchId, super.key});

  final String matchId;

  @override
  ConsumerState<RevengePanel> createState() => _RevengePanelState();
}

class _RevengePanelState extends ConsumerState<RevengePanel> {
  StreamSubscription<Object>? _sub;
  String? _sentId;
  var _declined = false;

  @override
  void initState() {
    super.initState();
    _sub = ref.read(realtimeClientProvider).frames.listen((f) {
      if (f.type != 'challenge.received' && f.type != 'challenge.updated') {
        return;
      }
      ref.invalidate(incomingRevengeProvider(widget.matchId));
      if (f.type == 'challenge.updated' && f.data['challenge_id'] == _sentId) {
        if (f.data['match_id'] case final String id
            when f.data['status'] == 'started') {
          if (mounted) {
            context.go(Routes.match(id));
          }
        } else if (mounted) {
          setState(() => _declined = true);
        }
      }
    });
  }

  @override
  void dispose() {
    unawaited(_sub?.cancel());
    super.dispose();
  }

  Future<void> _respond(Challenge c, ChallengeAction action) async {
    final l10n = context.l10n;
    final messenger = ScaffoldMessenger.of(context)..hideCurrentSnackBar();
    try {
      final updated = await ref
          .read(lobbyRepositoryProvider)
          .respondChallenge(c.id, action);
      if (updated.matchId case final id? when mounted) {
        context.go(Routes.match(id));
        return;
      }
    } on Object catch (e) {
      messenger.showSnackBar(SnackBar(content: Text(playErrorText(l10n, e))));
    }
    ref.invalidate(incomingRevengeProvider(widget.matchId));
  }

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    final result = ref.watch(matchResultProvider(widget.matchId)).value;
    final incoming = ref.watch(incomingRevengeProvider(widget.matchId)).value;
    return Column(
      mainAxisSize: MainAxisSize.min,
      children: [
        if (incoming != null)
          BvCard(
            key: const Key('revenge-incoming'),
            padding: const EdgeInsets.all(14),
            child: Column(
              children: [
                Text(
                  l10n.revengeIncoming(incoming.challenger.displayName),
                  textAlign: TextAlign.center,
                  style: Theme.of(context).textTheme.titleMedium,
                ),
                const SizedBox(height: 10),
                Row(
                  mainAxisAlignment: MainAxisAlignment.center,
                  children: [
                    FilledButton.icon(
                      key: const Key('revenge-accept'),
                      icon: const Icon(Icons.sports_mma),
                      onPressed: () =>
                          _respond(incoming, ChallengeAction.accept),
                      label: Text(l10n.accept),
                    ),
                    const SizedBox(width: 8),
                    TextButton(
                      key: const Key('revenge-decline'),
                      onPressed: () =>
                          _respond(incoming, ChallengeAction.decline),
                      child: Text(l10n.decline),
                    ),
                  ],
                ),
              ],
            ),
          ),
        if (result?.revenge case RevengeInfo(available: true))
          RevengeButton(
            matchId: widget.matchId,
            onSent: (c) => setState(() => _sentId = c.id),
          ),
        if (_declined)
          Padding(
            padding: const EdgeInsets.only(top: 8),
            child: Text(l10n.revengeDeclined, key: const Key('revenge-no')),
          ),
      ],
    );
  }
}

/// Revenge (D-045): the losing side picks a home category and challenges
/// the winners.
class RevengeButton extends ConsumerStatefulWidget {
  const RevengeButton({required this.matchId, this.onSent, super.key});

  final String matchId;
  final void Function(Challenge)? onSent;

  @override
  ConsumerState<RevengeButton> createState() => _RevengeButtonState();
}

class _RevengeButtonState extends ConsumerState<RevengeButton> {
  var _sent = false;

  Future<void> _ask() async {
    final l10n = context.l10n;
    final messenger = ScaffoldMessenger.of(context)..hideCurrentSnackBar();
    try {
      final cats = await ref.read(categoriesProvider.future);
      if (!mounted) {
        return;
      }
      final cat = await showDialog<String>(
        context: context,
        builder: (context) => SimpleDialog(
          title: Text(l10n.revengePickCategory),
          children: [
            for (final c in cats)
              SimpleDialogOption(
                key: Key('revenge-cat-${c.id}'),
                onPressed: () => Navigator.pop(context, c.id),
                child: Text(c.name),
              ),
          ],
        ),
      );
      if (cat == null) {
        return;
      }
      final c = await ref
          .read(lobbyRepositoryProvider)
          .revenge(widget.matchId, cat);
      if (mounted) {
        setState(() => _sent = true);
        widget.onSent?.call(c);
      }
      messenger.showSnackBar(SnackBar(content: Text(l10n.revengeSent)));
    } on Object catch (e) {
      messenger.showSnackBar(SnackBar(content: Text(playErrorText(l10n, e))));
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    return Padding(
      padding: const EdgeInsets.only(top: 12),
      child: OutlinedButton.icon(
        key: const Key('revenge'),
        icon: const Icon(Icons.sports_mma),
        onPressed: _sent ? null : _ask,
        label: Text(_sent ? l10n.revengeWaiting : l10n.revengeButton),
      ),
    );
  }
}

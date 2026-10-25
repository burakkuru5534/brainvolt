import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../app/router.dart';
import '../../../app/theme.dart';
import '../../../core/l10n/l10n.dart';
import '../../../core/widgets/bv.dart';
import '../../../core/widgets/dialogs.dart';
import '../../match/presentation/match_type_name.dart';
import '../../match/presentation/play_error_text.dart';
import '../../store/presentation/plus_screen.dart';
import '../../store/presentation/volt_screen.dart';
import '../data/tournament_providers.dart';
import 'tournament_screens.dart';

/// The set-up fee in Volt; Plus sets up for free (D-109).
const friendTournamentFee = 300;

/// Sets up a friend tournament (D-109): a name, the match type, 4/8/16
/// places and the start; only the owner's friends join.
class FriendTournamentScreen extends ConsumerStatefulWidget {
  const FriendTournamentScreen({super.key});

  @override
  ConsumerState<FriendTournamentScreen> createState() =>
      _FriendTournamentScreenState();
}

class _FriendTournamentScreenState
    extends ConsumerState<FriendTournamentScreen> {
  final _name = TextEditingController();
  var _type = 'quick_duel';
  var _size = 8;
  var _start = 1;
  var _busy = false;

  @override
  void dispose() {
    _name.dispose();
    super.dispose();
  }

  DateTime _startsAt() {
    final now = DateTime.now();
    return switch (_start) {
      0 => now.add(const Duration(minutes: 30)),
      1 => now.add(const Duration(hours: 1)),
      2 => now.add(const Duration(hours: 3)),
      _ => DateTime(now.year, now.month, now.day + 1, 20),
    };
  }

  Future<void> _submit() async {
    final l10n = context.l10n;
    setState(() => _busy = true);
    try {
      final id = await ref
          .read(tournamentRepositoryProvider)
          .createFriends(
            name: _name.text.trim(),
            matchType: _type,
            maxEntrants: _size,
            startsAt: _startsAt(),
          );
      ref
        ..invalidate(tournamentsProvider)
        ..invalidate(walletProvider);
      if (mounted) {
        context.pushReplacement(Routes.tournament(id));
      }
    } on Object catch (e) {
      if (mounted) {
        showSnack(context, playErrorText(l10n, e));
      }
    } finally {
      if (mounted) {
        setState(() => _busy = false);
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    final bv = BvColors.of(context);
    final text = Theme.of(context).textTheme;
    final plus = ref.watch(entitlementsProvider).value?.plus ?? false;
    final balance = ref.watch(walletProvider).value?.balance;
    final starts = [
      l10n.friendStart30m,
      l10n.friendStart1h,
      l10n.friendStart3h,
      l10n.friendStartTomorrow,
    ];
    Widget chips<T>(List<(T, String)> options, T value, void Function(T) set) =>
        Wrap(
          spacing: 8,
          runSpacing: 8,
          children: [
            for (final (v, label) in options)
              ChoiceChip(
                key: Key('ft-$v'),
                label: Text(label),
                selected: v == value,
                onSelected: (_) => setState(() => set(v)),
              ),
          ],
        );
    return Scaffold(
      appBar: AppBar(title: Text(l10n.friendTournamentTitle)),
      body: ListView(
        padding: const EdgeInsets.fromLTRB(16, 0, 16, 24),
        children: [
          NoticeBar(l10n.friendTournamentRules, icon: Icons.group_rounded),
          const SizedBox(height: 12),
          TextField(
            key: const Key('ft-name'),
            controller: _name,
            maxLength: 40,
            decoration: InputDecoration(labelText: l10n.friendTournamentName),
            onChanged: (_) => setState(() {}),
          ),
          SectionTitle(l10n.friendTournamentType),
          chips(
            [
              for (final t in ['quick_duel', 'sudden_death', 'marathon'])
                (t, matchTypeName(l10n, t)),
            ],
            _type,
            (v) => _type = v,
          ),
          SectionTitle(l10n.friendTournamentSize),
          chips(
            [
              for (final n in [4, 8, 16]) (n, '$n'),
            ],
            _size,
            (v) => _size = v,
          ),
          SectionTitle(l10n.friendTournamentStart),
          chips(
            [for (final (i, s) in starts.indexed) (i, s)],
            _start,
            (v) => _start = v,
          ),
          const SizedBox(height: 20),
          Row(
            children: [
              Icon(
                plus ? Icons.workspace_premium_rounded : voltIcon,
                color: plus ? bv.gold : bv.accent,
              ),
              const SizedBox(width: 8),
              Expanded(
                child: Text(
                  plus
                      ? l10n.friendTournamentFree
                      : l10n.friendTournamentFee(friendTournamentFee),
                  key: const Key('ft-fee'),
                  style: text.titleSmall,
                ),
              ),
              if (!plus && balance != null)
                Pill('$balance', icon: voltIcon, color: bv.accent),
            ],
          ),
          const SizedBox(height: 12),
          GradientButton(
            key: const Key('ft-submit'),
            icon: Icons.emoji_events_rounded,
            label: l10n.friendTournamentSubmit,
            onPressed:
                _busy ||
                    _name.text.trim().length < 3 ||
                    (!plus && (balance ?? 0) < friendTournamentFee)
                ? null
                : _submit,
          ),
        ],
      ),
    );
  }
}

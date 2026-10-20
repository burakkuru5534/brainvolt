import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../app/router.dart';
import '../../../core/l10n/l10n.dart';
import '../../../core/widgets/bv.dart';
import '../../lobby/data/lobby_providers.dart';
import '../../lobby/domain/lobby_repository.dart';
import '../../match/presentation/play_error_text.dart';
import '../data/creator_providers.dart';
import '../domain/creator_repository.dart';

/// The caller's quizzes.
final myQuizzesProvider = FutureProvider.autoDispose<List<Quiz>>(
  (ref) => ref.watch(creatorRepositoryProvider).mine(),
);

/// Published quizzes, most liked first.
final discoverQuizzesProvider = FutureProvider.autoDispose<List<Quiz>>(
  (ref) => ref.watch(creatorRepositoryProvider).discover(),
);

/// A status as the app names it.
String quizStatusName(AppLocalizations l10n, String status) => switch (status) {
  'submitted' => l10n.quizStatusSubmitted,
  'published' => l10n.quizStatusPublished,
  'rejected' => l10n.quizStatusRejected,
  'unpublished' => l10n.quizStatusUnpublished,
  _ => l10n.quizStatusDraft,
};

/// Opens a Quick Duel room playing [quiz] (D-083).
Future<void> playQuizInRoom(
  BuildContext context,
  WidgetRef ref,
  Quiz quiz,
) async {
  final l10n = context.l10n;
  final messenger = ScaffoldMessenger.of(context);
  try {
    final room = await ref
        .read(lobbyRepositoryProvider)
        .createRoom(MatchType.quickDuel, creatorQuizId: quiz.id);
    if (context.mounted) {
      unawaited(context.push(Routes.room(room.id)));
    }
  } on Object catch (e) {
    messenger.showSnackBar(SnackBar(content: Text(playErrorText(l10n, e))));
  }
}

/// Creator Quiz: the caller's quizzes and the published ones.
class QuizzesScreen extends ConsumerWidget {
  const QuizzesScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = context.l10n;
    return DefaultTabController(
      length: 2,
      child: Scaffold(
        appBar: AppBar(
          title: Text(l10n.quizzesTitle),
          bottom: TabBar(
            tabs: [
              Tab(key: const Key('tab-mine'), text: l10n.quizzesMine),
              Tab(key: const Key('tab-discover'), text: l10n.quizzesDiscover),
            ],
          ),
        ),
        floatingActionButton: FloatingActionButton.extended(
          key: const Key('quiz-new'),
          icon: const Icon(Icons.add),
          label: Text(l10n.quizNew),
          onPressed: () async {
            await context.push(Routes.quizNew);
            ref.invalidate(myQuizzesProvider);
          },
        ),
        body: const TabBarView(children: [_Mine(), _Discover()]),
      ),
    );
  }
}

class _Mine extends ConsumerWidget {
  const _Mine();

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = context.l10n;
    return switch (ref.watch(myQuizzesProvider)) {
      AsyncData(:final value) when value.isEmpty => EmptyState(
        icon: Icons.edit_note,
        text: l10n.quizzesEmptyMine,
      ),
      AsyncData(:final value) => ListView(
        padding: const EdgeInsets.only(bottom: 88),
        children: [for (final q in value) _MyQuiz(q)],
      ),
      AsyncError(:final error) => Center(
        child: Text(playErrorText(l10n, error)),
      ),
      _ => const Center(child: CircularProgressIndicator()),
    };
  }
}

class _MyQuiz extends ConsumerWidget {
  const _MyQuiz(this.quiz);

  final Quiz quiz;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = context.l10n;
    final repo = ref.read(creatorRepositoryProvider);
    Future<void> act(Future<void> Function() fn, [String? done]) async {
      final messenger = ScaffoldMessenger.of(context);
      try {
        await fn();
        ref.invalidate(myQuizzesProvider);
        if (done != null) {
          messenger.showSnackBar(SnackBar(content: Text(done)));
        }
      } on Object catch (e) {
        messenger.showSnackBar(SnackBar(content: Text(playErrorText(l10n, e))));
      }
    }

    return ListTile(
      key: Key('my-quiz-${quiz.id}'),
      title: Text(quiz.title),
      subtitle: Text(
        [
          '${quizStatusName(l10n, quiz.status)} · '
              '${l10n.quizLikes(quiz.likeCount, quiz.playCount)}',
          if (quiz.rejectionReason case final r? when quiz.status == 'rejected')
            l10n.quizRejectedReason(r),
        ].join('\n'),
      ),
      onTap: () async {
        await context.push(Routes.quiz(quiz.id));
        ref.invalidate(myQuizzesProvider);
      },
      trailing: PopupMenuButton<String>(
        key: Key('quiz-menu-${quiz.id}'),
        onSelected: (v) async {
          switch (v) {
            case 'play':
              if (!quiz.published) {
                ScaffoldMessenger.of(context).showSnackBar(
                  SnackBar(content: Text(l10n.quizUnpublishedNote)),
                );
              }
              await playQuizInRoom(context, ref, quiz);
            case 'submit':
              await act(() => repo.submit(quiz.id), l10n.quizSubmitted);
            case 'delete':
              final ok = await showDialog<bool>(
                context: context,
                builder: (context) => AlertDialog(
                  content: Text(l10n.quizDeleteConfirm),
                  actions: [
                    TextButton(
                      onPressed: () => Navigator.pop(context, false),
                      child: Text(
                        MaterialLocalizations.of(context).cancelButtonLabel,
                      ),
                    ),
                    FilledButton(
                      key: const Key('quiz-delete-confirm'),
                      onPressed: () => Navigator.pop(context, true),
                      child: Text(l10n.quizDelete),
                    ),
                  ],
                ),
              );
              if (ok ?? false) {
                await act(() => repo.delete(quiz.id));
              }
          }
        },
        itemBuilder: (context) => [
          PopupMenuItem(
            key: const Key('quiz-play'),
            value: 'play',
            child: Text(l10n.quizPlayInRoom),
          ),
          if (quiz.status != 'submitted' && !quiz.published)
            PopupMenuItem(
              key: const Key('quiz-submit'),
              value: 'submit',
              child: Text(l10n.quizSubmit),
            ),
          PopupMenuItem(
            key: const Key('quiz-delete'),
            value: 'delete',
            child: Text(l10n.quizDelete),
          ),
        ],
      ),
    );
  }
}

class _Discover extends ConsumerWidget {
  const _Discover();

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = context.l10n;
    return switch (ref.watch(discoverQuizzesProvider)) {
      AsyncData(:final value) when value.isEmpty => EmptyState(
        icon: Icons.travel_explore,
        text: l10n.quizzesEmptyDiscover,
      ),
      AsyncData(:final value) => ListView(
        padding: const EdgeInsets.only(bottom: 88),
        children: [for (final q in value) _PublishedQuiz(q)],
      ),
      AsyncError(:final error) => Center(
        child: Text(playErrorText(l10n, error)),
      ),
      _ => const Center(child: CircularProgressIndicator()),
    };
  }
}

class _PublishedQuiz extends ConsumerWidget {
  const _PublishedQuiz(this.quiz);

  final Quiz quiz;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = context.l10n;
    return ListTile(
      key: Key('quiz-${quiz.id}'),
      title: Text(quiz.title),
      subtitle: Text(
        '${l10n.quizMeta(quiz.questionCount, quiz.authorName)}\n'
        '${l10n.quizLikes(quiz.likeCount, quiz.playCount)}',
      ),
      isThreeLine: true,
      trailing: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          IconButton(
            key: Key('quiz-like-${quiz.id}'),
            tooltip: l10n.quizLike,
            icon: Icon(quiz.liked ? Icons.favorite : Icons.favorite_border),
            onPressed: () async {
              final messenger = ScaffoldMessenger.of(context);
              try {
                await ref
                    .read(creatorRepositoryProvider)
                    .like(quiz.id, like: !quiz.liked);
                ref.invalidate(discoverQuizzesProvider);
              } on Object catch (e) {
                messenger.showSnackBar(
                  SnackBar(content: Text(playErrorText(l10n, e))),
                );
              }
            },
          ),
          IconButton(
            key: Key('quiz-room-${quiz.id}'),
            tooltip: l10n.quizPlayInRoom,
            icon: const Icon(Icons.meeting_room),
            onPressed: () => playQuizInRoom(context, ref, quiz),
          ),
        ],
      ),
    );
  }
}

import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../app/session_controller.dart';
import '../../../app/theme.dart';
import '../../../core/l10n/l10n.dart';
import '../../../core/realtime/realtime_client.dart';
import '../../../core/realtime/realtime_providers.dart';
import '../../../core/widgets/bv.dart';
import '../../match/presentation/play_error_text.dart';
import '../../profile/domain/avatars.dart';
import '../data/chat_providers.dart';
import '../domain/chat_repository.dart';

/// A quick phrase as the app shows it.
String quickPhraseText(AppLocalizations l10n, String id) => switch (id) {
  'good_luck' => l10n.chatQuickGoodLuck,
  'well_played' => l10n.chatQuickWellPlayed,
  'nice' => l10n.chatQuickNice,
  'thanks' => l10n.chatQuickThanks,
  'oops' => l10n.chatQuickOops,
  'lets_go' => l10n.chatQuickLetsGo,
  'gg' => l10n.chatQuickGg,
  'rematch' => l10n.chatQuickRematch,
  _ => id,
};

/// The chat of a team or of the caller's couple (D-088): history over
/// REST, live messages and typing over the WebSocket, sending with
/// `chat.send`.
class ChatScreen extends ConsumerStatefulWidget {
  const ChatScreen({this.teamId, this.coupleId, this.title, super.key});

  final String? teamId;
  final String? coupleId;
  final String? title;

  @override
  ConsumerState<ChatScreen> createState() => _ChatScreenState();
}

class _ChatScreenState extends ConsumerState<ChatScreen> {
  final _text = TextEditingController();
  ChatChannel? _channel;
  Object? _error;

  /// Oldest first.
  final _messages = <ChatMessage>[];
  String? _typing;
  Timer? _typingTimer;
  DateTime _lastTypingSent = DateTime(2000);
  StreamSubscription<ServerFrame>? _frames;
  RealtimeLease? _lease;

  RealtimeClient get _rt => ref.read(realtimeClientProvider);

  @override
  void initState() {
    super.initState();
    unawaited(_load());
  }

  @override
  void dispose() {
    _lease?.release();
    _text.dispose();
    _typingTimer?.cancel();
    unawaited(_frames?.cancel());
    super.dispose();
  }

  Future<void> _load() async {
    final repo = ref.read(chatRepositoryProvider);
    try {
      final c = await repo.open(
        teamId: widget.teamId,
        coupleId: widget.coupleId,
      );
      final history = await repo.history(c.id);
      if (!mounted) {
        return;
      }
      setState(() {
        _channel = c;
        _messages
          ..clear()
          ..addAll(history.reversed);
      });
      _lease ??= _rt.acquire('chat');
      _frames = _rt.frames.listen(_onFrame);
      _markRead();
    } on Object catch (e) {
      if (mounted) {
        setState(() => _error = e);
      }
    }
  }

  void _markRead() {
    final c = _channel;
    if (c == null || _messages.isEmpty) {
      return;
    }
    unawaited(
      ref
          .read(chatRepositoryProvider)
          .read(c.id, _messages.last.id)
          .catchError((Object _) {}),
    );
  }

  void _onFrame(ServerFrame f) {
    final c = _channel;
    if (c == null || !f.type.startsWith('chat.')) {
      return;
    }
    switch (f.type) {
      case 'chat.message':
        final m = ChatMessage.fromJson(
          (f.data['message'] as Map).cast<String, dynamic>(),
        );
        if (m.channelId != c.id || _messages.any((x) => x.id == m.id)) {
          return;
        }
        setState(() {
          _messages.add(m);
          _typing = null;
        });
        _markRead();
      case 'chat.message_removed':
        if (f.data['channel_id'] != c.id) {
          return;
        }
        setState(() {
          final i = _messages.indexWhere((x) => x.id == f.data['id']);
          if (i >= 0) {
            _messages[i] = _messages[i].asRemoved();
          }
        });
      case 'chat.typing':
        if (f.data['channel_id'] != c.id) {
          return;
        }
        final who = c.members
            .where((m) => m.userId == f.data['user_id'])
            .firstOrNull;
        setState(() => _typing = who?.displayName);
        _typingTimer?.cancel();
        _typingTimer = Timer(const Duration(seconds: 4), () {
          if (mounted) {
            setState(() => _typing = null);
          }
        });
    }
  }

  Future<void> _send({String? body, String? quick}) async {
    final c = _channel;
    if (c == null) {
      return;
    }
    final l10n = context.l10n;
    final messenger = ScaffoldMessenger.of(context);
    try {
      await _rt.send('chat.send', {
        'channel_id': c.id,
        'body': ?body,
        'quick_id': ?quick,
      });
      if (body != null) {
        _text.clear();
      }
    } on Object catch (e) {
      messenger.showSnackBar(SnackBar(content: Text(playErrorText(l10n, e))));
    }
  }

  void _onTyping(String _) {
    final c = _channel;
    final now = DateTime.now();
    if (c == null || now.difference(_lastTypingSent).inSeconds < 3) {
      return;
    }
    _lastTypingSent = now;
    unawaited(
      _rt
          .send('chat.typing', {'channel_id': c.id})
          .then((_) {}, onError: (_) {}),
    );
  }

  Future<void> _delete(ChatMessage m) async {
    final l10n = context.l10n;
    final messenger = ScaffoldMessenger.of(context);
    try {
      await ref.read(chatRepositoryProvider).remove(m.id);
      setState(() {
        final i = _messages.indexWhere((x) => x.id == m.id);
        if (i >= 0) {
          _messages[i] = _messages[i].asRemoved();
        }
      });
    } on Object catch (e) {
      messenger.showSnackBar(SnackBar(content: Text(playErrorText(l10n, e))));
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    final me = switch (ref.watch(sessionControllerProvider).value) {
      SignedIn(:final session) => session.user.id,
      _ => '',
    };
    return Scaffold(
      appBar: AppBar(title: Text(widget.title ?? l10n.chatTitle)),
      body: _channel == null
          ? Center(
              child: _error == null
                  ? const CircularProgressIndicator()
                  : Text(playErrorText(l10n, _error!)),
            )
          : Column(
              children: [
                Expanded(
                  child: _messages.isEmpty
                      ? EmptyState(
                          icon: Icons.chat_bubble_outline,
                          text: l10n.chatEmpty,
                        )
                      : ListView.builder(
                          key: const Key('chat-list'),
                          reverse: true,
                          padding: const EdgeInsets.all(12),
                          itemCount: _messages.length,
                          itemBuilder: (context, i) {
                            final m = _messages[_messages.length - 1 - i];
                            final mine = m.sender.userId == me;
                            return _Bubble(
                              message: m,
                              mine: mine,
                              onDelete: mine && !m.removed
                                  ? () => _delete(m)
                                  : null,
                            );
                          },
                        ),
                ),
                if (_typing case final name?)
                  Padding(
                    padding: const EdgeInsets.symmetric(horizontal: 16),
                    child: Align(
                      alignment: Alignment.centerLeft,
                      child: Text(
                        l10n.chatTyping(name),
                        key: const Key('chat-typing'),
                        style: Theme.of(context).textTheme.bodySmall,
                      ),
                    ),
                  ),
                SizedBox(
                  height: 48,
                  child: ListView(
                    scrollDirection: Axis.horizontal,
                    padding: const EdgeInsets.symmetric(horizontal: 12),
                    children: [
                      for (final q in quickPhrases)
                        Padding(
                          padding: const EdgeInsets.only(right: 8),
                          child: ActionChip(
                            key: Key('quick-$q'),
                            label: Text(quickPhraseText(l10n, q)),
                            onPressed: () => _send(quick: q),
                          ),
                        ),
                    ],
                  ),
                ),
                SafeArea(
                  top: false,
                  child: Padding(
                    padding: const EdgeInsets.fromLTRB(12, 4, 12, 12),
                    child: Row(
                      children: [
                        Expanded(
                          child: TextField(
                            key: const Key('chat-input'),
                            controller: _text,
                            maxLength: 300,
                            minLines: 1,
                            maxLines: 4,
                            onChanged: _onTyping,
                            decoration: InputDecoration(
                              hintText: l10n.chatHint,
                              counterText: '',
                            ),
                          ),
                        ),
                        const SizedBox(width: 8),
                        IconButton.filled(
                          key: const Key('chat-send'),
                          tooltip: l10n.chatSend,
                          icon: const Icon(Icons.send),
                          onPressed: () {
                            final body = _text.text.trim();
                            if (body.isNotEmpty) {
                              unawaited(_send(body: body));
                            }
                          },
                        ),
                      ],
                    ),
                  ),
                ),
              ],
            ),
    );
  }
}

class _Bubble extends StatelessWidget {
  const _Bubble({required this.message, required this.mine, this.onDelete});

  final ChatMessage message;
  final bool mine;
  final VoidCallback? onDelete;

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    final bv = BvColors.of(context);
    final m = message;
    final text = m.removed
        ? l10n.chatRemoved
        : m.quickId != null
        ? quickPhraseText(l10n, m.quickId!)
        : m.body ?? '';
    final bubble = Container(
      key: Key('chat-${m.id}'),
      constraints: const BoxConstraints(maxWidth: 280),
      padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 10),
      decoration: BoxDecoration(
        gradient: mine && !m.removed ? bv.gradient : null,
        color: mine && !m.removed ? null : bv.card,
        border: mine ? null : Border.all(color: bv.cardBorder),
        borderRadius: BorderRadius.circular(18),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          if (!mine)
            Text(
              m.sender.displayName,
              style: Theme.of(context).textTheme.labelMedium
                  ?.copyWith(color: bv.badgeInk),
            ),
          Text(
            text,
            style: TextStyle(
              color: mine && !m.removed ? Colors.white : bv.ink,
              fontStyle: m.removed ? FontStyle.italic : null,
              fontSize: m.quickId != null ? 18 : null,
            ),
          ),
        ],
      ),
    );
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 4),
      child: Row(
        mainAxisAlignment: mine
            ? MainAxisAlignment.end
            : MainAxisAlignment.start,
        crossAxisAlignment: CrossAxisAlignment.end,
        children: [
          if (!mine) ...[
            AvatarBubble(avatarFor(m.sender.avatarId), size: 32, ring: false),
            const SizedBox(width: 6),
          ],
          GestureDetector(
            onLongPress: onDelete == null
                ? null
                : () async {
                    final ok = await showDialog<bool>(
                      context: context,
                      builder: (context) => AlertDialog(
                        content: Text(l10n.chatDelete),
                        actions: [
                          TextButton(
                            onPressed: () => Navigator.pop(context, false),
                            child: Text(l10n.cancel),
                          ),
                          FilledButton(
                            key: const Key('chat-delete-confirm'),
                            onPressed: () => Navigator.pop(context, true),
                            child: Text(l10n.chatDelete),
                          ),
                        ],
                      ),
                    );
                    if (ok ?? false) {
                      onDelete!();
                    }
                  },
            child: bubble,
          ),
        ],
      ),
    );
  }
}

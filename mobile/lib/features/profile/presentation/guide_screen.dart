import 'package:flutter/material.dart';

import '../../../app/theme.dart';
import '../../../core/l10n/l10n.dart';
import '../../../core/widgets/bv.dart';

/// The game guide (D-102): modes, scoring, leagues, XP and fair play, one
/// collapsible card per topic. Texts live in the ARB files; a body is
/// paragraphs separated by new lines, and lines starting with "• " are
/// shown as bullets.
class GuideScreen extends StatelessWidget {
  const GuideScreen({super.key});

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    final sections = [
      (
        'basics',
        Icons.touch_app_rounded,
        l10n.guideBasicsTitle,
        l10n.guideBasicsBody,
      ),
      (
        'scoring',
        Icons.calculate_rounded,
        l10n.guideScoringTitle,
        l10n.guideScoringBody,
      ),
      (
        'types',
        Icons.sports_esports_rounded,
        l10n.guideTypesTitle,
        l10n.guideTypesBody,
      ),
      (
        'category',
        Icons.category_rounded,
        l10n.guideCategoryTitle,
        l10n.guideCategoryBody,
      ),
      (
        'teams',
        Icons.groups_rounded,
        l10n.guideTeamsTitle,
        l10n.guideTeamsBody,
      ),
      ('daily', Icons.today_rounded, l10n.guideDailyTitle, l10n.guideDailyBody),
      (
        'rooms',
        Icons.meeting_room_rounded,
        l10n.guideRoomsTitle,
        l10n.guideRoomsBody,
      ),
      (
        'tournaments',
        Icons.emoji_events_rounded,
        l10n.guideTournamentsTitle,
        l10n.guideTournamentsBody,
      ),
      ('lab', Icons.edit_note_rounded, l10n.guideLabTitle, l10n.guideLabBody),
      (
        'leagues',
        Icons.military_tech_rounded,
        l10n.guideLeaguesTitle,
        l10n.guideLeaguesBody,
      ),
      ('xp', Icons.bolt_rounded, l10n.guideXpTitle, l10n.guideXpBody),
      (
        'boards',
        Icons.leaderboard_rounded,
        l10n.guideBoardsTitle,
        l10n.guideBoardsBody,
      ),
      (
        'fair',
        Icons.verified_user_rounded,
        l10n.guideFairTitle,
        l10n.guideFairBody,
      ),
    ];
    return Scaffold(
      appBar: AppBar(title: Text(l10n.guideTitle)),
      body: ListView(
        padding: const EdgeInsets.fromLTRB(16, 0, 16, 24),
        children: [
          BvCard(
            gradient: BvColors.of(context).heroGradient,
            padding: const EdgeInsets.all(18),
            child: Row(
              children: [
                const IconBadge(
                  Icons.menu_book_rounded,
                  size: 56,
                  onGradient: true,
                ),
                const SizedBox(width: 14),
                Expanded(
                  child: Text(
                    l10n.guideIntro,
                    key: const Key('guide-intro'),
                    style: Theme.of(context).textTheme.bodyLarge
                        ?.copyWith(color: Colors.white),
                  ),
                ),
              ],
            ),
          ),
          const SizedBox(height: 12),
          for (final (id, icon, title, body) in sections)
            Padding(
              padding: const EdgeInsets.only(bottom: 10),
              child: _Section(id: id, icon: icon, title: title, body: body),
            ),
        ],
      ),
    );
  }
}

class _Section extends StatelessWidget {
  const _Section({
    required this.id,
    required this.icon,
    required this.title,
    required this.body,
  });

  final String id;
  final IconData icon;
  final String title;
  final String body;

  @override
  Widget build(BuildContext context) {
    final bv = BvColors.of(context);
    final text = Theme.of(context).textTheme;
    return BvCard(
      radius: 20,
      padding: EdgeInsets.zero,
      child: Theme(
        // No divider lines around the expanded body.
        data: Theme.of(context).copyWith(dividerColor: Colors.transparent),
        child: ExpansionTile(
          key: Key('guide-$id'),
          tilePadding: const EdgeInsets.symmetric(horizontal: 14, vertical: 4),
          childrenPadding: const EdgeInsets.fromLTRB(16, 0, 16, 16),
          leading: IconBadge(icon, size: 42),
          title: Text(title, style: text.titleMedium),
          iconColor: bv.accent,
          collapsedIconColor: bv.muted,
          expandedCrossAxisAlignment: CrossAxisAlignment.start,
          children: [
            for (final line in body.split('\n'))
              if (line.startsWith('• '))
                Padding(
                  padding: const EdgeInsets.only(top: 6),
                  child: Row(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Padding(
                        padding: const EdgeInsets.only(top: 7, right: 10),
                        child: Container(
                          width: 6,
                          height: 6,
                          decoration: BoxDecoration(
                            shape: BoxShape.circle,
                            gradient: bv.gradient,
                          ),
                        ),
                      ),
                      Expanded(
                        child: Text(line.substring(2), style: text.bodyMedium),
                      ),
                    ],
                  ),
                )
              else
                Padding(
                  padding: const EdgeInsets.only(top: 6),
                  child: Text(line, style: text.bodyMedium),
                ),
          ],
        ),
      ),
    );
  }
}

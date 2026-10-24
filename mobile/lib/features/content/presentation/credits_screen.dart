import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

import '../../../app/theme.dart';
import '../../../core/l10n/l10n.dart';
import '../../../core/network/api_exception.dart';
import '../../../core/widgets/bv.dart';
import '../../../core/widgets/dialogs.dart';
import '../data/content_providers.dart';
import '../domain/credit.dart';

part 'credits_screen.g.dart';

/// Sources of the questions (D-006).
@riverpod
Future<List<Credit>> credits(Ref ref) =>
    ref.watch(contentRepositoryProvider).credits();

/// Credits and licenses: question sources with their attribution, required
/// by CC BY-SA for OpenTDB questions (schema-content §6), and the licenses
/// of the open-source packages the app uses.
class CreditsScreen extends ConsumerWidget {
  const CreditsScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = context.l10n;
    final credits = ref.watch(creditsProvider);
    return Scaffold(
      appBar: AppBar(title: Text(l10n.creditsTitle)),
      body: ListView(
        padding: const EdgeInsets.fromLTRB(16, 0, 16, 24),
        children: [
          SectionTitle(l10n.creditsQuestions),
          ...switch (credits) {
            AsyncValue(value: final list?) => [
              for (final c in list) _CreditTile(credit: c),
            ],
            AsyncValue(:final error?) => [
              ErrorView(
                message: error is ApiException
                    ? error.localizedMessage(l10n)
                    : l10n.errorUnknown,
                retryLabel: l10n.retry,
                onRetry: () => ref.invalidate(creditsProvider),
              ),
            ],
            _ => [
              const Padding(padding: EdgeInsets.all(16), child: LoadingView()),
            ],
          },
          const SizedBox(height: 8),
          GlassSection(
            children: [
              ListTile(
                key: const Key('open-source-licenses'),
                leading: const IconBadge(Icons.code_rounded, size: 36),
                title: Text(l10n.creditsOpenSource),
                trailing: const Icon(Icons.chevron_right_rounded),
                onTap: () => showLicensePage(
                  context: context,
                  applicationName: l10n.appTitle,
                ),
              ),
            ],
          ),
        ],
      ),
    );
  }
}

class _CreditTile extends StatelessWidget {
  const _CreditTile({required this.credit});

  final Credit credit;

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    final c = credit;
    final links = [?c.url, ?c.licenseUrl];
    return TileCard(
      child: ListTile(
        isThreeLine: true,
        leading: const IconBadge(Icons.menu_book_rounded, size: 40),
        title: Text(c.name),
        subtitle: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(c.attributionText),
            const SizedBox(height: 4),
            Text(l10n.creditsLicense(c.licenseCode)),
            if (c.shareAlike) Text(l10n.creditsShareAlike),
            for (final link in links)
              InkWell(
                onTap: () async {
                  await Clipboard.setData(ClipboardData(text: link));
                  if (context.mounted) {
                    showSnack(context, l10n.creditsLinkCopied);
                  }
                },
                child: Text(
                  link,
                  style: TextStyle(color: BvColors.of(context).accent),
                ),
              ),
          ],
        ),
      ),
    );
  }
}

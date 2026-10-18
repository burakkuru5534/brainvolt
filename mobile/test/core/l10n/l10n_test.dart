import 'dart:convert';
import 'dart:io';

import 'package:brainvolt/core/l10n/generated/app_localizations_en.dart';
import 'package:brainvolt/core/l10n/l10n.dart';
import 'package:brainvolt/core/network/error_messages.g.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('resolveLocale: override, then device, then Turkish', () {
    expect(
      resolveLocale(const Locale('en'), const [Locale('tr')]),
      const Locale('en'),
    );
    expect(
      resolveLocale(null, const [Locale('de'), Locale('en', 'GB')]),
      const Locale('en'),
    );
    expect(resolveLocale(null, const [Locale('tr', 'TR')]), const Locale('tr'));
    expect(resolveLocale(null, const [Locale('de')]), const Locale('tr'));
    expect(resolveLocale(null, null), const Locale('tr'));
  });

  // Drift check: every code of docs/api/errors.md has TR and EN texts and an
  // entry in the generated switch (run `make gen` after adding codes).
  test('every server error code has app texts', () {
    final doc = File('../docs/api/errors.md').readAsStringSync();
    final codes =
        RegExp(r'^\| (`[A-Z_]+`(?: / `[A-Z_]+`)*) \|', multiLine: true)
            .allMatches(doc)
            .expand(
              (m) =>
                  RegExp('`([A-Z_]+)`')
                      .allMatches(m.group(1)!)
                      .map((c) => c.group(1)!),
            )
            .toSet();
    expect(codes.length, greaterThan(60));
    expect(serverErrorCodes, codes, reason: 'run `make gen`');

    for (final name in ['app_tr.arb', 'app_en.arb']) {
      final arb = jsonDecode(
        File('lib/core/l10n/arb/$name').readAsStringSync(),
      ) as Map<String, dynamic>;
      for (final c in codes) {
        expect(
          arb['error_$c'],
          isA<String>().having((s) => s.isNotEmpty, 'non-empty', isTrue),
          reason: '$name: $c',
        );
      }
    }
    final en = AppLocalizationsEn();
    for (final c in codes) {
      expect(serverErrorMessage(en, c), isNotNull, reason: c);
    }
  });
}

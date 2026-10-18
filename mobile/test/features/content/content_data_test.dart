import 'package:brainvolt/core/network/api_client.dart';
import 'package:brainvolt/core/network/token_source.dart';
import 'package:brainvolt/features/content/data/api_content_repository.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../core/network/fake_adapter.dart';

void main() {
  test('GET /v1/credits', () async {
    final adapter = FakeAdapter(
      (_) => const FakeResponse(200, {
        'sources': [
          {
            'name': 'Open Trivia Database',
            'url': 'https://opentdb.com',
            'license_code': 'CC-BY-SA-4.0',
            'license_url': 'https://creativecommons.org/licenses/by-sa/4.0/',
            'attribution_text': 'Open Trivia Database, CC BY-SA 4.0',
            'share_alike': true,
          },
        ],
      }),
    );
    final repo = ApiContentRepository(
      ApiClient(
        buildDio(
          baseUrl: Uri.parse('http://api.test'),
          appVersion: '1.0.0+1',
          platform: 'android',
          locale: () => 'tr',
          tokens: const SignedOutTokenSource(),
          adapter: adapter,
        ),
      ),
    );
    final credits = await repo.credits();
    expect(adapter.requests.single.path, '/v1/credits');
    expect(credits.single.licenseCode, 'CC-BY-SA-4.0');
    expect(credits.single.shareAlike, isTrue);
    expect(credits.single.licenseUrl, contains('by-sa'));
  });
}

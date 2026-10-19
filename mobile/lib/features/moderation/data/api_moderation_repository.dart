import 'package:uuid/uuid.dart';

import '../../../core/network/api_client.dart';
import '../domain/moderation_repository.dart';

/// [ModerationRepository] over the REST API.
class ApiModerationRepository implements ModerationRepository {
  const ApiModerationRepository(this.api);

  final ApiClient api;

  @override
  Future<void> reportQuestion({
    required String questionId,
    required String reason,
    String? matchId,
    String? comment,
  }) => api.postJson(
    '/v1/reports',
    body: {
      'target_kind': 'question',
      'target_id': questionId,
      'reason': reason,
      'match_id': ?matchId,
      if (comment != null && comment.isNotEmpty) 'comment': comment,
    },
    idempotencyKey: const Uuid().v7(),
  );

  @override
  Future<List<Sanction>> mySanctions() async {
    final j = await api.getJson('/v1/me/sanctions');
    return [
      for (final s in (j['sanctions'] as List? ?? const []))
        Sanction.fromJson((s as Map).cast<String, dynamic>()),
    ];
  }

  @override
  Future<void> appeal(String sanctionId, String text) => api.postJson(
    '/v1/me/sanctions/$sanctionId/appeal',
    body: {'text': text},
    idempotencyKey: const Uuid().v7(),
  );
}

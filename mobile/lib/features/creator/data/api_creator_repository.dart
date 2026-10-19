import 'package:uuid/uuid.dart';

import '../../../core/network/api_client.dart';
import '../domain/creator_repository.dart';

class ApiCreatorRepository implements CreatorRepository {
  const ApiCreatorRepository(this.api);

  final ApiClient api;

  List<Quiz> _list(Map<String, dynamic> j) => [
    for (final q in (j['quizzes'] as List? ?? const []))
      Quiz.fromJson((q as Map).cast<String, dynamic>()),
  ];

  @override
  Future<List<Quiz>> mine() async => _list(await api.getJson('/v1/me/quizzes'));

  @override
  Future<List<Quiz>> discover({String? locale}) async => _list(
    await api.getJson(
      Uri(
        path: '/v1/creator/quizzes',
        queryParameters: {'locale': ?locale, 'limit': '50'},
      ).toString(),
    ),
  );

  @override
  Future<Quiz> quiz(String id) async =>
      Quiz.fromJson(await api.getJson('/v1/creator/quizzes/$id'));

  @override
  Future<Quiz> create(QuizDraft draft) async => Quiz.fromJson(
    (await api.postJson(
      '/v1/creator/quizzes',
      body: draft.toJson(),
      idempotencyKey: const Uuid().v7(),
    ))!,
  );

  @override
  Future<Quiz> update(String id, int version, QuizDraft draft) async =>
      Quiz.fromJson(
        await api.patchJson(
          '/v1/creator/quizzes/$id',
          body: draft.toJson(),
          ifMatch: '$version',
        ),
      );

  @override
  Future<void> delete(String id) => api.delete('/v1/creator/quizzes/$id');

  @override
  Future<Quiz> submit(String id) async =>
      Quiz.fromJson((await api.postJson('/v1/creator/quizzes/$id/submit'))!);

  @override
  Future<void> like(String id, {required bool like}) => like
      ? api.postJson('/v1/creator/quizzes/$id/like')
      : api.delete('/v1/creator/quizzes/$id/like');
}

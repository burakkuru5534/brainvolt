import 'package:riverpod_annotation/riverpod_annotation.dart';

import '../../../core/network/api_client.dart';
import '../domain/progress_repository.dart';
import 'api_progress_repository.dart';

part 'progress_providers.g.dart';

/// The ranking and progression endpoints. Tests override it with a fake.
@Riverpod(keepAlive: true)
ProgressRepository progressRepository(Ref ref) =>
    ApiProgressRepository(ref.watch(apiClientProvider));

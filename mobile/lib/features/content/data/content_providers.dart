import 'package:riverpod_annotation/riverpod_annotation.dart';

import '../../../core/network/api_client.dart';
import '../domain/content_repository.dart';
import 'api_content_repository.dart';

part 'content_providers.g.dart';

/// The content endpoints. Tests override it with a fake.
@Riverpod(keepAlive: true)
ContentRepository contentRepository(Ref ref) =>
    ApiContentRepository(ref.watch(apiClientProvider));

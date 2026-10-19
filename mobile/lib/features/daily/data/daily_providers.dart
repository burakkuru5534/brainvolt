import 'package:riverpod_annotation/riverpod_annotation.dart';

import '../../../core/network/api_client.dart';
import '../domain/daily_repository.dart';
import 'api_daily_repository.dart';

part 'daily_providers.g.dart';

/// The Daily Challenge endpoints. Tests override it with a fake.
@Riverpod(keepAlive: true)
DailyRepository dailyRepository(Ref ref) =>
    ApiDailyRepository(ref.watch(apiClientProvider));

import 'credit.dart';

/// The content endpoints of the backend (rest.md §2, `content` module).
abstract interface class ContentRepository {
  /// `GET /v1/credits`: every question source with its attribution.
  Future<List<Credit>> credits();
}

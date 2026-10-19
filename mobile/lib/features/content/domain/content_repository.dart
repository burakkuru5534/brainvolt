import 'credit.dart';

/// A top-level category (Category Master, Mixed Battle picks).
class CategoryInfo {
  const CategoryInfo(this.id, this.name);
  final String id;
  final String name;
}

/// The content endpoints of the backend (rest.md §2, `content` module).
abstract interface class ContentRepository {
  /// `GET /v1/credits`: every question source with its attribution.
  Future<List<Credit>> credits();

  /// `GET /v1/categories`: the active categories in the app's language.
  Future<List<CategoryInfo>> categories();
}

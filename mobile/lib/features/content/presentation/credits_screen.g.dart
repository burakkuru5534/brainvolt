// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'credits_screen.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning
/// Sources of the questions (D-006).

@ProviderFor(credits)
final creditsProvider = CreditsProvider._();

/// Sources of the questions (D-006).

final class CreditsProvider
    extends
        $FunctionalProvider<
          AsyncValue<List<Credit>>,
          List<Credit>,
          FutureOr<List<Credit>>
        >
    with $FutureModifier<List<Credit>>, $FutureProvider<List<Credit>> {
  /// Sources of the questions (D-006).
  CreditsProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'creditsProvider',
        isAutoDispose: true,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$creditsHash();

  @$internal
  @override
  $FutureProviderElement<List<Credit>> $createElement(
    $ProviderPointer pointer,
  ) => $FutureProviderElement(pointer);

  @override
  FutureOr<List<Credit>> create(Ref ref) {
    return credits(ref);
  }
}

String _$creditsHash() => r'df473caebeebe01a1310458695b6c7d6e3ae1191';

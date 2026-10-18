// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'result_screen.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning
/// GET /v1/matches/{id} for the result screen.

@ProviderFor(matchResult)
final matchResultProvider = MatchResultFamily._();

/// GET /v1/matches/{id} for the result screen.

final class MatchResultProvider
    extends
        $FunctionalProvider<
          AsyncValue<MatchResult>,
          MatchResult,
          FutureOr<MatchResult>
        >
    with $FutureModifier<MatchResult>, $FutureProvider<MatchResult> {
  /// GET /v1/matches/{id} for the result screen.
  MatchResultProvider._({
    required MatchResultFamily super.from,
    required String super.argument,
  }) : super(
         retry: null,
         name: r'matchResultProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$matchResultHash();

  @override
  String toString() {
    return r'matchResultProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  $FutureProviderElement<MatchResult> $createElement(
    $ProviderPointer pointer,
  ) => $FutureProviderElement(pointer);

  @override
  FutureOr<MatchResult> create(Ref ref) {
    final argument = this.argument as String;
    return matchResult(ref, argument);
  }

  @override
  bool operator ==(Object other) {
    return other is MatchResultProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$matchResultHash() => r'7266a0e50ec512d10ae74b8c975011d99d32d377';

/// GET /v1/matches/{id} for the result screen.

final class MatchResultFamily extends $Family
    with $FunctionalFamilyOverride<FutureOr<MatchResult>, String> {
  MatchResultFamily._()
    : super(
        retry: null,
        name: r'matchResultProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  /// GET /v1/matches/{id} for the result screen.

  MatchResultProvider call(String matchId) =>
      MatchResultProvider._(argument: matchId, from: this);

  @override
  String toString() => r'matchResultProvider';
}

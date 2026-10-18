import '../../../core/l10n/l10n.dart';
import '../../../core/network/api_exception.dart';
import '../../../core/network/error_messages.g.dart';
import '../../../core/realtime/realtime_client.dart';

/// Text for a failed play action (REST or WebSocket intent).
String playErrorText(AppLocalizations l, Object error) => switch (error) {
  ApiException() => error.localizedMessage(l),
  RealtimeError(code: RealtimeError.disconnected) => l.errorNetwork,
  RealtimeError(:final code, :final message) =>
    serverErrorMessage(l, code) ?? (message.isEmpty ? l.errorUnknown : message),
  _ => l.errorUnknown,
};

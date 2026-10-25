import '../../../core/widgets/cosmetics.dart';

/// The preset avatars (D-021, D-059) and the emoji the app draws for them
/// until the artwork arrives.
const avatarEmoji = <String, String>{
  'fox': '🦊',
  'owl': '🦉',
  'cat': '🐱',
  'panda': '🐼',
  'lion': '🦁',
  'penguin': '🐧',
  'rabbit': '🐰',
  'bear': '🐻',
  'koala': '🐨',
  'tiger': '🐯',
  'wolf': '🐺',
  'dolphin': '🐬',
};

/// Emoji of [avatarId]; unknown ids (newer server) get a neutral face.
String avatarFor(String avatarId) =>
    avatarEmoji[avatarId] ?? specialAvatarEmoji[avatarId] ?? '🙂';

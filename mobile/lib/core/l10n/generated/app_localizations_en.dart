// ignore: unused_import
import 'package:intl/intl.dart' as intl;

import 'app_localizations.dart';

// ignore_for_file: type=lint

/// The translations for English (`en`).
class AppLocalizationsEn extends AppLocalizations {
  AppLocalizationsEn([String locale = 'en']) : super(locale);

  @override
  String get appTitle => 'BrainVolt';

  @override
  String get homeWelcome => 'Welcome to the quiz!';

  @override
  String get settingsTitle => 'Settings';

  @override
  String get settingsLanguage => 'Language';

  @override
  String get languageSystem => 'Device language';

  @override
  String get languageTurkish => 'Türkçe';

  @override
  String get languageEnglish => 'English';

  @override
  String get retry => 'Try again';

  @override
  String get cancel => 'Cancel';

  @override
  String get continueButton => 'Continue';

  @override
  String get save => 'Save';

  @override
  String get splashError => 'We couldn\'t reach the server.';

  @override
  String get ageGateTitle => 'Welcome!';

  @override
  String get ageGateBody =>
      'You must be at least 13 years old to use BrainVolt.';

  @override
  String get ageGateConfirm => 'I am 13 or older';

  @override
  String homeGreeting(String name) {
    return 'Hi, $name!';
  }

  @override
  String get homeGuestBanner =>
      'You\'re playing as a guest. Link an account to keep your progress.';

  @override
  String get homeVerifyBanner =>
      'Verify your email address; guest limits apply until you do.';

  @override
  String get linkAccountAction => 'Link account';

  @override
  String get verifyAction => 'Verify';

  @override
  String get accountTitle => 'Account';

  @override
  String get accountGuestInfo =>
      'Guest account. Link a sign-in method to keep your progress and play on other devices.';

  @override
  String accountSignedInAs(String email) {
    return 'Signed in as $email';
  }

  @override
  String get accountSignedIn => 'Your account is linked.';

  @override
  String get accountLinked => 'Your account is linked. Your progress is safe.';

  @override
  String get signInWithGoogle => 'Continue with Google';

  @override
  String get signInWithApple => 'Continue with Apple';

  @override
  String get signUpWithEmail => 'Sign up with email';

  @override
  String get signInWithEmail => 'I have an account: sign in with email';

  @override
  String get emailSignInTitle => 'Sign in with email';

  @override
  String get emailLabel => 'Email';

  @override
  String get passwordLabel => 'Password';

  @override
  String get passwordHint => 'At least 6 characters';

  @override
  String get signUpButton => 'Sign up';

  @override
  String get signInButton => 'Sign in';

  @override
  String get signOut => 'Sign out';

  @override
  String get signOutConfirm =>
      'After signing out you\'ll continue on this device as a new guest.';

  @override
  String get deleteAccount => 'Delete my account';

  @override
  String get deleteAccountConfirm =>
      'Your account will be permanently deleted after 14 days. You can sign in and cancel within that time.';

  @override
  String get linkConflictTitle => 'Account already registered';

  @override
  String get linkConflictBody =>
      'This account is already registered. Your guest progress will be lost. Continue?';

  @override
  String get verifyEmailTitle => 'Verify your email';

  @override
  String verifyEmailBody(String email) {
    return 'We sent a verification link to $email. Tap below after opening it.';
  }

  @override
  String get verifyEmailDone => 'I\'ve verified';

  @override
  String get verifyEmailResend => 'Send again';

  @override
  String get verifyEmailSent => 'Verification email sent.';

  @override
  String get verifyEmailNotYet => 'Your email isn\'t verified yet.';

  @override
  String get verifyEmailVerified => 'Your email is verified.';

  @override
  String get authErrorInvalidCredential => 'Wrong email or password.';

  @override
  String get authErrorWeakPassword =>
      'The password must be at least 6 characters.';

  @override
  String get authErrorInvalidEmail => 'Enter a valid email address.';

  @override
  String get authErrorUserNotFound => 'There\'s no account with this email.';

  @override
  String get authErrorTooManyRequests => 'Too many attempts. Try again later.';

  @override
  String get authErrorUnsupported =>
      'This sign-in method isn\'t available on this device.';

  @override
  String get accountSuspendedTitle => 'Your account is suspended';

  @override
  String get accountSuspendedBody => 'Contact support for details.';

  @override
  String get deletionPendingTitle => 'Your account will be deleted';

  @override
  String deletionPendingBody(DateTime date) {
    final intl.DateFormat dateDateFormat = intl.DateFormat.yMMMMd(localeName);
    final String dateString = dateDateFormat.format(date);

    return 'Your account will be permanently deleted on $dateString.';
  }

  @override
  String get cancelDeletion => 'Cancel deletion';

  @override
  String get profileTitle => 'Profile';

  @override
  String get displayNameLabel => 'Display name';

  @override
  String get displayNameHint => '2–24 characters';

  @override
  String get usernameLabel => 'Username';

  @override
  String get usernameHint =>
      '3–20 characters: lowercase letters, digits, _ and .';

  @override
  String get usernameGuestHint =>
      'Link and verify an account to change your username.';

  @override
  String usernameAvailableAt(DateTime date) {
    final intl.DateFormat dateDateFormat = intl.DateFormat.yMMMMd(localeName);
    final String dateString = dateDateFormat.format(date);

    return 'You can change your username after $dateString.';
  }

  @override
  String get avatarLabel => 'Avatar';

  @override
  String get profileSaved => 'Profile saved.';

  @override
  String get settingsGame => 'Game';

  @override
  String get settingsSound => 'Sound effects';

  @override
  String get settingsMusic => 'Music';

  @override
  String get settingsHaptics => 'Haptics';

  @override
  String get settingsPrivacy => 'Privacy';

  @override
  String get settingsShowOnline => 'Show my online status';

  @override
  String get settingsAllowFriendRequests => 'Allow friend requests';

  @override
  String get settingsChallengesFrom => 'Challenges from';

  @override
  String get challengesEveryone => 'Everyone';

  @override
  String get challengesFriends => 'Friends only';

  @override
  String get challengesNobody => 'Nobody';

  @override
  String get creditsTitle => 'Credits and Licenses';

  @override
  String get creditsQuestions => 'Question sources';

  @override
  String creditsLicense(String license) {
    return 'License: $license';
  }

  @override
  String get creditsShareAlike =>
      'Questions from this source and their translations are shared under the same license.';

  @override
  String get creditsLinkCopied => 'Link copied.';

  @override
  String get creditsOpenSource => 'Open-source licenses';

  @override
  String get errorNetwork =>
      'Couldn\'t connect. Check your internet connection.';

  @override
  String get errorTimeout =>
      'The server didn\'t answer in time. Please try again.';

  @override
  String get errorUnknown => 'Something went wrong. Please try again.';

  @override
  String get error_ACCOUNT_DELETION_PENDING =>
      'Your account is being deleted. You can only cancel the deletion.';

  @override
  String get error_ACCOUNT_SUSPENDED => 'Your account is suspended.';

  @override
  String get error_ACCOUNT_UPGRADE_REQUIRED =>
      'Upgrade your account to do this.';

  @override
  String get error_AD_REWARD_LIMIT =>
      'You\'ve reached today\'s ad reward limit.';

  @override
  String get error_AGE_CONFIRMATION_REQUIRED =>
      'Please confirm your age to continue.';

  @override
  String get error_ALREADY_FRIENDS => 'You\'re already friends.';

  @override
  String get error_ALREADY_IN_MATCH => 'You already have a match in progress.';

  @override
  String get error_ALREADY_QUEUED =>
      'You\'re already in the matchmaking queue.';

  @override
  String get error_ALREADY_REPORTED => 'You\'ve already reported this.';

  @override
  String get error_ANSWER_DUPLICATE =>
      'You\'ve already answered this question.';

  @override
  String get error_ANSWER_INVALID_OPTION => 'Invalid answer option.';

  @override
  String get error_ANSWER_LATE => 'Time\'s up for this answer.';

  @override
  String get error_APPEAL_EXISTS => 'You\'ve already appealed this sanction.';

  @override
  String get error_APP_VERSION_UNSUPPORTED =>
      'Please update the app to continue.';

  @override
  String get error_BLOCKED => 'You can\'t interact with this user.';

  @override
  String get error_CHALLENGES_DISABLED =>
      'This user isn\'t accepting challenges right now.';

  @override
  String get error_CHALLENGE_EXPIRED => 'The challenge has expired.';

  @override
  String get error_CHANGE_TOO_SOON =>
      'You can\'t change this yet. Please try again later.';

  @override
  String get error_COUPLE_COOLDOWN =>
      'You need to wait before forming a new couple.';

  @override
  String get error_COUPLE_EXISTS => 'You already have a couple.';

  @override
  String get error_DAILY_ATTEMPT_IN_PROGRESS =>
      'You have a daily attempt in progress.';

  @override
  String get error_DAILY_ATTEMPT_USED =>
      'You\'ve already used today\'s official attempt.';

  @override
  String get error_DAILY_NOT_AVAILABLE =>
      'Today\'s challenge isn\'t ready yet.';

  @override
  String get error_FORBIDDEN => 'You are not allowed to do this.';

  @override
  String get error_FRIEND_LIMIT_REACHED => 'You\'ve reached the friend limit.';

  @override
  String get error_IDEMPOTENCY_KEY_REUSED =>
      'This request key was already used for a different request.';

  @override
  String get error_IDEMPOTENT_REQUEST_IN_PROGRESS =>
      'This request is still being processed. Try again in a moment.';

  @override
  String get error_INTERNAL => 'Something went wrong. Please try again.';

  @override
  String get error_INVITE_CODE_EXPIRED => 'The invite code has expired.';

  @override
  String get error_INVITE_CODE_INVALID => 'The invite code is invalid.';

  @override
  String get error_MALFORMED_REQUEST => 'The request could not be read.';

  @override
  String get error_MATCH_NOT_FOUND => 'Match not found.';

  @override
  String get error_MATCH_STATE_INVALID =>
      'This can\'t be done at the current stage of the match.';

  @override
  String get error_METHOD_NOT_ALLOWED =>
      'This action is not supported at this address.';

  @override
  String get error_MODE_NOT_ALLOWED =>
      'This combination of game modes isn\'t supported.';

  @override
  String get error_NOT_A_PARTICIPANT => 'You\'re not a player in this match.';

  @override
  String get error_NOT_CAPTAIN => 'Only the team captain can do this.';

  @override
  String get error_NOT_ENOUGH_QUESTIONS =>
      'There aren\'t enough questions for these settings.';

  @override
  String get error_NOT_FOUND => 'We couldn\'t find what you were looking for.';

  @override
  String get error_NOT_YOUR_TURN => 'It\'s not your turn.';

  @override
  String get error_NO_ACTIVE_SEASON => 'There is no active season right now.';

  @override
  String get error_PAYLOAD_TOO_LARGE => 'The request is too large.';

  @override
  String get error_PICK_NOT_ALLOWED => 'This category can\'t be picked.';

  @override
  String get error_PURCHASE_ALREADY_PROCESSED =>
      'This purchase has already been processed.';

  @override
  String get error_PURCHASE_INVALID => 'The purchase could not be verified.';

  @override
  String get error_QUESTION_DUPLICATE => 'This question already exists.';

  @override
  String get error_QUESTION_IN_USE =>
      'This question has been used in matches and can\'t be changed this way.';

  @override
  String get error_QUESTION_SOURCE_REQUIRED =>
      'The question\'s source and license are required.';

  @override
  String get error_QUESTION_STATE_INVALID =>
      'This isn\'t possible in the question\'s current status.';

  @override
  String get error_QUEUE_COOLDOWN => 'You can\'t join the queue for a while.';

  @override
  String get error_QUIZ_LIMIT_REACHED => 'You\'ve reached the quiz limit.';

  @override
  String get error_QUIZ_SIZE_INVALID =>
      'The number of quiz questions is outside the allowed range.';

  @override
  String get error_RATE_LIMITED =>
      'Too many requests. Please wait a moment and try again.';

  @override
  String get error_REPORT_LIMIT_REACHED =>
      'You\'ve reached today\'s report limit.';

  @override
  String get error_REQUEST_PENDING => 'A friend request is already pending.';

  @override
  String get error_REVENGE_NOT_ALLOWED =>
      'A revenge match isn\'t possible right now.';

  @override
  String get error_ROOM_CLOSED => 'The room has closed.';

  @override
  String get error_ROOM_FRIENDS_ONLY =>
      'Only the host\'s friends can join this room.';

  @override
  String get error_ROOM_FULL => 'The room is full.';

  @override
  String get error_SANCTIONED =>
      'This feature is temporarily restricted for your account.';

  @override
  String get error_SEAT_TAKEN_ELSEWHERE =>
      'You joined this match from another device.';

  @override
  String get error_SELF_REVIEW_FORBIDDEN => 'You can\'t review your own case.';

  @override
  String get error_TEAM_FULL => 'The team is full.';

  @override
  String get error_TEAM_LIMIT_REACHED => 'You\'ve reached the team limit.';

  @override
  String get error_TEXT_REJECTED => 'This text doesn\'t follow our rules.';

  @override
  String get error_TOKEN_EXPIRED =>
      'Your session has expired. Please try again.';

  @override
  String get error_TOURNAMENT_FULL => 'The tournament is full.';

  @override
  String get error_TOURNAMENT_REGISTRATION_CLOSED =>
      'Tournament registration is closed.';

  @override
  String get error_TOURNAMENT_REQUIREMENTS_NOT_MET =>
      'You don\'t meet this tournament\'s requirements.';

  @override
  String get error_UNAUTHENTICATED => 'You need to sign in.';

  @override
  String get error_UNAVAILABLE =>
      'The service is temporarily unavailable. Please try again soon.';

  @override
  String get error_USERNAME_TAKEN => 'This username is already taken.';

  @override
  String get error_VALIDATION_FAILED =>
      'Some of the submitted data is invalid.';

  @override
  String get error_VERSION_CONFLICT =>
      'Someone else changed this. Refresh and try again.';
}

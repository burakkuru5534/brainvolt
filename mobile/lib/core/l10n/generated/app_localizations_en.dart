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
  String get playQuickDuel => 'Quick Duel';

  @override
  String get playPractice => 'Practice vs bot';

  @override
  String get searchingTitle => 'Looking for an opponent…';

  @override
  String searchingElapsed(int seconds) {
    return '$seconds s';
  }

  @override
  String get searchingCancel => 'Cancel search';

  @override
  String get matchFoundTitle => 'Match found — ready?';

  @override
  String get readyButton => 'I\'m ready';

  @override
  String readyDeadline(int seconds) {
    return 'Get ready within $seconds s';
  }

  @override
  String get waitingForOpponent => 'Waiting for the opponent…';

  @override
  String matchStartsIn(int seconds) {
    return '$seconds';
  }

  @override
  String roundOf(int round, int total) {
    return 'Question $round/$total';
  }

  @override
  String get you => 'You';

  @override
  String get opponentAnswered => 'Opponent answered';

  @override
  String get answerCorrect => 'Correct!';

  @override
  String get answerWrong => 'Wrong';

  @override
  String get answerTimeout => 'Time\'s up';

  @override
  String pointsGained(int points) {
    return '+$points points';
  }

  @override
  String get matchWon => 'You won!';

  @override
  String get matchLost => 'You lost';

  @override
  String get matchDraw => 'Draw';

  @override
  String get matchCancelled => 'Match cancelled';

  @override
  String get opponentLeft => 'Your opponent left the match';

  @override
  String get backHome => 'Back to home';

  @override
  String get showDetails => 'See the questions';

  @override
  String get leaveMatchTitle => 'Leave the match?';

  @override
  String get leaveMatchBody => 'If you leave, you lose the match.';

  @override
  String get leaveButton => 'Leave';

  @override
  String get reconnecting => 'Reconnecting…';

  @override
  String get opponentDisconnected => 'Your opponent lost connection';

  @override
  String get seatMoved => 'The match was opened on another device';

  @override
  String get matchResumed => 'Server switched, the question was replaced';

  @override
  String get resultTitle => 'Match result';

  @override
  String resultCorrectAnswer(String answer) {
    return 'Correct answer: $answer';
  }

  @override
  String get resumeMatch => 'Back to your match';

  @override
  String get homeFriends => 'Friends';

  @override
  String get homeChallenges => 'Challenges';

  @override
  String get friendsTab => 'Friends';

  @override
  String get requestsTab => 'Requests';

  @override
  String get friendsEmpty =>
      'No friends yet. Add someone by username or friend code.';

  @override
  String get requestsEmpty => 'No pending requests.';

  @override
  String get friendsUpgrade => 'Link your account to add friends.';

  @override
  String friendCode(String code) {
    return 'Your friend code: $code';
  }

  @override
  String get copied => 'Copied';

  @override
  String get addFriend => 'Add friend';

  @override
  String get addByUsername => 'Username';

  @override
  String get addByCode => 'Friend code';

  @override
  String get send => 'Send';

  @override
  String get friendRequestSent => 'Friend request sent.';

  @override
  String get friendAdded => 'You are friends now!';

  @override
  String get online => 'Online';

  @override
  String get incomingLabel => 'Incoming';

  @override
  String get outgoingLabel => 'Sent';

  @override
  String get accept => 'Accept';

  @override
  String get decline => 'Decline';

  @override
  String get withdraw => 'Withdraw';

  @override
  String get unfriend => 'Remove friend';

  @override
  String get blockUser => 'Block';

  @override
  String blockConfirm(String name) {
    return 'Block $name? Your friendship and pending requests end.';
  }

  @override
  String get challengeAction => 'Challenge';

  @override
  String challengeTitle(String name) {
    return 'Challenge $name';
  }

  @override
  String get challengeSent => 'Challenge sent.';

  @override
  String get matchTypeQuickDuel => 'Quick Duel';

  @override
  String get matchTypeSuddenDeath => 'Sudden Death';

  @override
  String get matchTypeMarathon => 'Marathon';

  @override
  String get challengesTitle => 'Challenges';

  @override
  String get challengesEmpty => 'No pending challenges.';

  @override
  String challengeFrom(String name) {
    return '$name challenged you';
  }

  @override
  String challengeTo(String name) {
    return 'To $name';
  }

  @override
  String challengeReceived(String name) {
    return '$name challenged you!';
  }

  @override
  String get challengeAccepted => 'Your challenge was accepted!';

  @override
  String get view => 'View';

  @override
  String get playNow => 'Play';

  @override
  String get rematchButton => 'Rematch';

  @override
  String get rematchWaiting => 'Waiting for your opponent…';

  @override
  String get rematchOffered => 'Your opponent wants a rematch!';

  @override
  String get rematchDeclined => 'No rematch this time.';

  @override
  String get playModes => 'More modes';

  @override
  String get modesTitle => 'Choose a mode';

  @override
  String get matchTypeRanked => 'Ranked';

  @override
  String get categoryModeLabel => 'Category mode';

  @override
  String get categoryModeNone => 'Standard';

  @override
  String get categoryModeBlind => 'Blind Challenge';

  @override
  String get categoryModeMaster => 'Category Master';

  @override
  String get categoryModeMixed => 'Mixed Battle';

  @override
  String get categoryLabel => 'Category';

  @override
  String get startSearch => 'Find an opponent';

  @override
  String get pickCategory => 'Pick a category';

  @override
  String opponentPicking(String name) {
    return '$name is picking a category…';
  }

  @override
  String get privateRoom => 'Private room';

  @override
  String get createRoom => 'Create room';

  @override
  String get joinRoom => 'Join with code';

  @override
  String get roomCodeLabel => 'Room code';

  @override
  String roomCode(String code) {
    return 'Room code: $code';
  }

  @override
  String sideLabel(int no) {
    return 'Side $no';
  }

  @override
  String get emptySeat => 'Empty seat — move here';

  @override
  String get host => 'Host';

  @override
  String get readyLabel => 'Ready';

  @override
  String get notReady => 'Not ready';

  @override
  String get startMatch => 'Start the match';

  @override
  String get roomWaitingPlayers => 'Waiting for players…';

  @override
  String get leaveRoom => 'Leave the room';

  @override
  String get roomClosed => 'The room was closed.';

  @override
  String get backToRoom => 'Back to the room';

  @override
  String get homeTeams => 'Teams';

  @override
  String get teamsUpgrade =>
      'Link your account to create teams and form a couple.';

  @override
  String get coupleSection => 'Couple profile';

  @override
  String get noCouple => 'You have no couple profile.';

  @override
  String get requestCouple => 'Form a couple';

  @override
  String coupleWith(String name) {
    return 'Couple with $name';
  }

  @override
  String get couplePending => 'Waiting for an answer';

  @override
  String get coupleActive => 'Active';

  @override
  String get dissolveCouple => 'Break up';

  @override
  String get teamInvites => 'Team invites';

  @override
  String get myTeams => 'My teams';

  @override
  String get teamsEmpty => 'You have no team yet.';

  @override
  String get createTeam => 'Create team';

  @override
  String get teamName => 'Team name';

  @override
  String get teamTag => 'Tag (2–4)';

  @override
  String get captain => 'Captain';

  @override
  String get makeCaptain => 'Make captain';

  @override
  String get kick => 'Remove from team';

  @override
  String get inviteToTeam => 'Invite a friend';

  @override
  String get teamInviteSent => 'Invite sent.';

  @override
  String get leaveTeam => 'Leave the team';

  @override
  String get disbandTeam => 'Disband the team';

  @override
  String get disbandConfirm => 'Disband the team? This cannot be undone.';

  @override
  String get pickFriend => 'Pick a friend';

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

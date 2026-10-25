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
  String get homeWelcome => 'Ready to test your knowledge today?';

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
  String get rematchButton => 'Play again';

  @override
  String get rematchWaiting => 'Waiting for the other players…';

  @override
  String get rematchOffered => 'You were asked to play again!';

  @override
  String get rematchDeclined => 'No replay this time.';

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
  String get yourTeam => 'Your team';

  @override
  String get opponentsTeam => 'Opponents';

  @override
  String get lockAnswer => 'Lock the answer';

  @override
  String get yourTurn => 'Your turn';

  @override
  String turnOf(String name) {
    return '$name\'s turn';
  }

  @override
  String get formatLabel => 'Format';

  @override
  String get formatCouple => 'Couple';

  @override
  String get teamModeLabel => 'Team mode';

  @override
  String get teamModeShared => 'Shared Answer';

  @override
  String get teamModeTurns => 'Individual Turns';

  @override
  String get teamModeFast => 'Fast Team Round';

  @override
  String get partyLabel => 'Your party';

  @override
  String get pickParty => 'Pick friends';

  @override
  String get playAsTeam => 'Play as my team';

  @override
  String get noTeam => 'No team';

  @override
  String get leaderboardsTitle => 'Leaderboards';

  @override
  String get questsTitle => 'Quests';

  @override
  String get badgesTitle => 'Badges';

  @override
  String get periodDaily => 'Daily';

  @override
  String get periodWeekly => 'Weekly';

  @override
  String get periodMonthly => 'Monthly';

  @override
  String get periodAllTime => 'All time';

  @override
  String get periodSeason => 'Season';

  @override
  String get entityUser => 'Individual';

  @override
  String get entityTeam => 'Teams';

  @override
  String get entityCouple => 'Couples';

  @override
  String get scopeGlobal => 'Global';

  @override
  String get scopeCountry => 'Country';

  @override
  String get scopeFriends => 'Friends';

  @override
  String get leaderboardEmpty => 'No one here yet.';

  @override
  String get leaderboardMe => 'You';

  @override
  String boardPoints(int count) {
    return '$count pts';
  }

  @override
  String levelLabel(int level) {
    return 'Level $level';
  }

  @override
  String xpProgress(int current, int total) {
    return '$current / $total XP';
  }

  @override
  String streakDays(int days) {
    return '$days-day streak';
  }

  @override
  String seasonPoints(int sp) {
    return '$sp SP';
  }

  @override
  String get leagueBronze => 'Bronze';

  @override
  String get leagueSilver => 'Silver';

  @override
  String get leagueGold => 'Gold';

  @override
  String get leaguePlatinum => 'Platinum';

  @override
  String get leagueDiamond => 'Diamond';

  @override
  String nextDivisionAt(int sp) {
    return 'Next division: $sp SP';
  }

  @override
  String get questsDaily => 'Daily quests';

  @override
  String get questsWeekly => 'Weekly quests';

  @override
  String get questClaim => 'Claim';

  @override
  String get questClaimed => 'Claimed';

  @override
  String get questReroll => 'Reroll';

  @override
  String questRerollsLeft(int count) {
    return 'Rerolls left today: $count';
  }

  @override
  String questXpGained(int xp) {
    return '+$xp XP earned';
  }

  @override
  String questReward(int xp) {
    return '$xp XP';
  }

  @override
  String get badgeLocked => 'Not earned yet';

  @override
  String get awardsTitle => 'Rewards';

  @override
  String get awardsPending => 'Calculating rewards…';

  @override
  String awardsXp(int xp) {
    return '+$xp XP';
  }

  @override
  String awardsLevelUp(int level) {
    return 'Level up: $level!';
  }

  @override
  String awardsSp(String sp) {
    return '$sp SP';
  }

  @override
  String awardsLeagueUp(String league) {
    return 'New league: $league!';
  }

  @override
  String awardsBadge(String name) {
    return 'New badge: $name';
  }

  @override
  String awardsQuests(int count) {
    return '$count quests completed';
  }

  @override
  String get revengeLabel => 'Revenge';

  @override
  String get dailyTitle => 'Daily Challenge';

  @override
  String get dailyIntro =>
      'Everyone answers the same 10 questions. One official attempt a day; after that, practice.';

  @override
  String get dailyStart => 'Start';

  @override
  String get dailyResume => 'Resume';

  @override
  String get dailyPractice => 'Practice';

  @override
  String get dailyNotAvailable => 'Today\'s questions aren\'t ready yet.';

  @override
  String dailyResult(int correct, String seconds) {
    return '$correct/10 correct · $seconds s';
  }

  @override
  String dailyRank(int rank) {
    return 'Your place: #$rank';
  }

  @override
  String get dailyBoardTitle => 'Today\'s board';

  @override
  String dailyQuestionOf(int position, int total) {
    return 'Question $position of $total';
  }

  @override
  String get dailyPracticeLabel => 'Practice';

  @override
  String get dailyDone => 'Finished!';

  @override
  String get dailyBack => 'Back to the Daily Challenge';

  @override
  String get dailyVoided => 'This attempt didn\'t count.';

  @override
  String get dailyCorrect => 'Correct!';

  @override
  String get dailyWrong => 'Wrong';

  @override
  String get dailyTimeout => 'Time\'s up';

  @override
  String get reportQuestion => 'Report question';

  @override
  String get reportReasonTitle => 'What\'s wrong?';

  @override
  String get reportReasonWrongAnswer => 'The answer is wrong';

  @override
  String get reportReasonAmbiguous => 'Ambiguous';

  @override
  String get reportReasonOutdated => 'Outdated';

  @override
  String get reportReasonTypo => 'Typo or translation error';

  @override
  String get reportReasonOffensive => 'Offensive content';

  @override
  String get reportReasonOther => 'Other';

  @override
  String get reportComment => 'Note (optional)';

  @override
  String get reportSend => 'Send';

  @override
  String get reportSent => 'Thanks for your report.';

  @override
  String get sanctionsTitle => 'My sanctions';

  @override
  String get sanctionsEmpty => 'No sanctions on your account.';

  @override
  String get sanctionWarning => 'Warning';

  @override
  String get sanctionChatMute => 'Chat mute';

  @override
  String get sanctionRankedBan => 'Ranked ban';

  @override
  String get sanctionMatchmakingBan => 'Matchmaking ban';

  @override
  String get sanctionSocialBan => 'Social ban';

  @override
  String get sanctionSuspension => 'Account suspension';

  @override
  String sanctionUntil(String date) {
    return 'Until $date';
  }

  @override
  String get sanctionPermanent => 'Permanent';

  @override
  String get sanctionEnded => 'Ended';

  @override
  String get appealButton => 'Appeal';

  @override
  String get appealHint => 'Explain why it\'s unfair (at least 10 characters)';

  @override
  String get appealSent => 'Your appeal was received.';

  @override
  String get appealPending => 'Appeal under review';

  @override
  String get appealAccepted => 'Appeal accepted';

  @override
  String get appealRejected => 'Appeal rejected';

  @override
  String get quizzesTitle => 'Question Lab';

  @override
  String get quizzesMine => 'My quizzes';

  @override
  String get quizzesDiscover => 'Discover';

  @override
  String get quizzesEmptyMine => 'You haven\'t written a quiz yet.';

  @override
  String get quizzesEmptyDiscover => 'No published quizzes yet.';

  @override
  String get quizNew => 'New quiz';

  @override
  String get quizEdit => 'Edit quiz';

  @override
  String get quizTitleField => 'Title';

  @override
  String get quizDescriptionField => 'Description (optional)';

  @override
  String get quizLanguage => 'Language';

  @override
  String get quizCategory => 'Category';

  @override
  String get quizWindow => 'Answer time';

  @override
  String quizSeconds(int seconds) {
    return '$seconds s';
  }

  @override
  String quizQuestionN(int n) {
    return 'Question $n';
  }

  @override
  String get quizPrompt => 'Question';

  @override
  String quizOptionN(int n) {
    return 'Option $n';
  }

  @override
  String get quizTrueFalse => 'True / False';

  @override
  String get quizTrue => 'True';

  @override
  String get quizFalse => 'False';

  @override
  String get quizCorrectHint => 'Mark the correct answer with the circle.';

  @override
  String get quizAddQuestion => 'Add question';

  @override
  String get quizRemoveQuestion => 'Remove question';

  @override
  String get quizSave => 'Save';

  @override
  String get quizSaved => 'Quiz saved.';

  @override
  String get quizInvalid =>
      'The title needs 3–60 characters and every question and option must be filled in.';

  @override
  String quizCountRule(int min, int max) {
    return 'You can add $min–$max questions.';
  }

  @override
  String get quizStatusDraft => 'Draft';

  @override
  String get quizStatusSubmitted => 'In review';

  @override
  String get quizStatusPublished => 'Published';

  @override
  String get quizStatusRejected => 'Rejected';

  @override
  String get quizStatusUnpublished => 'Unpublished';

  @override
  String get quizSubmit => 'Send to review';

  @override
  String get quizSubmitted => 'Quiz sent to review.';

  @override
  String get quizDelete => 'Delete';

  @override
  String get quizDeleteConfirm => 'Delete this quiz?';

  @override
  String get quizPlayInRoom => 'Play in a room';

  @override
  String quizMeta(int count, String author) {
    return '$count questions · $author';
  }

  @override
  String quizLikes(int count, int plays) {
    return '$count likes · $plays plays';
  }

  @override
  String get quizLike => 'Like';

  @override
  String quizRejectedReason(String reason) {
    return 'Rejected: $reason';
  }

  @override
  String get quizUnpublishedNote =>
      'An unpublished quiz is played with your friends only and gives no rewards.';

  @override
  String roomQuiz(int count) {
    return 'Quiz · $count questions';
  }

  @override
  String get roomFriendsOnly => 'Only the host\'s friends can join.';

  @override
  String get homeQuizzes => 'Question Lab';

  @override
  String get homeQuickDuelSub => 'Find your next opponent';

  @override
  String get homeModesSub => 'Explore all modes';

  @override
  String get homeRoomSub => 'Create or join a game';

  @override
  String get homeTeamsSub => 'Team up & win';

  @override
  String get homeDailySub => 'Today\'s 10 questions';

  @override
  String get homeQuizzesSub => 'Suggest questions, build quizzes';

  @override
  String get homePracticeSub => 'Sharpen your skills in a solo match';

  @override
  String get homeTotalWins => 'Total wins';

  @override
  String get homeCurrentRank => 'Current rank';

  @override
  String get homeNoRank => 'Unranked';

  @override
  String get roomSheetTitle => 'Private Room';

  @override
  String get roomSheetSub => 'Choose how you want to play';

  @override
  String get roomQuickDuelSub => 'Challenge friends in a classic match';

  @override
  String get roomSuddenDeathSub => 'One mistake can change everything';

  @override
  String get roomMarathonSub => 'Compete through many rounds';

  @override
  String get roomJoinWithCode => 'Join with code';

  @override
  String get roomCodeHint => 'Enter room code';

  @override
  String get roomJoinAction => 'Join';

  @override
  String get gameModeLabel => 'Game mode';

  @override
  String get queueFinding => 'Finding your next challenge';

  @override
  String get inboxTitle => 'Notifications';

  @override
  String get inboxEmpty => 'No notifications yet.';

  @override
  String get inboxMarkAll => 'Mark all read';

  @override
  String get notificationSettingsTitle => 'Notification settings';

  @override
  String get notifPush => 'Push';

  @override
  String get notifInApp => 'In-app';

  @override
  String get notifQuietHours => 'Quiet hours';

  @override
  String get notifQuietHoursHint =>
      'No push in these hours; notices wait in the inbox.';

  @override
  String get notifQuietStart => 'Starts';

  @override
  String get notifQuietEnd => 'Ends';

  @override
  String get notifLocked => 'Account notices cannot be turned off.';

  @override
  String get notifCatChallenge => 'Challenges';

  @override
  String get notifCatFriend => 'Friends';

  @override
  String get notifCatMatchFound => 'Match found';

  @override
  String get notifCatTeam => 'Teams and couples';

  @override
  String get notifCatChat => 'Chat';

  @override
  String get notifCatSeason => 'Season';

  @override
  String get notifCatQuest => 'Quests';

  @override
  String get notifCatTournament => 'Tournaments';

  @override
  String get notifCatModeration => 'Account notices';

  @override
  String get notifCatProductNews => 'News and offers';

  @override
  String get settingsNotifications => 'Notifications';

  @override
  String get shareAction => 'Share';

  @override
  String get shareResult => 'Share result';

  @override
  String shareMatchText(String score, String result) {
    return 'BrainVolt $score: $result Your turn!';
  }

  @override
  String shareDailyText(int correct) {
    return 'BrainVolt Daily Challenge: $correct/10 correct. Can you beat it?';
  }

  @override
  String shareCorrectOf(int correct, int total) {
    return '$correct/$total correct';
  }

  @override
  String get chatTitle => 'Chat';

  @override
  String get chatHint => 'Write a message…';

  @override
  String get chatSend => 'Send';

  @override
  String get chatEmpty => 'No messages yet. Say hello!';

  @override
  String get chatRemoved => 'This message was removed.';

  @override
  String chatTyping(String name) {
    return '$name is typing…';
  }

  @override
  String get chatDelete => 'Delete message';

  @override
  String get chatOpen => 'Open chat';

  @override
  String get chatQuickGoodLuck => 'Good luck!';

  @override
  String get chatQuickWellPlayed => 'Well played!';

  @override
  String get chatQuickNice => 'Nice!';

  @override
  String get chatQuickThanks => 'Thanks!';

  @override
  String get chatQuickOops => 'Oops!';

  @override
  String get chatQuickLetsGo => 'Let\'s go!';

  @override
  String get chatQuickGg => 'GG';

  @override
  String get chatQuickRematch => 'Play again?';

  @override
  String get tournamentsTitle => 'Tournaments';

  @override
  String get homeTournamentsSub => 'Win the cup';

  @override
  String get tournamentStatusRegistration => 'Registration open';

  @override
  String get tournamentStatusRunning => 'In progress';

  @override
  String get tournamentStatusFinished => 'Finished';

  @override
  String get tournamentStatusCancelled => 'Cancelled';

  @override
  String get tournamentStatusDraft => 'Draft';

  @override
  String get tournamentCheckedIn => 'Checked in';

  @override
  String get tournamentPlaying => 'Playing';

  @override
  String tournamentPlace(int place) {
    return 'Place $place';
  }

  @override
  String get tournamentChampion => 'Champion! 🏆';

  @override
  String get tournamentRegistered => 'Registered';

  @override
  String get tournamentKindCouple => 'Couples';

  @override
  String tournamentKindTeam(String format) {
    return 'Teams $format';
  }

  @override
  String get tournamentKindSolo => 'Solo';

  @override
  String get tournamentCreate => 'Create tournament';

  @override
  String get tournamentsEmpty => 'No tournaments right now. New ones soon!';

  @override
  String tournamentEntrants(int count, int max) {
    return '$count/$max entrants';
  }

  @override
  String get tournamentBracket => 'Bracket';

  @override
  String get tournamentRegister => 'Register';

  @override
  String get tournamentCheckedInInfo =>
      'You\'re checked in. Your match shows here when the tournament starts.';

  @override
  String get tournamentCheckInNow => 'Check-in is open! Check in to play.';

  @override
  String tournamentCheckInAt(String time) {
    return 'Check-in opens at $time. Only checked-in entries play.';
  }

  @override
  String get tournamentCheckIn => 'Check in';

  @override
  String get tournamentWithdraw => 'Withdraw';

  @override
  String tournamentYourMatch(int round, String name) {
    return 'Round $round: you play $name';
  }

  @override
  String get tournamentWaitingOpponent =>
      'You\'re in. Waiting for your opponent…';

  @override
  String tournamentReadyLeft(String time) {
    return 'Time left to show up: $time';
  }

  @override
  String get tournamentGoToMatch => 'Go to match';

  @override
  String get tournamentPlay => 'Play';

  @override
  String get tournamentWaitingRound => 'Waiting for your next round.';

  @override
  String tournamentStarts(String time) {
    return 'Starts: $time';
  }

  @override
  String tournamentRewardLine(int place, int xp) {
    return '#$place: $xp XP';
  }

  @override
  String get tournamentFinal => 'Final';

  @override
  String get tournamentSemiFinal => 'Semi-final';

  @override
  String tournamentRound(int round) {
    return 'Round $round';
  }

  @override
  String get tournamentBye => 'Bye';

  @override
  String get tournamentWalkover => 'Walkover';

  @override
  String tournamentPickLineup(int size) {
    return 'Pick your team and $size players';
  }

  @override
  String get tournamentNoCaptainTeam => 'You aren\'t captain of any team.';

  @override
  String get tournamentNameTr => 'Name (Turkish)';

  @override
  String get tournamentNameEn => 'Name (English, optional)';

  @override
  String get tournamentKind => 'Entrants';

  @override
  String get tournamentFormat => 'Team size';

  @override
  String get tournamentMatchType => 'Match type';

  @override
  String get tournamentMax => 'Max entrants';

  @override
  String get tournamentStartIn => 'Starts in';

  @override
  String tournamentMinutes(int minutes) {
    return '$minutes min';
  }

  @override
  String get tournamentWindow => 'Time to show up';

  @override
  String get tournamentCreateHint =>
      'Registration opens now and closes at the start. Check-in in the last hour. Rewards: 1st 300 XP, 2nd 150 XP, 3rd 75 XP, others 20 XP.';

  @override
  String get tournamentMatchStarted => 'Your tournament match is on!';

  @override
  String roomReturning(int seconds) {
    return 'Back to the room in $seconds s…';
  }

  @override
  String get roomSettingsTitle => 'Room settings';

  @override
  String get matchTypeLabel => 'Match type';

  @override
  String get roomSettingsSaved =>
      'Room settings updated — everyone readies up again';

  @override
  String get mmLobbyTitle => 'Match lobby';

  @override
  String mmLobbyWaiting(int count, int total) {
    return 'Finding players… ($count/$total)';
  }

  @override
  String get mmLobbyLeaderSoon =>
      'Once the lobby is full a random leader is picked; they start the match.';

  @override
  String get mmLobbyYouLead =>
      'You lead this lobby! Start the match once everyone is ready.';

  @override
  String mmLobbyLeaderIs(String name) {
    return 'Lobby leader: $name. Get ready — the leader starts the match.';
  }

  @override
  String get mmLobbyWaitReady => 'Waiting for everyone to be ready';

  @override
  String get mmLobbySearchingSeat => 'Finding a player…';

  @override
  String get mmLobbyDropped =>
      'You were removed from the lobby for not getting ready in time.';

  @override
  String get mmLobbyExpired =>
      'Not enough players were found; the lobby closed.';

  @override
  String get topicLabel => 'Topic';

  @override
  String get topicAll => 'All topics';

  @override
  String get topicAllSub => 'Mixed questions from every topic';

  @override
  String get topicPickTitle => 'Which topic do you want to play?';

  @override
  String get topicPickSub =>
      'You are matched only with players who chose the same topic.';

  @override
  String topicLine(String name) {
    return 'Topic: $name';
  }

  @override
  String get topicLoadFailed =>
      'Topics could not be loaded; you can still play All topics.';

  @override
  String get topicRankedNote => 'Ranked matches always use every topic.';

  @override
  String get brandTagline => 'TEST YOUR KNOWLEDGE, REACH THE TOP';

  @override
  String get homeModesTitle => 'Game modes';

  @override
  String get homeSeeAll => 'See all';

  @override
  String get homeMoreTitle => 'Explore';

  @override
  String get homeAllModes => 'All modes';

  @override
  String get homeRankedSub => 'Climb your league';

  @override
  String get homeDailyTag => 'TODAY';

  @override
  String get homeDailyPlay => 'Play';

  @override
  String get navHome => 'Home';

  @override
  String get navLeaderboards => 'Leaders';

  @override
  String get profileEdit => 'Edit profile';

  @override
  String get profileMatches => 'Matches';

  @override
  String get profileWins => 'Wins';

  @override
  String get profileWinRate => 'Win rate';

  @override
  String get profileBestStreak => 'Best streak';

  @override
  String get profileAchievements => 'Achievements';

  @override
  String get profileNoBadges =>
      'No badges yet. Play matches and finish quests!';

  @override
  String get badgesEmpty => 'No badges yet.';

  @override
  String get questsEmpty => 'No quests right now, new ones are on the way.';

  @override
  String get resultRounds => 'Questions';

  @override
  String get queueTip1 =>
      'The faster you answer correctly, the more points you score.';

  @override
  String get queueTip2 =>
      'In Sudden Death the first wrong answer ends the match: be sure before you tap.';

  @override
  String get queueTip3 =>
      'Marathon is a long match: keep your focus until the last question.';

  @override
  String get queueTip4 =>
      'Ranked matches move your league points; climb from Bronze to Diamond.';

  @override
  String get queueTip5 =>
      'Pick a topic and you are matched only with players who picked the same one.';

  @override
  String get queueTip6 =>
      'In the Daily Challenge everyone answers the same 10 questions; claim your rank.';

  @override
  String get queueTip7 =>
      'Finish quests for XP; you can swap a quest you do not like once a day.';

  @override
  String get queueTip8 => 'Play at least one match a day to keep your streak.';

  @override
  String get queueTip9 =>
      'Share your friend code, open a Private Room and play with friends.';

  @override
  String get queueTip10 =>
      'Build a team and compete together in 2v2, 3v3 and 4v4 matches.';

  @override
  String get queueTip11 =>
      'Suggest your own questions in the Question Lab; approved ones are played by everyone.';

  @override
  String get queueTipLabel => 'TIP';

  @override
  String get answerHiddenNote =>
      'You can see the right answer on the result screen after the match.';

  @override
  String get dailyAnswerNextDay =>
      'The right answers open tomorrow in \"Yesterday\'s answers\".';

  @override
  String get dailyYesterdayAnswers => 'Yesterday\'s answers';

  @override
  String get dailyYesterdaySub =>
      'The right answers to yesterday\'s 10 questions';

  @override
  String get dailyAnswersTitle => 'Yesterday\'s answers';

  @override
  String get dailyAnswersYours => 'Your answer';

  @override
  String get dailyAnswersNotPlayed => 'You didn\'t answer this one.';

  @override
  String get dailyAnswersEmpty => 'No answers for yesterday.';

  @override
  String get playerProfileTitle => 'Player profile';

  @override
  String get playerViewProfile => 'View profile';

  @override
  String get playerAddFriend => 'Add friend';

  @override
  String get playerAccept => 'Accept request';

  @override
  String get playerFriend => 'You are friends';

  @override
  String get playerRequestSent => 'Request sent';

  @override
  String get playerPrivate =>
      'This profile is private. Only their friends can see the stats.';

  @override
  String get playerPrivateTag => 'Private profile';

  @override
  String get playerBlocked => 'You can\'t see this player\'s profile.';

  @override
  String get playerStats => 'Stats';

  @override
  String playerLevel(int level) {
    return 'Level $level';
  }

  @override
  String get playerLevelLabel => 'Level';

  @override
  String get playerMatches => 'Matches';

  @override
  String get playerWins => 'Wins';

  @override
  String get playerWinRate => 'Win rate';

  @override
  String get playerBadges => 'Badges';

  @override
  String get playerStreak => 'Streak';

  @override
  String get settingsPrivateProfile => 'Private profile';

  @override
  String get settingsPrivateProfileSub =>
      'Only your friends see your stats. Your name still shows on leaderboards.';

  @override
  String get guideTitle => 'Game guide';

  @override
  String get guideSub => 'Modes, scoring, leagues and more';

  @override
  String get guideIntro =>
      'BrainVolt is a real-time quiz game. You and your opponent see the same question at the same moment; being right and fast wins. Tap a topic below to read more.';

  @override
  String get guideBasicsTitle => 'How to play';

  @override
  String get guideBasicsBody =>
      'Each question has four options (sometimes True / False) and a timer. Tap an option before time runs out.\n• Both sides see the same question at the same moment; network delay is compensated on the server.\n• A wrong answer or a timeout costs nothing, it just scores 0.\n• If you answer wrong, the right answer is not shown during the match; after the match you can see every question and answer on the result screen.\n• Questions you saw in the last 30 days are not asked again.\n• If your connection drops, the app rejoins the match by itself. Staying away too long or dropping too often loses the match.';

  @override
  String get guideScoringTitle => 'Scoring';

  @override
  String get guideScoringBody =>
      'A right answer earns points in three parts:\n• Base points: 80–120 depending on the question\'s difficulty.\n• Speed bonus: the sooner you answer, the more, up to +100. Answering at the last moment gives 0.\n• Streak bonus: +20 for 3–4 right in a row, +40 from 5.\nOne question is worth at most 260 points. True / False questions are worth half (at most 150).\nAt the end the higher total wins; on a tie, right answers and speed decide.';

  @override
  String get guideTypesTitle => 'Match types';

  @override
  String get guideTypesBody =>
      '• Quick Duel: 7 questions, 15 s each. A short, relaxed match.\n• Ranked: 10 questions, 15 s, multiple choice only. Played only through matchmaking; the only mode that changes your Season Points (SP).\n• Sudden Death: 10 s per question. When one side is right and the other wrong in a round, the match ends. Difficulty rises every 3 rounds; if nothing is decided after 30 rounds, the higher score wins.\n• Marathon: 25 questions, 12 s each, with a 15 s break after question 12.';

  @override
  String get guideCategoryTitle => 'Topic modes';

  @override
  String get guideCategoryBody =>
      'Outside Ranked you can choose how questions are picked:\n• Standard: questions come mixed from different topics.\n• Blind Challenge: a question\'s topic stays hidden until it is answered.\n• Category Master: every question comes from one topic you choose. In matchmaking you only meet players looking for the same topic.\n• Mixed Battle: 3 topics are set for the match; each round the sides take turns picking one of them (10 s).\n• Revenge: the losing side may ask for a revenge within 24 hours after the match; half of the questions come from a topic they choose. A series has at most 3 revenge matches.';

  @override
  String get guideTeamsTitle => 'Teams and couples';

  @override
  String get guideTeamsBody =>
      'Besides one-on-one you can play 2v2, 3v3, 4v4 and Couple formats. You can create a team and invite friends, or become a couple with a friend. Team matches have three answer modes:\n• Shared Answer: the team proposes an option together and one member locks it in. You get 5 extra seconds to agree.\n• Individual Turns: the next player in the rotation answers each question.\n• Fast Team Round: everyone answers alone; the team\'s points are the average of its members.\nA team scores on the same scale as a single player; a bigger side does not earn extra points.';

  @override
  String get guideDailyTitle => 'Daily Challenge';

  @override
  String get guideDailyBody =>
      'Every day everyone answers the same 10 questions in the same order. The day resets at midnight Istanbul time.\n• You have one official attempt a day; later runs are practice and don\'t affect the board.\n• The board ranks by right answers first, then by total answer time.\n• Right answers are not shown during the day. You can see them the next day in \"Yesterday\'s answers\".\n• If your connection drops, you can continue within 2 minutes; the clock keeps running meanwhile.\n• The official run earns XP (30 + 5 per right answer) and keeps your play streak.';

  @override
  String get guideRoomsTitle => 'Private rooms and challenges';

  @override
  String get guideRoomsBody =>
      '• Private room: create a room and share its code with friends. The host picks the match type, format and topic mode; the match starts when everyone is ready. After the match everyone returns to the room.\n• Challenge: send a match invite straight to a friend from your friends list.\n• These matches earn XP but no SP or leaderboard points, so friends can\'t farm points against each other.';

  @override
  String get guideTournamentsTitle => 'Tournaments';

  @override
  String get guideTournamentsBody =>
      'Tournaments are knockout brackets. Register, then check in during the last hour before the start; players who don\'t check in are left out.\n• Each round you get a notice when your match is ready; if you don\'t join in time you lose by walkover.\n• There are solo, team and couple tournaments.\n• Rewards: 1st 300 XP, 2nd 150 XP, 3rd 75 XP, everyone else 20 XP.';

  @override
  String get guideLabTitle => 'Question Lab';

  @override
  String get guideLabBody =>
      'Make your own quiz: start with one question and grow it to up to 30 with \"Add question\". You choose the answer time (10, 15 or 20 s).\n• Draft quizzes can only be played with friends in a private room and give no rewards.\n• Send a quiz for review; once approved it is published, anyone can discover and play it, and players earn XP.\n• You earn no XP from your own quiz.';

  @override
  String get guideLeaguesTitle => 'Leagues, seasons and SP';

  @override
  String get guideLeaguesBody =>
      'Season Points (SP) come only from Ranked: win +30, draw +10, loss −15. From your 3rd ranked win in a row a streak bonus of +5, +10 and at most +15 is added.\n• Leagues: Bronze (0), Silver (300), Gold (800), Platinum (1600) and Diamond (2800 SP). Every league except Diamond has three divisions (III, II, I).\n• You can\'t drop out of a league during a season; divisions inside it can go up and down.\n• A season lasts 8 weeks. At the end you get a badge for the highest league you reached, then everyone starts again in Bronze III.';

  @override
  String get guideXpTitle => 'XP, levels, streaks and quests';

  @override
  String get guideXpBody =>
      '• Every completed match earns XP: 20 + 5 per right answer + a result bonus (win 15, draw 8).\n• Each level needs a little more XP than the one before; the top level is 100. Level only shows your progress and never affects matchmaking or scoring.\n• Streak: complete at least one match or an official Daily Challenge every day to grow your streak; missing a day resets it.\n• Quests: 3 daily and 3 weekly quests. Claim the reward when you finish one; you can swap one quest a day for free.\n• Badges: earn badges for wins, accuracy, speed, streaks, topics and seasons.';

  @override
  String get guideBoardsTitle => 'Leaderboards';

  @override
  String get guideBoardsBody =>
      'There are daily, weekly, monthly and all-time boards, each for the world, your country and your friends. Teams and couples have their own boards too.\n• Leaderboard points come only from matchmade games: win 20, draw 10, +2 per right answer.\n• The season board is ranked by SP, and the Daily Challenge has its own board.\n• Tap a player to add them as a friend or view their profile. If you make your profile private, only your friends see your stats.';

  @override
  String get guideFairTitle => 'Fair play';

  @override
  String get guideFairBody =>
      '• Points and results are computed only on the server; changing the app cannot change a result.\n• Right answers are hidden during a match from players who answered wrong, and in the Daily Challenge from everyone until the day is over.\n• Matches between friends give no SP or leaderboard points.\n• If a question looks wrong, use \"Report question\" after the match. Breaking the rules can lead to sanctions.';

  @override
  String get adBonusXpButton => 'Watch an ad: +50% XP';

  @override
  String adBonusXpDone(int xp) {
    return '+$xp XP added!';
  }

  @override
  String get adNoFill => 'No ad to show right now. Try again in a bit.';

  @override
  String get adNotEarned => 'Watch the ad to the end to get the reward.';

  @override
  String get adPending => 'Your reward will arrive in a moment.';

  @override
  String get adQuestReroll => 'Watch an ad to swap';

  @override
  String adStreakTitle(int days) {
    return 'Your $days-day streak is broken!';
  }

  @override
  String get adStreakSub => 'Watch an ad to save it.';

  @override
  String adStreakDone(int days) {
    return 'Streak saved: $days days!';
  }

  @override
  String get adWatch => 'Watch';

  @override
  String get plusTitle => 'BrainVolt Plus';

  @override
  String get plusTagline => 'Play without ads and earn more rewards every day.';

  @override
  String get plusPerkNoAds => 'No ads: no interstitials';

  @override
  String get plusPerkRewards => 'Rewarded-ad rewards without watching ads';

  @override
  String get plusPerkQuests => '+1 quest and +1 free swap every day';

  @override
  String get plusPerkBadge => 'Plus badge on your home screen';

  @override
  String get plusMonthly => 'Monthly';

  @override
  String get plusAnnual => 'Yearly';

  @override
  String plusPerMonth(String price) {
    return '$price/month';
  }

  @override
  String plusPerYear(String price) {
    return '$price/year';
  }

  @override
  String plusTrial(int days) {
    return 'Try free for $days days';
  }

  @override
  String plusStartTrial(int days) {
    return 'Start $days-day free trial';
  }

  @override
  String get plusBuy => 'Get Plus';

  @override
  String get plusRestore => 'Restore purchases';

  @override
  String get plusLegal =>
      'The subscription renews automatically through Google Play at the end of each period unless cancelled. Cancel any time in Google Play.';

  @override
  String get plusActive => 'Plus is active.';

  @override
  String plusActiveUntil(String date) {
    return 'Plus is active. Renews on $date';
  }

  @override
  String get plusManage => 'Manage subscription';

  @override
  String get plusLinkFirst =>
      'Link your account to Google, Apple or email first so your subscription is never lost.';

  @override
  String get plusUnavailable =>
      'Purchases are not available on this device yet.';

  @override
  String get plusNoPlans => 'Plans couldn\'t be loaded. Try again in a bit.';

  @override
  String get plusWelcome => 'Welcome to Plus!';

  @override
  String get plusActivating => 'Purchase done; Plus turns on in a few seconds.';

  @override
  String get plusPending =>
      'Your payment is pending; Plus turns on once it is approved.';

  @override
  String get plusFailed => 'The purchase couldn\'t be completed. Try again.';

  @override
  String get plusNothingToRestore => 'No subscription found to restore.';

  @override
  String get plusCta => 'Get Plus';

  @override
  String get voltTitle => 'Volt Store';

  @override
  String get voltBalanceLabel => 'Your Volt';

  @override
  String get voltTagline => 'Earn it by playing, spend it in the store.';

  @override
  String voltAdTitle(int amount) {
    return 'Watch an ad, earn $amount Volt';
  }

  @override
  String voltAdLeft(int count) {
    return '$count left today';
  }

  @override
  String get voltAdNone => 'No more ads today';

  @override
  String voltAdDone(int amount) {
    return 'You earned $amount Volt';
  }

  @override
  String get voltShop => 'Store';

  @override
  String get voltItemShield => 'Streak Shield';

  @override
  String voltItemShieldSub(int count, int max) {
    return 'Keeps your streak through a missed day. You hold $count/$max';
  }

  @override
  String get voltItemReroll => 'Extra quest reroll';

  @override
  String get voltItemRerollSub =>
      'Swap one more quest once your free reroll is used.';

  @override
  String voltBuyTitle(String item) {
    return 'Buy $item?';
  }

  @override
  String voltBuyBody(int price) {
    return 'This spends $price Volt.';
  }

  @override
  String get voltBuy => 'Buy';

  @override
  String voltBought(String item) {
    return '$item bought';
  }

  @override
  String get voltEarnTitle => 'How to earn';

  @override
  String get voltEarnQuests => 'Daily quest +10 · Weekly quest +50';

  @override
  String get voltEarnDaily => 'Daily Challenge +10';

  @override
  String get voltEarnLevel => 'Each new level +25';

  @override
  String get voltEarnTournament => 'Tournament top three: +300 / +150 / +75';

  @override
  String get voltEarnSeason => 'Season-end league reward: +50 to +500';

  @override
  String get voltEarnPlus => 'Plus members get +400 every month';

  @override
  String get voltHistory => 'Recent activity';

  @override
  String get voltHistoryEmpty => 'Nothing here yet.';

  @override
  String get voltReasonQuest => 'Quest reward';

  @override
  String get voltReasonDaily => 'Daily Challenge';

  @override
  String voltReasonLevel(String level) {
    return 'Level $level';
  }

  @override
  String get voltReasonTournament => 'Tournament place';

  @override
  String get voltReasonSeason => 'Season reward';

  @override
  String get voltReasonAd => 'Ad reward';

  @override
  String get voltReasonPlus => 'Plus monthly Volt';

  @override
  String get voltReasonPlusRefund => 'Plus refund';

  @override
  String voltReasonItem(String item) {
    return 'Store: $item';
  }

  @override
  String get voltReasonOther => 'Adjustment';

  @override
  String voltRerollWith(int price) {
    return 'Swap for $price Volt';
  }

  @override
  String questVoltReward(int volt) {
    return '+$volt';
  }

  @override
  String get error_ACCOUNT_DELETION_PENDING =>
      'Your account is being deleted. You can only cancel the deletion.';

  @override
  String get error_ACCOUNT_SUSPENDED => 'Your account is suspended.';

  @override
  String get error_ACCOUNT_UPGRADE_REQUIRED =>
      'Upgrade your account to do this.';

  @override
  String get error_AD_LIMIT_REACHED =>
      'You\'ve reached today\'s limit for this reward.';

  @override
  String get error_AD_REWARD_LIMIT =>
      'You\'ve reached today\'s ad reward limit.';

  @override
  String get error_AD_REWARD_UNAVAILABLE =>
      'This reward isn\'t available right now.';

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
  String get error_DAILY_ANSWERS_NOT_OPEN =>
      'A day\'s answers open once the day is over.';

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
  String get error_ITEM_UNAVAILABLE => 'You can\'t get this item right now.';

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
  String get error_QUEST_ALREADY_CLAIMED => 'You already claimed this reward.';

  @override
  String get error_QUEST_NOT_COMPLETED => 'This quest isn\'t completed yet.';

  @override
  String get error_QUEST_NOT_REROLLABLE => 'This quest can\'t be rerolled.';

  @override
  String get error_QUEST_REROLL_LIMIT => 'No quest rerolls left today.';

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

  @override
  String get error_VOLT_INSUFFICIENT => 'You don\'t have enough Volt.';
}

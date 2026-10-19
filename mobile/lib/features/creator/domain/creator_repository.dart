/// Creator Quiz (D-044, D-083): player-made quizzes, their review and
/// discovery.
library;

/// One question of a quiz as its author wrote it.
class QuizQuestion {
  const QuizQuestion({
    required this.prompt,
    required this.options,
    required this.correct,
  });

  factory QuizQuestion.fromJson(Map<String, dynamic> j) => QuizQuestion(
    prompt: j['prompt'] as String? ?? '',
    options: [for (final o in (j['options'] as List? ?? const [])) '$o'],
    correct: (j['correct'] as num?)?.toInt() ?? 1,
  );

  final String prompt;

  /// Two options make a true/false question, four a multiple-choice one.
  final List<String> options;

  /// 1-based position of the correct option.
  final int correct;

  String get type => options.length == 2 ? 'true_false' : 'multiple_choice';

  Map<String, dynamic> toJson() => {
    'type': type,
    'prompt': prompt,
    'options': options,
    'correct': correct,
  };
}

/// A quiz; [questions] only reach its author.
class Quiz {
  const Quiz({
    required this.id,
    required this.title,
    required this.locale,
    required this.answerWindowMs,
    required this.questionCount,
    required this.status,
    required this.version,
    this.authorName = '',
    this.description,
    this.categoryId,
    this.rejectionReason,
    this.likeCount = 0,
    this.playCount = 0,
    this.liked = false,
    this.questions = const [],
  });

  factory Quiz.fromJson(Map<String, dynamic> j) => Quiz(
    id: j['id'] as String,
    title: j['title'] as String? ?? '',
    description: j['description'] as String?,
    locale: j['locale'] as String? ?? 'tr',
    categoryId: j['category_id'] as String?,
    answerWindowMs: (j['answer_window_ms'] as num?)?.toInt() ?? 15000,
    questionCount: (j['question_count'] as num?)?.toInt() ?? 0,
    status: j['status'] as String? ?? 'draft',
    rejectionReason: j['rejection_reason'] as String?,
    likeCount: (j['like_count'] as num?)?.toInt() ?? 0,
    playCount: (j['play_count'] as num?)?.toInt() ?? 0,
    liked: j['liked'] == true,
    version: (j['version'] as num?)?.toInt() ?? 1,
    authorName: (j['author'] as Map?)?['display_name'] as String? ?? '',
    questions: [
      for (final q in (j['questions'] as List? ?? const []))
        QuizQuestion.fromJson((q as Map).cast<String, dynamic>()),
    ],
  );

  final String id;
  final String title;
  final String? description;
  final String locale;
  final String? categoryId;
  final int answerWindowMs;
  final int questionCount;

  /// draft / submitted / published / rejected / unpublished.
  final String status;
  final String? rejectionReason;
  final int likeCount;
  final int playCount;
  final bool liked;
  final int version;
  final String authorName;
  final List<QuizQuestion> questions;

  bool get published => status == 'published';
}

/// What the editor sends.
class QuizDraft {
  const QuizDraft({
    required this.title,
    required this.locale,
    required this.categoryId,
    required this.answerWindowMs,
    required this.questions,
    this.description,
  });

  final String title;
  final String? description;
  final String locale;
  final String categoryId;
  final int answerWindowMs;
  final List<QuizQuestion> questions;

  Map<String, dynamic> toJson() => {
    'title': title,
    if (description != null && description!.isNotEmpty)
      'description': description,
    'locale': locale,
    'category_id': categoryId,
    'answer_window_ms': answerWindowMs,
    'questions': [for (final q in questions) q.toJson()],
  };
}

/// Quiz limits (D-031).
const quizMinQuestions = 5;
const quizMaxQuestions = 30;
const quizWindows = [10000, 15000, 20000];

abstract interface class CreatorRepository {
  Future<List<Quiz>> mine();

  /// Published quizzes, most liked first.
  Future<List<Quiz>> discover({String? locale});

  Future<Quiz> quiz(String id);

  Future<Quiz> create(QuizDraft draft);

  /// Replaces a quiz; [version] guards against a concurrent edit.
  Future<Quiz> update(String id, int version, QuizDraft draft);

  Future<void> delete(String id);

  Future<Quiz> submit(String id);

  Future<void> like(String id, {required bool like});
}

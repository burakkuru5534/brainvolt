import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/l10n/l10n.dart';
import '../../match/presentation/mode_picker_screen.dart';
import '../../match/presentation/play_error_text.dart';
import '../data/creator_providers.dart';
import '../domain/creator_repository.dart';

/// One question while it is edited.
class _Draft {
  _Draft({String prompt = '', List<String>? options, this.correct = 1})
    : prompt = TextEditingController(text: prompt),
      options = [
        for (final o in options ?? const ['', '', '', ''])
          TextEditingController(text: o),
      ];

  factory _Draft.of(QuizQuestion q) =>
      _Draft(prompt: q.prompt, options: q.options, correct: q.correct);

  final TextEditingController prompt;
  List<TextEditingController> options;
  int correct;

  bool get trueFalse => options.length == 2;

  void dispose() {
    prompt.dispose();
    for (final o in options) {
      o.dispose();
    }
  }
}

/// Writes a new quiz or edits one of the caller's ([quizId]): details,
/// answer window and 5–30 questions (D-031, D-083). Saving a submitted or
/// published quiz sends it back to review.
class QuizEditorScreen extends ConsumerStatefulWidget {
  const QuizEditorScreen({this.quizId, super.key});

  final String? quizId;

  @override
  ConsumerState<QuizEditorScreen> createState() => _QuizEditorScreenState();
}

class _QuizEditorScreenState extends ConsumerState<QuizEditorScreen> {
  final _title = TextEditingController();
  final _description = TextEditingController();
  // A new quiz starts in the app's language.
  String? _locale;
  String? _category;
  var _window = 15000;
  final _questions = <_Draft>[];
  Quiz? _quiz;
  var _loading = false;
  var _saving = false;

  @override
  void initState() {
    super.initState();
    if (widget.quizId case final id?) {
      _loading = true;
      unawaited(Future.microtask(() => _load(id)));
    } else {
      _questions.addAll(List.generate(quizMinQuestions, (_) => _Draft()));
    }
  }

  Future<void> _load(String id) async {
    try {
      final q = await ref.read(creatorRepositoryProvider).quiz(id);
      if (!mounted) {
        return;
      }
      setState(() {
        _quiz = q;
        _title.text = q.title;
        _description.text = q.description ?? '';
        _locale = q.locale;
        _category = q.categoryId;
        _window = q.answerWindowMs;
        _questions.addAll(q.questions.map(_Draft.of));
        _loading = false;
      });
    } on Object catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(SnackBar(content: Text(playErrorText(context.l10n, e))));
        Navigator.of(context).maybePop();
      }
    }
  }

  @override
  void dispose() {
    _title.dispose();
    _description.dispose();
    for (final q in _questions) {
      q.dispose();
    }
    super.dispose();
  }

  QuizDraft? _draft() {
    final title = _title.text.trim();
    final ok =
        title.length >= 3 &&
        title.length <= 60 &&
        _category != null &&
        _questions.every(
          (q) =>
              q.prompt.text.trim().isNotEmpty &&
              q.options.every((o) => o.text.trim().isNotEmpty),
        );
    if (!ok) {
      return null;
    }
    return QuizDraft(
      title: title,
      description: _description.text.trim(),
      locale: _locale ?? 'tr',
      categoryId: _category!,
      answerWindowMs: _window,
      questions: [
        for (final q in _questions)
          QuizQuestion(
            prompt: q.prompt.text.trim(),
            options: [for (final o in q.options) o.text.trim()],
            correct: q.correct,
          ),
      ],
    );
  }

  Future<void> _save() async {
    final l10n = context.l10n;
    final messenger = ScaffoldMessenger.of(context);
    final draft = _draft();
    if (draft == null) {
      messenger.showSnackBar(SnackBar(content: Text(l10n.quizInvalid)));
      return;
    }
    setState(() => _saving = true);
    final repo = ref.read(creatorRepositoryProvider);
    try {
      if (_quiz case final q?) {
        await repo.update(q.id, q.version, draft);
      } else {
        await repo.create(draft);
      }
      messenger.showSnackBar(SnackBar(content: Text(l10n.quizSaved)));
      if (mounted) {
        await Navigator.of(context).maybePop();
      }
    } on Object catch (e) {
      messenger.showSnackBar(SnackBar(content: Text(playErrorText(l10n, e))));
    } finally {
      if (mounted) {
        setState(() => _saving = false);
      }
    }
  }

  void _setTrueFalse(_Draft q, bool tf) {
    setState(() {
      for (final o in q.options) {
        o.dispose();
      }
      final l10n = context.l10n;
      q
        ..options = tf
            ? [
                TextEditingController(text: l10n.quizTrue),
                TextEditingController(text: l10n.quizFalse),
              ]
            : List.generate(4, (_) => TextEditingController())
        ..correct = 1;
    });
  }

  @override
  Widget build(BuildContext context) {
    final l10n = context.l10n;
    final categories = ref.watch(categoriesProvider).value ?? const [];
    _locale ??= Localizations.localeOf(context).languageCode == 'en'
        ? 'en'
        : 'tr';
    return Scaffold(
      appBar: AppBar(
        title: Text(_quiz == null ? l10n.quizNew : l10n.quizEdit),
        actions: [
          IconButton(
            key: const Key('quiz-save'),
            tooltip: l10n.quizSave,
            icon: const Icon(Icons.save),
            onPressed: _saving || _loading ? null : _save,
          ),
        ],
      ),
      body: _loading
          ? const Center(child: CircularProgressIndicator())
          : ListView(
              padding: const EdgeInsets.all(16),
              children: [
                TextField(
                  key: const Key('quiz-title'),
                  controller: _title,
                  maxLength: 60,
                  decoration: InputDecoration(labelText: l10n.quizTitleField),
                ),
                TextField(
                  key: const Key('quiz-description'),
                  controller: _description,
                  maxLength: 300,
                  maxLines: 2,
                  decoration: InputDecoration(
                    labelText: l10n.quizDescriptionField,
                  ),
                ),
                const SizedBox(height: 8),
                Text(l10n.quizLanguage),
                const SizedBox(height: 4),
                SegmentedButton<String>(
                  segments: const [
                    ButtonSegment(value: 'tr', label: Text('Türkçe')),
                    ButtonSegment(value: 'en', label: Text('English')),
                  ],
                  selected: {_locale ?? 'tr'},
                  // The language is fixed once the quiz exists.
                  onSelectionChanged: _quiz != null
                      ? null
                      : (s) => setState(() => _locale = s.first),
                ),
                const SizedBox(height: 16),
                DropdownButtonFormField<String>(
                  key: const Key('quiz-category'),
                  initialValue: categories.any((c) => c.id == _category)
                      ? _category
                      : null,
                  decoration: InputDecoration(labelText: l10n.quizCategory),
                  items: [
                    for (final c in categories)
                      DropdownMenuItem(value: c.id, child: Text(c.name)),
                  ],
                  onChanged: (v) => setState(() => _category = v),
                ),
                const SizedBox(height: 16),
                Text(l10n.quizWindow),
                const SizedBox(height: 4),
                SegmentedButton<int>(
                  segments: [
                    for (final w in quizWindows)
                      ButtonSegment(
                        value: w,
                        label: Text(l10n.quizSeconds(w ~/ 1000)),
                      ),
                  ],
                  selected: {_window},
                  onSelectionChanged: (s) => setState(() => _window = s.first),
                ),
                const SizedBox(height: 16),
                Text(
                  '${l10n.quizCorrectHint} '
                  '${l10n.quizCountRule(quizMinQuestions, quizMaxQuestions)}',
                  style: Theme.of(context).textTheme.bodySmall,
                ),
                for (final (i, q) in _questions.indexed) _question(i, q),
                const SizedBox(height: 8),
                OutlinedButton.icon(
                  key: const Key('quiz-add-question'),
                  icon: const Icon(Icons.add),
                  label: Text(l10n.quizAddQuestion),
                  onPressed: _questions.length >= quizMaxQuestions
                      ? null
                      : () => setState(() => _questions.add(_Draft())),
                ),
              ],
            ),
    );
  }

  Widget _question(int i, _Draft q) {
    final l10n = context.l10n;
    return Card(
      key: Key('quiz-q-$i'),
      margin: const EdgeInsets.only(top: 12),
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Row(
              children: [
                Expanded(
                  child: Text(
                    l10n.quizQuestionN(i + 1),
                    style: Theme.of(context).textTheme.titleSmall,
                  ),
                ),
                Text(l10n.quizTrueFalse),
                Switch(
                  key: Key('quiz-q-$i-tf'),
                  value: q.trueFalse,
                  onChanged: (v) => _setTrueFalse(q, v),
                ),
                IconButton(
                  key: Key('quiz-q-$i-remove'),
                  tooltip: l10n.quizRemoveQuestion,
                  icon: const Icon(Icons.delete_outline),
                  onPressed: _questions.length <= quizMinQuestions
                      ? null
                      : () => setState(() => _questions.removeAt(i).dispose()),
                ),
              ],
            ),
            TextField(
              key: Key('quiz-q-$i-prompt'),
              controller: q.prompt,
              maxLines: null,
              decoration: InputDecoration(labelText: l10n.quizPrompt),
            ),
            RadioGroup<int>(
              groupValue: q.correct,
              onChanged: (v) => setState(() => q.correct = v ?? 1),
              child: Column(
                children: [
                  for (final (j, o) in q.options.indexed)
                    Row(
                      children: [
                        Radio<int>(
                          key: Key('quiz-q-$i-correct-$j'),
                          value: j + 1,
                        ),
                        Expanded(
                          child: TextField(
                            key: Key('quiz-q-$i-option-$j'),
                            controller: o,
                            decoration: InputDecoration(
                              labelText: l10n.quizOptionN(j + 1),
                            ),
                          ),
                        ),
                      ],
                    ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}

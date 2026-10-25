import 'package:brainvolt/core/l10n/l10n.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('Turkish upper case keeps the dotted İ and dotless I', () {
    expect(upperFor('tr', 'Gizlilik'), 'GİZLİLİK');
    expect(upperFor('tr', 'Işık ve dil'), 'IŞIK VE DİL');
    expect(upperFor('tr', 'Oyun'), 'OYUN');
    expect(upperFor('en', 'Privacy'), 'PRIVACY');
  });
}

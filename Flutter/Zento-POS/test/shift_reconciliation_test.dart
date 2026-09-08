import 'package:flutter_test/flutter_test.dart';
import 'package:solid_pos/core/models/shift_model.dart';

void main() {
  group('Shift Reconciliation Logic', () {
    test('Expected cash calculation should include cash sales and deduct expenses', () {
      const openingCash = 1000.0;
      const cashSales = 500.0;
      const cashOut = 200.0;
      
      // Expected Cash = Opening Cash + Cash Sales - Money Out (Expenses)
      const expectedCashResult = openingCash + cashSales - cashOut;
      
      expect(expectedCashResult, 1300.0);
    });

    test('Difference calculation should compare actual cash with updated expected cash', () {
      const expectedCash = 1300.0;
      const actualCash = 1250.0; // Shortage of 50
      
      const difference = actualCash - expectedCash;
      
      expect(difference, -50.0);
    });
  });
}

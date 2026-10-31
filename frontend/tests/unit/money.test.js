import { test } from 'node:test';
import assert from 'node:assert/strict';
import { toKopecks, moneyInput, formatMoney } from '../../src/shared/utils/money.js';

test('money input converts decimal rubles to exact integer kopecks', () => {
  assert.equal(toKopecks('1.01'), 101);
  assert.equal(toKopecks('1000,09'), 100009);
  assert.equal(toKopecks('0'), 0);
  assert.equal(toKopecks('0.1'), 10);
  assert.equal(toKopecks('36000'), 3600000);
});
test('rejects negative, fractional kopecks, exponent and unsafe amounts', () => {
  for (const value of ['', '-1', '1.001', '1e3', 'Infinity', 'abc', '9007199254740992']) {
    assert.throws(() => toKopecks(value), undefined, value);
  }
});
test('retains kopecks in display and round trips inputs', () => {
  assert.equal(toKopecks(moneyInput(1234567)), 1234567);
  assert.equal(formatMoney(undefined), '—');
  assert.match(formatMoney(101), /1,01 ₽/);
});

import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readWithdrawal, saveWithdrawal } from '../../src/shared/utils/withdrawalAttempt.js';

test('retains the unresolved amount and ID separately for each client', () => {
  const data = new Map();
  globalThis.sessionStorage = { getItem: (key) => data.get(key) ?? null, setItem: (key, value) => data.set(key, value), removeItem: (key) => data.delete(key) };
  saveWithdrawal('anya', { id: 'w-1', amount: 101 });
  assert.deepEqual(readWithdrawal('anya'), { id: 'w-1', amount: 101 });
  assert.equal(readWithdrawal('other-client'), null);
  assert.equal(JSON.parse(data.get('zapas:withdrawal:anya')).id, 'w-1');
  saveWithdrawal('anya', null);
  assert.equal(readWithdrawal('anya'), null);
});
test('works without browser storage and ignores invalid persisted operations', () => {
  globalThis.sessionStorage = { getItem: () => '{"id":"bad","amount":-1}', setItem() { throw new Error('disabled'); }, removeItem() { throw new Error('disabled'); } };
  assert.equal(readWithdrawal('invalid'), null);
  saveWithdrawal('private', { id: 'w-private', amount: 10000 });
  assert.deepEqual(readWithdrawal('private'), { id: 'w-private', amount: 10000 });
  saveWithdrawal('private', null);
  assert.equal(readWithdrawal('private'), null);
});

import { test } from 'node:test';
import assert from 'node:assert/strict';
import { moscowDateTime, eventTimestamp } from '../../src/features/demo/time.js';

test('bank event times round trip between Moscow inputs and RFC3339 without relying on host timezone', () => {
  assert.equal(moscowDateTime('2026-10-20T07:00:00Z'), '2026-10-20T10:00:00');
  assert.equal(eventTimestamp('2026-10-20T10:00:00'), '2026-10-20T07:00:00.000Z');
  assert.equal(moscowDateTime('2026-10-20T22:30:01Z'), '2026-10-21T01:30:01');
  assert.equal(eventTimestamp('2026-10-21T01:30:01'), '2026-10-20T22:30:01.000Z');
  assert.throws(() => eventTimestamp(''));
});

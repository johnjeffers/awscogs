import assert from 'node:assert/strict';
import test from 'node:test';
import { sortData } from '../src/utils/sortData.ts';

const instances = [
  { name: 'single-a', multiAz: false },
  { name: 'multi', multiAz: true },
  { name: 'single-b', multiAz: false },
];

test('sorts boolean values in both directions', () => {
  const ascending = sortData(instances, { key: 'multiAz', direction: 'asc' });
  const descending = sortData(instances, { key: 'multiAz', direction: 'desc' });

  assert.deepEqual(
    ascending.map((instance) => instance.multiAz),
    [false, false, true],
  );
  assert.deepEqual(
    descending.map((instance) => instance.multiAz),
    [true, false, false],
  );
});

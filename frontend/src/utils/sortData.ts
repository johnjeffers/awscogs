export type SortDirection = 'asc' | 'desc';

export interface SortConfig {
  key: string;
  direction: SortDirection;
}

const NUMERIC_SORT_KEYS = new Set([
  'hourlyCost',
  'totalCost',
  'size',
  'desiredCount',
  'runningCount',
  'memorySize',
  'invocations',
  'averageDurationMs',
  'requestVolume',
  'bandwidthBytes',
]);

export function sortData<T>(data: T[], sortConfig: SortConfig): T[] {
  const isNumeric = NUMERIC_SORT_KEYS.has(sortConfig.key);
  return [...data].sort((a, b) => {
    let aVal = (a as Record<string, unknown>)[sortConfig.key];
    let bVal = (b as Record<string, unknown>)[sortConfig.key];

    // Treat missing numeric values as 0
    if (isNumeric) {
      if (aVal == null) aVal = 0;
      if (bVal == null) bVal = 0;
    } else {
      if (aVal == null && bVal == null) return 0;
      if (aVal == null) return 1;
      if (bVal == null) return -1;
    }
    if (Array.isArray(aVal)) aVal = aVal.join(' ');
    if (Array.isArray(bVal)) bVal = bVal.join(' ');

    let comparison = 0;
    if (typeof aVal === 'string' && typeof bVal === 'string') {
      comparison = aVal.localeCompare(bVal);
    } else if (typeof aVal === 'number' && typeof bVal === 'number') {
      comparison = aVal - bVal;
    } else if (typeof aVal === 'boolean' && typeof bVal === 'boolean') {
      comparison = Number(aVal) - Number(bVal);
    }

    return sortConfig.direction === 'asc' ? comparison : -comparison;
  });
}

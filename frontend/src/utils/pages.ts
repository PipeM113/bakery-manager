/** How many rows the API gives of a list that grows (quotations, sales): the most recent ones. */
export const PAGE_SIZE = 200;

/** A list as long as a page may be cut: there can be older rows the screen does not show. */
export const mayHaveMore = (list: readonly unknown[]): boolean => list.length >= PAGE_SIZE;

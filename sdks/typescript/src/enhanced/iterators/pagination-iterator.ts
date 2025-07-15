/**
 * Async iterator for paginated API responses
 */

export interface PaginatedResponse<T> {
  data: T[]
  meta: {
    hasMore: boolean
    nextCursor?: string
    totalCount?: number
  }
}

export interface PaginationOptions {
  limit?: number
  cursor?: string
}

export class PaginationIterator<T> {
  private cursor?: string
  private hasMore = true

  constructor(
    private fetchPage: (options: PaginationOptions) => Promise<PaginatedResponse<T>>,
    private options: PaginationOptions = {}
  ) {
    this.cursor = options.cursor
  }

  async *[Symbol.asyncIterator](): AsyncIterableIterator<T> {
    while (this.hasMore) {
      const response = await this.fetchPage({
        ...this.options,
        cursor: this.cursor
      })

      for (const item of response.data) {
        yield item
      }

      this.hasMore = response.meta.hasMore
      this.cursor = response.meta.nextCursor
    }
  }

  /**
   * Collect all items into an array
   */
  async toArray(): Promise<T[]> {
    const items: T[] = []
    for await (const item of this) {
      items.push(item)
    }
    return items
  }

  /**
   * Find the first item matching predicate
   */
  async find(predicate: (item: T) => boolean): Promise<T | undefined> {
    for await (const item of this) {
      if (predicate(item)) {
        return item
      }
    }
    return undefined
  }

  /**
   * Filter items using predicate
   */
  async *filter(predicate: (item: T) => boolean): AsyncIterableIterator<T> {
    for await (const item of this) {
      if (predicate(item)) {
        yield item
      }
    }
  }

  /**
   * Map items to a new type
   */
  async *map<U>(mapper: (item: T) => U): AsyncIterableIterator<U> {
    for await (const item of this) {
      yield mapper(item)
    }
  }
}

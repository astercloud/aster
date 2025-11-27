/**
 * 性能优化工具函数
 */

/**
 * 防抖函数
 * 在事件被触发n秒后再执行回调，如果在这n秒内又被触发，则重新计时
 */
export function debounce<T extends (...args: any[]) => any>(
  func: T,
  wait: number = 300
): (...args: Parameters<T>) => void {
  let timeoutId: ReturnType<typeof setTimeout> | null = null;

  return function (this: any, ...args: Parameters<T>) {
    const context = this;

    if (timeoutId !== null) {
      clearTimeout(timeoutId);
    }

    timeoutId = setTimeout(() => {
      func.apply(context, args);
      timeoutId = null;
    }, wait);
  };
}

/**
 * 节流函数
 * 规定在一个单位时间内，只能触发一次函数
 */
export function throttle<T extends (...args: any[]) => any>(
  func: T,
  wait: number = 300
): (...args: Parameters<T>) => void {
  let lastTime = 0;
  let timeoutId: ReturnType<typeof setTimeout> | null = null;

  return function (this: any, ...args: Parameters<T>) {
    const context = this;
    const now = Date.now();

    // 如果距离上次执行的时间超过了wait，则立即执行
    if (now - lastTime >= wait) {
      if (timeoutId !== null) {
        clearTimeout(timeoutId);
        timeoutId = null;
      }
      func.apply(context, args);
      lastTime = now;
    } else if (timeoutId === null) {
      // 否则设置定时器，在wait时间后执行
      timeoutId = setTimeout(() => {
        func.apply(context, args);
        lastTime = Date.now();
        timeoutId = null;
      }, wait - (now - lastTime));
    }
  };
}

/**
 * RAF 节流 - 使用 requestAnimationFrame 实现的节流
 * 适用于DOM操作和动画
 */
export function rafThrottle<T extends (...args: any[]) => any>(
  func: T
): (...args: Parameters<T>) => void {
  let rafId: number | null = null;

  return function (this: any, ...args: Parameters<T>) {
    const context = this;

    if (rafId !== null) {
      return;
    }

    rafId = requestAnimationFrame(() => {
      func.apply(context, args);
      rafId = null;
    });
  };
}

/**
 * 延迟执行 - Promise 版本的 setTimeout
 */
export function delay(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

/**
 * 分批处理大量数据
 * 将大数组分批处理，每批之间使用 setTimeout 让出主线程
 */
export async function processBatch<T, R>(
  items: T[],
  processor: (item: T, index: number) => R,
  batchSize: number = 100,
  delayMs: number = 0
): Promise<R[]> {
  const results: R[] = [];

  for (let i = 0; i < items.length; i += batchSize) {
    const batch = items.slice(i, i + batchSize);

    for (let j = 0; j < batch.length; j++) {
      results.push(processor(batch[j], i + j));
    }

    // 让出主线程
    if (delayMs > 0 && i + batchSize < items.length) {
      await delay(delayMs);
    }
  }

  return results;
}

/**
 * 空闲时执行 - 使用 requestIdleCallback 或降级到 setTimeout
 */
export function runWhenIdle(callback: () => void, timeout: number = 1000): number {
  if (typeof requestIdleCallback === 'function') {
    return requestIdleCallback(callback, { timeout });
  }

  // 降级方案
  return setTimeout(callback, 100) as unknown as number;
}

/**
 * 取消空闲回调
 */
export function cancelIdleCallback(id: number): void {
  if (typeof cancelIdleCallback === 'function') {
    window.cancelIdleCallback(id);
  } else {
    clearTimeout(id);
  }
}

/**
 * 内存化 - 缓存函数结果
 */
export function memoize<T extends (...args: any[]) => any>(
  func: T,
  resolver?: (...args: Parameters<T>) => string
): T {
  const cache = new Map<string, ReturnType<T>>();

  return function (this: any, ...args: Parameters<T>): ReturnType<T> {
    const key = resolver ? resolver(...args) : JSON.stringify(args);

    if (cache.has(key)) {
      return cache.get(key)!;
    }

    const result = func.apply(this, args);
    cache.set(key, result);
    return result;
  } as T;
}

/**
 * Aster UI Protocol - JSON Pointer 路径解析器
 *
 * 实现 RFC 6901 JSON Pointer 规范，支持数据绑定的路径解析。
 *
 * @module protocol/path-resolver
 */

import type { DataValue, DataMap } from '@/types/ui-protocol';

/**
 * 路径解析错误代码
 */
export const PathErrorCodes = {
  INVALID_PATH: 'INVALID_PATH',
  PATH_NOT_FOUND: 'PATH_NOT_FOUND',
  INVALID_ARRAY_INDEX: 'INVALID_ARRAY_INDEX',
  CANNOT_SET_VALUE: 'CANNOT_SET_VALUE',
} as const;

export type PathErrorCode = typeof PathErrorCodes[keyof typeof PathErrorCodes];

/**
 * 路径解析错误
 */
export class PathError extends Error {
  constructor(
    message: string,
    public code: PathErrorCode,
    public details?: Record<string, unknown>,
  ) {
    super(message);
    this.name = 'PathError';
  }
}

/**
 * 验证 JSON Pointer 路径是否有效
 *
 * 根据 RFC 6901，有效的 JSON Pointer 必须：
 * - 为空字符串（表示整个文档）
 * - 或以 "/" 开头
 *
 * @param path - 要验证的路径
 * @returns 是否为有效的 JSON Pointer
 */
export function isValidJsonPointer(path: string): boolean {
  if (typeof path !== 'string') {
    return false;
  }
  // 空字符串表示整个文档
  if (path === '') {
    return true;
  }
  // 必须以 "/" 开头
  return path.startsWith('/');
}

/**
 * 解码 JSON Pointer 转义字符
 *
 * 根据 RFC 6901：
 * - ~1 -> /
 * - ~0 -> ~
 *
 * @param token - 要解码的路径段
 * @returns 解码后的路径段
 */
export function decodeJsonPointerToken(token: string): string {
  return token.replace(/~1/g, '/').replace(/~0/g, '~');
}

/**
 * 编码 JSON Pointer 转义字符
 *
 * 根据 RFC 6901：
 * - ~ -> ~0
 * - / -> ~1
 *
 * @param token - 要编码的路径段
 * @returns 编码后的路径段
 */
export function encodeJsonPointerToken(token: string): string {
  return token.replace(/~/g, '~0').replace(/\//g, '~1');
}

/**
 * 解析 JSON Pointer 路径为路径段数组
 *
 * @param path - JSON Pointer 路径
 * @returns 路径段数组
 * @throws {PathError} 如果路径无效
 */
export function parseJsonPointer(path: string): string[] {
  if (!isValidJsonPointer(path)) {
    throw new PathError(
      `Invalid JSON Pointer: "${path}". Path must be empty or start with "/"`,
      PathErrorCodes.INVALID_PATH,
      { path },
    );
  }

  // 空字符串表示根路径
  if (path === '') {
    return [];
  }

  // 移除开头的 "/" 并分割
  const tokens = path.slice(1).split('/');

  // 解码每个路径段
  return tokens.map(decodeJsonPointerToken);
}

/**
 * 将路径段数组转换为 JSON Pointer 路径
 *
 * @param tokens - 路径段数组
 * @returns JSON Pointer 路径
 */
export function toJsonPointer(tokens: string[]): string {
  if (tokens.length === 0) {
    return '';
  }
  return '/' + tokens.map(encodeJsonPointerToken).join('/');
}

/**
 * 解析相对路径
 *
 * 支持以下格式：
 * - 绝对路径：以 "/" 开头，如 "/user/name"
 * - 相对路径：不以 "/" 开头，相对于 contextPath
 *
 * @param path - 要解析的路径
 * @param contextPath - 上下文路径（用于相对路径解析）
 * @returns 解析后的绝对路径
 */
export function resolvePath(path: string, contextPath?: string): string {
  if (typeof path !== 'string') {
    throw new PathError(
      'Path must be a string',
      PathErrorCodes.INVALID_PATH,
      { path },
    );
  }

  // 空路径表示根
  if (path === '') {
    return '';
  }

  // 绝对路径直接返回
  if (path.startsWith('/')) {
    return path;
  }

  // 相对路径需要上下文
  if (!contextPath) {
    // 没有上下文时，将相对路径转换为绝对路径
    return '/' + path;
  }

  // 解析上下文路径
  const contextTokens = parseJsonPointer(contextPath);

  // 解析相对路径段
  const relativeTokens = path.split('/');

  // 合并路径
  const resultTokens = [...contextTokens, ...relativeTokens];

  return toJsonPointer(resultTokens);
}

/**
 * 从数据模型中获取指定路径的数据
 *
 * @param data - 数据模型
 * @param path - JSON Pointer 路径
 * @returns 路径对应的数据值，如果路径不存在则返回 null
 */
export function getData(data: DataValue, path: string): DataValue | null {
  // 验证路径
  if (!isValidJsonPointer(path)) {
    return null;
  }

  // 空路径返回整个数据
  if (path === '') {
    return data;
  }

  // 解析路径
  const tokens = parseJsonPointer(path);

  // 遍历路径
  let current: DataValue | undefined = data;

  for (const token of tokens) {
    if (current === null || current === undefined) {
      return null;
    }

    // 处理数组
    if (Array.isArray(current)) {
      // 检查是否为有效的数组索引
      const index = parseInt(token, 10);
      if (isNaN(index) || index < 0 || index >= current.length) {
        return null;
      }
      current = current[index] as DataValue | undefined;
    }
    // 处理对象
    else if (typeof current === 'object') {
      const obj = current as DataMap;
      if (!(token in obj)) {
        return null;
      }
      current = obj[token];
    }
    // 基本类型无法继续遍历
    else {
      return null;
    }
  }

  return current ?? null;
}

/**
 * 在数据模型中设置指定路径的数据
 *
 * @param data - 数据模型（会被修改）
 * @param path - JSON Pointer 路径
 * @param value - 要设置的值
 * @returns 是否设置成功
 */
export function setData(data: DataMap, path: string, value: DataValue): boolean {
  // 验证路径
  if (!isValidJsonPointer(path)) {
    return false;
  }

  // 空路径无法设置（需要替换整个对象）
  if (path === '') {
    return false;
  }

  // 解析路径
  const tokens = parseJsonPointer(path);

  if (tokens.length === 0) {
    return false;
  }

  // 遍历到父节点
  let current: DataValue | undefined = data;
  const parentTokens = tokens.slice(0, -1);
  const lastToken = tokens[tokens.length - 1]!;

  for (const token of parentTokens) {
    if (current === null || current === undefined) {
      return false;
    }

    // 处理数组
    if (Array.isArray(current)) {
      const index = parseInt(token, 10);
      if (isNaN(index) || index < 0 || index >= current.length) {
        return false;
      }
      current = current[index] as DataValue | undefined;
    }
    // 处理对象
    else if (typeof current === 'object') {
      const obj = current as DataMap;
      if (!(token in obj)) {
        // 自动创建中间对象
        obj[token] = {};
      }
      current = obj[token];
    }
    // 基本类型无法继续遍历
    else {
      return false;
    }
  }

  // 设置值
  if (current === null || current === undefined) {
    return false;
  }

  // 处理数组
  if (Array.isArray(current)) {
    const index = parseInt(lastToken, 10);
    if (isNaN(index) || index < 0) {
      return false;
    }
    // 允许在数组末尾添加元素
    if (index > current.length) {
      return false;
    }
    current[index] = value;
    return true;
  }

  // 处理对象
  if (typeof current === 'object') {
    (current as DataMap)[lastToken] = value;
    return true;
  }

  return false;
}

/**
 * 检查路径是否存在于数据模型中
 *
 * @param data - 数据模型
 * @param path - JSON Pointer 路径
 * @returns 路径是否存在
 */
export function hasPath(data: DataValue, path: string): boolean {
  if (!isValidJsonPointer(path)) {
    return false;
  }

  if (path === '') {
    return true;
  }

  const tokens = parseJsonPointer(path);
  let current: DataValue | undefined = data;

  for (const token of tokens) {
    if (current === null || current === undefined) {
      return false;
    }

    if (Array.isArray(current)) {
      const index = parseInt(token, 10);
      if (isNaN(index) || index < 0 || index >= current.length) {
        return false;
      }
      current = current[index] as DataValue | undefined;
    }
    else if (typeof current === 'object') {
      const obj = current as DataMap;
      if (!(token in obj)) {
        return false;
      }
      current = obj[token];
    }
    else {
      return false;
    }
  }

  return true;
}

/**
 * 删除数据模型中指定路径的数据
 *
 * @param data - 数据模型（会被修改）
 * @param path - JSON Pointer 路径
 * @returns 是否删除成功
 */
export function deleteData(data: DataMap, path: string): boolean {
  if (!isValidJsonPointer(path) || path === '') {
    return false;
  }

  const tokens = parseJsonPointer(path);
  if (tokens.length === 0) {
    return false;
  }

  // 遍历到父节点
  let current: DataValue | undefined = data;
  const parentTokens = tokens.slice(0, -1);
  const lastToken = tokens[tokens.length - 1]!;

  for (const token of parentTokens) {
    if (current === null || current === undefined) {
      return false;
    }

    if (Array.isArray(current)) {
      const index = parseInt(token, 10);
      if (isNaN(index) || index < 0 || index >= current.length) {
        return false;
      }
      current = current[index] as DataValue | undefined;
    }
    else if (typeof current === 'object') {
      const obj = current as DataMap;
      if (!(token in obj)) {
        return false;
      }
      current = obj[token];
    }
    else {
      return false;
    }
  }

  // 删除值
  if (current === null || current === undefined) {
    return false;
  }

  if (Array.isArray(current)) {
    const index = parseInt(lastToken, 10);
    if (isNaN(index) || index < 0 || index >= current.length) {
      return false;
    }
    current.splice(index, 1);
    return true;
  }

  if (typeof current === 'object') {
    const obj = current as DataMap;
    if (!(lastToken in obj)) {
      return false;
    }
    delete obj[lastToken];
    return true;
  }

  return false;
}

/**
 * 在数据模型中添加数据（add 操作）
 *
 * 行为：
 * - 如果目标是数组：追加元素（value 为数组时追加所有元素）
 * - 如果目标是对象：合并属性（value 必须是对象）
 * - 如果路径不存在：创建路径并设置值
 *
 * @param data - 数据模型（会被修改）
 * @param path - JSON Pointer 路径
 * @param value - 要添加的值
 * @returns 是否添加成功
 */
export function addData(data: DataMap, path: string, value: DataValue): boolean {
  if (!isValidJsonPointer(path)) {
    return false;
  }

  // 空路径：合并到根对象
  if (path === '') {
    if (typeof value === 'object' && value !== null && !Array.isArray(value)) {
      Object.assign(data, value);
      return true;
    }
    return false;
  }

  // 获取当前值
  const current = getData(data, path);

  // 路径不存在：创建并设置
  if (current === null) {
    return setData(data, path, value);
  }

  // 目标是数组：追加元素
  if (Array.isArray(current)) {
    if (Array.isArray(value)) {
      current.push(...value);
    }
    else {
      current.push(value);
    }
    return true;
  }

  // 目标是对象：合并属性
  if (typeof current === 'object' && current !== null) {
    if (typeof value === 'object' && value !== null && !Array.isArray(value)) {
      Object.assign(current, value);
      return true;
    }
    return false;
  }

  // 基本类型：无法添加，改为替换
  return setData(data, path, value);
}

/**
 * 移除数据模型中指定路径的数据（remove 操作）
 *
 * 与 deleteData 相同，但提供更语义化的名称
 *
 * @param data - 数据模型（会被修改）
 * @param path - JSON Pointer 路径
 * @returns 是否移除成功
 */
export function removeData(data: DataMap, path: string): boolean {
  // 空路径：清空整个数据模型
  if (path === '' || path === '/') {
    const keys = Object.keys(data);
    for (const key of keys) {
      delete data[key];
    }
    return true;
  }

  return deleteData(data, path);
}

/**
 * 获取路径的父路径
 *
 * @param path - JSON Pointer 路径
 * @returns 父路径，如果是根路径则返回 null
 */
export function getParentPath(path: string): string | null {
  if (!isValidJsonPointer(path) || path === '') {
    return null;
  }

  const tokens = parseJsonPointer(path);
  if (tokens.length === 0) {
    return null;
  }

  return toJsonPointer(tokens.slice(0, -1));
}

/**
 * 获取路径的最后一个段
 *
 * @param path - JSON Pointer 路径
 * @returns 最后一个路径段，如果是根路径则返回 null
 */
export function getLastToken(path: string): string | null {
  if (!isValidJsonPointer(path) || path === '') {
    return null;
  }

  const tokens = parseJsonPointer(path);
  if (tokens.length === 0) {
    return null;
  }

  const lastToken = tokens[tokens.length - 1];
  return lastToken !== undefined ? lastToken : null;
}

/**
 * 合并两个路径
 *
 * @param basePath - 基础路径
 * @param relativePath - 相对路径
 * @returns 合并后的路径
 */
export function joinPaths(basePath: string, relativePath: string): string {
  if (!isValidJsonPointer(basePath)) {
    throw new PathError(
      `Invalid base path: "${basePath}"`,
      PathErrorCodes.INVALID_PATH,
      { basePath },
    );
  }

  // 如果相对路径是绝对路径，直接返回
  if (relativePath.startsWith('/')) {
    return relativePath;
  }

  const baseTokens = parseJsonPointer(basePath);
  const relativeTokens = relativePath.split('/').map(decodeJsonPointerToken);

  return toJsonPointer([...baseTokens, ...relativeTokens]);
}

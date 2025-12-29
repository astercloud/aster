/**
 * Aster UI Protocol - Client-to-Server Messages
 *
 * Handles client-side message creation for user interactions and errors.
 *
 * @module protocol/client-message
 */

import type {
  ClientMessage,
  UserActionMessage,
  ProtocolError,
  ValidationError,
  PropertyValue,
  DataMap,
} from '@/types/ui-protocol';
import { createValidationError, createGenericError } from '@/types/ui-protocol';
import { getData } from './path-resolver';

/**
 * Create a UserActionMessage for button clicks and other user interactions
 */
export function createUserAction(
  name: string,
  surfaceId: string,
  sourceComponentId: string,
  context: Record<string, unknown> = {},
): UserActionMessage {
  return {
    name,
    surfaceId,
    sourceComponentId,
    timestamp: new Date().toISOString(),
    context,
  };
}

/**
 * Create a ClientMessage containing a user action
 */
export function createUserActionMessage(
  name: string,
  surfaceId: string,
  sourceComponentId: string,
  context: Record<string, unknown> = {},
): ClientMessage {
  return {
    userAction: createUserAction(name, surfaceId, sourceComponentId, context),
  };
}

/**
 * Create a ClientMessage containing an error
 */
export function createErrorMessage(error: ProtocolError): ClientMessage {
  return {
    error,
  };
}

/**
 * Create a ClientMessage containing a validation error
 */
export function createValidationErrorMessage(
  surfaceId: string,
  path: string,
  message: string,
): ClientMessage {
  return {
    error: createValidationError(surfaceId, path, message),
  };
}

/**
 * Create a ClientMessage containing a generic error
 */
export function createGenericErrorMessage(
  code: string,
  surfaceId: string,
  message: string,
  details?: Record<string, unknown>,
): ClientMessage {
  return {
    error: createGenericError(code, surfaceId, message, details),
  };
}

/**
 * Resolve action context by replacing path references with actual values
 */
export function resolveActionContext(
  actionContext: Record<string, PropertyValue>,
  dataModel: DataMap,
): Record<string, unknown> {
  const resolved: Record<string, unknown> = {};

  for (const [key, value] of Object.entries(actionContext)) {
    resolved[key] = resolvePropertyValue(value, dataModel);
  }

  return resolved;
}

/**
 * Resolve a single PropertyValue to its actual value
 */
function resolvePropertyValue(value: PropertyValue, dataModel: DataMap): unknown {
  if ('literalString' in value) {
    return value.literalString;
  }
  if ('literalNumber' in value) {
    return value.literalNumber;
  }
  if ('literalBoolean' in value) {
    return value.literalBoolean;
  }
  if ('path' in value) {
    return getData(dataModel, value.path) ?? undefined;
  }
  return undefined;
}

/**
 * Serialize a ClientMessage to JSON string
 */
export function serializeClientMessage(message: ClientMessage): string {
  return JSON.stringify(message);
}

/**
 * Parse a JSON string to ClientMessage
 */
export function parseClientMessage(json: string): ClientMessage | null {
  try {
    return JSON.parse(json) as ClientMessage;
  }
  catch {
    return null;
  }
}

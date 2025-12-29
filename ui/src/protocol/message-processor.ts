/**
 * Aster UI Protocol - Message Processor
 *
 * Processes UI protocol messages and manages Surface state.
 *
 * @module protocol/message-processor
 */

import type {
  AsterUIMessage,
  CreateSurfaceMessage,
  SurfaceUpdateMessage,
  DataModelUpdateMessage,
  BeginRenderingMessage,
  DeleteSurfaceMessage,
  ComponentDefinition,
  Surface,
  DataValue,
  DataMap,
  AnyComponentNode,
  ComponentArrayReference,
  PropertyValue,
  ValidationError,
} from '@/types/ui-protocol';
import { getComponentTypeName, getComponentProps, isPathReference, createValidationError } from '@/types/ui-protocol';
import { getData, setData, addData, removeData } from './path-resolver';
import { getDefaultRegistry } from './standard-components';
import type { ComponentRegistry } from './registry';

/**
 * Message processor error codes
 */
export const ProcessorErrorCodes = {
  INVALID_MESSAGE: 'INVALID_MESSAGE',
  UNKNOWN_COMPONENT: 'UNKNOWN_COMPONENT',
  CIRCULAR_REFERENCE: 'CIRCULAR_REFERENCE',
  INVALID_SURFACE: 'INVALID_SURFACE',
} as const;

/**
 * Streaming rendering state
 * Tracks state that should be preserved during incremental updates
 */
export interface StreamingState {
  /** Whether streaming mode is active (beginRendering called before all components defined) */
  isStreaming: boolean;
  /** Component IDs that have been rendered */
  renderedComponentIds: Set<string>;
  /** Pending component IDs waiting to be rendered */
  pendingComponentIds: Set<string>;
}

export type ProcessorErrorCode = typeof ProcessorErrorCodes[keyof typeof ProcessorErrorCodes];

/**
 * Message processor error
 */
export class ProcessorError extends Error {
  constructor(
    message: string,
    public code: ProcessorErrorCode,
    public details?: Record<string, unknown>,
  ) {
    super(message);
    this.name = 'ProcessorError';
  }
}

/**
 * Create an empty Surface
 */
function createEmptySurface(catalogId?: string): Surface {
  return {
    rootComponentId: null,
    componentTree: null,
    dataModel: {},
    components: new Map(),
    styles: {},
    catalogId,
  };
}

/**
 * Create initial streaming state
 */
function createStreamingState(): StreamingState {
  return {
    isStreaming: false,
    renderedComponentIds: new Set(),
    pendingComponentIds: new Set(),
  };
}

/**
 * Message Processor
 *
 * Processes AsterUIMessage and manages Surface state.
 * Handles five message types: createSurface, surfaceUpdate, dataModelUpdate, beginRendering, deleteSurface.
 * Supports streaming rendering with incremental component updates.
 */
export class MessageProcessor {
  /** Surface map (surfaceId -> Surface) */
  private surfaces: Map<string, Surface> = new Map();

  /** Streaming state map (surfaceId -> StreamingState) */
  private streamingStates: Map<string, StreamingState> = new Map();

  /** Component registry for validation */
  private registry: ComponentRegistry;

  /** Event listeners for surface changes */
  private listeners: Map<string, Set<(surface: Surface) => void>> = new Map();

  /** Validation errors */
  private validationErrors: ValidationError[] = [];

  constructor(registry?: ComponentRegistry) {
    this.registry = registry ?? getDefaultRegistry();
  }

  /**
   * Get all surfaces
   */
  getSurfaces(): ReadonlyMap<string, Surface> {
    return this.surfaces;
  }

  /**
   * Get a specific surface
   */
  getSurface(surfaceId: string): Surface | undefined {
    return this.surfaces.get(surfaceId);
  }

  /**
   * Get streaming state for a surface
   */
  getStreamingState(surfaceId: string): StreamingState | undefined {
    return this.streamingStates.get(surfaceId);
  }

  /**
   * Check if a surface is in streaming mode
   */
  isStreaming(surfaceId: string): boolean {
    return this.streamingStates.get(surfaceId)?.isStreaming ?? false;
  }

  /**
   * Clear all surfaces
   */
  clearSurfaces(): void {
    this.surfaces.clear();
    this.streamingStates.clear();
    this.validationErrors = [];
  }

  /**
   * Get validation errors
   */
  getValidationErrors(): readonly ValidationError[] {
    return this.validationErrors;
  }

  /**
   * Clear validation errors
   */
  clearValidationErrors(): void {
    this.validationErrors = [];
  }

  /**
   * Validate a message and return errors if any
   */
  validateMessage(message: AsterUIMessage): ValidationError[] {
    const errors: ValidationError[] = [];

    // Check that exactly one message type is present
    const messageTypes = [
      message.createSurface,
      message.surfaceUpdate,
      message.dataModelUpdate,
      message.beginRendering,
      message.deleteSurface,
    ].filter(Boolean);

    if (messageTypes.length === 0) {
      errors.push(createValidationError('', '/', 'Message must contain exactly one operation type'));
    }
    else if (messageTypes.length > 1) {
      errors.push(createValidationError('', '/', 'Message must contain exactly one operation type'));
    }

    // Validate createSurface
    if (message.createSurface) {
      if (!message.createSurface.surfaceId) {
        errors.push(createValidationError('', '/createSurface/surfaceId', 'surfaceId is required'));
      }
    }

    // Validate surfaceUpdate
    if (message.surfaceUpdate) {
      if (!message.surfaceUpdate.surfaceId) {
        errors.push(createValidationError('', '/surfaceUpdate/surfaceId', 'surfaceId is required'));
      }
      if (!Array.isArray(message.surfaceUpdate.components)) {
        errors.push(createValidationError(
          message.surfaceUpdate.surfaceId || '',
          '/surfaceUpdate/components',
          'components must be an array',
        ));
      }
    }

    // Validate dataModelUpdate
    if (message.dataModelUpdate) {
      if (!message.dataModelUpdate.surfaceId) {
        errors.push(createValidationError('', '/dataModelUpdate/surfaceId', 'surfaceId is required'));
      }
      const op = message.dataModelUpdate.op;
      if (op && !['add', 'replace', 'remove'].includes(op)) {
        errors.push(createValidationError(
          message.dataModelUpdate.surfaceId || '',
          '/dataModelUpdate/op',
          'op must be "add", "replace", or "remove"',
        ));
      }
      if (op !== 'remove' && message.dataModelUpdate.contents === undefined) {
        errors.push(createValidationError(
          message.dataModelUpdate.surfaceId || '',
          '/dataModelUpdate/contents',
          'contents is required for add and replace operations',
        ));
      }
    }

    // Validate beginRendering
    if (message.beginRendering) {
      if (!message.beginRendering.surfaceId) {
        errors.push(createValidationError('', '/beginRendering/surfaceId', 'surfaceId is required'));
      }
      if (!message.beginRendering.root) {
        errors.push(createValidationError(
          message.beginRendering.surfaceId || '',
          '/beginRendering/root',
          'root is required',
        ));
      }
    }

    // Validate deleteSurface
    if (message.deleteSurface) {
      if (!message.deleteSurface.surfaceId) {
        errors.push(createValidationError('', '/deleteSurface/surfaceId', 'surfaceId is required'));
      }
    }

    return errors;
  }

  /**
   * Process a batch of messages
   */
  processMessages(messages: AsterUIMessage[]): void {
    for (const message of messages) {
      this.processMessage(message);
    }
  }

  /**
   * Process a single message
   */
  processMessage(message: AsterUIMessage): void {
    // Validate message first
    const errors = this.validateMessage(message);
    if (errors.length > 0) {
      this.validationErrors.push(...errors);
      console.warn('Message validation failed:', errors);
      return;
    }

    if (message.createSurface) {
      this.processCreateSurface(message.createSurface);
    }
    if (message.surfaceUpdate) {
      this.processSurfaceUpdate(message.surfaceUpdate);
    }
    if (message.dataModelUpdate) {
      this.processDataModelUpdate(message.dataModelUpdate);
    }
    if (message.beginRendering) {
      this.processBeginRendering(message.beginRendering);
    }
    if (message.deleteSurface) {
      this.processDeleteSurface(message.deleteSurface);
    }
  }

  /**
   * Process createSurface message
   * Creates a new surface with optional catalogId
   */
  private processCreateSurface(message: CreateSurfaceMessage): void {
    const { surfaceId, catalogId } = message;

    // Create new surface (or reset existing)
    const surface = createEmptySurface(catalogId);
    this.surfaces.set(surfaceId, surface);

    // Create streaming state
    const streamingState = createStreamingState();
    this.streamingStates.set(surfaceId, streamingState);

    this.notifyListeners(surfaceId, surface);
  }

  /**
   * Process surfaceUpdate message
   * Updates component definitions for a surface
   * Supports incremental component addition for streaming rendering
   */
  private processSurfaceUpdate(message: SurfaceUpdateMessage): void {
    const { surfaceId, components } = message;

    // Get or create surface
    let surface = this.surfaces.get(surfaceId);
    if (!surface) {
      surface = createEmptySurface();
      this.surfaces.set(surfaceId, surface);
    }

    // Get or create streaming state
    let streamingState = this.streamingStates.get(surfaceId);
    if (!streamingState) {
      streamingState = createStreamingState();
      this.streamingStates.set(surfaceId, streamingState);
    }

    // Merge components by ID (incremental update)
    for (const component of components) {
      // Validate component type
      const typeName = getComponentTypeName(component.component);
      if (!this.registry.has(typeName) && typeName !== 'Custom') {
        console.warn(`Unknown component type: ${typeName}. Skipping component ${component.id}.`);
        continue;
      }

      // Track if this is a new component
      const isNew = !surface.components.has(component.id);
      surface.components.set(component.id, component);

      if (isNew) {
        // Remove from pending if it was waiting
        streamingState.pendingComponentIds.delete(component.id);
      }
    }

    // Rebuild component tree if root is set
    if (surface.rootComponentId) {
      this.rebuildComponentTree(surface, streamingState);
    }

    this.notifyListeners(surfaceId, surface);
  }

  /**
   * Process dataModelUpdate message
   * Updates data model and triggers reactive UI updates
   * Supports add/replace/remove operations
   */
  private processDataModelUpdate(message: DataModelUpdateMessage): void {
    const { surfaceId, path, op, contents } = message;

    // Get or create surface
    let surface = this.surfaces.get(surfaceId);
    if (!surface) {
      surface = createEmptySurface();
      this.surfaces.set(surfaceId, surface);
    }

    // Get streaming state
    const streamingState = this.streamingStates.get(surfaceId);

    // Determine operation (default to replace)
    const operation = op ?? 'replace';
    const targetPath = path ?? '/';

    // Execute operation
    switch (operation) {
      case 'add':
        if (contents !== undefined) {
          if (targetPath === '/' || targetPath === '') {
            // Add to root: merge if object
            if (typeof contents === 'object' && contents !== null && !Array.isArray(contents)) {
              Object.assign(surface.dataModel, contents);
            }
          }
          else {
            addData(surface.dataModel, targetPath, contents);
          }
        }
        break;

      case 'remove':
        if (targetPath === '/' || targetPath === '') {
          // Clear entire data model
          surface.dataModel = {};
        }
        else {
          removeData(surface.dataModel, targetPath);
        }
        break;

      case 'replace':
      default:
        if (targetPath === '/' || targetPath === '') {
          // Replace entire data model
          if (typeof contents === 'object' && contents !== null && !Array.isArray(contents)) {
            surface.dataModel = contents as DataMap;
          }
        }
        else if (contents !== undefined) {
          setData(surface.dataModel, targetPath, contents);
        }
        break;
    }

    // Rebuild component tree to reflect data changes
    if (surface.rootComponentId) {
      this.rebuildComponentTree(surface, streamingState);
    }

    this.notifyListeners(surfaceId, surface);
  }

  /**
   * Process beginRendering message
   * Starts rendering with specified root component
   * Supports streaming mode and catalogId
   */
  private processBeginRendering(message: BeginRenderingMessage): void {
    const { surfaceId, root, styles, catalogId } = message;

    // Get or create surface
    let surface = this.surfaces.get(surfaceId);
    if (!surface) {
      surface = createEmptySurface(catalogId);
      this.surfaces.set(surfaceId, surface);
    }

    // Get or create streaming state
    let streamingState = this.streamingStates.get(surfaceId);
    if (!streamingState) {
      streamingState = createStreamingState();
      this.streamingStates.set(surfaceId, streamingState);
    }

    // Set root component
    surface.rootComponentId = root;

    // Set styles
    if (styles) {
      surface.styles = { ...surface.styles, ...styles };
    }

    // Set catalogId (override if provided)
    if (catalogId) {
      surface.catalogId = catalogId;
    }

    // Check if root component exists - if not, we're in streaming mode
    if (!surface.components.has(root)) {
      streamingState.isStreaming = true;
      streamingState.pendingComponentIds.add(root);
      console.warn(`Root component ${root} not found. Entering streaming mode.`);
      this.notifyListeners(surfaceId, surface);
      return;
    }

    // Build component tree (may be partial in streaming mode)
    this.rebuildComponentTree(surface, streamingState);

    this.notifyListeners(surfaceId, surface);
  }

  /** Delete listeners for surface deletion notifications */
  private deleteListeners: Map<string, Set<(surfaceId: string) => void>> = new Map();

  /**
   * Process deleteSurface message
   * Removes surface and cleans up resources
   * Notifies delete listeners before cleanup
   */
  private processDeleteSurface(message: DeleteSurfaceMessage): void {
    const { surfaceId } = message;

    // Notify delete listeners before cleanup
    this.notifyDeleteListeners(surfaceId);

    // Remove surface
    this.surfaces.delete(surfaceId);

    // Remove streaming state
    this.streamingStates.delete(surfaceId);

    // Remove listeners
    this.listeners.delete(surfaceId);

    // Remove delete listeners
    this.deleteListeners.delete(surfaceId);
  }

  /**
   * Subscribe to surface deletion events
   * @param surfaceId - The surface ID to watch for deletion
   * @param listener - Callback when surface is deleted
   * @returns Unsubscribe function
   */
  subscribeToDelete(surfaceId: string, listener: (surfaceId: string) => void): () => void {
    let listeners = this.deleteListeners.get(surfaceId);
    if (!listeners) {
      listeners = new Set();
      this.deleteListeners.set(surfaceId, listeners);
    }
    listeners.add(listener);

    // Return unsubscribe function
    return () => {
      listeners?.delete(listener);
    };
  }

  /**
   * Notify delete listeners of surface deletion
   */
  private notifyDeleteListeners(surfaceId: string): void {
    const listeners = this.deleteListeners.get(surfaceId);
    if (listeners) {
      for (const listener of listeners) {
        listener(surfaceId);
      }
    }
  }

  /**
   * Check if a surface exists
   */
  hasSurface(surfaceId: string): boolean {
    return this.surfaces.has(surfaceId);
  }

  /**
   * Rebuild component tree from adjacency list model
   * Supports streaming mode where not all components may be defined yet
   */
  private rebuildComponentTree(surface: Surface, streamingState?: StreamingState): void {
    if (!surface.rootComponentId) {
      surface.componentTree = null;
      return;
    }

    const rootComponent = surface.components.get(surface.rootComponentId);
    if (!rootComponent) {
      // In streaming mode, root component may not be defined yet
      if (streamingState) {
        streamingState.isStreaming = true;
        streamingState.pendingComponentIds.add(surface.rootComponentId);
      }
      console.warn(`Root component ${surface.rootComponentId} not found.`);
      surface.componentTree = null;
      return;
    }

    // Build tree with cycle detection
    const visited = new Set<string>();
    surface.componentTree = this.buildComponentNode(
      rootComponent,
      surface,
      visited,
      streamingState,
    );

    // Update streaming state
    if (streamingState) {
      // Check if all pending components are now available
      const allPendingResolved = streamingState.pendingComponentIds.size === 0
        || [...streamingState.pendingComponentIds].every(id => surface.components.has(id));

      if (allPendingResolved && streamingState.pendingComponentIds.size === 0) {
        streamingState.isStreaming = false;
      }
    }
  }

  /**
   * Build a component node recursively
   * In streaming mode, tracks pending components that are not yet defined
   */
  private buildComponentNode(
    definition: ComponentDefinition,
    surface: Surface,
    visited: Set<string>,
    streamingState?: StreamingState,
  ): AnyComponentNode | null {
    // Cycle detection
    if (visited.has(definition.id)) {
      console.warn(`Circular reference detected for component ${definition.id}.`);
      return null;
    }
    visited.add(definition.id);

    const typeName = getComponentTypeName(definition.component);
    const props = getComponentProps<Record<string, unknown>>(definition.component);

    // Resolve property values
    const resolvedProps = this.resolveProps(props, surface.dataModel);

    // Build children
    const children = this.buildChildren(props, surface, visited, streamingState);

    // Track rendered component in streaming state
    if (streamingState) {
      streamingState.renderedComponentIds.add(definition.id);
    }

    // Remove from visited after processing (allow same component in different branches)
    visited.delete(definition.id);

    return {
      id: definition.id,
      type: typeName,
      props: resolvedProps,
      children: children.length > 0 ? children : undefined,
    };
  }

  /**
   * Resolve property values from data model
   */
  private resolveProps(
    props: Record<string, unknown>,
    dataModel: DataMap,
  ): Record<string, unknown> {
    const resolved: Record<string, unknown> = {};

    for (const [key, value] of Object.entries(props)) {
      if (key === 'children') {
        // Skip children, handled separately
        continue;
      }

      if (this.isPropertyValue(value)) {
        resolved[key] = this.resolvePropertyValue(value as PropertyValue, dataModel);
      }
      else if (Array.isArray(value)) {
        // Handle arrays (like tabs)
        resolved[key] = value.map(item =>
          typeof item === 'object' && item !== null
            ? this.resolveProps(item as Record<string, unknown>, dataModel)
            : item,
        );
      }
      else if (typeof value === 'object' && value !== null) {
        // Handle nested objects
        resolved[key] = this.resolveProps(value as Record<string, unknown>, dataModel);
      }
      else {
        resolved[key] = value;
      }
    }

    return resolved;
  }

  /**
   * Check if value is a PropertyValue
   */
  private isPropertyValue(value: unknown): boolean {
    if (typeof value !== 'object' || value === null) {
      return false;
    }
    const obj = value as Record<string, unknown>;
    return (
      'literalString' in obj
      || 'literalNumber' in obj
      || 'literalBoolean' in obj
      || 'path' in obj
    );
  }

  /**
   * Resolve a single PropertyValue
   */
  private resolvePropertyValue(value: PropertyValue, dataModel: DataMap): unknown {
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
      const result = getData(dataModel, value.path);
      return result ?? undefined;
    }
    return undefined;
  }

  /**
   * Build children from ComponentArrayReference
   * In streaming mode, tracks pending child components that are not yet defined
   */
  private buildChildren(
    props: Record<string, unknown>,
    surface: Surface,
    visited: Set<string>,
    streamingState?: StreamingState,
  ): AnyComponentNode[] {
    const childrenRef = props.children as ComponentArrayReference | undefined;
    if (!childrenRef) {
      return [];
    }

    const children: AnyComponentNode[] = [];

    // Handle explicit list
    if (childrenRef.explicitList) {
      for (const childId of childrenRef.explicitList) {
        const childDef = surface.components.get(childId);
        if (!childDef) {
          // In streaming mode, track pending components
          if (streamingState) {
            streamingState.pendingComponentIds.add(childId);
            streamingState.isStreaming = true;
          }
          console.warn(`Child component ${childId} not found. Skipping.`);
          continue;
        }
        const childNode = this.buildComponentNode(childDef, surface, visited, streamingState);
        if (childNode) {
          children.push(childNode);
        }
      }
    }

    // Handle template
    if (childrenRef.template) {
      const { componentId, dataBinding } = childrenRef.template;
      const templateDef = surface.components.get(componentId);
      if (!templateDef) {
        // In streaming mode, track pending template component
        if (streamingState) {
          streamingState.pendingComponentIds.add(componentId);
          streamingState.isStreaming = true;
        }
        console.warn(`Template component ${componentId} not found.`);
        return children;
      }

      // Get data array from binding
      const dataArray = getData(surface.dataModel, dataBinding);
      if (!Array.isArray(dataArray)) {
        return children;
      }

      // Create instance for each data item
      for (let i = 0; i < dataArray.length; i++) {
        const itemPath = `${dataBinding}/${i}`;
        const childNode = this.buildTemplateInstance(
          templateDef,
          surface,
          visited,
          itemPath,
          i,
          streamingState,
        );
        if (childNode) {
          children.push(childNode);
        }
      }
    }

    return children;
  }

  /**
   * Build a template instance with data context
   */
  private buildTemplateInstance(
    definition: ComponentDefinition,
    surface: Surface,
    visited: Set<string>,
    itemPath: string,
    index: number,
    streamingState?: StreamingState,
  ): AnyComponentNode | null {
    const instanceId = `${definition.id}_${index}`;

    // Cycle detection
    if (visited.has(instanceId)) {
      console.warn(`Circular reference detected for template instance ${instanceId}.`);
      return null;
    }
    visited.add(instanceId);

    const typeName = getComponentTypeName(definition.component);
    const props = getComponentProps<Record<string, unknown>>(definition.component);

    // Resolve property values with item context
    const resolvedProps = this.resolvePropsWithContext(props, surface.dataModel, itemPath);

    // Build children
    const children = this.buildChildren(props, surface, visited, streamingState);

    // Track rendered component in streaming state
    if (streamingState) {
      streamingState.renderedComponentIds.add(instanceId);
    }

    visited.delete(instanceId);

    return {
      id: instanceId,
      type: typeName,
      props: resolvedProps,
      children: children.length > 0 ? children : undefined,
    };
  }

  /**
   * Resolve property values with data context
   */
  private resolvePropsWithContext(
    props: Record<string, unknown>,
    dataModel: DataMap,
    contextPath: string,
  ): Record<string, unknown> {
    const resolved: Record<string, unknown> = {};

    for (const [key, value] of Object.entries(props)) {
      if (key === 'children') {
        continue;
      }

      if (this.isPropertyValue(value)) {
        const pv = value as PropertyValue;
        if (isPathReference(pv)) {
          // Resolve relative paths
          const path = pv.path.startsWith('/')
            ? pv.path
            : `${contextPath}/${pv.path}`;
          const result = getData(dataModel, path);
          resolved[key] = result ?? undefined;
        }
        else {
          resolved[key] = this.resolvePropertyValue(pv, dataModel);
        }
      }
      else if (typeof value === 'object' && value !== null) {
        resolved[key] = this.resolvePropsWithContext(
          value as Record<string, unknown>,
          dataModel,
          contextPath,
        );
      }
      else {
        resolved[key] = value;
      }
    }

    return resolved;
  }

  /**
   * Get data from a surface's data model
   */
  getData(surfaceId: string, path: string): DataValue | null {
    const surface = this.surfaces.get(surfaceId);
    if (!surface) {
      return null;
    }
    return getData(surface.dataModel, path);
  }

  /**
   * Set data in a surface's data model
   */
  setData(surfaceId: string, path: string, value: DataValue): boolean {
    const surface = this.surfaces.get(surfaceId);
    if (!surface) {
      return false;
    }

    const result = setData(surface.dataModel, path, value);

    if (result && surface.rootComponentId) {
      const streamingState = this.streamingStates.get(surfaceId);
      this.rebuildComponentTree(surface, streamingState);
      this.notifyListeners(surfaceId, surface);
    }

    return result;
  }

  /**
   * Add data to a surface's data model
   */
  addDataToSurface(surfaceId: string, path: string, value: DataValue): boolean {
    const surface = this.surfaces.get(surfaceId);
    if (!surface) {
      return false;
    }

    const result = addData(surface.dataModel, path, value);

    if (result && surface.rootComponentId) {
      const streamingState = this.streamingStates.get(surfaceId);
      this.rebuildComponentTree(surface, streamingState);
      this.notifyListeners(surfaceId, surface);
    }

    return result;
  }

  /**
   * Remove data from a surface's data model
   */
  removeDataFromSurface(surfaceId: string, path: string): boolean {
    const surface = this.surfaces.get(surfaceId);
    if (!surface) {
      return false;
    }

    const result = removeData(surface.dataModel, path);

    if (result && surface.rootComponentId) {
      const streamingState = this.streamingStates.get(surfaceId);
      this.rebuildComponentTree(surface, streamingState);
      this.notifyListeners(surfaceId, surface);
    }

    return result;
  }

  /**
   * Resolve action context from a Button's actionContext
   */
  resolveActionContext(
    surfaceId: string,
    actionContext: Record<string, PropertyValue>,
  ): Record<string, unknown> {
    const surface = this.surfaces.get(surfaceId);
    if (!surface) {
      return {};
    }

    const resolved: Record<string, unknown> = {};
    for (const [key, value] of Object.entries(actionContext)) {
      resolved[key] = this.resolvePropertyValue(value, surface.dataModel);
    }
    return resolved;
  }

  /**
   * Subscribe to surface changes
   */
  subscribe(surfaceId: string, listener: (surface: Surface) => void): () => void {
    let listeners = this.listeners.get(surfaceId);
    if (!listeners) {
      listeners = new Set();
      this.listeners.set(surfaceId, listeners);
    }
    listeners.add(listener);

    // Return unsubscribe function
    return () => {
      listeners?.delete(listener);
    };
  }

  /**
   * Notify listeners of surface changes
   */
  private notifyListeners(surfaceId: string, surface: Surface): void {
    const listeners = this.listeners.get(surfaceId);
    if (listeners) {
      for (const listener of listeners) {
        listener(surface);
      }
    }
  }
}

/**
 * Create a new MessageProcessor instance
 */
export function createMessageProcessor(registry?: ComponentRegistry): MessageProcessor {
  return new MessageProcessor(registry);
}

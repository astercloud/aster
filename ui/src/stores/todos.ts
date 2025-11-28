/**
 * Todo 任务状态管理
 *
 * 管理 Agent 的任务列表
 */

import { defineStore } from 'pinia';
import { ref, computed } from 'vue';
import type { TodoItemData } from '@/types';
import { useWebSocket } from '@/composables/useWebSocket';

export const useTodosStore = defineStore('todos', () => {
  // ==================
  // State
  // ==================

  // Todo 列表
  const todos = ref<TodoItemData[]>([]);

  // ==================
  // Getters
  // ==================

  // 待处理任务数量
  const pendingCount = computed(() =>
    todos.value.filter(t => t.status === 'pending').length
  );

  // 进行中任务数量
  const inProgressCount = computed(() =>
    todos.value.filter(t => t.status === 'in_progress').length
  );

  // 已完成任务数量
  const completedCount = computed(() =>
    todos.value.filter(t => t.status === 'completed').length
  );

  // 总任务数量
  const totalCount = computed(() => todos.value.length);

  // 完成进度（0-1）
  const progress = computed(() => {
    if (totalCount.value === 0) return 0;
    return completedCount.value / totalCount.value;
  });

  // 是否有进行中的任务
  const hasInProgressTask = computed(() => inProgressCount.value > 0);

  // 当前进行中的任务
  const currentTask = computed(() =>
    todos.value.find(t => t.status === 'in_progress')
  );

  // ==================
  // Actions
  // ==================

  /**
   * 更新整个 Todo 列表（来自后端事件）
   */
  const updateTodos = (newTodos: TodoItemData[]) => {
    todos.value = newTodos;
  };

  /**
   * 添加单个 Todo
   */
  const addTodo = (todo: TodoItemData) => {
    todos.value.push(todo);
  };

  /**
   * 更新单个 Todo
   */
  const updateTodo = (id: string, updates: Partial<TodoItemData>) => {
    const index = todos.value.findIndex(t => t.id === id);
    if (index !== -1) {
      const todo = todos.value[index];
      if (todo) {
        todos.value[index] = {
          ...todo,
          ...updates,
        };
      }
    }
  };

  /**
   * 删除 Todo
   */
  const removeTodo = (id: string) => {
    const index = todos.value.findIndex(t => t.id === id);
    if (index !== -1) {
      todos.value.splice(index, 1);
    }
  };

  /**
   * 切换 Todo 状态
   *
   * 逻辑：
   * - pending → in_progress（确保同时只有一个 in_progress）
   * - in_progress → completed
   * - completed → pending
   */
  const toggleStatus = (id: string): boolean => {
    const todo = todos.value.find(t => t.id === id);
    if (!todo) return false;

    let newStatus: TodoItemData['status'];

    if (todo.status === 'pending') {
      // 检查是否已有进行中的任务
      if (hasInProgressTask.value) {
        console.warn('同时只能有一个任务处于进行中状态');
        return false;
      }
      newStatus = 'in_progress';
    } else if (todo.status === 'in_progress') {
      newStatus = 'completed';
    } else {
      newStatus = 'pending';
    }

    updateTodo(id, { status: newStatus });
    return true;
  };

  /**
   * 标记为进行中
   */
  const markAsInProgress = (id: string): boolean => {
    if (hasInProgressTask.value) {
      console.warn('同时只能有一个任务处于进行中状态');
      return false;
    }
    updateTodo(id, { status: 'in_progress' });
    return true;
  };

  /**
   * 标记为已完成
   */
  const markAsCompleted = (id: string) => {
    updateTodo(id, { status: 'completed' });
  };

  /**
   * 清除所有 Todo
   */
  const clearAllTodos = () => {
    todos.value = [];
  };

  /**
   * 清除已完成的 Todo
   */
  const clearCompletedTodos = () => {
    todos.value = todos.value.filter(t => t.status !== 'completed');
  };

  /**
   * 创建新 Todo (发送到后端)
   */
  const createTodo = async (todo: Omit<TodoItemData, 'id' | 'created_at' | 'updated_at'>) => {
    const { getInstance } = useWebSocket();
    const ws = getInstance();
    if (ws) {
      ws.send({
        type: 'todo_create',
        payload: {
          content: todo.content,
          active_form: todo.active_form,
          status: todo.status,
          priority: todo.priority,
        },
      });
    } else {
      console.error('WebSocket not connected, cannot create todo');
    }
  };

  /**
   * 更新 Todo 状态 (发送到后端)
   */
  const updateTodoStatus = async (todoId: string, status: 'pending' | 'in_progress' | 'completed') => {
    const { getInstance } = useWebSocket();
    const ws = getInstance();
    if (ws) {
      ws.send({
        type: 'todo_update',
        payload: {
          id: todoId,
          status,
        },
      });
    } else {
      console.error('WebSocket not connected, cannot update todo');
    }
  };

  /**
   * 删除 Todo (发送到后端)
   */
  const deleteTodo = async (todoId: string) => {
    const { getInstance } = useWebSocket();
    const ws = getInstance();
    if (ws) {
      ws.send({
        type: 'todo_delete',
        payload: {
          id: todoId,
        },
      });
    } else {
      console.error('WebSocket not connected, cannot delete todo');
    }
  };

  // ==================
  // Return
  // ==================

  return {
    // State
    todos,

    // Getters
    pendingCount,
    inProgressCount,
    completedCount,
    totalCount,
    progress,
    hasInProgressTask,
    currentTask,

    // Actions
    updateTodos,
    addTodo,
    updateTodo,
    removeTodo,
    toggleStatus,
    markAsInProgress,
    markAsCompleted,
    clearAllTodos,
    clearCompletedTodos,
    createTodo,
    updateTodoStatus,
    deleteTodo,
  };
});

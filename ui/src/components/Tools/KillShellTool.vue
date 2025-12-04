<template>
  <div class="kill-shell-tool">
    <!-- 工具头部 -->
    <div class="flex items-center justify-between mb-4">
      <div class="flex items-center space-x-3">
        <div class="w-8 h-8 bg-red-500 rounded-lg flex items-center justify-center">
          <svg class="w-5 h-5 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"></path>
          </svg>
        </div>
        <div>
          <h3 class="text-lg font-semibold text-gray-900 dark:text-gray-100">进程管理工具</h3>
          <p class="text-sm text-gray-500 dark:text-gray-400">管理和终止系统进程</p>
        </div>
      </div>

      <button @click="refreshProcesses" :disabled="loading" class="px-3 py-2 bg-blue-500 text-white rounded-lg hover:bg-blue-600 disabled:opacity-50 flex items-center space-x-2">
        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"></path>
        </svg>
        <span>刷新</span>
      </button>
    </div>

    <!-- 快速操作 -->
    <div class="mb-4 p-4 bg-gray-50 dark:bg-gray-800 rounded-lg">
      <div class="flex items-center justify-between mb-3">
        <h4 class="font-medium text-gray-900 dark:text-gray-100">快速操作</h4>
        <button @click="showQuickKill = !showQuickKill" class="text-blue-500 hover:text-blue-600 text-sm">
          {{ showQuickKill ? "隐藏" : "显示" }}
        </button>
      </div>

      <div v-if="showQuickKill" class="grid grid-cols-1 md:grid-cols-2 gap-3">
        <button @click="killByName('aster')" class="px-3 py-2 bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-300 rounded-lg hover:bg-red-200 dark:hover:bg-red-900/50 text-sm">终止所有 aster 进程</button>
        <button @click="killByPort(8080)" class="px-3 py-2 bg-orange-100 dark:bg-orange-900/30 text-orange-700 dark:text-orange-300 rounded-lg hover:bg-orange-200 dark:hover:bg-orange-900/50 text-sm">终止占用 8080 端口的进程</button>
        <button @click="killByPort(3000)" class="px-3 py-2 bg-yellow-100 dark:bg-yellow-900/30 text-yellow-700 dark:text-yellow-300 rounded-lg hover:bg-yellow-200 dark:hover:bg-yellow-900/50 text-sm">终止占用 3000 端口的进程</button>
        <button @click="killZombies" class="px-3 py-2 bg-purple-100 dark:bg-purple-900/30 text-purple-700 dark:text-purple-300 rounded-lg hover:bg-purple-200 dark:hover:bg-purple-900/50 text-sm">清理僵尸进程</button>
      </div>
    </div>

    <!-- 搜索和过滤 -->
    <div class="mb-4 space-y-3">
      <div class="flex space-x-3">
        <div class="flex-1 relative">
          <input
            v-model="searchQuery"
            type="text"
            placeholder="搜索进程名称、PID 或命令..."
            class="w-full pl-10 pr-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100"
          />
          <svg class="absolute left-3 top-2.5 w-5 h-5 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"></path>
          </svg>
        </div>

        <select v-model="filterType" class="px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100">
          <option value="all">所有进程</option>
          <option value="user">用户进程</option>
          <option value="system">系统进程</option>
          <option value="zombie">僵尸进程</option>
          <option value="sleeping">休眠进程</option>
          <option value="running">运行中</option>
        </select>
      </div>

      <div class="flex items-center space-x-3">
        <label class="flex items-center space-x-2 text-sm text-gray-600 dark:text-gray-400">
          <input v-model="showMyProcesses" type="checkbox" class="rounded border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-blue-500 focus:ring-blue-500" />
          <span>只显示我的进程</span>
        </label>

        <label class="flex items-center space-x-2 text-sm text-gray-600 dark:text-gray-400">
          <input v-model="sortBy" type="radio" value="cpu" class="border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-blue-500 focus:ring-blue-500" />
          <span>按 CPU 排序</span>
        </label>

        <label class="flex items-center space-x-2 text-sm text-gray-600 dark:text-gray-400">
          <input v-model="sortBy" type="radio" value="memory" class="border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-blue-500 focus:ring-blue-500" />
          <span>按内存排序</span>
        </label>
      </div>
    </div>

    <!-- 进程列表 -->
    <div class="space-y-3 mb-4">
      <div v-if="loading" class="text-center py-8 text-gray-500 dark:text-gray-400">
        <svg class="animate-spin h-8 w-8 mx-auto mb-2" fill="none" viewBox="0 0 24 24">
          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
        </svg>
        正在加载进程列表...
      </div>

      <div v-else-if="filteredProcesses.length === 0" class="text-center py-8 text-gray-500 dark:text-gray-400">
        <svg class="w-12 h-12 mx-auto mb-2 text-gray-300 dark:text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"></path>
        </svg>
        <p>没有找到进程</p>
      </div>

      <div v-else>
        <div v-for="process in filteredProcesses" :key="process.pid" class="p-4 bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 hover:shadow-md transition-all">
          <div class="flex items-center justify-between">
            <div class="flex items-center space-x-3">
              <!-- 进程状态图标 -->
              <div :class="getStatusIconClass(process.status)" class="w-8 h-8 rounded-full flex items-center justify-center">
                <svg class="w-4 h-4 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" :d="getStatusIcon(process.status)"></path>
                </svg>
              </div>

              <!-- 进程信息 -->
              <div class="flex-1 min-w-0">
                <div class="flex items-center space-x-2 mb-1">
                  <span class="font-medium text-gray-900 dark:text-gray-100 truncate">{{ process.name }}</span>
                  <span class="px-2 py-0.5 bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-400 text-xs rounded-full"> PID: {{ process.pid }} </span>
                  <span v-if="process.ppid" class="px-2 py-0.5 bg-blue-100 dark:bg-blue-900/30 text-blue-600 dark:text-blue-400 text-xs rounded-full"> PPID: {{ process.ppid }} </span>
                  <span v-if="process.user" class="px-2 py-0.5 bg-purple-100 dark:bg-purple-900/30 text-purple-600 dark:text-purple-400 text-xs rounded-full">
                    {{ process.user }}
                  </span>
                </div>

                <div class="flex items-center space-x-4 text-sm text-gray-500 dark:text-gray-400">
                  <span class="flex items-center space-x-1">
                    <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 3v2m6-2v2M9 19v2m6-2v2M5 9H3m2 6H3m18-6h-2m2 6h-2M7 19h10a2 2 0 002-2V7a2 2 0 00-2-2H7a2 2 0 00-2 2v10a2 2 0 002 2zM9 9h6v6H9V9z"></path>
                    </svg>
                    <span>CPU: {{ process.cpu }}%</span>
                  </span>
                  <span class="flex items-center space-x-1">
                    <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 14v3m4-3v3m4-3v3M3 21h18M3 10h18M3 7l9-4 9 4M4 10h16v11H4V10z"></path>
                    </svg>
                    <span>内存: {{ process.memory }}</span>
                  </span>
                  <span v-if="process.ports && process.ports.length > 0" class="flex items-center space-x-1">
                    <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9-3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9m-9 9a9 9 0 019-9"></path>
                    </svg>
                    <span>端口: {{ process.ports.join(", ") }}</span>
                  </span>
                </div>

                <div v-if="process.command" class="mt-1 text-xs text-gray-400 dark:text-gray-500 truncate">
                  {{ process.command }}
                </div>
              </div>
            </div>

            <!-- 操作按钮 -->
            <div class="flex items-center space-x-2">
              <button @click="viewProcessDetails(process)" class="p-2 text-blue-500 hover:bg-blue-50 dark:hover:bg-blue-900/20 rounded-lg" title="查看详情">
                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"></path>
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z"></path>
                </svg>
              </button>

              <button
                @click="confirmKill(process)"
                :disabled="process.protected"
                class="p-2 text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20 rounded-lg disabled:opacity-50 disabled:cursor-not-allowed"
                :title="process.protected ? '系统保护进程' : '终止进程'"
              >
                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"></path>
                </svg>
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 统计信息 -->
    <div class="grid grid-cols-2 md:grid-cols-4 gap-3 mb-4">
      <div class="p-3 bg-gray-50 dark:bg-gray-800 rounded-lg text-center">
        <div class="text-2xl font-bold text-gray-900 dark:text-gray-100">{{ stats.total }}</div>
        <div class="text-xs text-gray-500 dark:text-gray-400">总进程数</div>
      </div>
      <div class="p-3 bg-green-50 dark:bg-green-900/20 rounded-lg text-center">
        <div class="text-2xl font-bold text-green-600 dark:text-green-400">{{ stats.running }}</div>
        <div class="text-xs text-gray-500 dark:text-gray-400">运行中</div>
      </div>
      <div class="p-3 bg-yellow-50 dark:bg-yellow-900/20 rounded-lg text-center">
        <div class="text-2xl font-bold text-yellow-600 dark:text-yellow-400">{{ stats.sleeping }}</div>
        <div class="text-xs text-gray-500 dark:text-gray-400">休眠中</div>
      </div>
      <div class="p-3 bg-red-50 dark:bg-red-900/20 rounded-lg text-center">
        <div class="text-2xl font-bold text-red-600 dark:text-red-400">{{ stats.zombie }}</div>
        <div class="text-xs text-gray-500 dark:text-gray-400">僵尸进程</div>
      </div>
    </div>

    <!-- 高级操作 -->
    <div class="p-4 bg-gray-50 dark:bg-gray-800 rounded-lg">
      <h4 class="font-medium text-gray-900 dark:text-gray-100 mb-3">高级操作</h4>
      <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
        <div class="flex space-x-2">
          <input v-model="customKillSignal" type="text" placeholder="信号 (如: SIGTERM, SIGKILL)" class="flex-1 px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100" />
          <input v-model="customPid" type="text" placeholder="PID" class="w-20 px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100" />
          <button @click="customKill" :disabled="!customPid || !customKillSignal" class="px-4 py-2 bg-red-500 text-white rounded-lg hover:bg-red-600 disabled:opacity-50">发送信号</button>
        </div>

        <button @click="showSystemInfo = !showSystemInfo" class="px-4 py-2 bg-blue-500 text-white rounded-lg hover:bg-blue-600">{{ showSystemInfo ? "隐藏" : "显示" }}系统信息</button>
      </div>

      <div v-if="showSystemInfo" class="mt-4 p-3 bg-white dark:bg-gray-900 rounded-lg">
        <div class="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
          <div>
            <div class="text-gray-500 dark:text-gray-400">系统负载</div>
            <div class="font-medium text-gray-900 dark:text-gray-100">{{ systemInfo.loadAverage || "N/A" }}</div>
          </div>
          <div>
            <div class="text-gray-500 dark:text-gray-400">内存使用</div>
            <div class="font-medium text-gray-900 dark:text-gray-100">{{ systemInfo.memoryUsage || "N/A" }}</div>
          </div>
          <div>
            <div class="text-gray-500 dark:text-gray-400">CPU 使用</div>
            <div class="font-medium text-gray-900 dark:text-gray-100">{{ systemInfo.cpuUsage || "N/A" }}</div>
          </div>
          <div>
            <div class="text-gray-500 dark:text-gray-400">运行时间</div>
            <div class="font-medium text-gray-900 dark:text-gray-100">{{ systemInfo.uptime || "N/A" }}</div>
          </div>
        </div>
      </div>
    </div>

    <!-- 进程详情对话框 -->
    <div v-if="selectedProcess" class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4">
      <div class="bg-white dark:bg-gray-800 rounded-lg max-w-4xl w-full max-h-[80vh] overflow-y-auto">
        <div class="p-6">
          <div class="flex items-center justify-between mb-4">
            <h3 class="text-lg font-semibold text-gray-900 dark:text-gray-100">进程详情 - {{ selectedProcess.name }} ({{ selectedProcess.pid }})</h3>
            <button @click="selectedProcess = null" class="text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200">
              <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path>
              </svg>
            </button>
          </div>

          <div class="space-y-4">
            <div class="grid grid-cols-2 gap-4">
              <div>
                <div class="text-sm text-gray-500 dark:text-gray-400">进程ID</div>
                <div class="font-medium text-gray-900 dark:text-gray-100">{{ selectedProcess.pid }}</div>
              </div>
              <div>
                <div class="text-sm text-gray-500 dark:text-gray-400">父进程ID</div>
                <div class="font-medium text-gray-900 dark:text-gray-100">{{ selectedProcess.ppid || "N/A" }}</div>
              </div>
              <div>
                <div class="text-sm text-gray-500 dark:text-gray-400">用户</div>
                <div class="font-medium text-gray-900 dark:text-gray-100">{{ selectedProcess.user || "N/A" }}</div>
              </div>
              <div>
                <div class="text-sm text-gray-500 dark:text-gray-400">状态</div>
                <div class="font-medium text-gray-900 dark:text-gray-100">{{ selectedProcess.status }}</div>
              </div>
              <div>
                <div class="text-sm text-gray-500 dark:text-gray-400">CPU使用率</div>
                <div class="font-medium text-gray-900 dark:text-gray-100">{{ selectedProcess.cpu }}%</div>
              </div>
              <div>
                <div class="text-sm text-gray-500 dark:text-gray-400">内存使用</div>
                <div class="font-medium text-gray-900 dark:text-gray-100">{{ selectedProcess.memory }}</div>
              </div>
              <div>
                <div class="text-sm text-gray-500 dark:text-gray-400">启动时间</div>
                <div class="font-medium text-gray-900 dark:text-gray-100">{{ selectedProcess.startTime || "N/A" }}</div>
              </div>
              <div>
                <div class="text-sm text-gray-500 dark:text-gray-400">运行时长</div>
                <div class="font-medium text-gray-900 dark:text-gray-100">{{ selectedProcess.duration || "N/A" }}</div>
              </div>
            </div>

            <div>
              <div class="text-sm text-gray-500 dark:text-gray-400 mb-1">命令行</div>
              <div class="p-3 bg-gray-50 dark:bg-gray-900 rounded-lg font-mono text-sm text-gray-900 dark:text-gray-100 break-all">
                {{ selectedProcess.command || "N/A" }}
              </div>
            </div>

            <div v-if="selectedProcess.environment">
              <div class="text-sm text-gray-500 dark:text-gray-400 mb-1">环境变量</div>
              <div class="p-3 bg-gray-50 dark:bg-gray-900 rounded-lg font-mono text-sm text-gray-900 dark:text-gray-100 max-h-40 overflow-y-auto">
                <div v-for="(value, key) in selectedProcess.environment" :key="key" class="mb-1">
                  <span class="text-blue-600 dark:text-blue-400">{{ key }}=</span>{{ value }}
                </div>
              </div>
            </div>

            <div v-if="selectedProcess.openFiles">
              <div class="text-sm text-gray-500 dark:text-gray-400 mb-1">打开的文件</div>
              <div class="p-3 bg-gray-50 dark:bg-gray-900 rounded-lg text-sm text-gray-900 dark:text-gray-100 max-h-40 overflow-y-auto">
                <div v-for="file in selectedProcess.openFiles" :key="file" class="mb-1">
                  {{ file }}
                </div>
              </div>
            </div>

            <div v-if="selectedProcess.networkConnections">
              <div class="text-sm text-gray-500 dark:text-gray-400 mb-1">网络连接</div>
              <div class="p-3 bg-gray-50 dark:bg-gray-900 rounded-lg text-sm text-gray-900 dark:text-gray-100">
                <div v-for="conn in selectedProcess.networkConnections" :key="conn.id" class="mb-2 flex justify-between">
                  <span>{{ conn.protocol }} {{ conn.localAddress }}:{{ conn.localPort }}</span>
                  <span class="text-gray-500 dark:text-gray-400">{{ conn.remoteAddress }}:{{ conn.remotePort }}</span>
                </div>
              </div>
            </div>
          </div>

          <div class="flex justify-end space-x-3 mt-6">
            <button @click="confirmKill(selectedProcess)" :disabled="selectedProcess.protected" class="px-4 py-2 bg-red-500 text-white rounded-lg hover:bg-red-600 disabled:opacity-50">终止进程</button>
            <button @click="selectedProcess = null" class="px-4 py-2 bg-gray-500 text-white rounded-lg hover:bg-gray-600">关闭</button>
          </div>
        </div>
      </div>
    </div>

    <!-- 确认对话框 -->
    <div v-if="showConfirmDialog" class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4">
      <div class="bg-white dark:bg-gray-800 rounded-lg max-w-md w-full p-6">
        <div class="flex items-center space-x-3 mb-4">
          <div class="w-10 h-10 bg-red-100 dark:bg-red-900/30 rounded-full flex items-center justify-center">
            <svg class="w-6 h-6 text-red-600 dark:text-red-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L4.082 16.5c-.77.833.192 2.5 1.732 2.5z"></path>
            </svg>
          </div>
          <div>
            <h3 class="text-lg font-semibold text-gray-900 dark:text-gray-100">确认终止进程</h3>
            <p class="text-sm text-gray-500 dark:text-gray-400">此操作不可撤销</p>
          </div>
        </div>

        <div v-if="processToKill" class="mb-4 p-3 bg-gray-50 dark:bg-gray-900 rounded-lg">
          <div class="text-sm text-gray-600 dark:text-gray-400 mb-1">将要终止的进程：</div>
          <div class="font-medium text-gray-900 dark:text-gray-100">{{ processToKill.name }} (PID: {{ processToKill.pid }})</div>
          <div v-if="processToKill.command" class="text-xs text-gray-500 dark:text-gray-400 mt-1 truncate">
            {{ processToKill.command }}
          </div>
        </div>

        <div class="flex justify-end space-x-3">
          <button
            @click="
              showConfirmDialog = false;
              processToKill = null;
            "
            class="px-4 py-2 bg-gray-500 text-white rounded-lg hover:bg-gray-600"
          >
            取消
          </button>
          <button @click="executeKill" :disabled="killing" class="px-4 py-2 bg-red-500 text-white rounded-lg hover:bg-red-600 disabled:opacity-50">
            {{ killing ? "终止中..." : "确认终止" }}
          </button>
        </div>
      </div>
    </div>

    <!-- 操作结果 -->
    <div
      v-if="actionResult"
      :class="actionResult.type === 'success' ? 'bg-green-50 dark:bg-green-900/20 border-green-200 dark:border-green-800 text-green-800 dark:text-green-200' : 'bg-red-50 dark:bg-red-900/20 border-red-200 dark:border-red-800 text-red-800 dark:text-red-200'"
      class="p-4 rounded-lg border mb-4"
    >
      <div class="flex items-center space-x-2">
        <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path v-if="actionResult.type === 'success'" stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"></path>
          <path v-else stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path>
        </svg>
        <span>{{ actionResult.message }}</span>
      </div>
      <div v-if="actionResult.details" class="mt-2 text-sm">
        {{ actionResult.details }}
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from "vue";

interface Process {
  pid: number;
  ppid?: number;
  name: string;
  user?: string;
  status: string;
  cpu: number;
  memory: string;
  command?: string;
  ports?: number[];
  protected?: boolean;
  startTime?: string;
  duration?: string;
  environment?: Record<string, string>;
  openFiles?: string[];
  networkConnections?: Array<{
    id: string;
    protocol: string;
    localAddress: string;
    localPort: number;
    remoteAddress: string;
    remotePort: number;
  }>;
}

interface SystemInfo {
  loadAverage: string;
  memoryUsage: string;
  cpuUsage: string;
  uptime: string;
}

const props = defineProps<{
  socket?: WebSocket;
  sessionId?: string;
}>();

const emit = defineEmits<{
  execute: [
    {
      command: "kill-shell";
      parameters: {
        pid?: number;
        signal?: string;
        name?: string;
        port?: number;
      };
    },
  ];
}>();

// 状态管理
const loading = ref(false);
const processes = ref<Process[]>([]);
const searchQuery = ref("");
const filterType = ref("all");
const showMyProcesses = ref(false);
const sortBy = ref("cpu");
const showQuickKill = ref(true);
const showSystemInfo = ref(false);
const selectedProcess = ref<Process | null>(null);
const showConfirmDialog = ref(false);
const processToKill = ref<Process | null>(null);
const killing = ref(false);
const actionResult = ref<{ type: "success" | "error"; message: string; details?: string } | null>(null);

// 高级操作
const customKillSignal = ref("SIGTERM");
const customPid = ref("");

// 系统信息
const systemInfo = ref<SystemInfo>({
  loadAverage: "",
  memoryUsage: "",
  cpuUsage: "",
  uptime: "",
});

// 自动刷新
const autoRefreshInterval = ref<NodeJS.Timeout | null>(null);

// 统计信息
const stats = computed(() => {
  const total = processes.value.length;
  const running = processes.value.filter((p) => p.status === "R" || p.status === "Running").length;
  const sleeping = processes.value.filter((p) => p.status === "S" || p.status === "Sleeping").length;
  const zombie = processes.value.filter((p) => p.status === "Z" || p.status === "Zombie").length;

  return { total, running, sleeping, zombie };
});

// 过滤后的进程列表
const filteredProcesses = computed(() => {
  let filtered = processes.value;

  // 搜索过滤
  if (searchQuery.value) {
    const query = searchQuery.value.toLowerCase();
    filtered = filtered.filter((process) => process.name.toLowerCase().includes(query) || process.pid.toString().includes(query) || (process.command && process.command.toLowerCase().includes(query)));
  }

  // 类型过滤
  if (filterType.value !== "all") {
    filtered = filtered.filter((process) => {
      switch (filterType.value) {
        case "user":
          return process.user && !["root", "system"].includes(process.user);
        case "system":
          return process.user === "root" || process.user === "system";
        case "zombie":
          return process.status === "Z" || process.status === "Zombie";
        case "sleeping":
          return process.status === "S" || process.status === "Sleeping";
        case "running":
          return process.status === "R" || process.status === "Running";
        default:
          return true;
      }
    });
  }

  // 只显示当前用户的进程
  if (showMyProcesses.value) {
    filtered = filtered.filter((process) => process.user && !["root", "system"].includes(process.user));
  }

  // 排序
  if (sortBy.value === "cpu") {
    filtered = [...filtered].sort((a, b) => b.cpu - a.cpu);
  } else if (sortBy.value === "memory") {
    filtered = [...filtered].sort((a, b) => {
      const memA = parseFloat(a.memory.replace(/[^0-9.]/g, ""));
      const memB = parseFloat(b.memory.replace(/[^0-9.]/g, ""));
      return memB - memA;
    });
  }

  return filtered;
});

// 获取状态图标
const getStatusIcon = (status: string): string => {
  switch (status) {
    case "R":
    case "Running":
      return "M13 10V3L4 14h7v7l9-11h-7z";
    case "S":
    case "Sleeping":
      return "M20.354 15.354A9 9 0 018.646 3.646 9.003 9.003 0 0012 21a9.003 9.003 0 008.354-5.646z";
    case "Z":
    case "Zombie":
      return "M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z";
    default:
      return "M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2";
  }
};

// 获取状态图标样式
const getStatusIconClass = (status: string): string => {
  switch (status) {
    case "R":
    case "Running":
      return "bg-green-500";
    case "S":
    case "Sleeping":
      return "bg-blue-500";
    case "Z":
    case "Zombie":
      return "bg-red-500";
    default:
      return "bg-gray-500";
  }
};

// 获取进程列表
const loadProcesses = async () => {
  loading.value = true;
  try {
    const response = await fetch("/api/shells/processes");
    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`);
    }
    const data = await response.json();
    processes.value = data.processes || [];

    // 更新系统信息
    if (data.systemInfo) {
      systemInfo.value = data.systemInfo;
    }
  } catch (error) {
    console.error("Error loading processes:", error);
    showActionResult("error", "获取进程列表失败", error instanceof Error ? error.message : String(error));
  } finally {
    loading.value = false;
  }
};

// 刷新进程列表
const refreshProcesses = () => {
  loadProcesses();
};

// 查看进程详情
const viewProcessDetails = async (process: Process) => {
  try {
    const response = await fetch(`/api/shells/processes/${process.pid}`);
    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`);
    }
    const details = await response.json();
    selectedProcess.value = { ...process, ...details };
  } catch (error) {
    console.error("Error getting process details:", error);
    showActionResult("error", "获取进程详情失败", error instanceof Error ? error.message : String(error));
  }
};

// 确认终止进程
const confirmKill = (process: Process) => {
  if (process.protected) {
    showActionResult("error", "无法终止系统保护进程");
    return;
  }
  processToKill.value = process;
  showConfirmDialog.value = true;
};

// 执行终止进程
const executeKill = async () => {
  if (!processToKill.value) return;

  killing.value = true;
  try {
    emit("execute", {
      command: "kill-shell",
      parameters: {
        pid: processToKill.value.pid,
      },
    });

    showConfirmDialog.value = false;
    showActionResult("success", `已终止进程 ${processToKill.value.name} (${processToKill.value.pid})`);

    // 刷新进程列表
    setTimeout(() => {
      loadProcesses();
    }, 1000);

    processToKill.value = null;
  } catch (error) {
    console.error("Error killing process:", error);
    showActionResult("error", "终止进程失败", error instanceof Error ? error.message : String(error));
  } finally {
    killing.value = false;
  }
};

// 按名称终止进程
const killByName = (name: string) => {
  emit("execute", {
    command: "kill-shell",
    parameters: {
      name: name,
    },
  });

  showActionResult("success", `正在终止名称包含 "${name}" 的进程`);

  setTimeout(() => {
    loadProcesses();
  }, 1000);
};

// 按端口终止进程
const killByPort = (port: number) => {
  emit("execute", {
    command: "kill-shell",
    parameters: {
      port: port,
    },
  });

  showActionResult("success", `正在终止占用端口 ${port} 的进程`);

  setTimeout(() => {
    loadProcesses();
  }, 1000);
};

// 清理僵尸进程
const killZombies = () => {
  const zombieProcesses = processes.value.filter((p) => p.status === "Z" || p.status === "Zombie");

  if (zombieProcesses.length === 0) {
    showActionResult("error", "没有发现僵尸进程");
    return;
  }

  zombieProcesses.forEach((process) => {
    emit("execute", {
      command: "kill-shell",
      parameters: {
        pid: process.pid,
      },
    });
  });

  showActionResult("success", `正在清理 ${zombieProcesses.length} 个僵尸进程`);

  setTimeout(() => {
    loadProcesses();
  }, 1000);
};

// 自定义信号终止
const customKill = () => {
  if (!customPid.value || !customKillSignal.value) return;

  const pid = parseInt(customPid.value);
  if (isNaN(pid)) {
    showActionResult("error", "无效的 PID");
    return;
  }

  emit("execute", {
    command: "kill-shell",
    parameters: {
      pid: pid,
      signal: customKillSignal.value,
    },
  });

  showActionResult("success", `正在向进程 ${pid} 发送 ${customKillSignal.value} 信号`);

  setTimeout(() => {
    loadProcesses();
  }, 1000);

  customPid.value = "";
};

// 显示操作结果
const showActionResult = (type: "success" | "error", message: string, details?: string) => {
  actionResult.value = { type, message, details };
  setTimeout(() => {
    actionResult.value = null;
  }, 5000);
};

// 监听 WebSocket 消息
const handleSocketMessage = (event: MessageEvent) => {
  try {
    const data = JSON.parse(event.data);
    if (data.type === "shell_result" && data.command === "kill_shell") {
      if (data.success) {
        showActionResult("success", data.message || "操作成功");
      } else {
        showActionResult("error", data.error || "操作失败");
      }
    }
  } catch (error) {
    console.error("Error parsing WebSocket message:", error);
  }
};

// 生命周期
onMounted(() => {
  loadProcesses();

  // 启动自动刷新
  autoRefreshInterval.value = setInterval(() => {
    loadProcesses();
  }, 5000); // 每5秒刷新一次

  // 监听 WebSocket
  if (props.socket) {
    props.socket.addEventListener("message", handleSocketMessage);
  }
});

onUnmounted(() => {
  // 清理自动刷新
  if (autoRefreshInterval.value) {
    clearInterval(autoRefreshInterval.value);
  }

  // 移除 WebSocket 监听
  if (props.socket) {
    props.socket.removeEventListener("message", handleSocketMessage);
  }
});
</script>

<style scoped>
/* 自定义样式 */
.kill-shell-tool {
  @apply space-y-4;
}

/* 动画效果 */
.animate-pulse {
  animation: pulse 2s cubic-bezier(0.4, 0, 0.6, 1) infinite;
}

@keyframes pulse {
  0%,
  100% {
    opacity: 1;
  }
  50% {
    opacity: 0.5;
  }
}
</style>

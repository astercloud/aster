<template>
  <div class="skill-call-tool">
    <!-- 工具头部 -->
    <div class="flex items-center justify-between mb-4">
      <div class="flex items-center space-x-3">
        <div class="w-8 h-8 bg-purple-500 rounded-lg flex items-center justify-center">
          <svg class="w-5 h-5 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z"></path>
          </svg>
        </div>
        <div>
          <h3 class="text-lg font-semibold text-gray-900 dark:text-gray-100">技能执行工具</h3>
          <p class="text-sm text-gray-500 dark:text-gray-400">执行和管理AI技能</p>
        </div>
      </div>

      <div class="flex items-center space-x-2">
        <button @click="refreshSkills" :disabled="loading" class="px-3 py-2 bg-blue-500 text-white rounded-lg hover:bg-blue-600 disabled:opacity-50 flex items-center space-x-2">
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"></path>
          </svg>
          <span>刷新技能</span>
        </button>

        <button @click="showSkillStore = !showSkillStore" class="px-3 py-2 bg-purple-500 text-white rounded-lg hover:bg-purple-600 flex items-center space-x-2">
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 11V7a4 4 0 00-8 0v4M5 9h14l1 12H4L5 9z"></path>
          </svg>
          <span>技能商店</span>
        </button>
      </div>
    </div>

    <!-- 技能搜索和过滤 -->
    <div class="mb-4 space-y-3">
      <div class="flex space-x-3">
        <div class="flex-1 relative">
          <input
            v-model="searchQuery"
            type="text"
            placeholder="搜索技能名称或描述..."
            class="w-full pl-10 pr-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-purple-500 focus:border-transparent bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100"
          />
          <svg class="absolute left-3 top-2.5 w-5 h-5 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"></path>
          </svg>
        </div>

        <select v-model="categoryFilter" class="px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100">
          <option value="all">所有分类</option>
          <option value="development">开发工具</option>
          <option value="productivity">生产力</option>
          <option value="data">数据处理</option>
          <option value="media">媒体处理</option>
          <option value="system">系统管理</option>
          <option value="security">安全工具</option>
          <option value="automation">自动化</option>
        </select>

        <select v-model="sortBy" class="px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100">
          <option value="name">按名称排序</option>
          <option value="usage">按使用频率</option>
          <option value="rating">按评分</option>
          <option value="recent">按最近使用</option>
        </select>
      </div>

      <div class="flex items-center space-x-3">
        <label class="flex items-center space-x-2 text-sm text-gray-600 dark:text-gray-400">
          <input v-model="showFavorites" type="checkbox" class="rounded border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-purple-500 focus:ring-purple-500" />
          <span>只显示收藏</span>
        </label>

        <label class="flex items-center space-x-2 text-sm text-gray-600 dark:text-gray-400">
          <input v-model="showInstalled" type="checkbox" class="rounded border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-purple-500 focus:ring-purple-500" />
          <span>只显示已安装</span>
        </label>

        <label class="flex items-center space-x-2 text-sm text-gray-600 dark:text-gray-400">
          <input v-model="showBuiltin" type="checkbox" class="rounded border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-purple-500 focus:ring-purple-500" />
          <span>显示内置技能</span>
        </label>
      </div>
    </div>

    <!-- 技能卡片网格 -->
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4 mb-4">
      <div v-if="loading" class="col-span-full text-center py-8 text-gray-500 dark:text-gray-400">
        <svg class="animate-spin h-8 w-8 mx-auto mb-2" fill="none" viewBox="0 0 24 24">
          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
        </svg>
        正在加载技能列表...
      </div>

      <div v-else-if="filteredSkills.length === 0" class="col-span-full text-center py-8 text-gray-500 dark:text-gray-400">
        <svg class="w-12 h-12 mx-auto mb-2 text-gray-300 dark:text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M19.428 15.428a2 2 0 00-1.022-.547l-2.387-.477a6 6 0 00-3.86.517l-.318.158a6 6 0 01-3.86.517L6.05 15.21a2 2 0 00-1.806.547M8 4h8l-1 1v5.172a2 2 0 00.586 1.414l5 5c1.26 1.26.367 3.414-1.415 3.414H4.828c-1.782 0-2.674-2.154-1.414-3.414l5-5A2 2 0 009 10.172V5L8 4z"
          ></path>
        </svg>
        <p>没有找到技能</p>
      </div>

      <div v-else v-for="skill in filteredSkills" :key="skill.id" class="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 hover:shadow-md transition-all">
        <div class="p-4">
          <!-- 技能头部 -->
          <div class="flex items-start justify-between mb-3">
            <div class="flex items-center space-x-3">
              <div :class="getSkillIconClass(skill.category)" class="w-10 h-10 rounded-lg flex items-center justify-center">
                <svg class="w-6 h-6 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" :d="getSkillIcon(skill.category)"></path>
                </svg>
              </div>
              <div>
                <h4 class="font-semibold text-gray-900 dark:text-gray-100">{{ skill.name }}</h4>
                <p class="text-sm text-gray-500 dark:text-gray-400">{{ skill.category }}</p>
              </div>
            </div>

            <button @click="toggleFavorite(skill)" class="text-yellow-400 hover:text-yellow-500">
              <svg :class="skill.favorite ? 'fill-current' : 'fill-none'" class="w-5 h-5" stroke="currentColor" viewBox="0 0 24 24">
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  stroke-width="2"
                  d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z"
                ></path>
              </svg>
            </button>
          </div>

          <!-- 技能描述 -->
          <p class="text-sm text-gray-600 dark:text-gray-400 mb-3 line-clamp-2">{{ skill.description }}</p>

          <!-- 技能信息 -->
          <div class="flex items-center justify-between mb-3 text-sm">
            <div class="flex items-center space-x-3">
              <span v-if="skill.version" class="px-2 py-1 bg-blue-100 dark:bg-blue-900/30 text-blue-600 dark:text-blue-400 rounded-full text-xs"> v{{ skill.version }} </span>
              <span v-if="skill.installed" class="px-2 py-1 bg-green-100 dark:bg-green-900/30 text-green-600 dark:text-green-400 rounded-full text-xs"> 已安装 </span>
              <span v-else class="px-2 py-1 bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-400 rounded-full text-xs"> 未安装 </span>
              <span v-if="skill.rating" class="flex items-center space-x-1">
                <svg class="w-4 h-4 text-yellow-400" fill="currentColor" viewBox="0 0 20 20">
                  <path
                    d="M9.049 2.927c.3-.921 1.603-.921 1.902 0l1.07 3.292a1 1 0 00.95.69h3.462c.969 0 1.371 1.24.588 1.81l-2.8 2.034a1 1 0 00-.364 1.118l1.07 3.292c.3.921-.755 1.688-1.54 1.118l-2.8-2.034a1 1 0 00-1.175 0l-2.8 2.034c-.784.57-1.838-.197-1.539-1.118l1.07-3.292a1 1 0 00-.364-1.118L2.98 8.72c-.783-.57-.38-1.81.588-1.81h3.461a1 1 0 00.951-.69l1.07-3.292z"
                  ></path>
                </svg>
                <span class="text-gray-600 dark:text-gray-400">{{ skill.rating.toFixed(1) }}</span>
              </span>
            </div>

            <div class="text-gray-500 dark:text-gray-400">{{ getUsageCount(skill) }} 次使用</div>
          </div>

          <!-- 操作按钮 -->
          <div class="flex space-x-2">
            <button v-if="!skill.installed" @click="installSkill(skill)" :disabled="installing" class="flex-1 px-3 py-2 bg-blue-500 text-white rounded-lg hover:bg-blue-600 disabled:opacity-50 text-sm">
              {{ installing ? "安装中..." : "安装" }}
            </button>

            <button v-if="skill.installed" @click="executeSkill(skill)" :disabled="executing" class="flex-1 px-3 py-2 bg-purple-500 text-white rounded-lg hover:bg-purple-600 disabled:opacity-50 text-sm">
              {{ executing ? "执行中..." : "执行" }}
            </button>

            <button @click="showSkillDetails(skill)" class="px-3 py-2 bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-400 rounded-lg hover:bg-gray-200 dark:hover:bg-gray-600 text-sm">详情</button>
          </div>
        </div>
      </div>
    </div>

    <!-- 技能商店 -->
    <div v-if="showSkillStore" class="mb-4 p-4 bg-gray-50 dark:bg-gray-800 rounded-lg">
      <div class="flex items-center justify-between mb-4">
        <h4 class="font-semibold text-gray-900 dark:text-gray-100">技能商店</h4>
        <button @click="showSkillStore = false" class="text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200">
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path>
          </svg>
        </button>
      </div>

      <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div class="p-3 bg-white dark:bg-gray-900 rounded-lg">
          <h5 class="font-medium text-gray-900 dark:text-gray-100 mb-2">热门技能</h5>
          <div class="space-y-2">
            <div v-for="skill in popularSkills" :key="skill.id" class="flex items-center justify-between p-2 hover:bg-gray-50 dark:hover:bg-gray-800 rounded cursor-pointer" @click="installSkill(skill)">
              <div class="flex items-center space-x-2">
                <div class="w-6 h-6 bg-purple-100 dark:bg-purple-900/30 rounded flex items-center justify-center">
                  <svg class="w-4 h-4 text-purple-600 dark:text-purple-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 7h8m0 0v8m0-8l-8 8-4-4-6 6"></path>
                  </svg>
                </div>
                <span class="text-sm text-gray-900 dark:text-gray-100">{{ skill.name }}</span>
              </div>
              <span class="text-xs text-gray-500 dark:text-gray-400">{{ skill.downloads }} 下载</span>
            </div>
          </div>
        </div>

        <div class="p-3 bg-white dark:bg-gray-900 rounded-lg">
          <h5 class="font-medium text-gray-900 dark:text-gray-100 mb-2">最近更新</h5>
          <div class="space-y-2">
            <div v-for="skill in recentSkills" :key="skill.id" class="flex items-center justify-between p-2 hover:bg-gray-50 dark:hover:bg-gray-800 rounded cursor-pointer" @click="showSkillDetails(skill)">
              <div class="flex items-center space-x-2">
                <div class="w-6 h-6 bg-green-100 dark:bg-green-900/30 rounded flex items-center justify-center">
                  <svg class="w-4 h-4 text-green-600 dark:text-green-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"></path>
                  </svg>
                </div>
                <span class="text-sm text-gray-900 dark:text-gray-100">{{ skill.name }}</span>
              </div>
              <span class="text-xs text-gray-500 dark:text-gray-400">{{ skill.updatedAt }}</span>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 技能执行历史 -->
    <div class="mb-4 p-4 bg-gray-50 dark:bg-gray-800 rounded-lg">
      <div class="flex items-center justify-between mb-3">
        <h4 class="font-medium text-gray-900 dark:text-gray-100">执行历史</h4>
        <button @click="clearHistory" class="text-red-500 hover:text-red-600 text-sm">清空历史</button>
      </div>

      <div v-if="executionHistory.length === 0" class="text-center py-4 text-gray-500 dark:text-gray-400">暂无执行历史</div>

      <div v-else class="space-y-2 max-h-40 overflow-y-auto">
        <div v-for="record in executionHistory" :key="record.id" class="flex items-center justify-between p-2 bg-white dark:bg-gray-900 rounded">
          <div class="flex items-center space-x-2">
            <div :class="getStatusClass(record.status)" class="w-2 h-2 rounded-full"></div>
            <span class="text-sm text-gray-900 dark:text-gray-100">{{ record.skillName }}</span>
          </div>
          <div class="flex items-center space-x-2">
            <span class="text-xs text-gray-500 dark:text-gray-400">{{ formatTime(record.timestamp) }}</span>
            <button @click="viewExecutionResult(record)" class="text-blue-500 hover:text-blue-600 text-sm">查看</button>
          </div>
        </div>
      </div>
    </div>

    <!-- 技能详情对话框 -->
    <div v-if="selectedSkill" class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4">
      <div class="bg-white dark:bg-gray-800 rounded-lg max-w-4xl w-full max-h-[80vh] overflow-y-auto">
        <div class="p-6">
          <div class="flex items-center justify-between mb-4">
            <div class="flex items-center space-x-3">
              <div :class="getSkillIconClass(selectedSkill.category)" class="w-12 h-12 rounded-lg flex items-center justify-center">
                <svg class="w-8 h-8 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" :d="getSkillIcon(selectedSkill.category)"></path>
                </svg>
              </div>
              <div>
                <h3 class="text-xl font-semibold text-gray-900 dark:text-gray-100">{{ selectedSkill.name }}</h3>
                <p class="text-gray-500 dark:text-gray-400">{{ selectedSkill.category }} • v{{ selectedSkill.version }}</p>
              </div>
            </div>
            <button @click="selectedSkill = null" class="text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200">
              <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path>
              </svg>
            </button>
          </div>

          <div class="space-y-4">
            <!-- 描述 -->
            <div>
              <h4 class="font-medium text-gray-900 dark:text-gray-100 mb-2">描述</h4>
              <p class="text-gray-600 dark:text-gray-400">{{ selectedSkill.description }}</p>
            </div>

            <!-- 作者信息 -->
            <div v-if="selectedSkill.author">
              <h4 class="font-medium text-gray-900 dark:text-gray-100 mb-2">作者信息</h4>
              <div class="flex items-center space-x-3">
                <div class="w-8 h-8 bg-gray-200 dark:bg-gray-700 rounded-full flex items-center justify-center">
                  <svg class="w-4 h-4 text-gray-600 dark:text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z"></path>
                  </svg>
                </div>
                <div>
                  <div class="font-medium text-gray-900 dark:text-gray-100">{{ selectedSkill.author.name }}</div>
                  <div class="text-sm text-gray-500 dark:text-gray-400">{{ selectedSkill.author.email }}</div>
                </div>
              </div>
            </div>

            <!-- 参数说明 -->
            <div v-if="selectedSkill.parameters && selectedSkill.parameters.length > 0">
              <h4 class="font-medium text-gray-900 dark:text-gray-100 mb-2">参数说明</h4>
              <div class="space-y-2">
                <div v-for="param in selectedSkill.parameters" :key="param.name" class="p-3 bg-gray-50 dark:bg-gray-900 rounded-lg">
                  <div class="flex items-center justify-between mb-1">
                    <span class="font-medium text-gray-900 dark:text-gray-100">{{ param.name }}</span>
                    <span class="px-2 py-1 bg-blue-100 dark:bg-blue-900/30 text-blue-600 dark:text-blue-400 rounded text-xs">
                      {{ param.type }}
                    </span>
                  </div>
                  <p class="text-sm text-gray-600 dark:text-gray-400">{{ param.description }}</p>
                  <p v-if="param.defaultValue" class="text-xs text-gray-500 dark:text-gray-400">默认值: {{ param.defaultValue }}</p>
                </div>
              </div>
            </div>

            <!-- 使用示例 -->
            <div v-if="selectedSkill.examples">
              <h4 class="font-medium text-gray-900 dark:text-gray-100 mb-2">使用示例</h4>
              <div class="space-y-2">
                <div v-for="example in selectedSkill.examples" :key="example.title" class="p-3 bg-gray-50 dark:bg-gray-900 rounded-lg">
                  <h5 class="font-medium text-gray-900 dark:text-gray-100 mb-1">{{ example.title }}</h5>
                  <p class="text-sm text-gray-600 dark:text-gray-400 mb-2">{{ example.description }}</p>
                  <div class="p-2 bg-gray-900 rounded font-mono text-sm text-green-400">
                    {{ example.command }}
                  </div>
                </div>
              </div>
            </div>

            <!-- 评分和统计 -->
            <div v-if="selectedSkill.rating || selectedSkill.downloads">
              <h4 class="font-medium text-gray-900 dark:text-gray-100 mb-2">评分和统计</h4>
              <div class="grid grid-cols-3 gap-4">
                <div class="text-center">
                  <div class="text-2xl font-bold text-yellow-500">⭐ {{ selectedSkill.rating?.toFixed(1) || "N/A" }}</div>
                  <div class="text-sm text-gray-500 dark:text-gray-400">用户评分</div>
                </div>
                <div class="text-center">
                  <div class="text-2xl font-bold text-blue-500">{{ selectedSkill.downloads || 0 }}</div>
                  <div class="text-sm text-gray-500 dark:text-gray-400">下载次数</div>
                </div>
                <div class="text-center">
                  <div class="text-2xl font-bold text-green-500">{{ selectedSkill.stars || 0 }}</div>
                  <div class="text-sm text-gray-500 dark:text-gray-400">收藏次数</div>
                </div>
              </div>
            </div>
          </div>

          <div class="flex justify-end space-x-3 mt-6">
            <button v-if="!selectedSkill.installed" @click="installSkill(selectedSkill)" :disabled="installing" class="px-4 py-2 bg-blue-500 text-white rounded-lg hover:bg-blue-600 disabled:opacity-50">
              {{ installing ? "安装中..." : "安装技能" }}
            </button>
            <button v-else @click="executeSkill(selectedSkill)" :disabled="executing" class="px-4 py-2 bg-purple-500 text-white rounded-lg hover:bg-purple-600 disabled:opacity-50">
              {{ executing ? "执行中..." : "执行技能" }}
            </button>
            <button @click="selectedSkill = null" class="px-4 py-2 bg-gray-500 text-white rounded-lg hover:bg-gray-600">关闭</button>
          </div>
        </div>
      </div>
    </div>

    <!-- 执行结果对话框 -->
    <div v-if="executionResult" class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4">
      <div class="bg-white dark:bg-gray-800 rounded-lg max-w-4xl w-full max-h-[80vh] overflow-y-auto">
        <div class="p-6">
          <div class="flex items-center justify-between mb-4">
            <h3 class="text-lg font-semibold text-gray-900 dark:text-gray-100">执行结果</h3>
            <button @click="executionResult = null" class="text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200">
              <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path>
              </svg>
            </button>
          </div>

          <div class="space-y-4">
            <div class="flex items-center space-x-2">
              <div :class="getStatusClass(executionResult.status)" class="w-3 h-3 rounded-full"></div>
              <span class="font-medium text-gray-900 dark:text-gray-100">{{ executionResult.skillName }}</span>
              <span class="text-sm text-gray-500 dark:text-gray-400">• {{ formatTime(executionResult.timestamp) }}</span>
            </div>

            <div v-if="executionResult.output">
              <h4 class="font-medium text-gray-900 dark:text-gray-100 mb-2">输出结果</h4>
              <div class="p-3 bg-gray-50 dark:bg-gray-900 rounded-lg font-mono text-sm whitespace-pre-wrap text-gray-900 dark:text-gray-100 max-h-60 overflow-y-auto">
                {{ executionResult.output }}
              </div>
            </div>

            <div v-if="executionResult.error">
              <h4 class="font-medium text-red-600 dark:text-red-400 mb-2">错误信息</h4>
              <div class="p-3 bg-red-50 dark:bg-red-900/20 rounded-lg font-mono text-sm text-red-800 dark:text-red-200 max-h-60 overflow-y-auto">
                {{ executionResult.error }}
              </div>
            </div>

            <div v-if="executionResult.duration">
              <h4 class="font-medium text-gray-900 dark:text-gray-100 mb-2">执行信息</h4>
              <div class="grid grid-cols-2 gap-4 text-sm">
                <div>
                  <span class="text-gray-500 dark:text-gray-400">执行时长:</span>
                  <span class="ml-2 text-gray-900 dark:text-gray-100">{{ executionResult.duration }}ms</span>
                </div>
                <div>
                  <span class="text-gray-500 dark:text-gray-400">内存使用:</span>
                  <span class="ml-2 text-gray-900 dark:text-gray-100">{{ executionResult.memoryUsage || "N/A" }}</span>
                </div>
              </div>
            </div>
          </div>

          <div class="flex justify-end mt-6">
            <button @click="executionResult = null" class="px-4 py-2 bg-gray-500 text-white rounded-lg hover:bg-gray-600">关闭</button>
          </div>
        </div>
      </div>
    </div>

    <!-- 操作结果提示 -->
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
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from "vue";

interface Skill {
  id: string;
  name: string;
  category: string;
  description: string;
  version: string;
  installed: boolean;
  favorite: boolean;
  rating?: number;
  downloads?: number;
  stars?: number;
  usageCount?: number;
  updatedAt?: string;
  parameters?: Array<{
    name: string;
    type: string;
    description: string;
    required: boolean;
    defaultValue?: string;
  }>;
  examples?: Array<{
    title: string;
    description: string;
    command: string;
  }>;
  author?: {
    name: string;
    email: string;
  };
}

interface ExecutionRecord {
  id: string;
  skillName: string;
  skillId: string;
  status: "success" | "error" | "running";
  timestamp: Date;
  output?: string;
  error?: string;
  duration?: number;
  memoryUsage?: string;
}

const props = defineProps<{
  socket?: WebSocket;
  sessionId?: string;
}>();

const emit = defineEmits<{
  execute: [
    {
      command: "skill-call";
      parameters: {
        skill: string;
        parameters?: Record<string, any>;
      };
    },
  ];
}>();

// 状态管理
const loading = ref(false);
const installing = ref(false);
const executing = ref(false);
const skills = ref<Skill[]>([]);
const searchQuery = ref("");
const categoryFilter = ref("all");
const sortBy = ref("name");
const showFavorites = ref(false);
const showInstalled = ref(false);
const showBuiltin = ref(true);
const showSkillStore = ref(false);
const selectedSkill = ref<Skill | null>(null);
const executionResult = ref<ExecutionRecord | null>(null);
const executionHistory = ref<ExecutionRecord[]>([]);
const actionResult = ref<{ type: "success" | "error"; message: string } | null>(null);

// 示例数据
const popularSkills = ref<Skill[]>([
  { id: "skill1", name: "文件转换器", category: "data", description: "支持多种文件格式转换", version: "1.0.0", installed: false, favorite: false, downloads: 1234 },
  { id: "skill2", name: "API测试器", category: "development", description: "RESTful API测试工具", version: "2.1.0", installed: true, favorite: true, downloads: 987 },
  { id: "skill3", name: "代码生成器", category: "development", description: "基于模板的代码生成", version: "1.5.0", installed: false, favorite: false, downloads: 756 },
]);

const recentSkills = ref<Skill[]>([
  { id: "skill4", name: "文档解析器", category: "productivity", description: "智能文档内容解析", version: "1.2.0", installed: true, favorite: false, updatedAt: "2天前" },
  { id: "skill5", name: "图像压缩器", category: "media", description: "批量图像压缩优化", version: "2.0.1", installed: false, favorite: true, updatedAt: "1周前" },
  { id: "skill6", name: "日志分析器", category: "system", description: "系统日志智能分析", version: "1.8.0", installed: true, favorite: false, updatedAt: "2周前" },
]);

// 过滤后的技能列表
const filteredSkills = computed(() => {
  let filtered = skills.value;

  // 搜索过滤
  if (searchQuery.value) {
    const query = searchQuery.value.toLowerCase();
    filtered = filtered.filter((skill) => skill.name.toLowerCase().includes(query) || skill.description.toLowerCase().includes(query) || skill.category.toLowerCase().includes(query));
  }

  // 分类过滤
  if (categoryFilter.value !== "all") {
    filtered = filtered.filter((skill) => skill.category === categoryFilter.value);
  }

  // 收藏过滤
  if (showFavorites.value) {
    filtered = filtered.filter((skill) => skill.favorite);
  }

  // 安装状态过滤
  if (showInstalled.value) {
    filtered = filtered.filter((skill) => skill.installed);
  }

  // 内置技能过滤
  if (!showBuiltin.value) {
    filtered = filtered.filter((skill) => skill.category !== "builtin");
  }

  // 排序
  if (sortBy.value === "name") {
    filtered = [...filtered].sort((a, b) => a.name.localeCompare(b.name));
  } else if (sortBy.value === "usage") {
    filtered = [...filtered].sort((a, b) => (b.usageCount || 0) - (a.usageCount || 0));
  } else if (sortBy.value === "rating") {
    filtered = [...filtered].sort((a, b) => (b.rating || 0) - (a.rating || 0));
  } else if (sortBy.value === "recent") {
    filtered = [...filtered].sort((a, b) => {
      const recentA = executionHistory.value.filter((r) => r.skillId === a.id)[0]?.timestamp || new Date(0);
      const recentB = executionHistory.value.filter((r) => r.skillId === b.id)[0]?.timestamp || new Date(0);
      return recentB.getTime() - recentA.getTime();
    });
  }

  return filtered;
});

// 获取技能图标
const getSkillIcon = (category: string): string => {
  switch (category) {
    case "development":
      return "M10 20l4-16m4 4l4 4-4 4M6 16l-4-4 4-4";
    case "productivity":
      return "M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-3 7h3m-3 4h3m-6-4h.01M9 16h.01";
    case "data":
      return "M9 17v1a1 1 0 001 1h4a1 1 0 001-1v-1m3-2V8a2 2 0 00-2-2H8a2 2 0 00-2 2v8a2 2 0 002 2h4a2 2 0 002-2zm-1-9a1 1 0 011 1v6a1 1 0 11-2 0V8a1 1 0 011-1z";
    case "media":
      return "M15 10l4.553-2.276A1 1 0 0121 8.618v6.764a1 1 0 01-1.447.894L15 14M5 18h8a2 2 0 002-2V8a2 2 0 00-2-2H5a2 2 0 00-2 2v8a2 2 0 002 2z";
    case "system":
      return "M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z";
    case "security":
      return "M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z";
    case "automation":
      return "M13 10V3L4 14h7v7l9-11h-7z";
    default:
      return "M19.428 15.428a2 2 0 00-1.022-.547l-2.387-.477a6 6 0 00-3.86.517l-.318.158a6 6 0 01-3.86.517L6.05 15.21a2 2 0 00-1.806.547M8 4h8l-1 1v5.172a2 2 0 00.586 1.414l5 5c1.26 1.26.367 3.414-1.415 3.414H4.828c-1.782 0-2.674-2.154-1.414-3.414l5-5A2 2 0 009 10.172V5L8 4z";
  }
};

// 获取技能图标样式
const getSkillIconClass = (category: string): string => {
  switch (category) {
    case "development":
      return "bg-blue-500";
    case "productivity":
      return "bg-green-500";
    case "data":
      return "bg-yellow-500";
    case "media":
      return "bg-pink-500";
    case "system":
      return "bg-gray-500";
    case "security":
      return "bg-red-500";
    case "automation":
      return "bg-purple-500";
    default:
      return "bg-indigo-500";
  }
};

// 获取使用次数
const getUsageCount = (skill: Skill): number => {
  return skill.usageCount || executionHistory.value.filter((r) => r.skillId === skill.id).length;
};

// 获取状态样式
const getStatusClass = (status: string): string => {
  switch (status) {
    case "success":
      return "bg-green-500";
    case "error":
      return "bg-red-500";
    case "running":
      return "bg-blue-500";
    default:
      return "bg-gray-500";
  }
};

// 格式化时间
const formatTime = (timestamp: Date): string => {
  return new Intl.DateTimeFormat("zh-CN", {
    year: "numeric",
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  }).format(timestamp);
};

// 加载技能列表
const loadSkills = async () => {
  loading.value = true;
  try {
    const response = await fetch("/api/skills");
    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`);
    }
    const data = await response.json();
    skills.value = data.skills || [];

    // 添加示例数据
    if (skills.value.length === 0) {
      skills.value = [
        {
          id: "builtin-file-converter",
          name: "文件转换器",
          category: "data",
          description: "支持多种文件格式之间的转换，包括PDF、Word、Excel、图片等格式",
          version: "1.0.0",
          installed: true,
          favorite: true,
          rating: 4.5,
          downloads: 1234,
          usageCount: 89,
        },
        {
          id: "builtin-api-tester",
          name: "API测试器",
          category: "development",
          description: "RESTful API测试工具，支持多种HTTP方法和认证方式",
          version: "2.1.0",
          installed: true,
          favorite: false,
          rating: 4.2,
          downloads: 987,
          usageCount: 56,
        },
        {
          id: "market-code-generator",
          name: "代码生成器",
          category: "development",
          description: "基于模板和AI的代码生成工具，支持多种编程语言",
          version: "1.5.0",
          installed: false,
          favorite: true,
          rating: 4.8,
          downloads: 756,
          usageCount: 0,
        },
      ];
    }
  } catch (error) {
    console.error("Error loading skills:", error);
    showActionResult("error", "加载技能列表失败");
  } finally {
    loading.value = false;
  }
};

// 刷新技能列表
const refreshSkills = () => {
  loadSkills();
};

// 切换收藏状态
const toggleFavorite = async (skill: Skill) => {
  skill.favorite = !skill.favorite;
  showActionResult("success", skill.favorite ? "已添加到收藏" : "已从收藏移除");
};

// 安装技能
const installSkill = async (skill: Skill) => {
  installing.value = true;
  try {
    const response = await fetch(`/api/skills/${skill.id}/install`, {
      method: "POST",
    });

    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`);
    }

    skill.installed = true;
    showActionResult("success", `技能 "${skill.name}" 安装成功`);
  } catch (error) {
    console.error("Error installing skill:", error);
    showActionResult("error", "安装技能失败");
  } finally {
    installing.value = false;
  }
};

// 执行技能
const executeSkill = async (skill: Skill) => {
  if (!skill.installed) {
    showActionResult("error", "请先安装该技能");
    return;
  }

  executing.value = true;
  try {
    const record: ExecutionRecord = {
      id: Date.now().toString(),
      skillName: skill.name,
      skillId: skill.id,
      status: "running",
      timestamp: new Date(),
    };

    executionHistory.value.unshift(record);

    emit("execute", {
      command: "skill-call",
      parameters: {
        skill: skill.id,
      },
    });

    showActionResult("success", `正在执行技能 "${skill.name}"`);

    // 模拟执行完成
    setTimeout(() => {
      const index = executionHistory.value.findIndex((r) => r.id === record.id);
      const historyRecord = executionHistory.value[index];
      if (index !== -1 && historyRecord) {
        historyRecord.status = "success";
        historyRecord.output = "技能执行成功，结果已生成。";
        historyRecord.duration = Math.floor(Math.random() * 5000) + 1000;
      }

      if (skill.usageCount !== undefined) {
        skill.usageCount++;
      }

      executing.value = false;
    }, 2000);
  } catch (error) {
    console.error("Error executing skill:", error);
    showActionResult("error", "执行技能失败");
    executing.value = false;
  }
};

// 显示技能详情
const showSkillDetails = (skill: Skill) => {
  selectedSkill.value = skill;
};

// 查看执行结果
const viewExecutionResult = (record: ExecutionRecord) => {
  executionResult.value = record;
};

// 清空历史记录
const clearHistory = () => {
  executionHistory.value = [];
  showActionResult("success", "已清空执行历史");
};

// 显示操作结果
const showActionResult = (type: "success" | "error", message: string) => {
  actionResult.value = { type, message };
  setTimeout(() => {
    actionResult.value = null;
  }, 3000);
};

// 监听 WebSocket 消息
const handleSocketMessage = (event: MessageEvent) => {
  try {
    const data = JSON.parse(event.data);
    if (data.type === "skill_result") {
      const index = executionHistory.value.findIndex((r) => r.id === data.id);
      const historyRecord = executionHistory.value[index];
      if (index !== -1 && historyRecord) {
        historyRecord.status = data.success ? "success" : "error";
        historyRecord.output = data.output;
        historyRecord.error = data.error;
        historyRecord.duration = data.duration;
      }
    }
  } catch (error) {
    console.error("Error parsing WebSocket message:", error);
  }
};

// 生命周期
onMounted(() => {
  loadSkills();

  // 监听 WebSocket
  if (props.socket) {
    props.socket.addEventListener("message", handleSocketMessage);
  }
});

onUnmounted(() => {
  // 移除 WebSocket 监听
  if (props.socket) {
    props.socket.removeEventListener("message", handleSocketMessage);
  }
});
</script>

<style scoped>
/* 自定义样式 */
.skill-call-tool {
  @apply space-y-4;
}

/* 文本截断 */
.line-clamp-2 {
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
</style>

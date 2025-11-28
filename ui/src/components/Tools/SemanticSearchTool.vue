<template>
  <div class="semantic-search-tool">
    <!-- 头部工具栏 -->
    <div class="search-header">
      <div class="header-title">
        <Icon type="brain" size="sm" />
        <span>语义搜索</span>
      </div>
      <div class="header-actions">
        <button
          class="action-button"
          title="搜索设置"
          @click="toggleSettings"
        >
          <Icon type="settings" size="sm" />
        </button>
        <button
          class="action-button"
          title="向量可视化"
          @click="toggleVisualization"
        >
          <Icon type="bar-chart" size="sm" />
        </button>
      </div>
    </div>

    <!-- 设置面板 -->
    <div v-if="showSettings" class="settings-panel">
      <div class="settings-content">
        <h4>语义搜索设置</h4>
        <div class="setting-group">
          <label>向量模型</label>
          <select v-model="searchSettings.model" class="setting-select">
            <option value="text-embedding-ada-002">OpenAI Ada-002</option>
            <option value="text-embedding-3-small">OpenAI Embedding-3-Small</option>
            <option value="text-embedding-3-large">OpenAI Embedding-3-Large</option>
            <option value="sentence-transformers">Sentence Transformers</option>
            <option value="custom">自定义模型</option>
          </select>
        </div>
        <div class="setting-group">
          <label>搜索范围</label>
          <select v-model="searchSettings.scope" class="setting-select">
            <option value="workspace">工作空间</option>
            <option value="project">当前项目</option>
            <option value="documents">文档库</option>
            <option value="web">网络内容</option>
            <option value="custom">自定义范围</option>
          </select>
        </div>
        <div class="setting-group">
          <label>相似度阈值</label>
          <input
            v-model.number="searchSettings.threshold"
            type="range"
            min="0"
            max="1"
            step="0.1"
            class="setting-range"
          />
          <span class="threshold-value">{{ searchSettings.threshold }}</span>
        </div>
        <div class="setting-group">
          <label>结果数量</label>
          <select v-model="searchSettings.maxResults" class="setting-select">
            <option value="5">5条</option>
            <option value="10">10条</option>
            <option value="20">20条</option>
            <option value="50">50条</option>
          </select>
        </div>
        <div class="setting-group">
          <label>
            <input
              v-model="searchSettings.includeMetadata"
              type="checkbox"
              class="setting-checkbox"
            />
            包含元数据
          </label>
        </div>
        <div class="setting-group">
          <label>
            <input
              v-model="searchSettings.enableClustering"
              type="checkbox"
              class="setting-checkbox"
            />
            启用结果聚类
          </label>
        </div>
        <div class="setting-actions">
          <button class="setting-btn" @click="resetSettings">重置</button>
          <button class="setting-btn primary" @click="saveSettings">保存</button>
        </div>
      </div>
    </div>

    <!-- 搜索区域 -->
    <div class="search-area">
      <div class="search-input-section">
        <div class="search-mode-selector">
          <button
            v-for="mode in searchModes"
            :key="mode.key"
            :class="['mode-btn', { active: currentMode === mode.key }]"
            @click="currentMode = mode.key"
          >
            <Icon :type="(mode.icon as any)" size="xs" />
            {{ mode.label }}
          </button>
        </div>

        <div class="search-input-wrapper">
          <Icon type="search" size="sm" class="search-icon" />
          <textarea
            v-model="searchQuery"
            ref="searchInput"
            :placeholder="getSearchPlaceholder()"
            class="search-textarea"
            rows="3"
            @keydown.ctrl.enter="performSearch"
            @input="handleSearchInput"
          ></textarea>
          <button
            class="search-button"
            :disabled="!searchQuery.trim() || isSearching"
            @click="performSearch"
          >
            <Icon v-if="isSearching" type="spinner" size="sm" class="animate-spin" />
            <Icon v-else type="search" size="sm" />
          </button>
        </div>
      </div>

      <!-- 搜索过滤器 -->
      <div class="search-filters">
        <div class="filter-group">
          <label class="filter-label">内容类型:</label>
          <div class="filter-options">
            <label
              v-for="type in contentTypes"
              :key="type.value"
              class="filter-option"
            >
              <input
                v-model="selectedContentTypes"
                type="checkbox"
                :value="type.value"
                class="filter-checkbox"
              />
              <Icon :type="(type.icon as any)" size="xs" />
              {{ type.label }}
            </label>
          </div>
        </div>

        <div class="filter-group">
          <label class="filter-label">时间范围:</label>
          <select v-model="timeRange" class="filter-select">
            <option value="">不限</option>
            <option value="1d">今天</option>
            <option value="7d">过去一周</option>
            <option value="30d">过去一月</option>
            <option value="90d">过去三月</option>
            <option value="1y">过去一年</option>
          </select>
        </div>

        <div class="filter-group">
          <label class="filter-label">文件大小:</label>
          <select v-model="fileSize" class="filter-select">
            <option value="">不限</option>
            <option value="<1MB">小于1MB</option>
            <option value="1-10MB">1-10MB</option>
            <option value="10-100MB">10-100MB</option>
            <option value=">100MB">大于100MB</option>
          </select>
        </div>
      </div>
    </div>

    <!-- 向量可视化 -->
    <div v-if="showVisualization && searchResults.length > 0" class="visualization-panel">
      <div class="viz-header">
        <h4>向量可视化</h4>
        <button
          class="viz-close-btn"
          @click="showVisualization = false"
        >
          <Icon type="close" size="xs" />
        </button>
      </div>
      <div class="viz-content">
        <canvas ref="vectorCanvas" class="vector-canvas"></canvas>
        <div class="viz-legend">
          <div class="legend-item">
            <div class="legend-color search-color"></div>
            <span>搜索查询</span>
          </div>
          <div class="legend-item">
            <div class="legend-color result-color"></div>
            <span>搜索结果</span>
          </div>
        </div>
      </div>
    </div>

    <!-- 搜索状态 -->
    <div v-if="isSearching" class="search-status">
      <div class="status-content">
        <Icon type="brain" size="lg" class="animate-pulse" />
        <p>正在进行语义分析...</p>
        <div class="status-progress">
          <div class="progress-bar">
            <div class="progress-fill" :style="{ width: searchProgress + '%' }"></div>
          </div>
          <span class="progress-text">{{ searchProgress }}%</span>
        </div>
      </div>
    </div>

    <!-- 搜索结果 -->
    <div v-if="searchResults.length > 0" class="search-results">
      <!-- 结果统计 -->
      <div class="results-header">
        <div class="results-info">
          <span class="results-count">找到 {{ searchResults.length }} 个相似内容</span>
          <span class="search-time">耗时 {{ searchTime }}ms</span>
          <span class="avg-similarity">平均相似度: {{ averageSimilarity.toFixed(3) }}</span>
        </div>
        <div class="results-actions">
          <button
            class="action-btn"
            title="导出结果"
            @click="exportResults"
          >
            <Icon type="download" size="xs" />
          </button>
          <button
            class="action-btn"
            title="聚类分析"
            @click="showClusters = !showClusters"
          >
            <Icon type="layers" size="xs" />
          </button>
          <button
            class="action-btn"
            title="重新搜索"
            @click="performSearch"
          >
            <Icon type="refresh" size="xs" />
          </button>
        </div>
      </div>

      <!-- 聚类视图 -->
      <div v-if="showClusters && clusters.length > 0" class="clusters-view">
        <div
          v-for="(cluster, index) in clusters"
          :key="index"
          class="cluster-group"
        >
          <div class="cluster-header">
            <div class="cluster-title">
              <Icon type="folder" size="sm" />
              <span>{{ cluster.name }}</span>
              <span class="cluster-count">({{ cluster.items.length }})</span>
            </div>
            <div class="cluster-similarity">
              相似度: {{ cluster.avgSimilarity.toFixed(3) }}
            </div>
          </div>
          <div class="cluster-items">
            <div
              v-for="(item, itemIndex) in cluster.items"
              :key="itemIndex"
              class="cluster-result-item"
              @click="openResult(item)"
            >
              <div class="cluster-result-title">{{ item.title }}</div>
              <div class="cluster-result-score">相似度: {{ item.similarity.toFixed(3) }}</div>
            </div>
          </div>
        </div>
      </div>

      <!-- 列表视图 -->
      <div v-else class="results-list">
        <div
          v-for="(result, index) in searchResults"
          :key="index"
          class="result-item"
          @click="openResult(result)"
        >
          <div class="result-header">
            <div class="result-title">
              <h3>{{ result.title }}</h3>
            </div>
            <div class="result-similarity">
              <div class="similarity-bar">
                <div
                  class="similarity-fill"
                  :style="{ width: (result.similarity * 100) + '%' }"
                ></div>
              </div>
              <span class="similarity-text">{{ (result.similarity * 100).toFixed(1) }}%</span>
            </div>
          </div>

          <div class="result-content">
            <div class="result-snippet">{{ result.snippet }}</div>

            <!-- 关键词高亮 -->
            <div v-if="result.keywords && result.keywords.length > 0" class="result-keywords">
              <div class="keywords-label">关键词:</div>
              <div class="keywords-list">
                <span
                  v-for="(keyword, kIndex) in result.keywords"
                  :key="kIndex"
                  class="keyword-tag"
                  :style="{ fontSize: getKeywordFontSize(keyword.score) }"
                >
                  {{ keyword.word }}
                </span>
              </div>
            </div>

            <!-- 向量信息 -->
            <div v-if="result.vectorInfo" class="vector-info">
              <div class="vector-dimensions">维度: {{ result.vectorInfo.dimensions }}</div>
              <div class="vector-distance">距离: {{ result.vectorInfo.distance.toFixed(6) }}</div>
            </div>
          </div>

          <div class="result-meta">
            <div class="meta-item">
              <Icon type="file" size="xs" />
              <span class="meta-value">{{ result.fileType }}</span>
            </div>
            <div class="meta-item">
              <Icon type="folder" size="xs" />
              <span class="meta-value">{{ result.path }}</span>
            </div>
            <div v-if="result.size" class="meta-item">
              <Icon type="hard-drive" size="xs" />
              <span class="meta-value">{{ formatFileSize(result.size) }}</span>
            </div>
            <div v-if="result.modified" class="meta-item">
              <Icon type="clock" size="xs" />
              <span class="meta-value">{{ formatDate(result.modified) }}</span>
            </div>
            <div class="meta-item">
              <Icon type="hash" size="xs" />
              <span class="meta-value">{{ result.hash }}</span>
            </div>
          </div>

          <!-- 操作按钮 -->
          <div class="result-actions">
            <button
              class="result-action-btn"
              title="查看详情"
              @click.stop="showResultDetails(result)"
            >
              <Icon type="eye" size="xs" />
            </button>
            <button
              class="result-action-btn"
              title="复制内容"
              @click.stop="copyContent(result)"
            >
              <Icon type="copy" size="xs" />
            </button>
            <button
              class="result-action-btn"
              title="相关搜索"
              @click.stop="findSimilar(result)"
            >
              <Icon type="search" size="xs" />
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- 空状态 -->
    <div v-if="!isSearching && searchResults.length === 0 && searchQuery" class="empty-results">
      <Icon type="brain" size="lg" />
      <h3>未找到相似内容</h3>
      <p class="empty-hint">
        尝试调整搜索词或降低相似度阈值
      </p>
      <div class="suggestions">
        <p class="suggestions-title">建议:</p>
        <ul class="suggestions-list">
          <li>使用更具体的描述</li>
          <li>尝试不同的关键词组合</li>
          <li>检查内容类型筛选器</li>
          <li>降低相似度阈值</li>
        </ul>
      </div>
    </div>

    <!-- 初始状态 -->
    <div v-if="!isSearching && searchResults.length === 0 && !searchQuery" class="initial-state">
      <Icon type="brain" size="lg" />
      <h3>智能语义搜索</h3>
      <p class="initial-hint">
        基于AI的智能搜索，理解内容语义而非关键词匹配
      </p>

      <!-- 搜索示例 -->
      <div class="search-examples">
        <h4>搜索示例</h4>
        <div class="example-grid">
          <button
            v-for="(example, index) in searchExamples"
            :key="index"
            class="example-card"
            @click="useExample(example)"
          >
            <div class="example-icon">
              <Icon :type="(example.icon as any)" size="sm" />
            </div>
            <div class="example-content">
              <div class="example-title">{{ example.title }}</div>
              <div class="example-description">{{ example.description }}</div>
            </div>
          </button>
        </div>
      </div>

      <!-- 搜索统计 -->
      <div v-if="searchStats" class="search-stats">
        <h4>数据库统计</h4>
        <div class="stats-grid">
          <div class="stat-item">
            <div class="stat-value">{{ searchStats.totalDocuments }}</div>
            <div class="stat-label">文档总数</div>
          </div>
          <div class="stat-item">
            <div class="stat-value">{{ searchStats.totalVectors }}</div>
            <div class="stat-label">向量总数</div>
          </div>
          <div class="stat-item">
            <div class="stat-value">{{ searchStats.averageDimension }}</div>
            <div class="stat-label">平均维度</div>
          </div>
          <div class="stat-item">
            <div class="stat-value">{{ searchStats.lastUpdated }}</div>
            <div class="stat-label">最后更新</div>
          </div>
        </div>
      </div>
    </div>

    <!-- 结果详情模态框 -->
    <div v-if="showDetailsModal" class="modal-overlay" @click="closeDetailsModal">
      <div class="modal-content" @click.stop>
        <div class="modal-header">
          <h3>内容详情</h3>
          <button
            class="modal-close-btn"
            @click="closeDetailsModal"
          >
            <Icon type="close" size="sm" />
          </button>
        </div>
        <div class="modal-body">
          <div v-if="selectedResult" class="result-details">
            <div class="detail-section">
              <h4>基本信息</h4>
              <div class="detail-grid">
                <div class="detail-item">
                  <label>标题:</label>
                  <span>{{ selectedResult.title }}</span>
                </div>
                <div class="detail-item">
                  <label>路径:</label>
                  <span>{{ selectedResult.path }}</span>
                </div>
                <div class="detail-item">
                  <label>相似度:</label>
                  <span>{{ (selectedResult.similarity * 100).toFixed(2) }}%</span>
                </div>
                <div class="detail-item">
                  <label>向量维度:</label>
                  <span>{{ selectedResult.vectorInfo?.dimensions || 'N/A' }}</span>
                </div>
              </div>
            </div>

            <div class="detail-section">
              <h4>内容预览</h4>
              <div class="content-preview">
                <pre>{{ selectedResult.content }}</pre>
              </div>
            </div>

            <div v-if="selectedResult.keywords" class="detail-section">
              <h4>关键词权重</h4>
              <div class="keyword-analysis">
                <div
                  v-for="(keyword, index) in selectedResult.keywords"
                  :key="index"
                  class="keyword-analysis-item"
                >
                  <span class="keyword-word">{{ keyword.word }}</span>
                  <div class="keyword-score-bar">
                    <div
                      class="keyword-score-fill"
                      :style="{ width: (keyword.score * 100) + '%' }"
                    ></div>
                  </div>
                  <span class="keyword-score">{{ keyword.score.toFixed(3) }}</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick, onMounted, onUnmounted } from 'vue';
import Icon from '../ChatUI/Icon.vue';

interface SearchResult {
  id: string;
  title: string;
  content: string;
  snippet: string;
  path: string;
  fileType: string;
  size?: number;
  modified?: string;
  hash: string;
  similarity: number;
  keywords?: Array<{
    word: string;
    score: number;
  }>;
  vectorInfo?: {
    dimensions: number;
    distance: number;
  };
}

interface SearchCluster {
  name: string;
  avgSimilarity: number;
  items: SearchResult[];
}

interface SearchSettings {
  model: string;
  scope: string;
  threshold: number;
  maxResults: number;
  includeMetadata: boolean;
  enableClustering: boolean;
}

interface SearchStats {
  totalDocuments: number;
  totalVectors: number;
  averageDimension: number;
  lastUpdated: string;
}

interface SearchExample {
  title: string;
  description: string;
  icon: string;
  query: string;
}

interface Props {
  wsUrl?: string;
  sessionId?: string;
}

const props = withDefaults(defineProps<Props>(), {
  wsUrl: 'ws://localhost:8080/ws',
  sessionId: 'default',
});

const emit = defineEmits<{
  searchPerformed: [query: string, results: SearchResult[]];
  resultSelected: [result: SearchResult];
}>();

// 响应式数据
const searchQuery = ref('');
const currentMode = ref('semantic');
const searchResults = ref<SearchResult[]>([]);
const clusters = ref<SearchCluster[]>([]);
const isSearching = ref(false);
const searchProgress = ref(0);
const searchTime = ref(0);
const showSettings = ref(false);
const showVisualization = ref(false);
const showClusters = ref(false);
const showDetailsModal = ref(false);
const selectedResult = ref<SearchResult | null>(null);

// 搜索过滤器
const selectedContentTypes = ref(['text', 'code', 'markdown']);
const timeRange = ref('');
const fileSize = ref('');

// 搜索设置
const searchSettings = ref<SearchSettings>({
  model: 'text-embedding-ada-002',
  scope: 'workspace',
  threshold: 0.7,
  maxResults: 10,
  includeMetadata: true,
  enableClustering: true,
});

// 搜索模式
const searchModes = ref([
  { key: 'semantic', label: '语义', icon: 'brain' },
  { key: 'hybrid', label: '混合', icon: 'layers' },
  { key: 'keyword', label: '关键词', icon: 'type' },
]);

// 内容类型
const contentTypes = ref([
  { value: 'text', label: '文本', icon: 'file-text' },
  { value: 'code', label: '代码', icon: 'file-code' },
  { value: 'markdown', label: 'Markdown', icon: 'file-text' },
  { value: 'json', label: 'JSON', icon: 'file-code' },
  { value: 'config', label: '配置', icon: 'settings' },
  { value: 'documentation', label: '文档', icon: 'book-open' },
]);

// 搜索示例
const searchExamples = ref<SearchExample[]>([
  {
    title: '查找数据库配置',
    description: '搜索项目中所有数据库相关配置文件',
    icon: 'database',
    query: '数据库连接配置 settings database',
  },
  {
    title: '错误处理逻辑',
    description: '查找代码中的错误处理和异常管理',
    icon: 'alert-triangle',
    query: '错误处理 try catch exception error handling',
  },
  {
    title: '用户认证功能',
    description: '搜索用户登录、注册和权限管理相关代码',
    icon: 'shield',
    query: '用户认证 登录注册 权限管理 authentication',
  },
  {
    title: 'API接口文档',
    description: '查找REST API接口定义和文档说明',
    icon: 'globe',
    query: 'API接口 REST endpoint 文档 接口说明',
  },
]);

// 搜索统计
const searchStats = ref<SearchStats | null>(null);

const searchInput = ref<HTMLTextAreaElement>();
const vectorCanvas = ref<HTMLCanvasElement>();
const websocket = ref<WebSocket | null>(null);

// 计算属性
const averageSimilarity = computed(() => {
  if (searchResults.value.length === 0) return 0;
  const total = searchResults.value.reduce((sum, result) => sum + result.similarity, 0);
  return total / searchResults.value.length;
});

// WebSocket 连接
const connectWebSocket = () => {
  try {
    websocket.value = new WebSocket(`${props.wsUrl}?session=${props.sessionId}`);

    websocket.value.onopen = () => {
      console.log('SemanticSearchTool WebSocket connected');
      loadSearchStats();
    };

    websocket.value.onmessage = (event) => {
      try {
        const message = JSON.parse(event.data);
        handleWebSocketMessage(message);
      } catch (error) {
        console.error('Failed to parse WebSocket message:', error);
      }
    };

    websocket.value.onclose = () => {
      console.log('SemanticSearchTool WebSocket disconnected');
      setTimeout(connectWebSocket, 5000);
    };

    websocket.value.onerror = (error) => {
      console.error('SemanticSearchTool WebSocket error:', error);
    };
  } catch (error) {
    console.error('Failed to connect WebSocket:', error);
  }
};

const handleWebSocketMessage = (message: any) => {
  switch (message.type) {
    case 'semantic_search_results':
      if (message.results) {
        handleSearchResults(message.results);
      }
      break;
    case 'search_stats':
      if (message.stats) {
        searchStats.value = message.stats;
      }
      break;
  }
};

const sendWebSocketMessage = (message: any) => {
  if (websocket.value && websocket.value.readyState === WebSocket.OPEN) {
    websocket.value.send(JSON.stringify(message));
  }
};

// 搜索功能
const performSearch = async () => {
  if (!searchQuery.value.trim() || isSearching.value) return;

  isSearching.value = true;
  searchProgress.value = 0;
  const startTime = Date.now();

  try {
    // 模拟搜索进度
    const progressInterval = setInterval(() => {
      searchProgress.value = Math.min(searchProgress.value + 10, 90);
    }, 100);

    // 构建搜索参数
    const searchParams = {
      query: searchQuery.value.trim(),
      mode: currentMode.value,
      scope: searchSettings.value.scope,
      threshold: searchSettings.value.threshold,
      maxResults: searchSettings.value.maxResults,
      includeMetadata: searchSettings.value.includeMetadata,
      contentTypes: selectedContentTypes.value,
      timeRange: timeRange.value,
      fileSize: fileSize.value,
      model: searchSettings.value.model,
    };

    // 发送搜索请求
    sendWebSocketMessage({
      type: 'semantic_search',
      params: searchParams,
    });

    // 模拟搜索结果
    const mockResults = await mockSemanticSearch(searchParams);

    clearInterval(progressInterval);
    searchProgress.value = 100;

    const endTime = Date.now();
    searchTime.value = endTime - startTime;

    handleSearchResults(mockResults);

    // 启用聚类
    if (searchSettings.value.enableClustering && mockResults.length > 0) {
      generateClusters(mockResults);
    }

    emit('searchPerformed', searchQuery.value.trim(), mockResults);

  } catch (error) {
    console.error('Search failed:', error);
    searchResults.value = [];
  } finally {
    isSearching.value = false;
    setTimeout(() => {
      searchProgress.value = 0;
    }, 500);
  }
};

const mockSemanticSearch = async (params: any): Promise<SearchResult[]> => {
  // 模拟搜索延迟
  await new Promise(resolve => setTimeout(resolve, 1500 + Math.random() * 1000));

  // 生成模拟搜索结果
  const mockResults: SearchResult[] = [
    {
      id: '1',
      title: `与 "${params.query}" 相关的配置文件`,
      content: `这是关于 ${params.query} 的详细配置内容，包含了相关的参数设置和配置选项...`,
      snippet: `配置文件包含了 ${params.query} 相关的重要参数和设置`,
      path: '/config/app.json',
      fileType: 'json',
      size: 2048,
      modified: new Date(Date.now() - 86400000).toISOString(),
      hash: 'abc123def456',
      similarity: 0.92,
      keywords: [
        { word: params.query, score: 0.95 },
        { word: '配置', score: 0.87 },
        { word: '参数', score: 0.76 },
      ],
      vectorInfo: {
        dimensions: 1536,
        distance: 0.123456,
      },
    },
    {
      id: '2',
      title: `${params.query} 实现代码`,
      content: `这是一个实现 ${params.query} 功能的核心代码模块，包含了完整的业务逻辑和错误处理...`,
      snippet: `代码实现了 ${params.query} 的核心功能，具有良好的错误处理机制`,
      path: '/src/components/FeatureComponent.vue',
      fileType: 'vue',
      size: 8192,
      modified: new Date(Date.now() - 172800000).toISOString(),
      hash: 'def456ghi789',
      similarity: 0.88,
      keywords: [
        { word: params.query, score: 0.91 },
        { word: '实现', score: 0.82 },
        { word: '代码', score: 0.78 },
      ],
      vectorInfo: {
        dimensions: 1536,
        distance: 0.234567,
      },
    },
    {
      id: '3',
      title: `${params.query} 相关的文档说明`,
      content: `这份文档详细说明了 ${params.query} 的使用方法、最佳实践和常见问题解决方案...`,
      snippet: `文档提供了 ${params.query} 的完整使用指南和示例代码`,
      path: '/docs/api-reference.md',
      fileType: 'markdown',
      size: 4096,
      modified: new Date(Date.now() - 259200000).toISOString(),
      hash: 'ghi789jkl012',
      similarity: 0.85,
      keywords: [
        { word: params.query, score: 0.89 },
        { word: '文档', score: 0.84 },
        { word: '说明', score: 0.79 },
      ],
      vectorInfo: {
        dimensions: 1536,
        distance: 0.345678,
      },
    },
    {
      id: '4',
      title: `${params.query} 测试用例`,
      content: `这是一套完整的 ${params.query} 测试用例，包含了单元测试、集成测试和端到端测试...`,
      snippet: `测试用例覆盖了 ${params.query} 的各种使用场景和边界条件`,
      path: '/tests/feature.test.js',
      fileType: 'javascript',
      size: 6144,
      modified: new Date(Date.now() - 345600000).toISOString(),
      hash: 'jkl012mno345',
      similarity: 0.79,
      keywords: [
        { word: params.query, score: 0.83 },
        { word: '测试', score: 0.88 },
        { word: '用例', score: 0.75 },
      ],
      vectorInfo: {
        dimensions: 1536,
        distance: 0.456789,
      },
    },
    {
      id: '5',
      title: `${params.query} 性能优化`,
      content: `本文档介绍了 ${params.query} 的性能优化策略和技巧，包括缓存机制、数据库优化等...`,
      snippet: `性能优化方案显著提升了 ${params.query} 的执行效率和响应速度`,
      path: '/docs/performance.md',
      fileType: 'markdown',
      size: 3072,
      modified: new Date(Date.now() - 432000000).toISOString(),
      hash: 'mno345pqr678',
      similarity: 0.75,
      keywords: [
        { word: params.query, score: 0.80 },
        { word: '性能', score: 0.86 },
        { word: '优化', score: 0.82 },
      ],
      vectorInfo: {
        dimensions: 1536,
        distance: 0.567890,
      },
    },
  ];

  return mockResults.filter(result => result.similarity >= params.threshold);
};

const handleSearchResults = (results: SearchResult[]) => {
  searchResults.value = results.sort((a, b) => b.similarity - a.similarity);
};

// 聚类分析
const generateClusters = (results: SearchResult[]) => {
  // 简化的聚类算法
  const newClusters: SearchCluster[] = [];
  const usedIndices = new Set<number>();

  results.forEach((result, index) => {
    if (usedIndices.has(index)) return;

    // 基于文件类型和相似性创建聚类
    const clusterItems = [result];
    usedIndices.add(index);

    // 查找相似的其他结果
    results.forEach((otherResult, otherIndex) => {
      if (usedIndices.has(otherIndex) || index === otherIndex) return;

      // 简单的聚类逻辑：相同文件类型且相似度高的归为一类
      if (result.fileType === otherResult.fileType &&
          Math.abs(result.similarity - otherResult.similarity) < 0.1) {
        clusterItems.push(otherResult);
        usedIndices.add(otherIndex);
      }
    });

    if (clusterItems.length > 0) {
      const avgSimilarity = clusterItems.reduce((sum, item) => sum + item.similarity, 0) / clusterItems.length;

      newClusters.push({
        name: `${result.fileType} 相关内容`,
        avgSimilarity,
        items: clusterItems,
      });
    }
  });

  clusters.value = newClusters;
};

const findSimilar = async (result: SearchResult) => {
  searchQuery.value = result.snippet.substring(0, 100);
  await performSearch();
};

// 向量可视化
const drawVectorVisualization = () => {
  if (!vectorCanvas.value || searchResults.value.length === 0) return;

  const canvas = vectorCanvas.value;
  const ctx = canvas.getContext('2d');
  if (!ctx) return;

  // 设置画布大小
  canvas.width = canvas.offsetWidth;
  canvas.height = canvas.offsetHeight;

  // 清空画布
  ctx.clearRect(0, 0, canvas.width, canvas.height);

  // 绘制搜索查询点
  const centerX = canvas.width / 2;
  const centerY = canvas.height / 2;

  ctx.fillStyle = '#3B82F6';
  ctx.beginPath();
  ctx.arc(centerX, centerY, 8, 0, 2 * Math.PI);
  ctx.fill();

  // 绘制搜索结果点
  const angleStep = (2 * Math.PI) / searchResults.value.length;

  searchResults.value.forEach((result, index) => {
    const angle = index * angleStep;
    const distance = (1 - result.similarity) * 150; // 相似度越高，距离越近
    const x = centerX + Math.cos(angle) * distance;
    const y = centerY + Math.sin(angle) * distance;

    // 根据相似度设置颜色
    const opacity = result.similarity;
    ctx.fillStyle = `rgba(34, 197, 94, ${opacity})`;
    ctx.beginPath();
    ctx.arc(x, y, 5, 0, 2 * Math.PI);
    ctx.fill();

    // 绘制连接线
    ctx.strokeStyle = `rgba(156, 163, 175, ${opacity * 0.3})`;
    ctx.beginPath();
    ctx.moveTo(centerX, centerY);
    ctx.lineTo(x, y);
    ctx.stroke();
  });
};

// 结果操作
const openResult = (result: SearchResult) => {
  // 这里可以打开文件或跳转到相应位置
  console.log('Open result:', result);
  emit('resultSelected', result);
};

const showResultDetails = (result: SearchResult) => {
  selectedResult.value = result;
  showDetailsModal.value = true;
};

const closeDetailsModal = () => {
  showDetailsModal.value = false;
  selectedResult.value = null;
};

const copyContent = async (result: SearchResult) => {
  try {
    await navigator.clipboard.writeText(result.content);
  } catch (error) {
    console.error('Failed to copy content:', error);
  }
};

const exportResults = () => {
  const exportData = {
    query: searchQuery.value,
    mode: currentMode.value,
    settings: searchSettings.value,
    timestamp: new Date().toISOString(),
    results: searchResults.value,
    clusters: clusters.value,
    searchTime: searchTime.value,
    averageSimilarity: averageSimilarity.value,
  };

  const dataStr = JSON.stringify(exportData, null, 2);
  const blob = new Blob([dataStr], { type: 'application/json' });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = `semantic-search-${Date.now()}.json`;
  a.click();
  URL.revokeObjectURL(url);
};

// 工具方法
const getSearchPlaceholder = () => {
  const placeholders: Record<string, string> = {
    semantic: '输入自然语言描述，如："查找用户登录相关的代码和配置"',
    hybrid: '结合语义和关键词搜索，如："API接口文档 REST endpoint"',
    keyword: '输入关键词搜索，如："database config settings"',
  };
  return placeholders[currentMode.value] || '输入搜索内容...';
};

const getKeywordFontSize = (score: number): string => {
  if (score > 0.8) return '1.2em';
  if (score > 0.6) return '1em';
  if (score > 0.4) return '0.9em';
  return '0.8em';
};

const formatFileSize = (bytes: number): string => {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
};

const formatDate = (dateString: string): string => {
  const date = new Date(dateString);
  return date.toLocaleDateString('zh-CN');
};

const useExample = (example: SearchExample) => {
  searchQuery.value = example.query;
  nextTick(() => {
    performSearch();
  });
};

const handleSearchInput = () => {
  // 可以在这里实现实时搜索建议
};

const toggleSettings = () => {
  showSettings.value = !showSettings.value;
};

const toggleVisualization = () => {
  showVisualization.value = !showVisualization.value;
  if (showVisualization.value) {
    nextTick(() => {
      drawVectorVisualization();
    });
  }
};

// 设置管理
const saveSettings = () => {
  localStorage.setItem('semantic-search-settings', JSON.stringify(searchSettings.value));
  showSettings.value = false;
};

const resetSettings = () => {
  searchSettings.value = {
    model: 'text-embedding-ada-002',
    scope: 'workspace',
    threshold: 0.7,
    maxResults: 10,
    includeMetadata: true,
    enableClustering: true,
  };
};

const loadSearchStats = () => {
  sendWebSocketMessage({
    type: 'get_search_stats',
  });

  // 模拟搜索统计
  searchStats.value = {
    totalDocuments: 1247,
    totalVectors: 1247,
    averageDimension: 1536,
    lastUpdated: new Date().toLocaleDateString('zh-CN'),
  };
};

// 生命周期
onMounted(() => {
  connectWebSocket();

  // 加载设置
  try {
    const saved = localStorage.getItem('semantic-search-settings');
    if (saved) {
      searchSettings.value = { ...searchSettings.value, ...JSON.parse(saved) };
    }
  } catch (error) {
    console.warn('Failed to load search settings:', error);
  }

  // 自动聚焦搜索框
  nextTick(() => {
    searchInput.value?.focus();
  });
});

onUnmounted(() => {
  // 清理资源
});

// 监听可视化变化
watch(showVisualization, (newValue) => {
  if (newValue) {
    nextTick(() => {
      drawVectorVisualization();
    });
  }
});

// 监听搜索结果变化
watch(searchResults, () => {
  if (showVisualization.value) {
    nextTick(() => {
      drawVectorVisualization();
    });
  }
});
</script>

<style scoped>
.semantic-search-tool {
  @apply flex flex-col h-full bg-surface dark:bg-surface-dark border border-border dark:border-border-dark rounded-lg;
}

.search-header {
  @apply flex items-center justify-between px-4 py-3 border-b border-border dark:border-border-dark bg-surface dark:bg-surface-dark;
}

.header-title {
  @apply flex items-center gap-2 font-semibold text-text dark:text-text-dark;
}

.header-actions {
  @apply flex gap-1;
}

.action-button {
  @apply p-1.5 text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-700 rounded transition-colors;
}

.settings-panel {
  @apply border-b border-border dark:border-border-dark bg-gray-50 dark:bg-gray-700/30;
}

.settings-content {
  @apply p-4 space-y-4;
}

.settings-content h4 {
  @apply text-sm font-semibold text-text dark:text-text-dark mb-3;
}

.setting-group {
  @apply flex items-center justify-between;
}

.setting-group label {
  @apply text-sm text-gray-700 dark:text-gray-300;
}

.setting-select {
  @apply ml-2 px-2 py-1 text-sm border border-border dark:border-border-dark rounded focus:outline-none focus:ring-2 focus:ring-blue-500 dark:bg-gray-800 dark:text-white;
}

.setting-range {
  @apply flex-1 mx-2;
}

.threshold-value {
  @apply ml-2 text-sm font-mono text-gray-600 dark:text-gray-400;
}

.setting-actions {
  @apply flex gap-2 justify-end mt-4;
}

.setting-btn {
  @apply px-3 py-1 text-sm border border-border dark:border-border-dark rounded transition-colors;
}

.setting-btn.primary {
  @apply bg-blue-500 hover:bg-blue-600 text-white border-blue-500;
}

.search-area {
  @apply p-4 border-b border-gray-200 dark:border-gray-600 bg-white dark:bg-gray-800;
}

.search-input-section {
  @apply space-y-4;
}

.search-mode-selector {
  @apply flex gap-2;
}

.mode-btn {
  @apply flex items-center gap-2 px-4 py-2 text-sm text-gray-600 dark:text-gray-400 border border-gray-200 dark:border-gray-600 rounded-lg hover:text-gray-800 dark:hover:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-600 transition-colors;
}

.mode-btn.active {
  @apply text-blue-600 dark:text-blue-400 bg-blue-50 dark:bg-blue-900/20 border-blue-200 dark:border-blue-600;
}

.search-input-wrapper {
  @apply relative;
}

.search-icon {
  @apply absolute left-3 top-3 text-gray-400 dark:text-gray-500;
}

.search-textarea {
  @apply w-full pl-10 pr-12 py-3 border border-gray-200 dark:border-gray-600 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 dark:bg-gray-700 dark:text-white resize-none;
}

.search-button {
  @apply absolute right-3 top-3 p-2 text-blue-500 hover:text-blue-600 hover:bg-blue-50 dark:hover:bg-blue-900/20 rounded transition-colors disabled:opacity-50 disabled:cursor-not-allowed;
}

.search-filters {
  @apply flex gap-4 flex-wrap;
}

.filter-group {
  @apply flex items-center gap-2;
}

.filter-label {
  @apply text-sm font-medium text-gray-700 dark:text-gray-300;
}

.filter-options {
  @apply flex gap-3;
}

.filter-option {
  @apply flex items-center gap-1 text-sm text-gray-600 dark:text-gray-400 cursor-pointer hover:text-gray-800 dark:hover:text-gray-200;
}

.filter-checkbox {
  @apply mr-1;
}

.filter-select {
  @apply px-2 py-1 text-sm border border-gray-200 dark:border-gray-600 rounded focus:outline-none focus:ring-1 focus:ring-blue-500 dark:bg-gray-700 dark:text-white;
}

.visualization-panel {
  @apply border-b border-gray-200 dark:border-gray-600 bg-white dark:bg-gray-800;
}

.viz-header {
  @apply flex items-center justify-between p-4 border-b border-gray-200 dark:border-gray-600;
}

.viz-header h4 {
  @apply text-sm font-semibold text-gray-800 dark:text-gray-200;
}

.viz-close-btn {
  @apply p-1 text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-600 rounded transition-colors;
}

.viz-content {
  @apply p-4;
}

.vector-canvas {
  @apply w-full h-64 border border-gray-200 dark:border-gray-600 rounded;
}

.viz-legend {
  @apply flex gap-4 mt-2 text-xs;
}

.legend-item {
  @apply flex items-center gap-2;
}

.legend-color {
  @apply w-3 h-3 rounded-full;
}

.search-color {
  @apply bg-blue-500;
}

.result-color {
  @apply bg-green-500;
}

.search-status {
  @apply flex-1 flex items-center justify-center p-8;
}

.status-content {
  @apply flex flex-col items-center text-gray-500 dark:text-gray-400;
}

.status-progress {
  @apply flex items-center gap-3 mt-4 w-full max-w-xs;
}

.progress-bar {
  @apply flex-1 h-2 bg-gray-200 dark:bg-gray-600 rounded-full overflow-hidden;
}

.progress-fill {
  @apply h-full bg-blue-500 transition-all duration-300;
}

.progress-text {
  @apply text-sm font-mono;
}

.search-results {
  @apply flex-1 flex flex-col overflow-hidden;
}

.results-header {
  @apply flex items-center justify-between px-4 py-3 border-b border-gray-200 dark:border-gray-600 bg-gray-50 dark:bg-gray-700;
}

.results-info {
  @apply flex items-center gap-3 text-sm text-gray-600 dark:text-gray-400;
}

.results-actions {
  @apply flex gap-1;
}

.action-btn {
  @apply p-1.5 text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-600 rounded transition-colors;
}

.clusters-view {
  @apply flex-1 overflow-y-auto p-4 space-y-4;
}

.cluster-group {
  @apply bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-600 rounded-lg;
}

.cluster-header {
  @apply flex items-center justify-between p-4 border-b border-gray-200 dark:border-gray-600;
}

.cluster-title {
  @apply flex items-center gap-2;
}

.cluster-count {
  @apply text-xs text-gray-500 dark:text-gray-400;
}

.cluster-similarity {
  @apply text-sm text-green-600 dark:text-green-400;
}

.cluster-items {
  @apply p-2 space-y-1;
}

.cluster-result-item {
  @apply p-2 bg-gray-50 dark:bg-gray-700 rounded cursor-pointer hover:bg-gray-100 dark:hover:bg-gray-600 transition-colors;
}

.cluster-result-title {
  @apply text-sm font-medium text-gray-800 dark:text-gray-200;
}

.cluster-result-score {
  @apply text-xs text-gray-500 dark:text-gray-400;
}

.results-list {
  @apply flex-1 overflow-y-auto p-4 space-y-4;
}

.result-item {
  @apply p-4 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-600 rounded-lg hover:shadow-md hover:border-blue-300 dark:hover:border-blue-600 transition-all cursor-pointer;
}

.result-header {
  @apply flex items-start justify-between mb-3;
}

.result-title h3 {
  @apply text-lg font-medium text-gray-800 dark:text-gray-200 mb-1;
}

.result-similarity {
  @apply flex items-center gap-2;
}

.similarity-bar {
  @apply w-24 h-2 bg-gray-200 dark:bg-gray-600 rounded-full overflow-hidden;
}

.similarity-fill {
  @apply h-full bg-green-500;
}

.similarity-text {
  @apply text-sm font-mono text-gray-600 dark:text-gray-400;
}

.result-content {
  @apply space-y-3;
}

.result-snippet {
  @apply text-gray-700 dark:text-gray-300 line-clamp-3;
}

.result-keywords {
  @apply space-y-2;
}

.keywords-label {
  @apply text-xs font-medium text-gray-600 dark:text-gray-400;
}

.keywords-list {
  @apply flex flex-wrap gap-2;
}

.keyword-tag {
  @apply px-2 py-1 bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300 rounded;
}

.vector-info {
  @apply flex gap-4 text-xs text-gray-500 dark:text-gray-400;
}

.result-meta {
  @apply flex items-center gap-3 flex-wrap text-xs text-gray-500 dark:text-gray-400 mb-3;
}

.meta-item {
  @apply flex items-center gap-1;
}

.result-actions {
  @apply flex gap-1;
}

.result-action-btn {
  @apply p-1.5 text-gray-400 dark:text-gray-500 hover:text-gray-600 dark:hover:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-600 rounded transition-colors;
}

.empty-results, .initial-state {
  @apply flex-1 flex flex-col items-center justify-center p-8 text-gray-400 dark:text-gray-500;
}

.empty-results h3, .initial-state h3 {
  @apply text-lg font-medium mb-2;
}

.empty-hint, .initial-hint {
  @apply text-sm text-center mb-6;
}

.suggestions {
  @apply text-center;
}

.suggestions-title {
  @apply font-medium mb-2;
}

.suggestions-list {
  @apply text-left inline-block text-sm;
}

.suggestions-list li {
  @apply mb-1;
}

.search-examples {
  @apply w-full max-w-2xl mt-6;
}

.search-examples h4 {
  @apply text-sm font-semibold text-gray-600 dark:text-gray-400 mb-3;
}

.example-grid {
  @apply grid grid-cols-1 sm:grid-cols-2 gap-3;
}

.example-card {
  @apply flex items-center gap-3 p-3 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-600 rounded-lg hover:border-blue-300 dark:hover:border-blue-600 hover:bg-blue-50 dark:hover:bg-blue-900/20 transition-colors cursor-pointer;
}

.example-icon {
  @apply flex-shrink-0 w-10 h-10 bg-blue-100 dark:bg-blue-900/30 rounded-lg flex items-center justify-center text-blue-600 dark:text-blue-400;
}

.example-content {
  @apply flex-1 min-w-0;
}

.example-title {
  @apply text-sm font-medium text-gray-800 dark:text-gray-200;
}

.example-description {
  @apply text-xs text-gray-500 dark:text-gray-400;
}

.search-stats {
  @apply w-full max-w-lg mt-6;
}

.search-stats h4 {
  @apply text-sm font-semibold text-gray-600 dark:text-gray-400 mb-3;
}

.stats-grid {
  @apply grid grid-cols-2 sm:grid-cols-4 gap-3;
}

.stat-item {
  @apply text-center;
}

.stat-value {
  @apply text-lg font-semibold text-gray-800 dark:text-gray-200;
}

.stat-label {
  @apply text-xs text-gray-500 dark:text-gray-400;
}

/* 模态框样式 */
.modal-overlay {
  @apply fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50;
}

.modal-content {
  @apply bg-white dark:bg-gray-800 rounded-lg shadow-xl max-w-4xl w-full mx-4 max-h-[90vh] overflow-hidden;
}

.modal-header {
  @apply flex items-center justify-between p-4 border-b border-gray-200 dark:border-gray-600;
}

.modal-header h3 {
  @apply text-lg font-semibold text-gray-800 dark:text-gray-200;
}

.modal-close-btn {
  @apply p-1 text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-600 rounded transition-colors;
}

.modal-body {
  @apply p-6 overflow-y-auto max-h-[calc(90vh-8rem)];
}

.result-details {
  @apply space-y-6;
}

.detail-section h4 {
  @apply text-sm font-semibold text-gray-700 dark:text-gray-300 mb-3;
}

.detail-grid {
  @apply grid grid-cols-1 sm:grid-cols-2 gap-3;
}

.detail-item {
  @apply flex flex-col;
}

.detail-item label {
  @apply text-xs text-gray-500 dark:text-gray-400 mb-1;
}

.detail-item span {
  @apply text-sm text-gray-800 dark:text-gray-200;
}

.content-preview {
  @apply bg-gray-50 dark:bg-gray-700 rounded p-3;
}

.content-preview pre {
  @apply text-xs text-gray-700 dark:text-gray-300 whitespace-pre-wrap;
}

.keyword-analysis {
  @apply space-y-2;
}

.keyword-analysis-item {
  @apply flex items-center gap-3;
}

.keyword-word {
  @apply text-sm font-medium text-gray-800 dark:text-gray-200 min-w-0 flex-1;
}

.keyword-score-bar {
  @apply w-24 h-2 bg-gray-200 dark:bg-gray-600 rounded-full overflow-hidden;
}

.keyword-score-fill {
  @apply h-full bg-blue-500;
}

.keyword-score {
  @apply text-sm font-mono text-gray-600 dark:text-gray-400;
}

.animate-spin {
  @apply animate-spin;
}

.animate-pulse {
  @apply animate-pulse;
}
</style>
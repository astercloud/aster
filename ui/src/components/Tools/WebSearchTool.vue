<template>
  <div class="web-search-tool">
    <!-- 头部工具栏 -->
    <div class="search-header">
      <div class="header-title">
        <Icon type="search" size="sm" />
        <span>网络搜索</span>
      </div>
      <div class="header-actions">
        <button class="action-button" title="搜索设置" @click="toggleSettings">
          <Icon type="settings" size="sm" />
        </button>
        <button class="action-button" title="搜索历史" @click="toggleHistory">
          <Icon type="clock" size="sm" />
        </button>
      </div>
    </div>

    <!-- 设置面板 -->
    <div v-if="showSettings" class="settings-panel">
      <div class="settings-content">
        <h4>搜索设置</h4>
        <div class="setting-group">
          <label>搜索引擎</label>
          <select v-model="searchSettings.searchEngine" class="setting-select">
            <option value="google">Google</option>
            <option value="bing">Bing</option>
            <option value="duckduckgo">DuckDuckGo</option>
            <option value="baidu">百度</option>
            <option value="yahoo">Yahoo</option>
          </select>
        </div>
        <div class="setting-group">
          <label>结果数量</label>
          <select v-model="searchSettings.resultsCount" class="setting-select">
            <option value="10">10条</option>
            <option value="20">20条</option>
            <option value="50">50条</option>
            <option value="100">100条</option>
          </select>
        </div>
        <div class="setting-group">
          <label>语言</label>
          <select v-model="searchSettings.language" class="setting-select">
            <option value="zh-CN">中文</option>
            <option value="en">English</option>
            <option value="ja">日本語</option>
            <option value="ko">한국어</option>
            <option value="auto">自动检测</option>
          </select>
        </div>
        <div class="setting-group">
          <label>地区</label>
          <select v-model="searchSettings.region" class="setting-select">
            <option value="cn">中国</option>
            <option value="us">美国</option>
            <option value="uk">英国</option>
            <option value="jp">日本</option>
            <option value="global">全球</option>
          </select>
        </div>
        <div class="setting-group">
          <label>安全搜索</label>
          <select v-model="searchSettings.safeSearch" class="setting-select">
            <option value="off">关闭</option>
            <option value="moderate">适中</option>
            <option value="strict">严格</option>
          </select>
        </div>
        <div class="setting-group">
          <label>
            <input v-model="searchSettings.saveHistory" type="checkbox" class="setting-checkbox" />
            保存搜索历史
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
        <div class="search-engine-selector">
          <select v-model="currentSearchEngine" class="engine-select">
            <option value="google">Google</option>
            <option value="bing">Bing</option>
            <option value="duckduckgo">DuckDuckGo</option>
            <option value="baidu">百度</option>
            <option value="yahoo">Yahoo</option>
          </select>
        </div>
        <div class="search-input-wrapper">
          <Icon type="search" size="sm" class="search-icon" />
          <input
            v-model="searchQuery"
            ref="searchInput"
            type="text"
            placeholder="输入搜索关键词..."
            class="search-input"
            @keydown.enter="performSearch"
            @keydown.down="highlightSuggestion('down')"
            @keydown.up="highlightSuggestion('up')"
            @input="handleSearchInput"
          />
          <button class="search-button" :disabled="!searchQuery.trim() || isSearching" @click="performSearch">
            <Icon v-if="isSearching" type="spinner" size="sm" class="animate-spin" />
            <Icon v-else type="search" size="sm" />
          </button>
        </div>
      </div>

      <!-- 搜索建议 -->
      <div v-if="showSuggestions && suggestions.length > 0" class="suggestions-dropdown">
        <div v-for="(suggestion, index) in suggestions" :key="index" :class="['suggestion-item', { highlighted: highlightedSuggestionIndex === index }]" @click="selectSuggestion(suggestion)" @mouseenter="highlightedSuggestionIndex = index">
          <Icon type="search" size="xs" class="suggestion-icon" />
          <span class="suggestion-text">{{ suggestion }}</span>
        </div>
      </div>

      <!-- 快速筛选器 -->
      <div class="quick-filters">
        <div class="filter-group">
          <label class="filter-label">时间范围:</label>
          <select v-model="timeFilter" class="filter-select">
            <option value="">不限</option>
            <option value="1h">过去1小时</option>
            <option value="24h">过去24小时</option>
            <option value="7d">过去一周</option>
            <option value="30d">过去一个月</option>
            <option value="1y">过去一年</option>
          </select>
        </div>
        <div class="filter-group">
          <label class="filter-label">结果类型:</label>
          <select v-model="resultType" class="filter-select">
            <option value="">全部</option>
            <option value="news">新闻</option>
            <option value="images">图片</option>
            <option value="videos">视频</option>
            <option value="books">图书</option>
            <option value="academic">学术</option>
          </select>
        </div>
        <div class="filter-group">
          <label class="filter-label">排序:</label>
          <select v-model="sortBy" class="filter-select">
            <option value="relevance">相关性</option>
            <option value="date">时间</option>
            <option value="popularity">热度</option>
          </select>
        </div>
      </div>
    </div>

    <!-- 搜索状态 -->
    <div v-if="isSearching" class="search-status">
      <div class="status-content">
        <Icon type="spinner" size="lg" class="animate-spin" />
        <p>正在搜索...</p>
        <p class="status-hint">使用 {{ getSearchEngineName(currentSearchEngine) }} 搜索</p>
      </div>
    </div>

    <!-- 搜索结果 -->
    <div v-if="searchResults.length > 0" class="search-results">
      <!-- 结果统计 -->
      <div class="results-header">
        <div class="results-info">
          <span class="results-count">找到约 {{ totalResults }} 条结果</span>
          <span class="search-time">耗时 {{ searchTime }}ms</span>
          <span class="search-query">"{{ lastSearchQuery }}"</span>
        </div>
        <div class="results-actions">
          <button class="action-btn" title="导出结果" @click="exportResults">
            <Icon type="download" size="xs" />
            导出
          </button>
          <button class="action-btn" title="重新搜索" @click="performSearch">
            <Icon type="refresh" size="xs" />
            刷新
          </button>
          <button class="action-btn" title="清空结果" @click="clearResults">
            <Icon type="trash" size="xs" />
            清空
          </button>
        </div>
      </div>

      <!-- 结果列表 -->
      <div class="results-list">
        <div v-for="(result, index) in searchResults" :key="index" class="result-item" @click="openResult(result)">
          <div class="result-header">
            <div class="result-title">
              <h3 v-html="result.title"></h3>
            </div>
            <div class="result-actions">
              <button class="result-action-btn" title="在新标签打开" @click.stop="openInNewTab(result)">
                <Icon type="external-link" size="xs" />
              </button>
              <button class="result-action-btn" title="复制链接" @click.stop="copyLink(result)">
                <Icon type="copy" size="xs" />
              </button>
              <button class="result-action-btn favorite-btn" :class="{ active: isFavorite(result) }" title="收藏" @click.stop="toggleFavorite(result)">
                <Icon type="star" size="xs" />
              </button>
            </div>
          </div>

          <div class="result-url">
            <Icon type="link" size="xs" class="url-icon" />
            <span class="url-text">{{ result.url }}</span>
          </div>

          <div class="result-snippet" v-html="result.snippet"></div>

          <div class="result-meta">
            <span v-if="result.date" class="result-date">{{ formatDate(result.date) }}</span>
            <span v-if="result.domain" class="result-domain">{{ result.domain }}</span>
            <span v-if="result.language" class="result-language">{{ result.language.toUpperCase() }}</span>
            <span v-if="result.type" class="result-type">{{ result.type }}</span>
          </div>

          <!-- 相关信息 -->
          <div v-if="result.richData" class="result-rich-data">
            <div v-if="result.richData.thumbnail" class="rich-thumbnail">
              <img :src="result.richData.thumbnail" :alt="result.title" />
            </div>
            <div v-if="result.richData.rating" class="rich-rating">
              <Icon type="star" size="xs" class="rating-star" />
              <span>{{ result.richData.rating }}</span>
              <span v-if="result.richData.reviewCount" class="review-count"> ({{ result.richData.reviewCount }} 评价) </span>
            </div>
            <div v-if="result.richData.price" class="rich-price">
              <span class="price-amount">{{ result.richData.price }}</span>
              <span v-if="result.richData.currency" class="price-currency">{{ result.richData.currency }}</span>
            </div>
          </div>
        </div>
      </div>

      <!-- 分页 -->
      <div v-if="totalPages > 1" class="pagination">
        <button class="pagination-btn" :disabled="currentPage === 1" @click="goToPage(currentPage - 1)">
          <Icon type="chevron-left" size="xs" />
          上一页
        </button>

        <div class="pagination-info">第 {{ currentPage }} 页，共 {{ totalPages }} 页</div>

        <button class="pagination-btn" :disabled="currentPage === totalPages" @click="goToPage(currentPage + 1)">
          下一页
          <Icon type="chevron-right" size="xs" />
        </button>
      </div>
    </div>

    <!-- 空状态 -->
    <div v-if="!isSearching && searchResults.length === 0 && lastSearchQuery" class="empty-results">
      <Icon type="search" size="lg" />
      <h3>未找到相关结果</h3>
      <p class="empty-hint">试试调整搜索关键词或筛选条件</p>
      <button class="retry-btn" @click="clearResults">重新搜索</button>
    </div>

    <!-- 初始状态 -->
    <div v-if="!isSearching && searchResults.length === 0 && !lastSearchQuery" class="initial-state">
      <Icon type="search" size="lg" />
      <h3>开始网络搜索</h3>
      <p class="initial-hint">输入关键词开始搜索，支持多种搜索方式和高级筛选</p>

      <!-- 热门搜索 -->
      <div class="trending-searches">
        <h4>热门搜索</h4>
        <div class="trending-tags">
          <button v-for="(trend, index) in trendingSearches" :key="index" class="trending-tag" @click="searchForTrend(trend)">
            {{ trend }}
          </button>
        </div>
      </div>

      <!-- 搜索历史 -->
      <div v-if="searchHistory.length > 0" class="recent-searches">
        <h4>最近搜索</h4>
        <div class="recent-list">
          <div v-for="(item, index) in searchHistory.slice(0, 5)" :key="index" class="recent-item" @click="searchFromHistory(item)">
            <Icon type="clock" size="xs" />
            <span class="recent-query">{{ item.query }}</span>
            <span class="recent-time">{{ formatTime(item.timestamp) }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- 搜索历史面板 -->
    <div v-if="showHistory" class="history-panel">
      <div class="history-header">
        <h4>搜索历史</h4>
        <button class="history-close-btn" @click="showHistory = false">
          <Icon type="close" size="xs" />
        </button>
      </div>
      <div class="history-actions">
        <button class="history-action-btn" @click="clearHistory">
          <Icon type="trash" size="xs" />
          清空历史
        </button>
      </div>
      <div class="history-list">
        <div v-for="(item, index) in searchHistory" :key="index" class="history-item" @click="searchFromHistory(item)">
          <div class="history-query">{{ item.query }}</div>
          <div class="history-meta">
            <span class="history-engine">{{ getSearchEngineName(item.engine) }}</span>
            <span class="history-time">{{ formatTime(item.timestamp) }}</span>
            <span class="history-results">{{ item.resultsCount }} 个结果</span>
          </div>
          <button class="history-delete-btn" @click.stop="deleteHistoryItem(index)">
            <Icon type="close" size="xs" />
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick, onMounted, onUnmounted } from "vue";
import Icon from "../ChatUI/Icon.vue";

interface SearchResult {
  title: string;
  url: string;
  snippet: string;
  domain: string;
  date?: string;
  language?: string;
  type?: string;
  richData?: {
    thumbnail?: string;
    rating?: number;
    reviewCount?: number;
    price?: string;
    currency?: string;
  };
}

interface SearchHistory {
  query: string;
  engine: string;
  timestamp: number;
  resultsCount: number;
}

interface SearchSettings {
  searchEngine: string;
  resultsCount: number;
  language: string;
  region: string;
  safeSearch: string;
  saveHistory: boolean;
}

interface Props {
  wsUrl?: string;
  sessionId?: string;
}

const props = withDefaults(defineProps<Props>(), {
  wsUrl: "ws://localhost:8080/ws",
  sessionId: "default",
});

const emit = defineEmits<{
  searchPerformed: [query: string, results: SearchResult[]];
  resultSelected: [result: SearchResult];
}>();

// 响应式数据
const searchQuery = ref("");
const currentSearchEngine = ref("google");
const searchResults = ref<SearchResult[]>([]);
const suggestions = ref<string[]>([]);
const searchHistory = ref<SearchHistory[]>([]);
const favorites = ref<SearchResult[]>([]);
const isSearching = ref(false);
const showSettings = ref(false);
const showHistory = ref(false);
const showSuggestions = ref(false);
const highlightedSuggestionIndex = ref(-1);

// 搜索参数
const timeFilter = ref("");
const resultType = ref("");
const sortBy = ref("relevance");
const currentPage = ref(1);
const totalResults = ref(0);
const totalPages = ref(0);
const searchTime = ref(0);
const lastSearchQuery = ref("");

// 搜索设置
const searchSettings = ref<SearchSettings>({
  searchEngine: "google",
  resultsCount: 10,
  language: "zh-CN",
  region: "cn",
  safeSearch: "moderate",
  saveHistory: true,
});

// 热门搜索
const trendingSearches = ref(["AI 最新发展", "Vue 3 新特性", "TypeScript 教程", "Web 开发最佳实践", "开源项目推荐"]);

const searchInput = ref<HTMLInputElement>();
const websocket = ref<WebSocket | null>(null);
let searchDebounceTimer: ReturnType<typeof setTimeout> | null = null;

// 计算属性
const hasSearchResults = computed(() => searchResults.value.length > 0);

// WebSocket 连接
const connectWebSocket = () => {
  try {
    websocket.value = new WebSocket(`${props.wsUrl}?session=${props.sessionId}`);

    websocket.value.onopen = () => {
      console.log("WebSearchTool WebSocket connected");
    };

    websocket.value.onmessage = (event) => {
      try {
        const message = JSON.parse(event.data);
        handleWebSocketMessage(message);
      } catch (error) {
        console.error("Failed to parse WebSocket message:", error);
      }
    };

    websocket.value.onclose = () => {
      console.log("WebSearchTool WebSocket disconnected");
      setTimeout(connectWebSocket, 5000);
    };

    websocket.value.onerror = (error) => {
      console.error("WebSearchTool WebSocket error:", error);
    };
  } catch (error) {
    console.error("Failed to connect WebSocket:", error);
  }
};

const handleWebSocketMessage = (message: any) => {
  switch (message.type) {
    case "search_results":
      if (message.results) {
        handleSearchResults(message.results);
      }
      break;
    case "search_suggestions":
      if (message.suggestions) {
        suggestions.value = message.suggestions;
        showSuggestions.value = true;
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
  lastSearchQuery.value = searchQuery.value.trim();
  const startTime = Date.now();

  try {
    // 构建搜索参数
    const searchParams = {
      query: lastSearchQuery.value,
      engine: currentSearchEngine.value,
      count: searchSettings.value.resultsCount,
      language: searchSettings.value.language,
      region: searchSettings.value.region,
      safeSearch: searchSettings.value.safeSearch,
      timeFilter: timeFilter.value,
      resultType: resultType.value,
      sortBy: sortBy.value,
      page: currentPage.value,
    };

    // 发送搜索请求
    sendWebSocketMessage({
      type: "web_search",
      params: searchParams,
    });

    // 模拟搜索结果（实际应该通过WebSocket获取）
    const mockResults = await mockWebSearch(searchParams);

    const endTime = Date.now();
    searchTime.value = endTime - startTime;

    handleSearchResults(mockResults);

    // 保存到历史记录
    if (searchSettings.value.saveHistory) {
      addToHistory(lastSearchQuery.value, currentSearchEngine.value, mockResults.length);
    }

    emit("searchPerformed", lastSearchQuery.value, mockResults);
  } catch (error) {
    console.error("Search failed:", error);
    searchResults.value = [];
  } finally {
    isSearching.value = false;
    showSuggestions.value = false;
  }
};

const mockWebSearch = async (params: any): Promise<SearchResult[]> => {
  // 模拟搜索延迟
  await new Promise((resolve) => setTimeout(resolve, 800 + Math.random() * 1200));

  // 模拟搜索结果
  const mockResults: SearchResult[] = [
    {
      title: `<b>${params.query}</b> - 搜索结果 1`,
      url: `https://example.com/result1?q=${encodeURIComponent(params.query)}`,
      snippet: `这是关于 ${params.query} 的第一个搜索结果。包含了相关的信息和详细内容...`,
      domain: "example.com",
      date: new Date().toISOString(),
      language: "zh-CN",
      type: resultType.value || "web",
      richData: {
        thumbnail: "https://via.placeholder.com/100x60",
        rating: 4.5,
        reviewCount: 128,
      },
    },
    {
      title: `<b>${params.query}</b> - 官方文档`,
      url: `https://docs.example.com/${params.query}`,
      snippet: `官方文档提供了关于 ${params.query} 的权威信息和技术细节...`,
      domain: "docs.example.com",
      date: new Date().toISOString(),
      language: "zh-CN",
      type: resultType.value || "web",
    },
    {
      title: `深入了解 ${params.query}`,
      url: `https://blog.example.com/${params.query}`,
      snippet: `在这篇详细的文章中，我们将深入探讨 ${params.query} 的各个方面...`,
      domain: "blog.example.com",
      date: new Date(Date.now() - 86400000).toISOString(),
      language: "zh-CN",
      type: resultType.value || "web",
      richData: {
        rating: 4.8,
        reviewCount: 256,
      },
    },
    {
      title: `${params.query} 相关教程`,
      url: `https://tutorial.example.com/${params.query}`,
      snippet: `逐步教程教你如何使用 ${params.query}，包含实例和最佳实践...`,
      domain: "tutorial.example.com",
      date: new Date(Date.now() - 172800000).toISOString(),
      language: "zh-CN",
      type: resultType.value || "web",
    },
    {
      title: `${params.query} 社区讨论`,
      url: `https://community.example.com/${params.query}`,
      snippet: `社区成员分享的关于 ${params.query} 的经验和见解...`,
      domain: "community.example.com",
      date: new Date(Date.now() - 259200000).toISOString(),
      language: "zh-CN",
      type: resultType.value || "web",
    },
  ];

  totalResults.value = Math.floor(Math.random() * 1000000) + 1000;
  totalPages.value = Math.ceil(totalResults.value / params.count);

  return mockResults;
};

const handleSearchResults = (results: SearchResult[]) => {
  searchResults.value = results;
  currentPage.value = 1;
};

const handleSearchInput = () => {
  // 防抖搜索建议
  if (searchDebounceTimer) {
    clearTimeout(searchDebounceTimer);
  }
  searchDebounceTimer = setTimeout(() => {
    if (searchQuery.value.trim().length > 2) {
      fetchSuggestions();
    } else {
      showSuggestions.value = false;
      suggestions.value = [];
    }
  }, 300);
};

const fetchSuggestions = () => {
  if (!searchQuery.value.trim()) return;

  sendWebSocketMessage({
    type: "get_search_suggestions",
    query: searchQuery.value.trim(),
    engine: currentSearchEngine.value,
  });

  // 模拟搜索建议
  suggestions.value = [searchQuery.value + " 教程", searchQuery.value + " 下载", searchQuery.value + " 官网", searchQuery.value + " 评价"];
  showSuggestions.value = true;
};

// 搜索建议处理
const highlightSuggestion = (direction: "up" | "down") => {
  if (!showSuggestions.value || suggestions.value.length === 0) return;

  if (direction === "down") {
    highlightedSuggestionIndex.value = Math.min(highlightedSuggestionIndex.value + 1, suggestions.value.length - 1);
  } else {
    highlightedSuggestionIndex.value = Math.max(highlightedSuggestionIndex.value - 1, -1);
  }

  if (highlightedSuggestionIndex.value >= 0) {
    searchQuery.value = suggestions.value[highlightedSuggestionIndex.value] ?? "";
  }
};

const selectSuggestion = (suggestion: string) => {
  searchQuery.value = suggestion;
  showSuggestions.value = false;
  highlightedSuggestionIndex.value = -1;
  nextTick(() => {
    performSearch();
  });
};

// 结果操作
const openResult = (result: SearchResult) => {
  window.open(result.url, "_blank");
  emit("resultSelected", result);
};

const openInNewTab = (result: SearchResult) => {
  window.open(result.url, "_blank");
};

const copyLink = async (result: SearchResult) => {
  try {
    await navigator.clipboard.writeText(result.url);
  } catch (error) {
    console.error("Failed to copy link:", error);
  }
};

const isFavorite = (result: SearchResult): boolean => {
  return favorites.value.some((fav) => fav.url === result.url);
};

const toggleFavorite = (result: SearchResult) => {
  const index = favorites.value.findIndex((fav) => fav.url === result.url);
  if (index === -1) {
    favorites.value.push(result);
  } else {
    favorites.value.splice(index, 1);
  }
  saveFavorites();
};

// 分页
const goToPage = (page: number) => {
  currentPage.value = page;
  performSearch();
};

// 导出结果
const exportResults = () => {
  const exportData = {
    query: lastSearchQuery.value,
    engine: currentSearchEngine.value,
    timestamp: new Date().toISOString(),
    results: searchResults.value,
    totalResults: totalResults.value,
    searchTime: searchTime.value,
  };

  const dataStr = JSON.stringify(exportData, null, 2);
  const blob = new Blob([dataStr], { type: "application/json" });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = `search-results-${Date.now()}.json`;
  a.click();
  URL.revokeObjectURL(url);
};

const clearResults = () => {
  searchResults.value = [];
  searchQuery.value = "";
  lastSearchQuery.value = "";
  currentPage.value = 1;
  totalResults.value = 0;
  totalPages.value = 0;
  showSuggestions.value = false;
  suggestions.value = [];
};

// 历史记录管理
const addToHistory = (query: string, engine: string, resultsCount: number) => {
  const historyItem: SearchHistory = {
    query,
    engine,
    timestamp: Date.now(),
    resultsCount,
  };

  // 避免重复
  const existingIndex = searchHistory.value.findIndex((item) => item.query === query && item.engine === engine);
  if (existingIndex !== -1) {
    searchHistory.value.splice(existingIndex, 1);
  }

  searchHistory.value.unshift(historyItem);
  if (searchHistory.value.length > 100) {
    searchHistory.value = searchHistory.value.slice(0, 100);
  }

  saveHistory();
};

const searchFromHistory = (item: SearchHistory) => {
  searchQuery.value = item.query;
  currentSearchEngine.value = item.engine;
  showHistory.value = false;
  nextTick(() => {
    performSearch();
  });
};

const deleteHistoryItem = (index: number) => {
  searchHistory.value.splice(index, 1);
  saveHistory();
};

const clearHistory = () => {
  if (confirm("确定要清空所有搜索历史吗？")) {
    searchHistory.value = [];
    saveHistory();
  }
};

const saveHistory = () => {
  try {
    localStorage.setItem("web-search-history", JSON.stringify(searchHistory.value));
  } catch (error) {
    console.warn("Failed to save search history:", error);
  }
};

const loadHistory = () => {
  try {
    const saved = localStorage.getItem("web-search-history");
    if (saved) {
      searchHistory.value = JSON.parse(saved);
    }
  } catch (error) {
    console.warn("Failed to load search history:", error);
  }
};

const saveFavorites = () => {
  try {
    localStorage.setItem("web-search-favorites", JSON.stringify(favorites.value));
  } catch (error) {
    console.warn("Failed to save favorites:", error);
  }
};

const loadFavorites = () => {
  try {
    const saved = localStorage.getItem("web-search-favorites");
    if (saved) {
      favorites.value = JSON.parse(saved);
    }
  } catch (error) {
    console.warn("Failed to load favorites:", error);
  }
};

// 热门搜索
const searchForTrend = (trend: string) => {
  searchQuery.value = trend;
  nextTick(() => {
    performSearch();
  });
};

// 设置管理
const toggleSettings = () => {
  showSettings.value = !showSettings.value;
};

const saveSettings = () => {
  localStorage.setItem("web-search-settings", JSON.stringify(searchSettings.value));
  currentSearchEngine.value = searchSettings.value.searchEngine;
  showSettings.value = false;
};

const resetSettings = () => {
  searchSettings.value = {
    searchEngine: "google",
    resultsCount: 10,
    language: "zh-CN",
    region: "cn",
    safeSearch: "moderate",
    saveHistory: true,
  };
};

const toggleHistory = () => {
  showHistory.value = !showHistory.value;
};

// 工具方法
const getSearchEngineName = (engine: string): string => {
  const engineNames: Record<string, string> = {
    google: "Google",
    bing: "Bing",
    duckduckgo: "DuckDuckGo",
    baidu: "百度",
    yahoo: "Yahoo",
  };
  return engineNames[engine] || engine;
};

const formatDate = (dateString: string): string => {
  const date = new Date(dateString);
  const now = new Date();
  const diff = now.getTime() - date.getTime();
  const days = Math.floor(diff / (1000 * 60 * 60 * 24));

  if (days === 0) {
    return "今天";
  } else if (days === 1) {
    return "昨天";
  } else if (days < 7) {
    return `${days}天前`;
  } else if (days < 30) {
    return `${Math.floor(days / 7)}周前`;
  } else {
    return date.toLocaleDateString("zh-CN");
  }
};

const formatTime = (timestamp: number): string => {
  const date = new Date(timestamp);
  const now = new Date();
  const diff = now.getTime() - date.getTime();

  if (diff < 60000) {
    return "刚刚";
  } else if (diff < 3600000) {
    const minutes = Math.floor(diff / 60000);
    return `${minutes}分钟前`;
  } else if (diff < 86400000) {
    const hours = Math.floor(diff / 3600000);
    return `${hours}小时前`;
  } else {
    return date.toLocaleDateString("zh-CN");
  }
};

// 点击外部关闭搜索建议
const handleClickOutside = (event: MouseEvent) => {
  const target = event.target as Element;
  if (!target.closest(".search-input-wrapper") && !target.closest(".suggestions-dropdown")) {
    showSuggestions.value = false;
  }
};

// 生命周期
onMounted(() => {
  connectWebSocket();
  loadHistory();
  loadFavorites();

  // 绑定外部点击事件
  document.addEventListener("click", handleClickOutside);

  // 加载设置
  try {
    const saved = localStorage.getItem("web-search-settings");
    if (saved) {
      searchSettings.value = { ...searchSettings.value, ...JSON.parse(saved) };
      currentSearchEngine.value = searchSettings.value.searchEngine;
    }
  } catch (error) {
    console.warn("Failed to load search settings:", error);
  }

  // 自动聚焦搜索框
  nextTick(() => {
    searchInput.value?.focus();
  });
});

onUnmounted(() => {
  document.removeEventListener("click", handleClickOutside);
});

// 监听搜索设置变化
watch(
  searchSettings,
  (newSettings) => {
    currentSearchEngine.value = newSettings.searchEngine;
  },
  { deep: true },
);
</script>

<style scoped>
.web-search-tool {
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
  @apply flex gap-3 mb-4;
}

.engine-select {
  @apply px-3 py-2 border border-gray-200 dark:border-gray-600 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 dark:bg-gray-700 dark:text-white;
}

.search-input-wrapper {
  @apply flex-1 relative;
}

.search-icon {
  @apply absolute left-3 top-1/2 transform -translate-y-1/2 text-gray-400 dark:text-gray-500;
}

.search-input {
  @apply w-full pl-10 pr-12 py-3 border border-gray-200 dark:border-gray-600 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 dark:bg-gray-700 dark:text-white text-lg;
}

.search-button {
  @apply absolute right-2 top-1/2 transform -translate-y-1/2 p-2 text-blue-500 hover:text-blue-600 hover:bg-blue-50 dark:hover:bg-blue-900/20 rounded transition-colors disabled:opacity-50 disabled:cursor-not-allowed;
}

.suggestions-dropdown {
  @apply absolute top-full left-0 right-0 mt-1 bg-white dark:bg-gray-700 border border-gray-200 dark:border-gray-600 rounded-lg shadow-lg z-10;
}

.suggestion-item {
  @apply flex items-center gap-3 px-4 py-3 hover:bg-gray-50 dark:hover:bg-gray-600 cursor-pointer transition-colors;
}

.suggestion-item.highlighted {
  @apply bg-blue-50 dark:bg-blue-900/20;
}

.suggestion-icon {
  @apply text-gray-400 dark:text-gray-500;
}

.suggestion-text {
  @apply text-gray-800 dark:text-gray-200;
}

.quick-filters {
  @apply flex gap-4 flex-wrap;
}

.filter-group {
  @apply flex items-center gap-2;
}

.filter-label {
  @apply text-sm text-gray-600 dark:text-gray-400;
}

.filter-select {
  @apply px-2 py-1 text-sm border border-gray-200 dark:border-gray-600 rounded focus:outline-none focus:ring-1 focus:ring-blue-500 dark:bg-gray-700 dark:text-white;
}

.search-status {
  @apply flex-1 flex items-center justify-center p-8;
}

.status-content {
  @apply flex flex-col items-center text-gray-500 dark:text-gray-400;
}

.status-hint {
  @apply text-sm mt-1;
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

.results-count {
  @apply font-medium;
}

.search-time {
  @apply text-xs;
}

.search-query {
  @apply text-xs text-gray-500 dark:text-gray-500;
}

.results-actions {
  @apply flex gap-1;
}

.action-btn {
  @apply p-1.5 text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-600 rounded transition-colors;
}

.results-list {
  @apply flex-1 overflow-y-auto p-4 space-y-4;
}

.result-item {
  @apply p-4 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-600 rounded-lg hover:shadow-md hover:border-blue-300 dark:hover:border-blue-600 transition-all cursor-pointer;
}

.result-header {
  @apply flex items-start justify-between mb-2;
}

.result-title h3 {
  @apply text-lg font-medium text-blue-600 dark:text-blue-400 hover:text-blue-700 dark:hover:text-blue-300 mb-1;
  @apply leading-tight;
}

.result-actions {
  @apply flex gap-1 ml-4;
}

.result-action-btn {
  @apply p-1.5 text-gray-400 dark:text-gray-500 hover:text-gray-600 dark:hover:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-600 rounded transition-colors;
}

.result-action-btn.favorite-btn.active {
  @apply text-yellow-500 hover:text-yellow-600;
}

.result-url {
  @apply flex items-center gap-2 text-sm text-green-600 dark:text-green-400 mb-2;
}

.url-icon {
  @apply flex-shrink-0;
}

.url-text {
  @apply truncate hover:underline;
}

.result-snippet {
  @apply text-gray-700 dark:text-gray-300 mb-3 line-clamp-2;
}

.result-meta {
  @apply flex items-center gap-3 text-xs text-gray-500 dark:text-gray-400;
}

.result-date,
.result-domain,
.result-language,
.result-type {
  @apply px-2 py-1 bg-gray-100 dark:bg-gray-600 rounded;
}

.result-rich-data {
  @apply mt-3 flex items-center gap-4;
}

.rich-thumbnail img {
  @apply w-16 h-12 object-cover rounded;
}

.rich-rating {
  @apply flex items-center gap-1 text-xs text-yellow-500;
}

.rich-price {
  @apply text-sm font-semibold text-green-600 dark:text-green-400;
}

.pagination {
  @apply flex items-center justify-center gap-4 p-4 border-t border-gray-200 dark:border-gray-600;
}

.pagination-btn {
  @apply flex items-center gap-2 px-4 py-2 text-sm text-gray-600 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-600 rounded transition-colors disabled:opacity-50 disabled:cursor-not-allowed;
}

.pagination-info {
  @apply text-sm text-gray-600 dark:text-gray-400;
}

.empty-results,
.initial-state {
  @apply flex-1 flex flex-col items-center justify-center p-8 text-gray-400 dark:text-gray-500;
}

.empty-results h3,
.initial-state h3 {
  @apply text-lg font-medium mb-2;
}

.empty-hint,
.initial-hint {
  @apply text-sm text-center mb-6;
}

.retry-btn {
  @apply px-4 py-2 bg-blue-500 hover:bg-blue-600 text-white rounded transition-colors;
}

.trending-searches,
.recent-searches {
  @apply w-full max-w-md mt-6;
}

.trending-searches h4,
.recent-searches h4 {
  @apply text-sm font-semibold text-gray-600 dark:text-gray-400 mb-3;
}

.trending-tags {
  @apply flex flex-wrap gap-2;
}

.trending-tag {
  @apply px-3 py-1 text-sm bg-blue-100 dark:bg-blue-900/30 text-blue-600 dark:text-blue-400 rounded-full hover:bg-blue-200 dark:hover:bg-blue-800/40 transition-colors;
}

.recent-list {
  @apply space-y-2;
}

.recent-item,
.history-item {
  @apply flex items-center gap-3 p-3 bg-gray-50 dark:bg-gray-700 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-600 cursor-pointer transition-colors;
}

.recent-query,
.history-query {
  @apply flex-1 text-sm text-gray-800 dark:text-gray-200;
}

.recent-time,
.history-time {
  @apply text-xs text-gray-500 dark:text-gray-400;
}

.history-panel {
  @apply absolute top-0 right-0 w-96 h-full bg-white dark:bg-gray-800 border-l border-gray-200 dark:border-gray-600 shadow-lg z-20;
}

.history-header {
  @apply flex items-center justify-between p-4 border-b border-gray-200 dark:border-gray-600;
}

.history-header h4 {
  @apply text-sm font-semibold text-gray-800 dark:text-gray-200;
}

.history-close-btn {
  @apply p-1 text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-600 rounded transition-colors;
}

.history-actions {
  @apply p-4 border-b border-gray-200 dark:border-gray-600;
}

.history-action-btn {
  @apply flex items-center gap-2 px-3 py-2 text-sm text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20 rounded transition-colors;
}

.history-list {
  @apply h-full overflow-y-auto p-4 space-y-2;
}

.history-meta {
  @apply flex items-center gap-2 text-xs text-gray-500 dark:text-gray-400;
}

.history-delete-btn {
  @apply p-1 text-gray-400 dark:text-gray-500 hover:text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20 rounded transition-colors;
}

.animate-spin {
  @apply animate-spin;
}
</style>

<template>
  <div class="http-request-tool">
    <!-- 头部工具栏 -->
    <div class="request-header">
      <div class="header-title">
        <Icon type="globe" size="sm" />
        <span>HTTP请求工具</span>
      </div>
      <div class="header-actions">
        <button class="action-button" title="新建请求" @click="createNewRequest">
          <Icon type="plus" size="sm" />
        </button>
        <button class="action-button" title="导入请求" @click="importRequest">
          <Icon type="upload" size="sm" />
        </button>
        <button class="action-button" title="导出请求" :disabled="!currentRequest" @click="exportRequest">
          <Icon type="download" size="sm" />
        </button>
        <button class="action-button" title="设置" @click="toggleSettings">
          <Icon type="settings" size="sm" />
        </button>
      </div>
    </div>

    <!-- 设置面板 -->
    <div v-if="showSettings" class="settings-panel">
      <div class="settings-content">
        <h4>请求设置</h4>
        <div class="setting-group">
          <label>默认超时时间 (秒)</label>
          <input v-model.number="requestSettings.timeout" type="number" min="1" max="300" class="setting-input" />
        </div>
        <div class="setting-group">
          <label>重试次数</label>
          <input v-model.number="requestSettings.retryCount" type="number" min="0" max="10" class="setting-input" />
        </div>
        <div class="setting-group">
          <label>
            <input v-model="requestSettings.followRedirects" type="checkbox" class="setting-checkbox" />
            跟随重定向
          </label>
        </div>
        <div class="setting-group">
          <label>
            <input v-model="requestSettings.verifySSL" type="checkbox" class="setting-checkbox" />
            验证SSL证书
          </label>
        </div>
        <div class="setting-group">
          <label>
            <input v-model="requestSettings.saveToHistory" type="checkbox" class="setting-checkbox" />
            保存到历史记录
          </label>
        </div>
        <div class="setting-actions">
          <button class="setting-btn" @click="resetSettings">重置</button>
          <button class="setting-btn primary" @click="saveSettings">保存</button>
        </div>
      </div>
    </div>

    <!-- 请求构建区域 -->
    <div class="request-builder">
      <!-- URL和方式选择 -->
      <div class="url-section">
        <div class="method-selector">
          <select v-model="currentRequest.method" class="method-select">
            <option value="GET">GET</option>
            <option value="POST">POST</option>
            <option value="PUT">PUT</option>
            <option value="DELETE">DELETE</option>
            <option value="PATCH">PATCH</option>
            <option value="HEAD">HEAD</option>
            <option value="OPTIONS">OPTIONS</option>
          </select>
        </div>
        <div class="url-input-group">
          <input v-model="currentRequest.url" type="text" placeholder="输入请求URL (如: https://api.example.com/users)" class="url-input" @keydown.enter="sendRequest" />
          <button class="send-button" :disabled="!currentRequest.url.trim() || isSending" @click="sendRequest">
            <Icon v-if="isSending" type="spinner" size="sm" class="animate-spin" />
            <Icon v-else type="send" size="sm" />
            {{ isSending ? "发送中..." : "发送" }}
          </button>
        </div>
      </div>

      <!-- 选项卡 -->
      <div class="request-tabs">
        <button v-for="tab in requestTabs" :key="tab.key" :class="['tab-button', { active: activeTab === tab.key }]" @click="activeTab = tab.key">
          <Icon :type="tab.icon as any" size="xs" />
          {{ tab.label }}
          <span v-if="tab.badge" class="tab-badge">{{ tab.badge }}</span>
        </button>
      </div>

      <!-- 选项卡内容 -->
      <div class="tab-content">
        <!-- 参数选项卡 -->
        <div v-if="activeTab === 'params'" class="params-content">
          <div class="params-table">
            <div class="table-header">
              <div class="header-cell param-key">参数名</div>
              <div class="header-cell param-value">参数值</div>
              <div class="header-cell param-desc">描述</div>
              <div class="header-cell param-actions">操作</div>
            </div>
            <div v-for="(param, index) in currentRequest.params" :key="index" class="table-row">
              <div class="table-cell">
                <input v-model="param.key" type="text" placeholder="参数名" class="param-input" />
              </div>
              <div class="table-cell">
                <input v-model="param.value" type="text" placeholder="参数值" class="param-input" />
              </div>
              <div class="table-cell">
                <input v-model="param.description" type="text" placeholder="描述" class="param-input" />
              </div>
              <div class="table-cell">
                <button class="row-action-btn delete-btn" title="删除" @click="removeParam(index)">
                  <Icon type="close" size="xs" />
                </button>
              </div>
            </div>
          </div>
          <button class="add-param-btn" @click="addParam">
            <Icon type="plus" size="xs" />
            添加参数
          </button>
        </div>

        <!-- 头部选项卡 -->
        <div v-if="activeTab === 'headers'" class="headers-content">
          <div class="headers-table">
            <div class="table-header">
              <div class="header-cell header-key">头部名称</div>
              <div class="header-cell header-value">头部值</div>
              <div class="header-cell header-desc">描述</div>
              <div class="header-cell header-actions">操作</div>
            </div>
            <div v-for="(header, index) in currentRequest.headers" :key="index" class="table-row">
              <div class="table-cell">
                <select v-model="header.key" class="header-select">
                  <option value="">自定义头部</option>
                  <option value="Accept">Accept</option>
                  <option value="Accept-Encoding">Accept-Encoding</option>
                  <option value="Authorization">Authorization</option>
                  <option value="Content-Type">Content-Type</option>
                  <option value="Cookie">Cookie</option>
                  <option value="User-Agent">User-Agent</option>
                  <option value="X-Requested-With">X-Requested-With</option>
                </select>
              </div>
              <div class="table-cell">
                <input v-model="header.value" type="text" placeholder="头部值" class="header-input" />
              </div>
              <div class="table-cell">
                <input v-model="header.description" type="text" placeholder="描述" class="header-input" />
              </div>
              <div class="table-cell">
                <button class="row-action-btn delete-btn" title="删除" @click="removeHeader(index)">
                  <Icon type="close" size="xs" />
                </button>
              </div>
            </div>
          </div>
          <button class="add-header-btn" @click="addHeader">
            <Icon type="plus" size="xs" />
            添加头部
          </button>
        </div>

        <!-- 请求体选项卡 -->
        <div v-if="activeTab === 'body'" class="body-content">
          <div class="body-type-selector">
            <label class="body-type-label">请求体类型:</label>
            <select v-model="currentRequest.bodyType" class="body-type-select">
              <option value="none">无</option>
              <option value="form-data">form-data</option>
              <option value="x-www-form-urlencoded">x-www-form-urlencoded</option>
              <option value="raw">raw</option>
              <option value="binary">binary</option>
            </select>
            <select v-if="currentRequest.bodyType === 'raw'" v-model="currentRequest.bodyLanguage" class="body-language-select">
              <option value="text">Text</option>
              <option value="json">JSON</option>
              <option value="xml">XML</option>
              <option value="html">HTML</option>
            </select>
          </div>

          <!-- form-data -->
          <div v-if="currentRequest.bodyType === 'form-data'" class="form-data-content">
            <div class="form-data-table">
              <div class="table-header">
                <div class="header-cell form-key">键</div>
                <div class="header-cell form-value">值</div>
                <div class="header-cell form-type">类型</div>
                <div class="header-cell form-actions">操作</div>
              </div>
              <div v-for="(item, index) in currentRequest.formData" :key="index" class="table-row">
                <div class="table-cell">
                  <input v-model="item.key" type="text" placeholder="键名" class="form-input" />
                </div>
                <div class="table-cell">
                  <input v-if="item.type === 'text'" v-model="item.value" type="text" placeholder="值" class="form-input" />
                  <div v-else class="file-input-wrapper">
                    <input type="file" :ref="`fileInput-${index}`" class="file-input" @change="handleFileSelect(index, $event)" />
                    <button class="file-select-btn" @click="($refs[`fileInput-${index}`] as HTMLInputElement[])?.[0]?.click()">
                      <Icon type="file" size="xs" />
                      选择文件
                    </button>
                    <span v-if="item.filename" class="filename">{{ item.filename }}</span>
                  </div>
                </div>
                <div class="table-cell">
                  <select v-model="item.type" class="form-type-select">
                    <option value="text">Text</option>
                    <option value="file">File</option>
                  </select>
                </div>
                <div class="table-cell">
                  <button class="row-action-btn delete-btn" title="删除" @click="removeFormDataItem(index)">
                    <Icon type="close" size="xs" />
                  </button>
                </div>
              </div>
            </div>
            <button class="add-form-data-btn" @click="addFormDataItem">
              <Icon type="plus" size="xs" />
              添加表单项
            </button>
          </div>

          <!-- x-www-form-urlencoded -->
          <div v-if="currentRequest.bodyType === 'x-www-form-urlencoded'" class="urlencoded-content">
            <div class="urlencoded-table">
              <div class="table-header">
                <div class="header-cell url-key">键</div>
                <div class="header-cell url-value">值</div>
                <div class="header-cell url-actions">操作</div>
              </div>
              <div v-for="(item, index) in currentRequest.urlencoded" :key="index" class="table-row">
                <div class="table-cell">
                  <input v-model="item.key" type="text" placeholder="键名" class="url-input" />
                </div>
                <div class="table-cell">
                  <input v-model="item.value" type="text" placeholder="值" class="url-input" />
                </div>
                <div class="table-cell">
                  <button class="row-action-btn delete-btn" title="删除" @click="removeUrlencodedItem(index)">
                    <Icon type="close" size="xs" />
                  </button>
                </div>
              </div>
            </div>
            <button class="add-urlencoded-btn" @click="addUrlencodedItem">
              <Icon type="plus" size="xs" />
              添加参数
            </button>
          </div>

          <!-- raw -->
          <div v-if="currentRequest.bodyType === 'raw'" class="raw-content">
            <div class="raw-editor">
              <textarea v-model="currentRequest.bodyRaw" :placeholder="getBodyPlaceholder()" class="raw-textarea" :class="`language-${currentRequest.bodyLanguage}`"></textarea>
            </div>
            <div class="raw-actions">
              <button class="raw-action-btn" title="格式化JSON" @click="formatRawBody">
                <Icon type="align-left" size="xs" />
                格式化
              </button>
              <button class="raw-action-btn" title="验证JSON" @click="validateRawBody">
                <Icon type="check" size="xs" />
                验证
              </button>
            </div>
          </div>

          <!-- binary -->
          <div v-if="currentRequest.bodyType === 'binary'" class="binary-content">
            <div class="binary-input-wrapper">
              <input ref="binaryFileInput" type="file" class="binary-input" @change="handleBinaryFileSelect" />
              <button class="binary-select-btn" @click="(binaryFileInput as HTMLInputElement)?.click()">
                <Icon type="file" size="xs" />
                选择文件
              </button>
              <span v-if="currentRequest.binaryFile" class="filename">
                {{ currentRequest.binaryFile.name }}
              </span>
            </div>
          </div>
        </div>

        <!-- 认证选项卡 -->
        <div v-if="activeTab === 'auth'" class="auth-content">
          <div class="auth-type-selector">
            <label class="auth-type-label">认证类型:</label>
            <select v-model="currentRequest.authType" class="auth-type-select">
              <option value="none">无认证</option>
              <option value="bearer">Bearer Token</option>
              <option value="basic">Basic Auth</option>
              <option value="api-key">API Key</option>
            </select>
          </div>

          <!-- Bearer Token -->
          <div v-if="currentRequest.authType === 'bearer'" class="bearer-auth">
            <label class="auth-label">Token:</label>
            <input v-model="currentRequest.bearerToken" type="text" placeholder="输入Bearer Token" class="auth-input" />
          </div>

          <!-- Basic Auth -->
          <div v-if="currentRequest.authType === 'basic'" class="basic-auth">
            <label class="auth-label">用户名:</label>
            <input v-model="currentRequest.basicUsername" type="text" placeholder="输入用户名" class="auth-input" />
            <label class="auth-label">密码:</label>
            <input v-model="currentRequest.basicPassword" type="password" placeholder="输入密码" class="auth-input" />
          </div>

          <!-- API Key -->
          <div v-if="currentRequest.authType === 'api-key'" class="api-key-auth">
            <label class="auth-label">Key:</label>
            <input v-model="currentRequest.apiKey" type="text" placeholder="输入API Key" class="auth-input" />
            <label class="auth-label">Value:</label>
            <input v-model="currentRequest.apiValue" type="text" placeholder="输入API Value" class="auth-input" />
            <label class="auth-label">添加到:</label>
            <select v-model="currentRequest.apiAddTo" class="auth-select">
              <option value="header">Header</option>
              <option value="query">Query Parameter</option>
            </select>
          </div>
        </div>

        <!-- 测试选项卡 -->
        <div v-if="activeTab === 'tests'" class="tests-content">
          <div class="tests-editor">
            <label class="tests-label">测试脚本 (JavaScript):</label>
            <textarea
              v-model="currentRequest.tests"
              placeholder="// 编写测试脚本
// 示例:
pm.test('状态码是200', function () {
    pm.response.to.have.status(200);
});

pm.test('响应时间小于200ms', function () {
    pm.expect(pm.response.responseTime).to.be.below(200);
});"
              class="tests-textarea"
            ></textarea>
          </div>
          <div class="tests-actions">
            <button class="test-action-btn" title="运行测试" @click="runTests">
              <Icon type="play" size="xs" />
              运行测试
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- 响应区域 -->
    <div v-if="response" class="response-area">
      <!-- 响应头部 -->
      <div class="response-header">
        <div class="response-info">
          <div class="status-info">
            <span :class="['status-code', getStatusClass(response.status)]">
              {{ response.status }}
            </span>
            <span class="status-text">{{ response.statusText }}</span>
          </div>
          <div class="response-stats">
            <span class="response-time">{{ response.responseTime }}ms</span>
            <span class="response-size">{{ formatSize(response.size) }}</span>
          </div>
        </div>
        <div class="response-actions">
          <button class="response-action-btn" title="保存响应" @click="saveResponse">
            <Icon type="save" size="xs" />
          </button>
          <button class="response-action-btn" title="复制响应" @click="copyResponse">
            <Icon type="copy" size="xs" />
          </button>
          <button class="response-action-btn" title="下载响应" @click="downloadResponse">
            <Icon type="download" size="xs" />
          </button>
          <button class="response-action-btn" title="清空响应" @click="clearResponse">
            <Icon type="trash" size="xs" />
          </button>
        </div>
      </div>

      <!-- 响应选项卡 -->
      <div class="response-tabs">
        <button v-for="tab in responseTabs" :key="tab.key" :class="['response-tab-button', { active: activeResponseTab === tab.key }]" @click="activeResponseTab = tab.key">
          {{ tab.label }}
          <span v-if="tab.badge" class="tab-badge">{{ tab.badge }}</span>
        </button>
      </div>

      <!-- 响应内容 -->
      <div class="response-content">
        <!-- 响应体 -->
        <div v-if="activeResponseTab === 'body'" class="response-body">
          <div v-if="isJsonResponse(response)" class="json-viewer">
            <pre>{{ formatJson(response.data) }}</pre>
          </div>
          <div v-else class="raw-viewer">
            <pre>{{ response.data }}</pre>
          </div>
        </div>

        <!-- 响应头部 -->
        <div v-if="activeResponseTab === 'headers'" class="response-headers">
          <div class="headers-list">
            <div v-for="(value, key) in response.headers" :key="key" class="header-item">
              <div class="header-name">{{ key }}:</div>
              <div class="header-value">{{ value }}</div>
            </div>
          </div>
        </div>

        <!-- Cookies -->
        <div v-if="activeResponseTab === 'cookies'" class="response-cookies">
          <div class="cookies-list">
            <div v-for="(cookie, index) in response.cookies" :key="index" class="cookie-item">
              <div class="cookie-name">{{ cookie.name }}</div>
              <div class="cookie-value">{{ cookie.value }}</div>
              <div class="cookie-details">
                <span v-if="cookie.domain" class="cookie-domain">域名: {{ cookie.domain }}</span>
                <span v-if="cookie.path" class="cookie-path">路径: {{ cookie.path }}</span>
                <span v-if="cookie.expires" class="cookie-expires">过期: {{ cookie.expires }}</span>
              </div>
            </div>
          </div>
        </div>

        <!-- 测试结果 -->
        <div v-if="activeResponseTab === 'tests'" class="test-results">
          <div v-if="testResults.length === 0" class="no-tests">
            <p>暂无测试结果</p>
            <button class="run-tests-btn" @click="runTests">
              <Icon type="play" size="xs" />
              运行测试
            </button>
          </div>
          <div v-else class="test-list">
            <div v-for="(test, index) in testResults" :key="index" :class="['test-item', { passed: test.passed, failed: !test.passed }]">
              <Icon :type="test.passed ? 'check-circle' : 'x-circle'" size="sm" />
              <span class="test-name">{{ test.name }}</span>
              <span v-if="!test.passed" class="test-error">{{ test.error }}</span>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 历史记录面板 -->
    <div v-if="showHistory" class="history-panel">
      <div class="history-header">
        <h4>请求历史</h4>
        <button class="history-close-btn" @click="showHistory = false">
          <Icon type="close" size="xs" />
        </button>
      </div>
      <div class="history-list">
        <div v-for="(item, index) in requestHistory" :key="index" class="history-item" @click="loadHistoryRequest(item)">
          <div class="history-method">{{ item.method }}</div>
          <div class="history-url">{{ item.url }}</div>
          <div class="history-time">{{ formatTime(item.timestamp) }}</div>
          <div class="history-status">
            <span :class="['history-status-code', getStatusClass(item.status)]">
              {{ item.status }}
            </span>
          </div>
        </div>
      </div>
    </div>

    <!-- 历史按钮 -->
    <div v-if="requestHistory.length > 0" class="history-toggle">
      <button class="history-button" @click="showHistory = !showHistory">
        <Icon type="clock" size="xs" />
        历史 ({{ requestHistory.length }})
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick, onMounted } from "vue";
import Icon from "../ChatUI/Icon.vue";

interface RequestParam {
  key: string;
  value: string;
  description: string;
}

interface RequestHeader {
  key: string;
  value: string;
  description: string;
}

interface FormDataItem {
  key: string;
  value: string;
  type: "text" | "file";
  filename?: string;
}

interface UrlencodedItem {
  key: string;
  value: string;
}

interface TestResult {
  name: string;
  passed: boolean;
  error?: string;
}

interface HistoryItem {
  method: string;
  url: string;
  status: number;
  timestamp: number;
  request: any;
}

interface Response {
  status: number;
  statusText: string;
  headers: Record<string, string>;
  data: any;
  responseTime: number;
  size: number;
  cookies: Array<{
    name: string;
    value: string;
    domain?: string;
    path?: string;
    expires?: string;
  }>;
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
  requestSent: [request: any, response: Response];
}>();

// 响应式数据
const currentRequest = ref({
  method: "GET",
  url: "",
  params: [] as RequestParam[],
  headers: [] as RequestHeader[],
  bodyType: "none",
  bodyLanguage: "json",
  bodyRaw: "",
  formData: [] as FormDataItem[],
  urlencoded: [] as UrlencodedItem[],
  binaryFile: null as File | null,
  authType: "none",
  bearerToken: "",
  basicUsername: "",
  basicPassword: "",
  apiKey: "",
  apiValue: "",
  apiAddTo: "header",
  tests: "",
});

const response = ref<Response | null>(null);
const isSending = ref(false);
const showSettings = ref(false);
const showHistory = ref(false);
const activeTab = ref("params");
const activeResponseTab = ref("body");
const testResults = ref<TestResult[]>([]);
const requestHistory = ref<HistoryItem[]>([]);

// 请求设置
const requestSettings = ref({
  timeout: 30,
  retryCount: 0,
  followRedirects: true,
  verifySSL: true,
  saveToHistory: true,
});

// 选项卡配置
const requestTabs = ref<Array<{ key: string; label: string; icon: string; badge?: number }>>([
  { key: "params", label: "参数", icon: "hash" },
  { key: "headers", label: "头部", icon: "list" },
  { key: "body", label: "请求体", icon: "file-text" },
  { key: "auth", label: "认证", icon: "lock" },
  { key: "tests", label: "测试", icon: "check" },
]);

const responseTabs = ref<Array<{ key: string; label: string; icon: string; badge?: number }>>([
  { key: "body", label: "响应体", icon: "file-text" },
  { key: "headers", label: "头部", icon: "list" },
  { key: "cookies", label: "Cookies", icon: "disc" },
  { key: "tests", label: "测试", icon: "check" },
]);

const binaryFileInput = ref<HTMLInputElement>();

// 计算属性
const hasParams = computed(() => currentRequest.value.params.length > 0);
const hasHeaders = computed(() => currentRequest.value.headers.length > 0);

// 动态更新选项卡徽章
watch(
  () => currentRequest.value.params,
  (params) => {
    const tab = requestTabs.value.find((t) => t.key === "params");
    if (tab) {
      tab.badge = params.length > 0 ? params.length : undefined;
    }
  },
  { deep: true },
);

watch(
  () => currentRequest.value.headers,
  (headers) => {
    const tab = requestTabs.value.find((t) => t.key === "headers");
    if (tab) {
      tab.badge = headers.length > 0 ? headers.length : undefined;
    }
  },
  { deep: true },
);

// 参数管理
const addParam = () => {
  currentRequest.value.params.push({ key: "", value: "", description: "" });
};

const removeParam = (index: number) => {
  currentRequest.value.params.splice(index, 1);
};

// 头部管理
const addHeader = () => {
  currentRequest.value.headers.push({ key: "", value: "", description: "" });
};

const removeHeader = (index: number) => {
  currentRequest.value.headers.splice(index, 1);
};

// 表单数据管理
const addFormDataItem = () => {
  currentRequest.value.formData.push({ key: "", value: "", type: "text" });
};

const removeFormDataItem = (index: number) => {
  currentRequest.value.formData.splice(index, 1);
};

const handleFileSelect = (index: number, event: Event) => {
  const target = event.target as HTMLInputElement;
  const file = target.files?.[0];
  if (file && currentRequest.value.formData[index]) {
    currentRequest.value.formData[index].filename = file.name;
    currentRequest.value.formData[index].value = URL.createObjectURL(file);
  }
};

// URL编码数据管理
const addUrlencodedItem = () => {
  currentRequest.value.urlencoded.push({ key: "", value: "" });
};

const removeUrlencodedItem = (index: number) => {
  currentRequest.value.urlencoded.splice(index, 1);
};

// 二进制文件处理
const handleBinaryFileSelect = (event: Event) => {
  const target = event.target as HTMLInputElement;
  const file = target.files?.[0];
  if (file) {
    currentRequest.value.binaryFile = file;
  }
};

// 请求发送
const sendRequest = async () => {
  if (!currentRequest.value.url.trim() || isSending.value) return;

  isSending.value = true;
  const startTime = Date.now();

  try {
    // 构建请求配置
    const requestConfig = buildRequestConfig();

    // 这里应该通过WebSocket发送请求到后端
    // 模拟响应
    const mockResponse = await mockHttpRequest(requestConfig);

    const responseTime = Date.now() - startTime;
    response.value = {
      status: mockResponse.status ?? 0,
      statusText: mockResponse.statusText ?? "",
      headers: mockResponse.headers ?? {},
      data: mockResponse.data,
      cookies: mockResponse.cookies ?? [],
      responseTime,
      size: JSON.stringify(mockResponse.data).length,
    };

    // 保存到历史记录
    if (requestSettings.value.saveToHistory && response.value) {
      addToHistory(currentRequest.value.method, currentRequest.value.url, response.value.status);
    }

    // 运行测试
    if (currentRequest.value.tests.trim()) {
      runTests();
    }

    if (response.value) {
      emit("requestSent", requestConfig, response.value);
    }
  } catch (error: unknown) {
    console.error("Request failed:", error);
    response.value = {
      status: 0,
      statusText: "Request Failed",
      headers: {},
      data: error instanceof Error ? error.message : String(error),
      responseTime: Date.now() - startTime,
      size: 0,
      cookies: [],
    };
  } finally {
    isSending.value = false;
  }
};

const buildRequestConfig = () => {
  const config: any = {
    method: currentRequest.value.method,
    url: currentRequest.value.url,
    headers: {},
    timeout: requestSettings.value.timeout * 1000,
  };

  // 构建查询参数
  if (currentRequest.value.params.length > 0) {
    const params = new URLSearchParams();
    currentRequest.value.params.forEach((param) => {
      if (param.key && param.value) {
        params.append(param.key, param.value);
      }
    });
    const paramString = params.toString();
    if (paramString) {
      config.url += (config.url.includes("?") ? "&" : "?") + paramString;
    }
  }

  // 添加头部
  currentRequest.value.headers.forEach((header) => {
    if (header.key && header.value) {
      config.headers[header.key] = header.value;
    }
  });

  // 添加认证
  if (currentRequest.value.authType === "bearer" && currentRequest.value.bearerToken) {
    config.headers.Authorization = `Bearer ${currentRequest.value.bearerToken}`;
  } else if (currentRequest.value.authType === "basic" && currentRequest.value.basicUsername) {
    const credentials = btoa(`${currentRequest.value.basicUsername}:${currentRequest.value.basicPassword}`);
    config.headers.Authorization = `Basic ${credentials}`;
  } else if (currentRequest.value.authType === "api-key") {
    if (currentRequest.value.apiAddTo === "header") {
      config.headers[currentRequest.value.apiKey] = currentRequest.value.apiValue;
    }
  }

  // 构建请求体
  if (["POST", "PUT", "PATCH"].includes(currentRequest.value.method)) {
    if (currentRequest.value.bodyType === "raw" && currentRequest.value.bodyRaw) {
      config.body = currentRequest.value.bodyRaw;
      if (currentRequest.value.bodyLanguage === "json") {
        config.headers["Content-Type"] = "application/json";
      }
    } else if (currentRequest.value.bodyType === "x-www-form-urlencoded") {
      const formData = new URLSearchParams();
      currentRequest.value.urlencoded.forEach((item) => {
        if (item.key && item.value) {
          formData.append(item.key, item.value);
        }
      });
      config.body = formData.toString();
      config.headers["Content-Type"] = "application/x-www-form-urlencoded";
    }
    // form-data 和 binary 需要特殊处理
  }

  return config;
};

const mockHttpRequest = async (config: any): Promise<Partial<Response>> => {
  // 模拟HTTP请求延迟
  await new Promise((resolve) => setTimeout(resolve, 500 + Math.random() * 1000));

  // 简单的模拟响应
  try {
    const url = new URL(config.url);

    if (url.pathname.includes("/users")) {
      return {
        status: 200,
        statusText: "OK",
        headers: {
          "Content-Type": "application/json",
          "X-Powered-By": "AsterCloud",
        },
        data: {
          users: [
            { id: 1, name: "John Doe", email: "john@example.com" },
            { id: 2, name: "Jane Smith", email: "jane@example.com" },
          ],
        },
        cookies: [],
      };
    } else {
      return {
        status: 404,
        statusText: "Not Found",
        headers: {
          "Content-Type": "application/json",
        },
        data: { error: "Not Found" },
        cookies: [],
      };
    }
  } catch (error) {
    return {
      status: 400,
      statusText: "Bad Request",
      headers: {
        "Content-Type": "application/json",
      },
      data: { error: "Invalid URL" },
      cookies: [],
    };
  }
};

// 测试功能
const runTests = () => {
  if (!currentRequest.value.tests.trim() || !response.value) {
    testResults.value = [];
    return;
  }

  testResults.value = [];

  try {
    // 简化的测试执行器
    const tests = currentRequest.value.tests.split("\n").filter((line) => line.trim());

    tests.forEach((testLine) => {
      const match = testLine.match(/pm\.test\(['"`]([^'"`]+)['"`],/);
      if (match) {
        const testName = match[1] ?? "";
        let passed = false;
        let errorMsg = "";

        try {
          // 简单的状态码测试
          if (testName.includes("状态码")) {
            const statusMatch = testName.match(/(\d+)/);
            if (statusMatch) {
              const expectedStatus = parseInt(statusMatch[1] ?? "0");
              passed = response.value?.status === expectedStatus;
            }
          }

          // 简单的响应时间测试
          if (testName.includes("响应时间")) {
            const timeMatch = testName.match(/(\d+)ms/);
            if (timeMatch) {
              const maxTime = parseInt(timeMatch[1] ?? "0");
              passed = (response.value?.responseTime ?? 0) < maxTime;
            }
          }
        } catch (e: unknown) {
          errorMsg = e instanceof Error ? e.message : String(e);
        }

        testResults.value.push({
          name: testName,
          passed,
          error: errorMsg,
        });
      }
    });
  } catch (error: unknown) {
    testResults.value.push({
      name: "测试执行错误",
      passed: false,
      error: error instanceof Error ? error.message : String(error),
    });
  }
};

// 工具方法
const createNewRequest = () => {
  currentRequest.value = {
    method: "GET",
    url: "",
    params: [],
    headers: [],
    bodyType: "none",
    bodyLanguage: "json",
    bodyRaw: "",
    formData: [],
    urlencoded: [],
    binaryFile: null,
    authType: "none",
    bearerToken: "",
    basicUsername: "",
    basicPassword: "",
    apiKey: "",
    apiValue: "",
    apiAddTo: "header",
    tests: "",
  };
  response.value = null;
  testResults.value = [];
};

const importRequest = () => {
  // 实现导入功能
  const input = document.createElement("input");
  input.type = "file";
  input.accept = ".json";
  input.onchange = (e) => {
    const file = (e.target as HTMLInputElement).files?.[0];
    if (file) {
      const reader = new FileReader();
      reader.onload = (e) => {
        try {
          const data = JSON.parse(e.target?.result as string);
          currentRequest.value = { ...currentRequest.value, ...data };
        } catch (error) {
          console.error("Failed to import request:", error);
        }
      };
      reader.readAsText(file);
    }
  };
  input.click();
};

const exportRequest = () => {
  if (!currentRequest.value) return;

  const dataStr = JSON.stringify(currentRequest.value, null, 2);
  const blob = new Blob([dataStr], { type: "application/json" });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = `request-${Date.now()}.json`;
  a.click();
  URL.revokeObjectURL(url);
};

const getBodyPlaceholder = () => {
  const placeholders: Record<string, string> = {
    text: "输入纯文本内容",
    json: '{\n  "key": "value"\n}',
    xml: '<?xml version="1.0" encoding="UTF-8"?>\n<root></root>',
    html: "<!DOCTYPE html>\n<html><head></head><body></body></html>",
  };
  return placeholders[currentRequest.value.bodyLanguage] || "输入内容";
};

const formatRawBody = () => {
  if (currentRequest.value.bodyLanguage === "json") {
    try {
      const parsed = JSON.parse(currentRequest.value.bodyRaw);
      currentRequest.value.bodyRaw = JSON.stringify(parsed, null, 2);
    } catch (error) {
      alert("JSON格式错误");
    }
  }
};

const validateRawBody = () => {
  if (currentRequest.value.bodyLanguage === "json") {
    try {
      JSON.parse(currentRequest.value.bodyRaw);
      alert("JSON格式正确");
    } catch (error: unknown) {
      alert("JSON格式错误: " + (error instanceof Error ? error.message : String(error)));
    }
  }
};

const getStatusClass = (status: number) => {
  if (status >= 200 && status < 300) return "success";
  if (status >= 300 && status < 400) return "redirect";
  if (status >= 400 && status < 500) return "client-error";
  if (status >= 500) return "server-error";
  return "default";
};

const isJsonResponse = (response: Response | null) => {
  if (!response) return false;
  const contentType = response.headers["content-type"] || "";
  return contentType.includes("application/json");
};

const formatJson = (data: any) => {
  try {
    return JSON.stringify(data, null, 2);
  } catch (error) {
    return data;
  }
};

const formatSize = (bytes: number) => {
  if (bytes === 0) return "0 B";
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + " " + sizes[i];
};

const formatTime = (timestamp: number) => {
  const date = new Date(timestamp);
  return date.toLocaleString("zh-CN");
};

const saveResponse = () => {
  if (!response.value) return;

  const data = JSON.stringify(response.value, null, 2);
  const blob = new Blob([data], { type: "application/json" });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = `response-${Date.now()}.json`;
  a.click();
  URL.revokeObjectURL(url);
};

const copyResponse = () => {
  if (!response.value) return;

  const text = typeof response.value.data === "string" ? response.value.data : JSON.stringify(response.value.data, null, 2);

  navigator.clipboard.writeText(text);
};

const downloadResponse = () => {
  if (!response.value) return;

  const contentType = response.value.headers["content-type"] || "text/plain";
  const blob = new Blob([response.value.data], { type: contentType });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = `response-${Date.now()}`;
  a.click();
  URL.revokeObjectURL(url);
};

const clearResponse = () => {
  response.value = null;
  testResults.value = [];
};

// 历史记录管理
const addToHistory = (method: string, url: string, status: number) => {
  const historyItem: HistoryItem = {
    method,
    url,
    status,
    timestamp: Date.now(),
    request: JSON.parse(JSON.stringify(currentRequest.value)),
  };

  requestHistory.value.unshift(historyItem);
  if (requestHistory.value.length > 50) {
    requestHistory.value = requestHistory.value.slice(0, 50);
  }

  // 保存到本地存储
  try {
    localStorage.setItem("http-request-history", JSON.stringify(requestHistory.value));
  } catch (error) {
    console.warn("Failed to save history:", error);
  }
};

const loadHistoryRequest = (item: HistoryItem) => {
  currentRequest.value = JSON.parse(JSON.stringify(item.request));
  showHistory.value = false;
};

// 设置管理
const toggleSettings = () => {
  showSettings.value = !showSettings.value;
};

const saveSettings = () => {
  localStorage.setItem("http-request-settings", JSON.stringify(requestSettings.value));
  showSettings.value = false;
};

const resetSettings = () => {
  requestSettings.value = {
    timeout: 30,
    retryCount: 0,
    followRedirects: true,
    verifySSL: true,
    saveToHistory: true,
  };
};

// 生命周期
onMounted(() => {
  // 加载设置
  try {
    const saved = localStorage.getItem("http-request-settings");
    if (saved) {
      requestSettings.value = { ...requestSettings.value, ...JSON.parse(saved) };
    }
  } catch (error) {
    console.warn("Failed to load request settings:", error);
  }

  // 加载历史记录
  try {
    const saved = localStorage.getItem("http-request-history");
    if (saved) {
      requestHistory.value = JSON.parse(saved);
    }
  } catch (error) {
    console.warn("Failed to load request history:", error);
  }
});
</script>

<style scoped>
.http-request-tool {
  @apply flex flex-col h-full bg-surface dark:bg-surface-dark border border-border dark:border-border-dark rounded-lg;
}

.request-header {
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

.setting-input {
  @apply w-24 px-2 py-1 text-sm border border-border dark:border-border-dark rounded focus:outline-none focus:ring-2 focus:ring-blue-500 dark:bg-gray-800 dark:text-white;
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

.request-builder {
  @apply flex-1 flex flex-col overflow-hidden;
}

.url-section {
  @apply flex items-center gap-2 p-4 border-b border-gray-200 dark:border-gray-600;
}

.method-select {
  @apply px-3 py-2 border border-border dark:border-border-dark rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 dark:bg-gray-800 dark:text-white font-medium;
}

.url-input-group {
  @apply flex-1 flex items-center gap-2;
}

.url-input {
  @apply flex-1 px-3 py-2 border border-border dark:border-border-dark rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 dark:bg-gray-800 dark:text-white;
}

.send-button {
  @apply flex items-center gap-2 px-4 py-2 bg-blue-500 hover:bg-blue-600 text-white rounded-md transition-colors disabled:opacity-50 disabled:cursor-not-allowed;
}

.request-tabs {
  @apply flex border-b border-gray-200 dark:border-gray-600;
}

.tab-button {
  @apply flex items-center gap-2 px-4 py-2 text-sm text-gray-600 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-600 transition-colors;
}

.tab-button.active {
  @apply text-blue-600 dark:text-blue-400 bg-blue-50 dark:bg-blue-900/20;
}

.tab-badge {
  @apply px-1.5 py-0.5 text-xs bg-blue-100 dark:bg-blue-800 text-blue-600 dark:text-blue-300 rounded-full;
}

.tab-content {
  @apply flex-1 overflow-y-auto p-4;
}

/* 参数表格 */
.params-content,
.headers-content {
  @apply space-y-3;
}

.params-table,
.headers-table,
.form-data-table,
.urlencoded-table {
  @apply border border-gray-200 dark:border-gray-600 rounded-lg overflow-hidden;
}

.table-header {
  @apply flex bg-gray-50 dark:bg-gray-700 border-b border-gray-200 dark:border-gray-600;
}

.header-cell {
  @apply flex-1 px-3 py-2 text-xs font-medium text-gray-700 dark:text-gray-300;
}

.table-row {
  @apply flex border-b border-gray-100 dark:border-gray-700 last:border-b-0;
}

.table-cell {
  @apply flex-1 px-3 py-2;
}

.param-input,
.header-input,
.form-input,
.url-input {
  @apply w-full px-2 py-1 text-sm border border-gray-200 dark:border-gray-600 rounded focus:outline-none focus:ring-1 focus:ring-blue-500 dark:bg-gray-700 dark:text-white;
}

.header-select,
.form-type-select {
  @apply w-full px-2 py-1 text-sm border border-gray-200 dark:border-gray-600 rounded focus:outline-none focus:ring-1 focus:ring-blue-500 dark:bg-gray-700 dark:text-white;
}

.row-action-btn {
  @apply p-1 text-gray-400 dark:text-gray-500 hover:text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20 rounded transition-colors;
}

.add-param-btn,
.add-header-btn,
.add-form-data-btn,
.add-urlencoded-btn {
  @apply flex items-center gap-2 px-3 py-2 text-sm text-blue-600 dark:text-blue-400 border border-blue-200 dark:border-blue-600 rounded hover:bg-blue-50 dark:hover:bg-blue-900/20 transition-colors;
}

/* 请求体 */
.body-type-selector,
.auth-type-selector {
  @apply flex items-center gap-3 mb-4;
}

.body-type-label,
.auth-type-label {
  @apply text-sm font-medium text-gray-700 dark:text-gray-300;
}

.body-type-select,
.body-language-select,
.auth-type-select {
  @apply px-3 py-1 text-sm border border-gray-200 dark:border-gray-600 rounded focus:outline-none focus:ring-1 focus:ring-blue-500 dark:bg-gray-700 dark:text-white;
}

.raw-content,
.binary-content {
  @apply space-y-3;
}

.raw-editor {
  @apply border border-gray-200 dark:border-gray-600 rounded-lg overflow-hidden;
}

.raw-textarea {
  @apply w-full h-64 px-3 py-2 font-mono text-sm resize-none focus:outline-none bg-white dark:bg-gray-800 text-gray-800 dark:text-gray-200;
}

.raw-actions {
  @apply flex gap-2;
}

.raw-action-btn,
.test-action-btn {
  @apply flex items-center gap-1 px-3 py-1 text-sm text-gray-600 dark:text-gray-400 border border-gray-200 dark:border-gray-600 rounded hover:bg-gray-50 dark:hover:bg-gray-600 transition-colors;
}

.file-input-wrapper,
.binary-input-wrapper {
  @apply flex items-center gap-3 p-4 border border-gray-200 dark:border-gray-600 rounded-lg;
}

.file-input,
.binary-input {
  @apply hidden;
}

.file-select-btn,
.binary-select-btn {
  @apply flex items-center gap-2 px-3 py-1 text-sm text-blue-600 dark:text-blue-400 border border-blue-200 dark:border-blue-600 rounded hover:bg-blue-50 dark:hover:bg-blue-900/20 transition-colors;
}

.filename {
  @apply text-sm text-gray-600 dark:text-gray-400;
}

/* 认证 */
.auth-content {
  @apply space-y-4;
}

.auth-label {
  @apply block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1;
}

.auth-input,
.auth-select {
  @apply w-full px-3 py-2 text-sm border border-gray-200 dark:border-gray-600 rounded focus:outline-none focus:ring-1 focus:ring-blue-500 dark:bg-gray-700 dark:text-white;
}

/* 测试 */
.tests-content {
  @apply space-y-3;
}

.tests-label {
  @apply block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1;
}

.tests-textarea {
  @apply w-full h-48 px-3 py-2 font-mono text-sm border border-gray-200 dark:border-gray-600 rounded-lg resize-none focus:outline-none focus:ring-1 focus:ring-blue-500 dark:bg-gray-700 dark:text-white;
}

.tests-actions {
  @apply flex gap-2;
}

/* 响应区域 */
.response-area {
  @apply border-t border-gray-200 dark:border-gray-600;
}

.response-header {
  @apply flex items-center justify-between px-4 py-3 bg-gray-50 dark:bg-gray-700 border-b border-gray-200 dark:border-gray-600;
}

.response-info {
  @apply flex items-center gap-4;
}

.status-info {
  @apply flex items-center gap-2;
}

.status-code {
  @apply px-2 py-1 text-xs font-semibold rounded;
}

.status-code.success {
  @apply bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400;
}

.status-code.redirect {
  @apply bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400;
}

.status-code.client-error {
  @apply bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400;
}

.status-code.server-error {
  @apply bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400;
}

.status-text {
  @apply text-sm text-gray-600 dark:text-gray-400;
}

.response-stats {
  @apply flex items-center gap-3 text-xs text-gray-500 dark:text-gray-400;
}

.response-actions {
  @apply flex gap-1;
}

.response-action-btn {
  @apply p-1.5 text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-600 rounded transition-colors;
}

.response-tabs {
  @apply flex border-b border-gray-200 dark:border-gray-600;
}

.response-tab-button {
  @apply px-4 py-2 text-sm text-gray-600 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-600 transition-colors;
}

.response-tab-button.active {
  @apply text-blue-600 dark:text-blue-400 bg-blue-50 dark:bg-blue-900/20;
}

.response-content {
  @apply h-64 overflow-y-auto p-4;
}

.response-body {
  @apply h-full;
}

.json-viewer,
.raw-viewer {
  @apply h-full;
}

.json-viewer pre,
.raw-viewer pre {
  @apply text-sm font-mono text-gray-800 dark:text-gray-200 whitespace-pre-wrap break-all;
}

.response-headers {
  @apply space-y-2;
}

.header-item {
  @apply flex gap-2 text-sm;
}

.header-name {
  @apply font-medium text-gray-700 dark:text-gray-300 min-w-0;
}

.header-value {
  @apply text-gray-600 dark:text-gray-400 break-all;
}

.response-cookies {
  @apply space-y-3;
}

.cookie-item {
  @apply p-3 bg-gray-50 dark:bg-gray-700 rounded-lg;
}

.cookie-name {
  @apply font-medium text-gray-800 dark:text-gray-200;
}

.cookie-value {
  @apply text-sm text-gray-600 dark:text-gray-400 mb-2;
}

.cookie-details {
  @apply flex flex-wrap gap-3 text-xs text-gray-500 dark:text-gray-400;
}

.test-results {
  @apply h-full;
}

.no-tests {
  @apply flex flex-col items-center justify-center h-full text-gray-400 dark:text-gray-500;
}

.run-tests-btn {
  @apply flex items-center gap-2 px-4 py-2 mt-4 text-sm text-blue-600 dark:text-blue-400 border border-blue-200 dark:border-blue-600 rounded hover:bg-blue-50 dark:hover:bg-blue-900/20 transition-colors;
}

.test-list {
  @apply space-y-2;
}

.test-item {
  @apply flex items-center gap-2 p-3 rounded-lg;
}

.test-item.passed {
  @apply bg-green-50 dark:bg-green-900/20 text-green-700 dark:text-green-400;
}

.test-item.failed {
  @apply bg-red-50 dark:bg-red-900/20 text-red-700 dark:text-red-400;
}

.test-name {
  @apply text-sm font-medium;
}

.test-error {
  @apply text-xs text-red-600 dark:text-red-400;
}

/* 历史记录 */
.history-panel {
  @apply absolute top-0 right-0 w-80 h-full bg-white dark:bg-gray-800 border-l border-gray-200 dark:border-gray-600 shadow-lg z-10;
}

.history-header {
  @apply flex items-center justify-between px-4 py-3 border-b border-gray-200 dark:border-gray-600;
}

.history-header h4 {
  @apply text-sm font-semibold text-gray-800 dark:text-gray-200;
}

.history-close-btn {
  @apply p-1 text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-600 rounded transition-colors;
}

.history-list {
  @apply h-full overflow-y-auto p-2 space-y-2;
}

.history-item {
  @apply p-3 bg-gray-50 dark:bg-gray-700 rounded-lg cursor-pointer hover:bg-gray-100 dark:hover:bg-gray-600 transition-colors;
}

.history-method {
  @apply inline-block px-2 py-1 text-xs font-semibold text-blue-600 dark:text-blue-400 bg-blue-100 dark:bg-blue-900/30 rounded mb-1;
}

.history-url {
  @apply text-sm text-gray-800 dark:text-gray-200 truncate mb-1;
}

.history-time {
  @apply text-xs text-gray-500 dark:text-gray-400;
}

.history-status-code {
  @apply px-1.5 py-0.5 text-xs font-semibold rounded;
}

.history-toggle {
  @apply absolute bottom-4 right-4 z-20;
}

.history-button {
  @apply flex items-center gap-2 px-3 py-2 text-sm bg-blue-500 hover:bg-blue-600 text-white rounded-full shadow-lg transition-colors;
}

.animate-spin {
  @apply animate-spin;
}
</style>

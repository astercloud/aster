<template>
  <div class="chatui-components-page">
    <Navbar title="ChatUI 组件库">
      <template #menu>
        <a v-for="section in sections" :key="section.id" :href="`#${section.id}`" class="nav-link">
          {{ section.name }}
        </a>
      </template>
      <template #actions>
        <Button variant="primary">GitHub</Button>
      </template>
    </Navbar>

    <div class="page-container">
      <Sidebar title="组件导航" collapsible>
        <List :items="allComponents" @select="scrollToComponent">
          <template #default="{ item }">
            <div class="component-item">
              <span>{{ item.name }}</span>
              <Tag size="sm" :color="item.category === 'chat' ? 'primary' : 'default'">
                {{ item.category }}
              </Tag>
            </div>
          </template>
        </List>
      </Sidebar>

      <ScrollView class="main-content">
        <!-- Hero -->
        <section class="hero-section">
          <h1 class="hero-title">ChatUI 组件库</h1>
          <p class="hero-subtitle">参考 ChatUI 设计的完整对话界面组件库，专为 Aster Agent 打造</p>
          <Flex justify="center" gap="md">
            <Button variant="primary" size="lg">快速开始</Button>
            <Button variant="secondary" size="lg">查看文档</Button>
          </Flex>
        </section>

        <Divider>组件展示</Divider>

        <!-- 对话组件 -->
        <section id="chat" class="component-section">
          <h2 class="section-title">对话组件</h2>

          <div id="demo-bubble" class="demo-card" @click="showComponentDoc('bubble')">
            <div class="demo-header">
              <h3 class="demo-title">Bubble - 消息气泡</h3>
              <p v-if="hasDoc('bubble')" class="demo-description">
                {{ getDocDescription("bubble") }}
              </p>
            </div>
            <div class="demo-content" @click.stop>
              <Flex direction="column" gap="md">
                <Bubble content="你好！我是 Aster Agent" position="left" />
                <Bubble content="很高兴认识你" position="right" status="sent" />
              </Flex>
            </div>
            <div v-if="hasDoc('bubble')" class="demo-footer">
              <button class="view-docs-link" @click.stop="showComponentDoc('bubble')">查看完整文档 →</button>
            </div>
          </div>

          <div id="demo-think-bubble" class="demo-card">
            <h3 class="demo-title">ThinkBubble - 思考气泡</h3>
            <div class="demo-content">
              <ThinkBubble />
            </div>
          </div>

          <div id="demo-card" class="demo-card">
            <h3 class="demo-title">Card - 卡片消息</h3>
            <div class="demo-content">
              <Card
                title="推荐文章"
                content="这是一篇关于 AI Agent 的深度文章"
                :actions="[
                  { text: '查看详情', value: 'view' },
                  { text: '分享', value: 'share' },
                ]"
              />
            </div>
          </div>

          <div id="demo-system-message" class="demo-card">
            <h3 class="demo-title">SystemMessage - 系统消息</h3>
            <div class="demo-content">
              <SystemMessage content="Agent 已加入对话" />
            </div>
          </div>

          <!-- Chat 聊天容器 -->
          <div id="demo-chat" class="demo-card placeholder">
            <div class="demo-header">
              <h3 class="demo-title">Chat - 聊天容器</h3>
              <p v-if="hasDoc('chat')" class="demo-description">
                {{ getDocDescription("chat") }}
              </p>
            </div>
            <div class="placeholder-content">
              <p class="placeholder-text">完整的聊天容器组件，包含消息列表、输入框等功能</p>
              <router-link v-if="hasDoc('chat')" :to="`/docs/chat`" class="view-docs-button"> 查看完整文档和示例 → </router-link>
            </div>
          </div>

          <!-- TypingBubble 输入中 -->
          <div id="demo-typing-bubble" class="demo-card placeholder">
            <div class="demo-header">
              <h3 class="demo-title">TypingBubble - 输入中</h3>
              <p class="demo-description">显示对方正在输入的气泡组件</p>
            </div>
            <div class="placeholder-content">
              <p class="placeholder-text">Demo 开发中</p>
              <router-link :to="`/docs/typing-bubble`" class="view-docs-button"> 查看文档 → </router-link>
            </div>
          </div>

          <!-- FileCard 文件卡片 -->
          <div id="demo-file-card" class="demo-card placeholder">
            <div class="demo-header">
              <h3 class="demo-title">FileCard - 文件卡片</h3>
              <p class="demo-description">用于显示文件信息的卡片组件</p>
            </div>
            <div class="placeholder-content">
              <p class="placeholder-text">Demo 开发中</p>
              <router-link :to="`/docs/file-card`" class="view-docs-button"> 查看文档 → </router-link>
            </div>
          </div>

          <!-- MessageStatus 消息状态 -->
          <div id="demo-message-status" class="demo-card placeholder">
            <div class="demo-header">
              <h3 class="demo-title">MessageStatus - 消息状态</h3>
              <p class="demo-description">显示消息发送、已读等状态</p>
            </div>
            <div class="placeholder-content">
              <p class="placeholder-text">Demo 开发中</p>
              <router-link :to="`/docs/message-status`" class="view-docs-button"> 查看文档 → </router-link>
            </div>
          </div>
        </section>

        <!-- 基础组件 -->
        <section id="basic" class="component-section">
          <h2 class="section-title">基础组件</h2>

          <div id="demo-button" class="demo-card" @click="showComponentDoc('button')">
            <div class="demo-header">
              <h3 class="demo-title">Button - 按钮</h3>
              <p v-if="hasDoc('button')" class="demo-description">
                {{ getDocDescription("button") }}
              </p>
            </div>
            <div class="demo-content" @click.stop>
              <Flex gap="md" wrap>
                <Button variant="primary">主要按钮</Button>
                <Button variant="secondary">次要按钮</Button>
                <Button variant="text">文本按钮</Button>
                <Button variant="primary" icon="send">发送</Button>
              </Flex>
            </div>
            <div v-if="hasDoc('button')" class="demo-footer">
              <button class="view-docs-link" @click.stop="showComponentDoc('button')">查看完整文档 →</button>
            </div>
          </div>

          <div id="demo-avatar" class="demo-card" @click="showComponentDoc('avatar')">
            <div class="demo-header">
              <h3 class="demo-title">Avatar - 头像</h3>
              <p v-if="hasDoc('avatar')" class="demo-description">
                {{ getDocDescription("avatar") }}
              </p>
            </div>
            <div class="demo-content" @click.stop>
              <Flex gap="md" align="center">
                <Avatar alt="User" size="xs" />
                <Avatar alt="Agent" size="sm" status="online" />
                <Avatar alt="Bot" size="md" status="busy" />
                <Avatar alt="AI" size="lg" />
                <Avatar alt="System" size="xl" status="offline" />
              </Flex>
            </div>
            <div v-if="hasDoc('avatar')" class="demo-footer">
              <button class="view-docs-link" @click.stop="showComponentDoc('avatar')">查看完整文档 →</button>
            </div>
          </div>

          <div id="demo-tag" class="demo-card">
            <h3 class="demo-title">Tag - 标签</h3>
            <div class="demo-content">
              <Flex gap="sm" wrap>
                <Tag>默认</Tag>
                <Tag color="primary">主要</Tag>
                <Tag color="success">成功</Tag>
                <Tag color="warning">警告</Tag>
                <Tag color="error">错误</Tag>
                <Tag closable @close="console.log('closed')">可关闭</Tag>
              </Flex>
            </div>
          </div>

          <!-- Icon 图标 -->
          <div id="demo-icon" class="demo-card placeholder">
            <div class="demo-header">
              <h3 class="demo-title">Icon - 图标</h3>
              <p v-if="hasDoc('icon')" class="demo-description">
                {{ getDocDescription("icon") }}
              </p>
            </div>
            <div class="placeholder-content">
              <p class="placeholder-text">内置常用图标组件</p>
              <router-link v-if="hasDoc('icon')" :to="`/docs/icon`" class="view-docs-button"> 查看完整文档和示例 → </router-link>
            </div>
          </div>

          <!-- Image 图片 -->
          <div id="demo-image" class="demo-card placeholder">
            <div class="demo-header">
              <h3 class="demo-title">Image - 图片</h3>
              <p v-if="hasDoc('image')" class="demo-description">
                {{ getDocDescription("image") }}
              </p>
            </div>
            <div class="placeholder-content">
              <p class="placeholder-text">支持懒加载、预览的图片组件</p>
              <router-link v-if="hasDoc('image')" :to="`/docs/image`" class="view-docs-button"> 查看完整文档和示例 → </router-link>
            </div>
          </div>
        </section>

        <!-- 表单组件 -->
        <section id="form" class="component-section">
          <h2 class="section-title">表单组件</h2>

          <div id="demo-input" class="demo-card">
            <div class="demo-header">
              <h3 class="demo-title">Input - 输入框</h3>
              <p v-if="hasDoc('input')" class="demo-description">
                {{ getDocDescription("input") }}
              </p>
            </div>
            <div class="demo-content">
              <Input v-model="inputValue" label="用户名" placeholder="请输入用户名" />
            </div>
            <div v-if="hasDoc('input')" class="demo-footer">
              <router-link :to="`/docs/input`" class="view-docs-link"> 查看完整文档 → </router-link>
            </div>
          </div>

          <div id="demo-search" class="demo-card">
            <h3 class="demo-title">Search - 搜索框</h3>
            <div class="demo-content">
              <Search v-model="searchValue" placeholder="搜索组件..." @search="handleSearch" />
            </div>
          </div>

          <div id="demo-checkbox" class="demo-card">
            <h3 class="demo-title">Checkbox & Radio</h3>
            <div class="demo-content">
              <Flex direction="column" gap="md">
                <Checkbox v-model="checked">同意用户协议</Checkbox>
                <Flex gap="md">
                  <Radio v-model="radioValue" value="a" name="demo">选项 A</Radio>
                  <Radio v-model="radioValue" value="b" name="demo">选项 B</Radio>
                </Flex>
              </Flex>
            </div>
          </div>

          <!-- Radio 单选框 -->
          <div id="demo-radio" class="demo-card placeholder">
            <div class="demo-header">
              <h3 class="demo-title">Radio - 单选框</h3>
              <p v-if="hasDoc('radio')" class="demo-description">
                {{ getDocDescription("radio") }}
              </p>
            </div>
            <div class="placeholder-content">
              <p class="placeholder-text">单选框组件，支持分组</p>
              <router-link v-if="hasDoc('radio')" :to="`/docs/radio`" class="view-docs-button"> 查看完整文档和示例 → </router-link>
            </div>
          </div>

          <!-- Dropdown 下拉菜单 -->
          <div id="demo-dropdown" class="demo-card placeholder">
            <div class="demo-header">
              <h3 class="demo-title">Dropdown - 下拉菜单</h3>
              <p v-if="hasDoc('dropdown')" class="demo-description">
                {{ getDocDescription("dropdown") }}
              </p>
            </div>
            <div class="placeholder-content">
              <p class="placeholder-text">下拉菜单组件，支持多级菜单</p>
              <router-link v-if="hasDoc('dropdown')" :to="`/docs/dropdown`" class="view-docs-button"> 查看完整文档和示例 → </router-link>
            </div>
          </div>

          <!-- MultimodalInput 多模态输入 -->
          <div id="demo-multimodal-input" class="demo-card placeholder">
            <div class="demo-header">
              <h3 class="demo-title">MultimodalInput - 多模态输入</h3>
              <p class="demo-description">支持文本、图片、语音等多种输入方式</p>
            </div>
            <div class="placeholder-content">
              <p class="placeholder-text">Demo 开发中</p>
            </div>
          </div>
        </section>

        <!-- 反馈组件 -->
        <section id="feedback" class="component-section">
          <h2 class="section-title">反馈组件</h2>

          <div id="demo-notice" class="demo-card">
            <div class="demo-header">
              <h3 class="demo-title">Notice - 通知提示</h3>
              <p v-if="hasDoc('notice')" class="demo-description">
                {{ getDocDescription("notice") }}
              </p>
            </div>
            <div class="demo-content">
              <Flex direction="column" gap="md">
                <Notice type="info" content="这是一条信息提示" />
                <Notice type="success" title="成功" content="操作已成功完成" closable />
                <Notice type="warning" content="请注意检查输入内容" />
                <Notice type="error" content="发生了一个错误" />
              </Flex>
            </div>
            <div v-if="hasDoc('notice')" class="demo-footer">
              <router-link :to="`/docs/notice`" class="view-docs-link"> 查看完整文档 → </router-link>
            </div>
          </div>

          <div id="demo-progress" class="demo-card">
            <h3 class="demo-title">Progress - 进度条</h3>
            <div class="demo-content">
              <Flex direction="column" gap="md">
                <Progress :percent="30" label="上传中" />
                <Progress :percent="100" status="success" label="已完成" />
                <Progress :percent="50" status="error" label="上传失败" />
              </Flex>
            </div>
          </div>

          <div id="demo-tooltip" class="demo-card">
            <h3 class="demo-title">Tooltip - 工具提示</h3>
            <div class="demo-content">
              <Flex gap="md">
                <Tooltip content="顶部提示" position="top">
                  <Button>上</Button>
                </Tooltip>
                <Tooltip content="右侧提示" position="right">
                  <Button>右</Button>
                </Tooltip>
                <Tooltip content="底部提示" position="bottom">
                  <Button>下</Button>
                </Tooltip>
                <Tooltip content="左侧提示" position="left">
                  <Button>左</Button>
                </Tooltip>
              </Flex>
            </div>
          </div>

          <!-- Modal 对话框 -->
          <div id="demo-modal" class="demo-card placeholder">
            <div class="demo-header">
              <h3 class="demo-title">Modal - 对话框</h3>
              <p v-if="hasDoc('modal')" class="demo-description">
                {{ getDocDescription("modal") }}
              </p>
            </div>
            <div class="placeholder-content">
              <p class="placeholder-text">对话框组件，支持自定义内容</p>
              <router-link v-if="hasDoc('modal')" :to="`/docs/modal`" class="view-docs-button"> 查看完整文档和示例 → </router-link>
            </div>
          </div>

          <!-- Popover 气泡卡片 -->
          <div id="demo-popover" class="demo-card placeholder">
            <div class="demo-header">
              <h3 class="demo-title">Popover - 气泡卡片</h3>
              <p class="demo-description">气泡卡片组件，点击显示</p>
            </div>
            <div class="placeholder-content">
              <p class="placeholder-text">Demo 开发中</p>
            </div>
          </div>

          <!-- Typing 打字效果 -->
          <div id="demo-typing" class="demo-card placeholder">
            <div class="demo-header">
              <h3 class="demo-title">Typing - 打字效果</h3>
              <p class="demo-description">打字效果组件，逐字显示文本</p>
            </div>
            <div class="placeholder-content">
              <p class="placeholder-text">Demo 开发中</p>
            </div>
          </div>
        </section>

        <!-- 布局组件 -->
        <section id="layout" class="component-section">
          <h2 class="section-title">布局组件</h2>

          <div id="demo-tabs" class="demo-card">
            <div class="demo-header">
              <h3 class="demo-title">Tabs - 标签页</h3>
              <p v-if="hasDoc('tabs')" class="demo-description">
                {{ getDocDescription("tabs") }}
              </p>
            </div>
            <div class="demo-content">
              <Tabs
                :tabs="[
                  { key: 'tab1', label: '标签一' },
                  { key: 'tab2', label: '标签二' },
                  { key: 'tab3', label: '标签三' },
                ]"
                v-model="activeTab"
              >
                <div v-if="activeTab === 'tab1'">标签一的内容</div>
                <div v-if="activeTab === 'tab2'">标签二的内容</div>
                <div v-if="activeTab === 'tab3'">标签三的内容</div>
              </Tabs>
            </div>
            <div v-if="hasDoc('tabs')" class="demo-footer">
              <router-link :to="`/docs/tabs`" class="view-docs-link"> 查看完整文档 → </router-link>
            </div>
          </div>

          <div id="demo-divider" class="demo-card">
            <h3 class="demo-title">Divider - 分割线</h3>
            <div class="demo-content">
              <Flex direction="column" gap="md">
                <div>内容上方</div>
                <Divider />
                <div>内容下方</div>
                <Divider>带文字的分割线</Divider>
                <div>更多内容</div>
              </Flex>
            </div>
          </div>

          <!-- Flex 弹性布局 -->
          <div id="demo-flex" class="demo-card placeholder">
            <div class="demo-header">
              <h3 class="demo-title">Flex - 弹性布局</h3>
              <p v-if="hasDoc('flex')" class="demo-description">
                {{ getDocDescription("flex") }}
              </p>
            </div>
            <div class="placeholder-content">
              <p class="placeholder-text">弹性布局组件，快速实现 Flexbox 布局</p>
              <router-link v-if="hasDoc('flex')" :to="`/docs/flex`" class="view-docs-button"> 查看完整文档和示例 → </router-link>
            </div>
          </div>

          <!-- Navbar 导航栏 -->
          <div id="demo-navbar" class="demo-card placeholder">
            <div class="demo-header">
              <h3 class="demo-title">Navbar - 导航栏</h3>
              <p v-if="hasDoc('navbar')" class="demo-description">
                {{ getDocDescription("navbar") }}
              </p>
            </div>
            <div class="placeholder-content">
              <p class="placeholder-text">导航栏组件，顶部导航</p>
              <router-link v-if="hasDoc('navbar')" :to="`/docs/navbar`" class="view-docs-button"> 查看完整文档和示例 → </router-link>
            </div>
          </div>

          <!-- Sidebar 侧边栏 -->
          <div id="demo-sidebar" class="demo-card placeholder">
            <div class="demo-header">
              <h3 class="demo-title">Sidebar - 侧边栏</h3>
              <p v-if="hasDoc('sidebar')" class="demo-description">
                {{ getDocDescription("sidebar") }}
              </p>
            </div>
            <div class="placeholder-content">
              <p class="placeholder-text">侧边栏组件，支持折叠</p>
              <router-link v-if="hasDoc('sidebar')" :to="`/docs/sidebar`" class="view-docs-button"> 查看完整文档和示例 → </router-link>
            </div>
          </div>

          <!-- ScrollView 滚动视图 -->
          <div id="demo-scroll-view" class="demo-card placeholder">
            <div class="demo-header">
              <h3 class="demo-title">ScrollView - 滚动视图</h3>
              <p class="demo-description">滚动视图组件，优化滚动性能</p>
            </div>
            <div class="placeholder-content">
              <p class="placeholder-text">Demo 开发中</p>
            </div>
          </div>

          <!-- Carousel 轮播图 -->
          <div id="demo-carousel" class="demo-card placeholder">
            <div class="demo-header">
              <h3 class="demo-title">Carousel - 轮播图</h3>
              <p class="demo-description">轮播图组件，支持自动播放</p>
            </div>
            <div class="placeholder-content">
              <p class="placeholder-text">Demo 开发中</p>
            </div>
          </div>
        </section>

        <!-- 数据展示组件 -->
        <section id="data" class="component-section">
          <h2 class="section-title">数据展示</h2>

          <!-- List 列表 -->
          <div id="demo-list" class="demo-card placeholder">
            <div class="demo-header">
              <h3 class="demo-title">List - 列表</h3>
              <p v-if="hasDoc('list')" class="demo-description">
                {{ getDocDescription("list") }}
              </p>
            </div>
            <div class="placeholder-content">
              <p class="placeholder-text">列表组件，支持虚拟滚动</p>
              <router-link v-if="hasDoc('list')" :to="`/docs/list`" class="view-docs-button"> 查看完整文档和示例 → </router-link>
            </div>
          </div>

          <!-- RichText 富文本 -->
          <div id="demo-rich-text" class="demo-card placeholder">
            <div class="demo-header">
              <h3 class="demo-title">RichText - 富文本</h3>
              <p v-if="hasDoc('rich-text')" class="demo-description">
                {{ getDocDescription("rich-text") }}
              </p>
            </div>
            <div class="placeholder-content">
              <p class="placeholder-text">富文本组件，支持 HTML 渲染</p>
              <router-link v-if="hasDoc('rich-text')" :to="`/docs/rich-text`" class="view-docs-button"> 查看完整文档和示例 → </router-link>
            </div>
          </div>
        </section>

        <!-- Agent 专属组件 -->
        <section id="agent" class="component-section">
          <h2 class="section-title">Agent 专属组件</h2>

          <!-- AgentCard Agent卡片 -->
          <div id="demo-agent-card" class="demo-card placeholder">
            <div class="demo-header">
              <h3 class="demo-title">AgentCard - Agent卡片</h3>
              <p v-if="hasDoc('agent-card')" class="demo-description">
                {{ getDocDescription("agent-card") }}
              </p>
            </div>
            <div class="placeholder-content">
              <p class="placeholder-text">Agent 卡片组件，显示 Agent 信息</p>
              <router-link v-if="hasDoc('agent-card')" :to="`/docs/agent-card`" class="view-docs-button"> 查看完整文档和示例 → </router-link>
            </div>
          </div>

          <!-- ThinkingBlock 思考块 -->
          <div id="demo-thinking-block" class="demo-card placeholder">
            <div class="demo-header">
              <h3 class="demo-title">ThinkingBlock - 思考块</h3>
              <p v-if="hasDoc('thinking-block')" class="demo-description">
                {{ getDocDescription("thinking-block") }}
              </p>
            </div>
            <div class="placeholder-content">
              <p class="placeholder-text">思考块组件，可视化 AI 推理过程</p>
              <router-link v-if="hasDoc('thinking-block')" :to="`/docs/thinking-block`" class="view-docs-button"> 查看完整文档和示例 → </router-link>
            </div>
          </div>

          <!-- WorkflowTimeline 工作流 -->
          <div id="demo-workflow-timeline" class="demo-card placeholder">
            <div class="demo-header">
              <h3 class="demo-title">WorkflowTimeline - 工作流时间线</h3>
              <p v-if="hasDoc('workflow-timeline')" class="demo-description">
                {{ getDocDescription("workflow-timeline") }}
              </p>
            </div>
            <div class="placeholder-content">
              <p class="placeholder-text">工作流时间线组件，显示执行步骤</p>
              <router-link v-if="hasDoc('workflow-timeline')" :to="`/docs/workflow-timeline`" class="view-docs-button"> 查看完整文档和示例 → </router-link>
            </div>
          </div>

          <!-- ProjectCard 项目卡片 -->
          <div id="demo-project-card" class="demo-card placeholder">
            <div class="demo-header">
              <h3 class="demo-title">ProjectCard - 项目卡片</h3>
              <p v-if="hasDoc('project-card')" class="demo-description">
                {{ getDocDescription("project-card") }}
              </p>
            </div>
            <div class="placeholder-content">
              <p class="placeholder-text">项目卡片组件，项目管理</p>
              <router-link v-if="hasDoc('project-card')" :to="`/docs/project-card`" class="view-docs-button"> 查看完整文档和示例 → </router-link>
            </div>
          </div>

          <!-- RoomCard 房间卡片 -->
          <div id="demo-room-card" class="demo-card placeholder">
            <div class="demo-header">
              <h3 class="demo-title">RoomCard - 房间卡片</h3>
              <p v-if="hasDoc('room-card')" class="demo-description">
                {{ getDocDescription("room-card") }}
              </p>
            </div>
            <div class="placeholder-content">
              <p class="placeholder-text">房间卡片组件，协作房间</p>
              <router-link v-if="hasDoc('room-card')" :to="`/docs/room-card`" class="view-docs-button"> 查看完整文档和示例 → </router-link>
            </div>
          </div>

          <!-- WorkflowCard 工作流卡片 -->
          <div id="demo-workflow-card" class="demo-card placeholder">
            <div class="demo-header">
              <h3 class="demo-title">WorkflowCard - 工作流卡片</h3>
              <p v-if="hasDoc('workflow-card')" class="demo-description">
                {{ getDocDescription("workflow-card") }}
              </p>
            </div>
            <div class="placeholder-content">
              <p class="placeholder-text">工作流卡片组件，工作流管理</p>
              <router-link v-if="hasDoc('workflow-card')" :to="`/docs/workflow-card`" class="view-docs-button"> 查看完整文档和示例 → </router-link>
            </div>
          </div>

          <!-- EditorPanel 编辑器 -->
          <div id="demo-editor-panel" class="demo-card placeholder">
            <div class="demo-header">
              <h3 class="demo-title">EditorPanel - 编辑器面板</h3>
              <p class="demo-description">代码编辑器面板组件</p>
            </div>
            <div class="placeholder-content">
              <p class="placeholder-text">Demo 开发中</p>
            </div>
          </div>
        </section>

        <!-- 内联文档显示区域 -->
        <section v-if="selectedComponent" class="doc-section">
          <div class="doc-container">
            <div class="doc-header">
              <div class="doc-title-area">
                <h2 class="doc-title">
                  {{ getComponent(selectedComponent)?.name || selectedComponent }}
                </h2>
                <button @click="selectedComponent = null" class="close-doc-btn" aria-label="关闭文档">✕</button>
              </div>
              <div class="doc-actions">
                <router-link v-if="hasDoc(selectedComponent)" :to="`/docs/${selectedComponent}`" class="external-doc-link" target="_blank"> 在新窗口打开文档 ↗ </router-link>
              </div>
            </div>

            <div class="doc-content">
              <div v-if="hasDoc(selectedComponent)" class="markdown-content">
                <div v-html="renderMarkdown(getFullDoc(selectedComponent))"></div>
              </div>
              <div v-else class="no-doc-content">
                <div class="no-doc-icon">📝</div>
                <h3>文档开发中</h3>
                <p>该组件的详细文档正在编写中，敬请期待。</p>
                <div class="no-doc-suggestions">
                  <p>您可以：</p>
                  <ul>
                    <li>查看组件的基础演示</li>
                    <li>在组件源代码中了解 Props 接口</li>
                    <li>参考其他类似组件的文档</li>
                  </ul>
                </div>
              </div>
            </div>
          </div>
        </section>
      </ScrollView>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from "vue";
import { Navbar, Sidebar, ScrollView, Tabs, Divider, Flex, List, Button, Avatar, Tag, Input, Search, Checkbox, Radio, Bubble, ThinkBubble, Card, SystemMessage, Notice, Progress, Tooltip } from "@/components/ChatUI";

// 批量导入所有 Markdown 文档
const docModules = import.meta.glob("@/src/docs/components/*.md", {
  query: "?raw",
  import: "default",
  eager: true,
});

// 将文档转换为 key-value 映射
const docs: Record<string, string> = {};
Object.keys(docModules).forEach((path) => {
  // 从路径中提取文件名（不含扩展名）
  const match = path.match(/\/([^/]+)\.md$/);
  if (match && match[1]) {
    const filename = match[1];
    // 转换为 kebab-case (例如: Button -> button, AgentCard -> agent-card)
    const key = filename
      .replace(/([A-Z])/g, "-$1")
      .toLowerCase()
      .replace(/^-/, "");
    docs[key] = docModules[path] as string;
  }
});

const sections = [
  { id: "chat", name: "对话组件" },
  { id: "basic", name: "基础组件" },
  { id: "form", name: "表单组件" },
  { id: "feedback", name: "反馈组件" },
  { id: "layout", name: "布局组件" },
  { id: "data", name: "数据展示" },
  { id: "agent", name: "Agent 组件" },
];

const allComponents = [
  // Chat 对话组件
  { key: "chat", name: "Chat 聊天容器", category: "chat" },
  { key: "bubble", name: "Bubble 消息气泡", category: "chat" },
  { key: "think-bubble", name: "ThinkBubble 思考气泡", category: "chat" },
  { key: "typing-bubble", name: "TypingBubble 输入中", category: "chat" },
  { key: "card", name: "Card 卡片", category: "chat" },
  { key: "file-card", name: "FileCard 文件卡片", category: "chat" },
  { key: "message-status", name: "MessageStatus 消息状态", category: "chat" },
  { key: "system-message", name: "SystemMessage 系统消息", category: "chat" },

  // Basic 基础组件
  { key: "button", name: "Button 按钮", category: "basic" },
  { key: "icon", name: "Icon 图标", category: "basic" },
  { key: "avatar", name: "Avatar 头像", category: "basic" },
  { key: "tag", name: "Tag 标签", category: "basic" },
  { key: "image", name: "Image 图片", category: "basic" },
  { key: "divider", name: "Divider 分割线", category: "basic" },

  // Form 表单组件
  { key: "input", name: "Input 输入框", category: "form" },
  { key: "search", name: "Search 搜索框", category: "form" },
  { key: "checkbox", name: "Checkbox 复选框", category: "form" },
  { key: "radio", name: "Radio 单选框", category: "form" },
  { key: "dropdown", name: "Dropdown 下拉菜单", category: "form" },
  { key: "multimodal-input", name: "MultimodalInput 多模态输入", category: "form" },

  // Feedback 反馈组件
  { key: "modal", name: "Modal 对话框", category: "feedback" },
  { key: "notice", name: "Notice 通知", category: "feedback" },
  { key: "tooltip", name: "Tooltip 提示", category: "feedback" },
  { key: "popover", name: "Popover 气泡卡片", category: "feedback" },
  { key: "progress", name: "Progress 进度条", category: "feedback" },
  { key: "typing", name: "Typing 打字效果", category: "feedback" },

  // Layout 布局组件
  { key: "flex", name: "Flex 弹性布局", category: "layout" },
  { key: "navbar", name: "Navbar 导航栏", category: "layout" },
  { key: "sidebar", name: "Sidebar 侧边栏", category: "layout" },
  { key: "scroll-view", name: "ScrollView 滚动视图", category: "layout" },
  { key: "tabs", name: "Tabs 标签页", category: "layout" },
  { key: "carousel", name: "Carousel 轮播图", category: "layout" },

  // Data 数据展示
  { key: "list", name: "List 列表", category: "data" },
  { key: "rich-text", name: "RichText 富文本", category: "data" },

  // Agent 专属组件
  { key: "agent-card", name: "AgentCard Agent卡片", category: "agent" },
  { key: "thinking-block", name: "ThinkingBlock 思考块", category: "agent" },
  { key: "workflow-timeline", name: "WorkflowTimeline 工作流", category: "agent" },
  { key: "editor-panel", name: "EditorPanel 编辑器", category: "agent" },
  { key: "project-card", name: "ProjectCard 项目卡片", category: "agent" },
  { key: "room-card", name: "RoomCard 房间卡片", category: "agent" },
  { key: "workflow-card", name: "WorkflowCard 工作流卡片", category: "agent" },
];

const inputValue = ref("");
const searchValue = ref("");
const checked = ref(false);
const radioValue = ref("a");
const activeTab = ref("tab1");

const scrollToComponent = (item: any) => {
  const key = item.key;
  const element = document.getElementById(`demo-${key}`);

  if (element) {
    // 滚动到目标元素，考虑顶部导航栏的高度
    const navbarHeight = 64; // Navbar 高度
    const elementPosition = element.getBoundingClientRect().top;
    const offsetPosition = elementPosition + window.pageYOffset - navbarHeight - 20;

    window.scrollTo({
      top: offsetPosition,
      behavior: "smooth",
    });

    // 添加高亮效果
    element.classList.add("highlight");
    setTimeout(() => {
      element.classList.remove("highlight");
    }, 2000);
  } else {
    // 如果没有对应的 demo，滚动到对应的分类区域
    const categoryElement = document.getElementById(item.category);
    if (categoryElement) {
      categoryElement.scrollIntoView({ behavior: "smooth", block: "start" });
    }
  }
};

const handleSearch = (value: string) => {
  console.log("Search:", value);
};

// 从 Markdown 文档中提取描述（第一段文字）
const getDocDescription = (key: string): string => {
  const doc = docs[key];
  if (!doc) return "";

  // 提取第一个标题后的第一段文字
  const lines = doc.split("\n");
  for (let i = 0; i < lines.length; i++) {
    const line = lines[i]?.trim() ?? "";
    // 跳过标题行
    if (line.startsWith("#")) continue;
    // 跳过空行
    if (line === "") continue;
    // 返回第一个非空非标题行
    return line;
  }
  return "";
};

// 检查组件是否有文档
const hasDoc = (key: string): boolean => {
  return !!docs[key];
};

// 获取完整的文档内容
const getFullDoc = (key: string): string => {
  return docs[key] || "";
};

// 当前选中的组件（用于显示文档）
const selectedComponent = ref<string | null>(null);

// 显示组件文档
const showComponentDoc = (key: string) => {
  selectedComponent.value = selectedComponent.value === key ? null : key;
};

// 获取组件对象
const getComponent = (key: string) => {
  return allComponents.find((comp) => comp.key === key) || null;
};

// 简单的 Markdown 渲染函数
const renderMarkdown = (markdown: string): string => {
  if (!markdown) return "";

  return (
    markdown
      // 处理标题
      .replace(/^### (.+)$/gm, '<h3 class="text-xl font-semibold text-gray-900 dark:text-white mt-6 mb-3">$1</h3>')
      .replace(/^## (.+)$/gm, '<h2 class="text-2xl font-bold text-gray-900 dark:text-white mt-8 mb-4">$1</h2>')
      .replace(/^# (.+)$/gm, '<h1 class="text-3xl font-bold text-gray-900 dark:text-white mt-8 mb-6">$1</h1>')
      // 处理粗体和斜体
      .replace(/\*\*(.+?)\*\*/g, "<strong>$1</strong>")
      .replace(/\*(.+?)\*/g, "<em>$1</em>")
      // 处理行内代码
      .replace(/`([^`]+)`/g, '<code class="bg-gray-100 dark:bg-gray-800 px-2 py-1 rounded text-sm font-mono text-gray-800 dark:text-gray-200">$1</code>')
      // 处理代码块
      .replace(/```(\w+)?\n([\s\S]+?)```/g, '<pre class="bg-gray-900 dark:bg-gray-950 text-gray-100 p-4 rounded-lg overflow-x-auto my-4"><code>$2</code></pre>')
      // 处理段落
      .split("\n\n")
      .map((paragraph) => {
        const trimmed = paragraph.trim();
        if (trimmed.startsWith("<h") || trimmed.startsWith("<pre") || trimmed.startsWith("<code>")) {
          return trimmed;
        }
        if (trimmed.startsWith("- ")) {
          // 处理列表
          const items = trimmed
            .split("\n")
            .map((item) => item.replace(/^- (.+)$/, '<li class="ml-4">$1</li>'))
            .join("");
          return `<ul class="list-disc space-y-1 my-3">${items}</ul>`;
        }
        if (trimmed) {
          return `<p class="text-gray-700 dark:text-gray-300 my-3 leading-relaxed">${trimmed}</p>`;
        }
        return "";
      })
      .filter(Boolean)
      .join("\n")
  );
};
</script>

<style scoped>
.chatui-components-page {
  @apply min-h-screen bg-gray-50 dark:bg-gray-900;
}

.page-container {
  @apply flex h-[calc(100vh-64px)];
}

.main-content {
  @apply flex-1 p-8;
}

.hero-section {
  @apply text-center py-16 space-y-6;
}

.hero-title {
  @apply text-5xl font-bold text-gray-900 dark:text-white;
}

.hero-subtitle {
  @apply text-xl text-gray-600 dark:text-gray-400 max-w-2xl mx-auto;
}

.component-section {
  @apply py-12 space-y-8;
}

.section-title {
  @apply text-3xl font-bold text-gray-900 dark:text-white mb-8;
}

.demo-card {
  @apply bg-white dark:bg-gray-800 rounded-xl p-6 shadow-sm border border-gray-200 dark:border-gray-700 transition-all;
}

.demo-header {
  @apply mb-4;
}

.demo-title {
  @apply text-lg font-semibold text-gray-900 dark:text-white mb-2;
}

.demo-description {
  @apply text-sm text-gray-600 dark:text-gray-400;
}

.demo-content {
  @apply p-4 bg-gray-50 dark:bg-gray-900 rounded-lg mb-4;
}

.demo-footer {
  @apply pt-4 border-t border-gray-200 dark:border-gray-700;
}

.view-docs-link {
  @apply text-sm text-blue-600 dark:text-blue-400 hover:underline font-medium;
}

/* 占位卡片样式 */
.demo-card.placeholder {
  @apply bg-gradient-to-br from-gray-50 to-gray-100 dark:from-gray-800 dark:to-gray-900 border-dashed;
}

.placeholder-content {
  @apply p-6 text-center space-y-4;
}

.placeholder-text {
  @apply text-gray-600 dark:text-gray-400;
}

.view-docs-button {
  @apply inline-block px-4 py-2 bg-blue-600 dark:bg-blue-500 text-white rounded-lg hover:bg-blue-700 dark:hover:bg-blue-600 transition-colors font-medium text-sm;
}

.nav-link {
  @apply text-sm font-medium text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white transition-colors;
}

.component-item {
  @apply flex items-center justify-between;
}

/* 滚动高亮效果 */
.demo-card.highlight {
  @apply ring-2 ring-blue-500 ring-offset-2 dark:ring-offset-gray-900;
  animation: highlight-pulse 2s ease-in-out;
}

@keyframes highlight-pulse {
  0%,
  100% {
    @apply ring-opacity-0;
  }
  50% {
    @apply ring-opacity-100;
  }
}

/* 文档显示区域样式 */
.doc-section {
  @apply mt-12 mb-8;
}

.doc-container {
  @apply bg-white dark:bg-gray-800 rounded-xl border border-gray-200 dark:border-gray-700 shadow-sm;
}

.doc-header {
  @apply flex items-center justify-between p-6 border-b border-gray-200 dark:border-gray-700;
}

.doc-title-area {
  @apply flex items-center gap-4;
}

.doc-title {
  @apply text-2xl font-bold text-gray-900 dark:text-white m-0;
}

.close-doc-btn {
  @apply w-8 h-8 flex items-center justify-center text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors;
}

.doc-actions {
  @apply flex items-center gap-3;
}

.external-doc-link {
  @apply text-blue-600 dark:text-blue-400 hover:text-blue-700 dark:hover:text-blue-300 text-sm font-medium transition-colors;
}

.doc-content {
  @apply p-6;
}

.markdown-content {
  @apply max-w-none;
}

.markdown-content h1 {
  @apply text-3xl font-bold text-gray-900 dark:text-white mt-8 mb-6;
}

.markdown-content h2 {
  @apply text-2xl font-bold text-gray-900 dark:text-white mt-8 mb-4;
}

.markdown-content h3 {
  @apply text-xl font-semibold text-gray-900 dark:text-white mt-6 mb-3;
}

.markdown-content p {
  @apply text-gray-700 dark:text-gray-300 my-3 leading-relaxed;
}

.markdown-content code {
  @apply bg-gray-100 dark:bg-gray-800 px-2 py-1 rounded text-sm font-mono text-gray-800 dark:text-gray-200;
}

.markdown-content pre {
  @apply bg-gray-900 dark:bg-gray-950 text-gray-100 p-4 rounded-lg overflow-x-auto my-4;
}

.markdown-content pre code {
  @apply bg-transparent px-0 py-0 text-gray-100;
}

.markdown-content ul {
  @apply list-disc space-y-1 my-3 ml-4;
}

/* 无文档内容样式 */
.no-doc-content {
  @apply text-center py-12 space-y-4;
}

.no-doc-icon {
  @apply text-4xl mb-4;
}

.no-doc-content h3 {
  @apply text-xl font-semibold text-gray-900 dark:text-white mb-2;
}

.no-doc-content p {
  @apply text-gray-600 dark:text-gray-400 max-w-md mx-auto;
}

.no-doc-suggestions {
  @apply mt-8 text-left max-w-md mx-auto;
}

.no-doc-suggestions p {
  @apply font-medium text-gray-900 dark:text-white mb-2;
}

.no-doc-suggestions ul {
  @apply list-disc list-inside space-y-1 text-gray-600 dark:text-gray-400;
}

/* 组件卡片可点击样式 */
.demo-card {
  @apply cursor-pointer transition-all duration-200 hover:shadow-md hover:shadow-gray-200/50 dark:hover:shadow-gray-900/50;
}

.demo-card:hover {
  @apply transform -translate-y-0.5;
}

.demo-content {
  @apply cursor-auto;
}

.view-docs-link {
  @apply text-blue-600 dark:text-blue-400 hover:text-blue-700 dark:hover:text-blue-300 text-sm font-medium transition-colors bg-transparent border-none p-0 cursor-pointer;
}

.view-docs-link:hover {
  @apply underline;
}
</style>

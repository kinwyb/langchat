# LangChat UI

LangChat 的前端界面，基于 Vue 3 和 Vite 构建，提供类似 DeepSeek 的聊天体验。

## 功能特性

- 现代化的聊天界面，参考 DeepSeek 设计风格
- 实时流式响应（SSE）
- Markdown 渲染支持
- 代码语法高亮（支持 JavaScript、Python、Go、Bash 等）
- 历史对话管理
- 技能开关控制
- 响应式设计

## 安装依赖

```bash
npm install
```

## 开发模式

```bash
npm run dev
```

前端将在 http://localhost:3000 启动，并自动代理 API 请求到后端服务器（默认 http://localhost:8080）。

## 生产构建

```bash
npm run build
```

构建产物将输出到 `dist` 目录。

## 预览生产构建

```bash
npm run preview
```

## 配置

### 环境变量

创建 `.env` 文件来自定义配置：

```env
VITE_API_BASE_URL=/api
```

### API 代理

开发模式下，Vite 配置了代理将 `/api` 请求转发到后端服务器。如果需要修改后端地址，请编辑 `vite.config.js`。

## 技术栈

- **Vue 3** - 渐进式 JavaScript 框架
- **Vite** - 下一代前端构建工具
- **Marked** - Markdown 解析器
- **Highlight.js** - 代码语法高亮
- **Axios** - HTTP 客户端

## 项目结构

```
ui/
├── src/
│   ├── components/      # Vue 组件
│   │   ├── Sidebar.vue    # 侧边栏组件
│   │   ├── Message.vue    # 消息组件
│   │   └── ChatInput.vue  # 输入框组件
│   ├── App.vue          # 主应用组件
│   ├── main.js          # 应用入口
│   ├── api.js           # API 封装
│   └── style.css        # 全局样式
├── index.html           # HTML 模板
├── vite.config.js       # Vite 配置
└── package.json         # 项目配置
```

## 使用说明

1. 启动后端 API 服务器：
   ```bash
   cd ..
   go run examples/server/main.go
   ```

2. 启动前端开发服务器：
   ```bash
   cd ui
   npm run dev
   ```

3. 在浏览器中打开 http://localhost:3000

## 功能说明

### 侧边栏
- 显示历史对话列表
- 创建新对话
- 删除对话
- 选择对话切换

### 聊天区域
- 消息展示（用户和助手）
- 流式响应展示
- 代码块高亮
- 复制消息内容

### 输入框
- 自动调整高度
- 发送快捷键（Enter）
- 多行输入（Shift+Enter）
- 技能开关

## 开发注意事项

- 所有 API 请求通过 `api.js` 封装
- 组件使用 Composition API
- 使用 CSS 变量管理主题色
- 支持深色模式扩展
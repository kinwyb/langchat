<template>
  <div :class="['message', message.role]">
    <div class="message-container">
      <!-- 用户消息 -->
      <template v-if="message.role === 'user'">
        <div class="user-message-wrapper">
          <div class="user-content">
            {{ message.content }}
          </div>
          <!-- 用户头像 -->
          <div class="user-avatar">
            <svg width="28" height="28" viewBox="0 0 28 28" fill="none" xmlns="http://www.w3.org/2000/svg">
              <circle cx="14" cy="11" r="5" fill="white"/>
              <path d="M4 24C4 20.686 6.686 18 10 18H18C21.314 18 24 20.686 24 24" stroke="white" stroke-width="2.5" stroke-linecap="round"/>
            </svg>
          </div>
        </div>
      </template>

      <!-- AI消息 -->
      <template v-else>
        <div class="assistant-message-wrapper">
          <!-- AI头像 -->
          <div class="assistant-avatar">
            <svg width="28" height="28" viewBox="0 0 28 28" fill="none" xmlns="http://www.w3.org/2000/svg">
              <rect x="2" y="2" width="24" height="24" rx="8" fill="#4d6bfe"/>
              <path d="M8 14L12 18L20 10" stroke="white" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"/>
            </svg>
          </div>

          <div class="assistant-content">
            <!-- 思考过程（深度思考） -->
            <div v-if="message.reasoning_content" class="reasoning-section">
              <div class="reasoning-header" @click="toggleReasoning">
                <svg 
                  class="reasoning-arrow" 
                  :class="{ 'is-expanded': isReasoningExpanded }"
                  width="16" 
                  height="16" 
                  viewBox="0 0 16 16" 
                  fill="none"
                >
                  <path d="M6 12L10 8L6 4" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
                </svg>
                <span class="reasoning-title">深度思考</span>
                <span v-if="isStreamingReasoning" class="reasoning-status">思考中...</span>
              </div>
              <div v-show="isReasoningExpanded" class="reasoning-content">
                <div class="reasoning-text" v-html="renderedReasoning"></div>
              </div>
            </div>

            <!-- 主要内容 -->
            <div class="message-body">
              <!-- 加载状态 -->
              <div v-if="isLoading && !message.content" class="typing-indicator">
                <span></span>
                <span></span>
                <span></span>
              </div>

              <!-- 流式输出光标效果 -->
              <div v-else class="markdown-body" v-html="renderedContent"></div>
              
              <!-- 流式光标 -->
              <span v-if="isStreaming" class="stream-cursor"></span>
            </div>

            <!-- 错误信息 -->
            <div v-if="error" class="message-error">
              <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
                <circle cx="8" cy="8" r="6" stroke="#ef4444" stroke-width="1.5"/>
                <path d="M8 5V8M8 11V11.01" stroke="#ef4444" stroke-width="1.5" stroke-linecap="round"/>
              </svg>
              <span>{{ error }}</span>
            </div>

            <!-- 操作按钮 -->
            <div v-if="!isLoading && message.content" class="message-actions">
              <button class="action-btn" @click="copyContent" title="复制">
                <svg width="14" height="14" viewBox="0 0 14 14" fill="none">
                  <rect x="2" y="2" width="8" height="10" rx="1.5" stroke="currentColor" stroke-width="1.2"/>
                  <path d="M5 2V1.5C5 0.671573 5.67157 0 6.5 0H11C11.8284 0 12.5 0.671573 12.5 1.5V9C12.5 9.82843 11.8284 10.5 11 10.5H10" stroke="currentColor" stroke-width="1.2"/>
                </svg>
                <span>{{ copyText }}</span>
              </button>
              <button class="action-btn" @click="retryMessage" title="重新生成" v-if="error">
                <svg width="14" height="14" viewBox="0 0 14 14" fill="none">
                  <path d="M1 7C1 3.686 3.686 1 7 1C9.5 1 11.5 2.5 12.5 4.5M13 7C13 10.314 10.314 13 7 13C4.5 13 2.5 11.5 1.5 9.5" stroke="currentColor" stroke-width="1.2" stroke-linecap="round"/>
                  <path d="M12.5 2V4.5H10" stroke="currentColor" stroke-width="1.2" stroke-linecap="round" stroke-linejoin="round"/>
                  <path d="M1.5 12V9.5H4" stroke="currentColor" stroke-width="1.2" stroke-linecap="round" stroke-linejoin="round"/>
                </svg>
                <span>重试</span>
              </button>
            </div>
          </div>
        </div>
      </template>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { marked } from 'marked'
import { markedHighlight } from 'marked-highlight'
import hljs from 'highlight.js/lib/core'
import javascript from 'highlight.js/lib/languages/javascript'
import typescript from 'highlight.js/lib/languages/typescript'
import python from 'highlight.js/lib/languages/python'
import go from 'highlight.js/lib/languages/go'
import bash from 'highlight.js/lib/languages/bash'
import shell from 'highlight.js/lib/languages/shell'
import xml from 'highlight.js/lib/languages/xml'
import css from 'highlight.js/lib/languages/css'
import json from 'highlight.js/lib/languages/json'
import yaml from 'highlight.js/lib/languages/yaml'
import sql from 'highlight.js/lib/languages/sql'
import rust from 'highlight.js/lib/languages/rust'
import java from 'highlight.js/lib/languages/java'
import cpp from 'highlight.js/lib/languages/cpp'
import c from 'highlight.js/lib/languages/c'
import 'highlight.js/styles/atom-one-dark.css'

// 注册语言
hljs.registerLanguage('javascript', javascript)
hljs.registerLanguage('typescript', typescript)
hljs.registerLanguage('python', python)
hljs.registerLanguage('go', go)
hljs.registerLanguage('bash', bash)
hljs.registerLanguage('shell', shell)
hljs.registerLanguage('xml', xml)
hljs.registerLanguage('html', xml)
hljs.registerLanguage('css', css)
hljs.registerLanguage('json', json)
hljs.registerLanguage('yaml', yaml)
hljs.registerLanguage('sql', sql)
hljs.registerLanguage('rust', rust)
hljs.registerLanguage('java', java)
hljs.registerLanguage('cpp', cpp)
hljs.registerLanguage('c', c)

// 自定义renderer
const renderer = new marked.Renderer()

// 自定义代码块渲染
renderer.code = (code, language) => {
  const validLang = language && hljs.getLanguage(language) ? language : 'plaintext'
  const highlighted = validLang !== 'plaintext' 
    ? hljs.highlight(code, { language: validLang }).value 
    : code
  
  return `
    <div class="code-block">
      <div class="code-header">
        <span class="code-lang">${validLang}</span>
        <button class="code-copy-btn" onclick="copyCode(this)" data-code="${encodeURIComponent(code)}">
          <svg width="12" height="12" viewBox="0 0 14 14" fill="none">
            <rect x="2" y="2" width="8" height="10" rx="1.5" stroke="currentColor" stroke-width="1.2"/>
            <path d="M5 2V1.5C5 0.671573 5.67157 0 6.5 0H11C11.8284 0 12.5 0.671573 12.5 1.5V9C12.5 9.82843 11.8284 10.5 11 10.5H10" stroke="currentColor" stroke-width="1.2"/>
          </svg>
          <span>复制</span>
        </button>
      </div>
      <pre><code class="hljs language-${validLang}">${highlighted}</code></pre>
    </div>
  `
}

// 配置marked
marked.use(
  markedHighlight({
    langPrefix: 'hljs language-',
    highlight(code, lang) {
      const language = hljs.getLanguage(lang) ? lang : 'plaintext'
      return hljs.highlight(code, { language }).value
    }
  })
)

marked.setOptions({
  renderer,
  breaks: true,
  gfm: true,
  headerIds: false,
  mangle: false
})

const props = defineProps({
  message: {
    type: Object,
    required: true
  },
  isLoading: {
    type: Boolean,
    default: false
  },
  error: {
    type: String,
    default: ''
  },
  isStreaming: {
    type: Boolean,
    default: false
  }
})

const emit = defineEmits(['retry'])

const isReasoningExpanded = ref(true)
const copyText = ref('复制')

const isStreamingReasoning = computed(() => {
  return props.isStreaming && props.message.reasoning_content && 
         !props.message.content
})

const renderedContent = computed(() => {
  if (!props.message.content) return ''
  try {
    return marked.parse(props.message.content)
  } catch (e) {
    console.error('Markdown parse error:', e)
    return props.message.content
  }
})

const renderedReasoning = computed(() => {
  if (!props.message.reasoning_content) return ''
  // 思考内容简单渲染，支持换行
  return props.message.reasoning_content
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/\n/g, '<br>')
})

const toggleReasoning = () => {
  isReasoningExpanded.value = !isReasoningExpanded.value
}

const copyContent = async () => {
  try {
    await navigator.clipboard.writeText(props.message.content)
    copyText.value = '已复制'
    setTimeout(() => {
      copyText.value = '复制'
    }, 2000)
  } catch (err) {
    console.error('Copy failed:', err)
  }
}

const retryMessage = () => {
  emit('retry')
}

// 复制代码功能
if (typeof window !== 'undefined') {
  window.copyCode = async (btn) => {
    const code = decodeURIComponent(btn.getAttribute('data-code'))
    try {
      await navigator.clipboard.writeText(code)
      const originalText = btn.innerHTML
      btn.innerHTML = `
        <svg width="12" height="12" viewBox="0 0 14 14" fill="none">
          <path d="M2 7L5.5 10.5L12 4" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
        </svg>
        <span>已复制</span>
      `
      btn.classList.add('copied')
      setTimeout(() => {
        btn.innerHTML = originalText
        btn.classList.remove('copied')
      }, 2000)
    } catch (err) {
      console.error('Copy code failed:', err)
    }
  }
}

// 自动展开思考内容（新消息）
watch(() => props.message.reasoning_content, (newVal, oldVal) => {
  if (newVal && !oldVal) {
    isReasoningExpanded.value = true
  }
}, { immediate: true })
</script>

<style scoped>
.message {
  padding: 16px 0;
}

.message-container {
  max-width: 800px;
  margin: 0 auto;
  padding: 0 16px;
}

/* 用户消息样式 */
.user-message-wrapper {
  display: flex;
  justify-content: flex-end;
  align-items: flex-start;
  gap: 12px;
}

.user-content {
  background-color: #4d6bfe;
  color: white;
  padding: 12px 16px;
  border-radius: 16px;
  border-bottom-right-radius: 4px;
  max-width: 85%;
  font-size: 15px;
  line-height: 1.6;
  word-wrap: break-word;
}

.user-avatar {
  flex-shrink: 0;
  width: 32px;
  height: 32px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  background-color: #10b981;
  color: white;
}

/* AI消息样式 */
.assistant-message-wrapper {
  display: flex;
  gap: 12px;
}

.assistant-avatar {
  flex-shrink: 0;
  width: 32px;
  height: 32px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  background-color: #f0f2ff;
}

.assistant-content {
  flex: 1;
  min-width: 0;
  background-color: #f5f7fa;
  border-radius: 16px;
  border-top-left-radius: 4px;
  padding: 16px 20px;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.05);
}

/* 思考过程样式 */
.reasoning-section {
  margin-bottom: 12px;
  background-color: #f8f9fa;
  border-radius: 8px;
  overflow: hidden;
}

.reasoning-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 14px;
  cursor: pointer;
  user-select: none;
  transition: background-color 0.2s;
}

.reasoning-header:hover {
  background-color: #f0f1f2;
}

.reasoning-arrow {
  color: #6b7280;
  transition: transform 0.2s;
}

.reasoning-arrow.is-expanded {
  transform: rotate(90deg);
}

.reasoning-title {
  font-size: 13px;
  font-weight: 500;
  color: #374151;
}

.reasoning-status {
  font-size: 12px;
  color: #4d6bfe;
  margin-left: auto;
  animation: pulse 2s infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}

.reasoning-content {
  padding: 0 14px 14px;
}

.reasoning-text {
  font-size: 13px;
  line-height: 1.7;
  color: #6b7280;
  font-style: italic;
}

/* 消息主体 */
.message-body {
  font-size: 15px;
  line-height: 1.75;
  color: #1f2937;
}

/* 打字指示器 */
.typing-indicator {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 8px 0;
}

.typing-indicator span {
  width: 8px;
  height: 8px;
  background-color: #9ca3af;
  border-radius: 50%;
  animation: typing 1.4s infinite ease-in-out both;
}

.typing-indicator span:nth-child(1) {
  animation-delay: -0.32s;
}

.typing-indicator span:nth-child(2) {
  animation-delay: -0.16s;
}

@keyframes typing {
  0%, 80%, 100% {
    transform: scale(0);
    opacity: 0.5;
  }
  40% {
    transform: scale(1);
    opacity: 1;
  }
}

/* 流式光标 */
.stream-cursor {
  display: inline-block;
  width: 8px;
  height: 18px;
  background-color: #4d6bfe;
  margin-left: 2px;
  vertical-align: middle;
  animation: blink 1s infinite;
  border-radius: 1px;
}

@keyframes blink {
  0%, 50% {
    opacity: 1;
  }
  51%, 100% {
    opacity: 0;
  }
}

/* 错误信息 */
.message-error {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 12px;
  padding: 10px 14px;
  background-color: #fef2f2;
  border-radius: 8px;
  font-size: 13px;
  color: #dc2626;
}

/* 操作按钮 */
.message-actions {
  display: flex;
  gap: 8px;
  margin-top: 12px;
  opacity: 0;
  transition: opacity 0.2s;
}

.message:hover .message-actions {
  opacity: 1;
}

.action-btn {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 6px 10px;
  font-size: 12px;
  color: #6b7280;
  background-color: #f3f4f6;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.2s;
}

.action-btn:hover {
  background-color: #e5e7eb;
  color: #374151;
}

/* Markdown内容样式 */
:deep(.markdown-body) {
  word-wrap: break-word;
}

:deep(.markdown-body p) {
  margin: 0.8em 0;
}

:deep(.markdown-body p:first-child) {
  margin-top: 0;
}

:deep(.markdown-body p:last-child) {
  margin-bottom: 0;
}

:deep(.markdown-body h1),
:deep(.markdown-body h2),
:deep(.markdown-body h3),
:deep(.markdown-body h4),
:deep(.markdown-body h5),
:deep(.markdown-body h6) {
  margin: 1.5em 0 0.8em;
  font-weight: 600;
  line-height: 1.3;
  color: #111827;
}

:deep(.markdown-body h1) {
  font-size: 1.6em;
  border-bottom: 1px solid #e5e7eb;
  padding-bottom: 0.3em;
}

:deep(.markdown-body h2) {
  font-size: 1.4em;
  border-bottom: 1px solid #e5e7eb;
  padding-bottom: 0.3em;
}

:deep(.markdown-body h3) {
  font-size: 1.2em;
}

:deep(.markdown-body h4) {
  font-size: 1.1em;
}

:deep(.markdown-body strong) {
  font-weight: 600;
  color: #111827;
}

:deep(.markdown-body ul),
:deep(.markdown-body ol) {
  margin: 0.8em 0;
  padding-left: 1.8em;
}

:deep(.markdown-body li) {
  margin: 0.4em 0;
}

:deep(.markdown-body li > ul),
:deep(.markdown-body li > ol) {
  margin: 0.4em 0;
}

:deep(.markdown-body blockquote) {
  margin: 1em 0;
  padding: 0.8em 1em;
  border-left: 4px solid #4d6bfe;
  background-color: #f8f9fa;
  color: #4b5563;
  border-radius: 0 8px 8px 0;
}

:deep(.markdown-body blockquote p) {
  margin: 0.4em 0;
}

:deep(.markdown-body blockquote p:first-child) {
  margin-top: 0;
}

:deep(.markdown-body blockquote p:last-child) {
  margin-bottom: 0;
}

:deep(.markdown-body hr) {
  margin: 1.5em 0;
  border: none;
  border-top: 1px solid #e5e7eb;
}

:deep(.markdown-body a) {
  color: #4d6bfe;
  text-decoration: none;
}

:deep(.markdown-body a:hover) {
  text-decoration: underline;
}

:deep(.markdown-body img) {
  max-width: 100%;
  border-radius: 8px;
  margin: 0.8em 0;
}

:deep(.markdown-body table) {
  width: 100%;
  margin: 1em 0;
  border-collapse: collapse;
  font-size: 14px;
}

:deep(.markdown-body th),
:deep(.markdown-body td) {
  padding: 10px 14px;
  border: 1px solid #e5e7eb;
  text-align: left;
}

:deep(.markdown-body th) {
  background-color: #f9fafb;
  font-weight: 600;
  color: #374151;
}

:deep(.markdown-body tr:nth-child(even)) {
  background-color: #f9fafb;
}

:deep(.markdown-body code) {
  font-family: 'JetBrains Mono', 'Fira Code', 'SF Mono', Monaco, monospace;
  font-size: 0.9em;
  background-color: #f3f4f6;
  padding: 0.2em 0.4em;
  border-radius: 4px;
  color: #dc2626;
}

/* 代码块样式 */
:deep(.code-block) {
  margin: 1em 0;
  border-radius: 10px;
  overflow: hidden;
  background-color: #282c34;
}

:deep(.code-header) {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 14px;
  background-color: #21252b;
  border-bottom: 1px solid #3e4451;
}

:deep(.code-lang) {
  font-size: 12px;
  color: #abb2bf;
  text-transform: uppercase;
  font-weight: 500;
}

:deep(.code-copy-btn) {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 4px 10px;
  font-size: 12px;
  color: #abb2bf;
  background-color: transparent;
  border: 1px solid #3e4451;
  border-radius: 4px;
  cursor: pointer;
  transition: all 0.2s;
}

:deep(.code-copy-btn:hover) {
  background-color: #3e4451;
  color: #fff;
}

:deep(.code-copy-btn.copied) {
  background-color: #28a745;
  border-color: #28a745;
  color: #fff;
}

:deep(.code-block pre) {
  margin: 0;
  padding: 16px;
  overflow-x: auto;
  background-color: #282c34;
}

:deep(.code-block code) {
  font-family: 'JetBrains Mono', 'Fira Code', 'SF Mono', Monaco, monospace;
  font-size: 13px;
  line-height: 1.6;
  background-color: transparent;
  padding: 0;
  color: #abb2bf;
}

/* 复选框样式 */
:deep(.markdown-body input[type="checkbox"]) {
  margin-right: 0.5em;
}

/* 响应式 */
@media (max-width: 640px) {
  .message-container {
    padding: 0 12px;
  }
  
  .user-content {
    max-width: 90%;
    font-size: 14px;
  }
  
  .message-body {
    font-size: 14px;
  }
  
  .assistant-avatar {
    width: 28px;
    height: 28px;
  }
  
  .user-avatar {
    width: 28px;
    height: 28px;
  }
}
</style>
<template>
  <div class="app">
    <Sidebar
      :chat-history="chatHistory"
      :current-chat-id="currentChatId"
      @new-chat="newChat"
      @select-chat="selectChat"
      @delete-chat="deleteChat"
    />

    <div class="main-content">
      <div class="chat-container" ref="chatContainer">
        <div v-if="messages.length === 0" class="empty-state">
          <div class="empty-state-icon">
            <svg width="64" height="64" viewBox="0 0 64 64" fill="none" xmlns="http://www.w3.org/2000/svg">
              <rect x="8" y="8" width="48" height="48" rx="16" fill="#4d6bfe" fill-opacity="0.1"/>
              <path d="M16 32C16 23.1634 23.1634 16 32 16C40.8366 16 48 23.1634 48 32C48 40.8366 40.8366 48 32 48C23.1634 48 16 40.8366 16 32Z" fill="#4d6bfe" fill-opacity="0.2"/>
              <path d="M24 32L30 38L40 26" stroke="#4d6bfe" stroke-width="4" stroke-linecap="round" stroke-linejoin="round"/>
            </svg>
          </div>
          <h2 class="empty-state-title">开始新的对话</h2>
          <p class="empty-state-subtitle">输入消息开始与 AI 助手对话</p>
        </div>

        <Message
          v-for="message in messages"
          :key="message.id"
          :message="message"
          :is-loading="message.id === loadingMessageId && message.role === 'assistant'"
          :error="message.error"
          :is-streaming="isStreaming && message.id === loadingMessageId"
        />
      </div>

      <ChatInput
        :is-sending="isSending"
        @send="handleSend"
      />
    </div>
  </div>
</template>

<script setup>
import { ref, computed, nextTick, onMounted, watch } from 'vue'
import Sidebar from './components/Sidebar.vue'
import Message from './components/Message.vue'
import ChatInput from './components/ChatInput.vue'
import { chatStream } from './api.js'

const messages = ref([])
const chatHistory = ref([])
const currentChatId = ref(null)
const isSending = ref(false)
const isStreaming = ref(false)
const loadingMessageId = ref(null)
const chatContainer = ref(null)

let messageIdCounter = 0

const generateMessageId = () => {
  return `msg_${++messageIdCounter}_${Date.now()}`
}

const generateChatId = () => {
  return `chat_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`
}

const newChat = () => {
  currentChatId.value = null
  messages.value = []
}

const selectChat = (chatId) => {
  currentChatId.value = chatId
  const chat = chatHistory.value.find(c => c.id === chatId)
  if (chat) {
    messages.value = [...chat.messages]
  }
}

const deleteChat = (chatId) => {
  chatHistory.value = chatHistory.value.filter(c => c.id !== chatId)
  if (currentChatId.value === chatId) {
    newChat()
  }
}

const scrollToBottom = () => {
  nextTick(() => {
    if (chatContainer.value) {
      chatContainer.value.scrollTop = chatContainer.value.scrollHeight
    }
  })
}

const handleSend = async ({ text, enableSkills, enableMCP }) => {
  const userMessage = {
    id: generateMessageId(),
    role: 'user',
    content: text,
    timestamp: Date.now()
  }

  messages.value.push(userMessage)
  scrollToBottom()

  // Create or update chat in history
  if (!currentChatId.value) {
    const newChatId = generateChatId()
    currentChatId.value = newChatId
    chatHistory.value.unshift({
      id: newChatId,
      title: text.slice(0, 30) + (text.length > 30 ? '...' : ''),
      timestamp: Date.now(),
      messages: []
    })
  }

  const assistantMessage = {
    id: generateMessageId(),
    role: 'assistant',
    content: '',
    timestamp: Date.now()
  }

  messages.value.push(assistantMessage)
  loadingMessageId.value = assistantMessage.id
  scrollToBottom()

  isSending.value = true

  try {
    isStreaming.value = true
    await chatStream(
      text,
      enableSkills,
      enableMCP,
      {
        onChunk: (chunk) => {
          const msg = messages.value.find(m => m.id === assistantMessage.id)
          if (msg) {
            msg.content += chunk
            scrollToBottom()
          }
        },
        onReasoning: (reasoning) => {
          const msg = messages.value.find(m => m.id === assistantMessage.id)
          if (msg) {
            msg.reasoning_content = reasoning
          }
        },
        onDone: (fullResponse) => {
          const msg = messages.value.find(m => m.id === assistantMessage.id)
          if (msg) {
            msg.content = fullResponse
          }
        },
        onEnd: () => {
          isStreaming.value = false
          loadingMessageId.value = null
        },
        onError: (error) => {
          const msg = messages.value.find(m => m.id === assistantMessage.id)
          if (msg) {
            msg.error = error
          }
          isStreaming.value = false
          loadingMessageId.value = null
        }
      }
    )

    // Update chat history
    const chat = chatHistory.value.find(c => c.id === currentChatId.value)
    if (chat) {
      chat.messages = [...messages.value]
    }

  } catch (error) {
    console.error('Chat error:', error)
    const msg = messages.value.find(m => m.id === assistantMessage.id)
    if (msg) {
      msg.error = error.message || '发送消息失败'
    }
  } finally {
    isSending.value = false
    isStreaming.value = false
    loadingMessageId.value = null
  }
}

onMounted(() => {
  // Initialize with an empty chat
  messages.value = []
})
</script>

<style scoped>
.app {
  display: flex;
  width: 100vw;
  height: 100vh;
  background-color: var(--bg-primary);
}

.main-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.chat-container {
  flex: 1;
  overflow-y: auto;
  padding-bottom: 16px;
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  padding: 32px;
}

.empty-state-icon {
  margin-bottom: 24px;
}

.empty-state-title {
  font-size: 24px;
  font-weight: 600;
  color: var(--text-primary);
  margin-bottom: 8px;
}

.empty-state-subtitle {
  font-size: 14px;
  color: var(--text-secondary);
}
</style>
<template>
  <div class="sidebar">
    <div class="sidebar-header">
      <div class="logo">
        <svg width="32" height="32" viewBox="0 0 32 32" fill="none" xmlns="http://www.w3.org/2000/svg">
          <rect width="32" height="32" rx="8" fill="#4d6bfe"/>
          <path d="M8 16C8 11.5817 11.5817 8 16 8C20.4183 8 24 11.5817 24 16C24 20.4183 20.4183 24 16 24C11.5817 24 8 20.4183 8 16Z" fill="white"/>
          <path d="M12 16L15 19L20 13" stroke="#4d6bfe" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
        </svg>
        <span class="logo-text">LangChat</span>
      </div>
    </div>

    <div class="sidebar-content">
      <button class="new-chat-btn" @click="$emit('newChat')">
        <svg width="20" height="20" viewBox="0 0 20 20" fill="none" xmlns="http://www.w3.org/2000/svg">
          <path d="M10 4.16675V15.8334" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
          <path d="M4.16675 10H15.8334" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
        </svg>
        新建对话
      </button>

      <div class="chat-history">
        <div class="history-label">历史对话</div>
        <div class="history-list">
          <div
            v-for="chat in chatHistory"
            :key="chat.id"
            class="history-item"
            :class="{ active: chat.id === currentChatId }"
            @click="$emit('selectChat', chat.id)"
          >
            <div class="history-item-title">{{ chat.title }}</div>
            <button class="history-item-delete" @click.stop="$emit('deleteChat', chat.id)">
              <svg width="16" height="16" viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg">
                <path d="M3.33325 5.33333H12.6666" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
                <path d="M5.99992 5.33333V10.6667C5.99992 11.0203 6.1404 11.3594 6.39045 11.6095C6.64049 11.8595 6.97963 12 7.33325 12H8.66659C9.02021 12 9.35935 11.8595 9.6094 11.6095C9.85944 11.3594 9.99992 11.0203 9.99992 10.6667V5.33333" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
              </svg>
            </button>
          </div>

          <div v-if="chatHistory.length === 0" class="empty-history">
            暂无历史对话
          </div>
        </div>
      </div>
    </div>

    <div class="sidebar-footer">
      <div class="settings-btn">
        <svg width="20" height="20" viewBox="0 0 20 20" fill="none" xmlns="http://www.w3.org/2000/svg">
          <path d="M10.0001 5.83341C8.15915 5.83341 6.66675 7.32581 6.66675 9.16675C6.66675 11.0077 8.15915 12.5001 10.0001 12.5001C11.841 12.5001 13.3334 11.0077 13.3334 9.16675C13.3334 7.32581 11.841 5.83341 10.0001 5.83341Z" stroke="currentColor" stroke-width="1.5"/>
          <path d="M10.0001 1.66675C8.01683 1.66675 6.16675 2.45409 4.81045 3.81045C3.45409 5.16675 2.66675 7.01683 2.66675 9.00008C2.66675 10.9833 3.45409 12.8334 4.81045 14.1897C6.16675 15.5461 8.01683 16.3334 10.0001 16.3334C11.9833 16.3334 13.8334 15.5461 15.1897 14.1897C16.5461 12.8334 17.3334 10.9833 17.3334 9.00008C17.3334 7.01683 16.5461 5.16675 15.1897 3.81045C13.8334 2.45409 11.9833 1.66675 10.0001 1.66675Z" stroke="currentColor" stroke-width="1.5"/>
        </svg>
        <span>设置</span>
      </div>
    </div>
  </div>
</template>

<script setup>
defineProps({
  chatHistory: {
    type: Array,
    default: () => []
  },
  currentChatId: {
    type: String,
    default: null
  }
})

defineEmits(['newChat', 'selectChat', 'deleteChat'])
</script>

<style scoped>
.sidebar {
  width: 260px;
  height: 100%;
  background-color: var(--bg-secondary);
  border-right: 1px solid var(--border-color);
  display: flex;
  flex-direction: column;
}

.sidebar-header {
  padding: 16px;
  border-bottom: 1px solid var(--border-color);
}

.logo {
  display: flex;
  align-items: center;
  gap: 12px;
}

.logo-text {
  font-size: 18px;
  font-weight: 600;
  color: var(--text-primary);
}

.sidebar-content {
  flex: 1;
  overflow-y: auto;
  padding: 12px;
}

.new-chat-btn {
  width: 100%;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 16px;
  background-color: var(--bg-primary);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  color: var(--text-primary);
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
}

.new-chat-btn:hover {
  background-color: #f3f4f6;
  border-color: #d1d5db;
}

.chat-history {
  margin-top: 24px;
}

.history-label {
  font-size: 12px;
  color: var(--text-tertiary);
  font-weight: 500;
  margin-bottom: 8px;
  padding: 0 4px;
}

.history-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.history-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 12px;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.2s;
}

.history-item:hover {
  background-color: var(--bg-primary);
}

.history-item.active {
  background-color: #f0f3ff;
}

.history-item-title {
  flex: 1;
  font-size: 14px;
  color: var(--text-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.history-item-delete {
  opacity: 0;
  padding: 4px;
  background: none;
  border: none;
  color: var(--text-tertiary);
  cursor: pointer;
  transition: all 0.2s;
  display: flex;
  align-items: center;
}

.history-item:hover .history-item-delete {
  opacity: 1;
}

.history-item-delete:hover {
  color: #ef4444;
  background-color: rgba(239, 68, 68, 0.1);
  border-radius: 4px;
}

.empty-history {
  padding: 20px;
  text-align: center;
  color: var(--text-tertiary);
  font-size: 13px;
}

.sidebar-footer {
  padding: 12px;
  border-top: 1px solid var(--border-color);
}

.settings-btn {
  width: 100%;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 12px;
  background: none;
  border: none;
  border-radius: 6px;
  color: var(--text-secondary);
  font-size: 14px;
  cursor: pointer;
  transition: all 0.2s;
}

.settings-btn:hover {
  background-color: var(--bg-primary);
  color: var(--text-primary);
}
</style>
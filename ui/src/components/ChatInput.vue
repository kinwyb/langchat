<template>
  <div class="chat-input-container">
    <div class="input-wrapper">
      <textarea
        ref="textarea"
        v-model="inputText"
        class="chat-input"
        placeholder="输入消息..."
        rows="1"
        @keydown="handleKeydown"
        @input="autoResize"
      ></textarea>
      <div class="input-actions">
        <button
          class="send-btn"
          :disabled="!inputText.trim() || isSending"
          @click="send"
        >
          <svg v-if="isSending" class="loading" width="16" height="16" viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg">
            <path d="M8 1V15" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
            <path d="M1 8H15" stroke="currentColor" stroke-width="2" stroke-linecap="round"/>
          </svg>
          <svg v-else width="16" height="16" viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg">
            <path d="M13.5 2.5L2.5 8L6.5 9.5L8 13.5L13.5 2.5Z" stroke="currentColor" stroke-width="1.5" stroke-linejoin="round"/>
            <path d="M6.5 9.5L13.5 2.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
          </svg>
        </button>
      </div>
    </div>
    <div class="input-footer">
      <label class="toggle-label">
        <input v-model="enableSkills" type="checkbox" class="toggle">
        <span>启用技能</span>
      </label>
    </div>
  </div>
</template>

<script setup>
import { ref, watch, nextTick } from 'vue'

const props = defineProps({
  isSending: {
    type: Boolean,
    default: false
  }
})

const emit = defineEmits(['send'])

const inputText = ref('')
const textarea = ref(null)
const enableSkills = ref(true)

const autoResize = () => {
  const el = textarea.value
  if (!el) return

  el.style.height = 'auto'
  const newHeight = Math.min(el.scrollHeight, 200)
  el.style.height = newHeight + 'px'
}

const handleKeydown = (e) => {
  // Cmd/Ctrl + Enter to send, Enter for newline
  if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) {
    e.preventDefault()
    send()
  }
}

const send = () => {
  const text = inputText.value.trim()
  if (!text || props.isSending) return

  emit('send', {
    text,
    enableSkills: enableSkills.value,
    enableMCP: false
  })

  inputText.value = ''
  nextTick(() => {
    autoResize()
  })
}

// Auto-resize on mount
nextTick(() => {
  autoResize()
})
</script>

<style scoped>
.chat-input-container {
  background-color: var(--bg-secondary);
  border-top: 1px solid var(--border-color);
  padding: 16px 32px;
}

.input-wrapper {
  max-width: 900px;
  margin: 0 auto;
  position: relative;
  display: flex;
  align-items: flex-end;
  gap: 12px;
  background-color: var(--bg-primary);
  border: 1px solid var(--border-color);
  border-radius: 12px;
  padding: 8px 12px;
  transition: all 0.2s;
}

.input-wrapper:focus-within {
  border-color: var(--primary-color);
  box-shadow: 0 0 0 3px rgba(77, 107, 254, 0.1);
}

.chat-input {
  flex: 1;
  border: none;
  background: none;
  resize: none;
  font-size: 14px;
  line-height: 1.6;
  color: var(--text-primary);
  font-family: inherit;
  max-height: 200px;
  min-height: 24px;
  overflow-y: auto;
  outline: none;
}

.chat-input::placeholder {
  color: var(--text-tertiary);
}

.input-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.send-btn {
  width: 32px;
  height: 32px;
  border: none;
  border-radius: 8px;
  background-color: var(--primary-color);
  color: white;
  cursor: pointer;
  transition: all 0.2s;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.send-btn:hover:not(:disabled) {
  background-color: var(--primary-hover);
  transform: scale(1.05);
}

.send-btn:disabled {
  background-color: var(--text-tertiary);
  cursor: not-allowed;
  opacity: 0.6;
}

.send-btn .loading {
  animation: rotate 1s linear infinite;
}

@keyframes rotate {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}

.input-footer {
  max-width: 900px;
  margin: 12px auto 0;
  display: flex;
  align-items: center;
  gap: 16px;
}

.toggle-label {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  color: var(--text-secondary);
  cursor: pointer;
  user-select: none;
}

.toggle {
  width: 16px;
  height: 16px;
  border-radius: 4px;
  border: 1px solid #d1d5db;
  cursor: pointer;
  appearance: none;
  position: relative;
  transition: all 0.2s;
}

.toggle:checked {
  background-color: var(--primary-color);
  border-color: var(--primary-color);
}

.toggle:checked::after {
  content: '';
  position: absolute;
  left: 5px;
  top: 2px;
  width: 4px;
  height: 8px;
  border: solid white;
  border-width: 0 2px 2px 0;
  transform: rotate(45deg);
}
</style>
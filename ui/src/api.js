import axios from 'axios'

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api'

const api = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    'Content-Type': 'application/json'
  }
})

// Health check
export const checkHealth = async () => {
  const response = await api.get('/health')
  return response.data
}

// Non-streaming chat
export const chat = async (message, enableSkills = true, enableMCP = false) => {
  const response = await api.post('/chat', {
    message,
    enableSkills,
    enableMCP
  })
  return response.data
}

// Streaming chat using Server-Sent Events
export const chatStream = async (message, enableSkills = true, enableMCP = false, callbacks) => {
  const response = await fetch(`${API_BASE_URL}/chat/stream`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json'
    },
    body: JSON.stringify({
      message,
      enableSkills,
      enableMCP
    })
  })

  if (!response.ok) {
    throw new Error(`HTTP error! status: ${response.status}`)
  }

  const reader = response.body.getReader()
  const decoder = new TextDecoder()
  let buffer = ''
  let fullResponse = ''

  try {
    while (true) {
      const { done, value } = await reader.read()

      if (done) break

      buffer += decoder.decode(value, { stream: true })

      // Process SSE events
      const lines = buffer.split('\n')
      buffer = lines.pop() || ''

      for (const line of lines) {
        if (line.startsWith('data: ')) {
          const data = line.slice(6)
          if (callbacks.onChunk) {
            callbacks.onChunk(data)
          }
          fullResponse += data
        } else if (line.startsWith('event: ')) {
          const event = line.slice(7)
          if (event === 'done') {
            if (callbacks.onDone) {
              callbacks.onDone(fullResponse)
            }
          } else if (event === 'error') {
            const nextLine = lines.find(l => l.startsWith('data: '))
            if (nextLine && callbacks.onError) {
              callbacks.onError(nextLine.slice(6))
            }
          } else if (event === 'end') {
            if (callbacks.onEnd) {
              callbacks.onEnd()
            }
            // Break out of the loop and return
            return fullResponse
          }
        }
      }
    }
  } finally {
    reader.releaseLock()
  }

  return fullResponse
}

export default api
import { useState } from 'react'
import { X, Send, Loader2, AlertCircle, Zap } from 'lucide-react'
import type { ChatMessage, ChatResponse } from '../../api/types'
import { useChat, useWallet } from '../../api/hooks'
import { calculateChatCost, formatCredits } from '../../lib/pricing'

interface ChatDialogProps {
  isOpen: boolean
  onClose: () => void
  episodeId: string
}

export default function ChatDialog({ isOpen, onClose, episodeId }: ChatDialogProps) {
  const [messages, setMessages] = useState<ChatMessage[]>([])
  const [inputText, setInputText] = useState('')
  const [responses, setResponses] = useState<Map<string, ChatResponse>>(new Map())
  const [totalCost, setTotalCost] = useState(0)

  const chatMutation = useChat()
  const walletQuery = useWallet()

  const walletData = walletQuery.data?.wallet
  const availableBalance = walletData?.balance ?? 0
  const isInsufficientBalance = totalCost > availableBalance

  const handleSendMessage = async () => {
    if (!inputText.trim()) return

    const userMessage: ChatMessage = {
      role: 'user',
      content: inputText,
    }

    const newMessages = [...messages, userMessage]
    setMessages(newMessages)
    setInputText('')

    try {
      const result = await chatMutation.mutateAsync({
        episodeId,
        request: {
          messages: newMessages,
        },
      })

      const chatResponse = result.chat_response
      const assistantMessage: ChatMessage = {
        role: 'assistant',
        content: chatResponse.content,
      }

      // 保存响应数据以便计算成本
      responses.set(chatResponse.id, chatResponse)
      setResponses(new Map(responses))

      // 更新消息列表
      setMessages([...newMessages, assistantMessage])

      // 计算本次调用的成本
      const inputTokens = chatResponse.token_usage?.input_tokens ?? 0
      const outputTokens = chatResponse.token_usage?.output_tokens ?? 0
      const cost = calculateChatCost(inputTokens, outputTokens)
      setTotalCost(totalCost + cost)
    } catch (error: any) {
      console.error('Chat error:', error)
      // 移除用户消息（如果出错）
      setMessages(newMessages.slice(0, -1))
      setInputText(userMessage.content)
    }
  }

  const handleClear = () => {
    setMessages([])
    setResponses(new Map())
    setTotalCost(0)
    setInputText('')
  }

  if (!isOpen) return null

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
      <div className="w-full max-w-2xl h-[600px] rounded-lg bg-white dark:bg-slate-900 shadow-xl flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between border-b border-slate-200 dark:border-slate-700 p-6">
          <h2 className="text-lg font-semibold text-slate-900 dark:text-white">
            Chat Assistant
          </h2>
          <button
            onClick={onClose}
            className="text-slate-500 hover:text-slate-700 dark:text-slate-400 dark:hover:text-slate-200 transition-colors"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Messages Area */}
        <div className="flex-1 overflow-y-auto p-6 space-y-4">
          {messages.length === 0 ? (
            <div className="h-full flex items-center justify-center text-slate-500 dark:text-slate-400">
              <p>Start a conversation...</p>
            </div>
          ) : (
            messages.map((msg, idx) => (
              <div
                key={idx}
                className={`flex ${msg.role === 'user' ? 'justify-end' : 'justify-start'}`}
              >
                <div
                  className={`max-w-xs px-4 py-2 rounded-lg ${
                    msg.role === 'user'
                      ? 'bg-blue-500 text-white'
                      : 'bg-slate-100 dark:bg-slate-800 text-slate-900 dark:text-slate-100'
                  }`}
                >
                  <p className="text-sm">{msg.content}</p>
                </div>
              </div>
            ))
          )}
        </div>

        {/* Cost Info */}
        {totalCost > 0 && (
          <div className={`px-6 py-3 border-t border-slate-200 dark:border-slate-700 ${
            isInsufficientBalance ? 'bg-red-50 dark:bg-red-900/20' : 'bg-blue-50 dark:bg-blue-900/20'
          }`}>
            <div className="flex items-center gap-2">
              {isInsufficientBalance ? (
                <>
                  <AlertCircle className="w-4 h-4 text-red-600 dark:text-red-400" />
                  <span className="text-xs font-medium text-red-600 dark:text-red-400">
                    Insufficient balance: {formatCredits(totalCost)} needed, {formatCredits(availableBalance)} available
                  </span>
                </>
              ) : (
                <>
                  <Zap className="w-4 h-4 text-blue-600 dark:text-blue-400" />
                  <span className="text-xs font-medium text-blue-600 dark:text-blue-400">
                    Cost: {formatCredits(totalCost)} • Balance: {formatCredits(availableBalance)}
                  </span>
                </>
              )}
            </div>
          </div>
        )}

        {/* Input Area */}
        <div className="border-t border-slate-200 dark:border-slate-700 p-6">
          <div className="flex gap-3">
            <input
              type="text"
              value={inputText}
              onChange={(e) => setInputText(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter' && !e.shiftKey) {
                  e.preventDefault()
                  handleSendMessage()
                }
              }}
              placeholder="Type your message... (Shift+Enter for new line)"
              disabled={chatMutation.isPending || isInsufficientBalance}
              className="flex-1 px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-lg bg-white dark:bg-slate-800 text-slate-900 dark:text-white placeholder-slate-400 dark:placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50"
            />
            <button
              onClick={handleSendMessage}
              disabled={chatMutation.isPending || isInsufficientBalance || !inputText.trim()}
              className="px-4 py-2 bg-blue-500 hover:bg-blue-600 text-white font-medium rounded-lg transition-colors flex items-center justify-center gap-2 disabled:opacity-50"
            >
              {chatMutation.isPending ? (
                <Loader2 className="w-4 h-4 animate-spin" />
              ) : (
                <Send className="w-4 h-4" />
              )}
            </button>
          </div>
        </div>

        {/* Footer */}
        <div className="flex gap-3 border-t border-slate-200 dark:border-slate-700 p-6">
          <button
            onClick={handleClear}
            disabled={chatMutation.isPending}
            className="flex-1 py-2 px-4 border border-slate-300 dark:border-slate-600 text-slate-700 dark:text-slate-300 font-medium rounded-lg hover:bg-slate-50 dark:hover:bg-slate-800 transition-colors disabled:opacity-50"
          >
            Clear
          </button>
          <button
            onClick={onClose}
            disabled={chatMutation.isPending}
            className="flex-1 py-2 px-4 border border-slate-300 dark:border-slate-600 text-slate-700 dark:text-slate-300 font-medium rounded-lg hover:bg-slate-50 dark:hover:bg-slate-800 transition-colors disabled:opacity-50"
          >
            Close
          </button>
        </div>
      </div>
    </div>
  )
}

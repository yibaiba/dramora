/**
 * Pricing utilities for frontend cost calculation.
 * 
 * Mirrors backend pricing model:
 * - 1000 tokens = 10 credits (rounded up)
 * - CalculateChatCost = ceil(totalTokens / 1000) * 10
 */

const TOKEN_COST_PER_THOUSAND = 10

/**
 * Calculate chat operation cost in credits based on token count.
 * 
 * Formula: ceil(tokens / 1000) * 10
 * - 1-999 tokens = 10 credits (minimum)
 * - 1000-1999 tokens = 20 credits
 * - 2000-2999 tokens = 30 credits, etc.
 */
export function calculateChatCost(
  inputTokens: number = 0,
  outputTokens: number = 0
): number {
  const totalTokens = inputTokens + outputTokens
  if (totalTokens <= 0) return 0
  
  // Ceiling division: (tokens + 999) / 1000 * 10
  const thousands = Math.ceil(totalTokens / 1000)
  return thousands * TOKEN_COST_PER_THOUSAND
}

/**
 * Format token count with readable suffix.
 * Examples: "150 tokens", "1.2k tokens"
 */
export function formatTokens(count: number): string {
  if (count < 1000) {
    return `${count} tokens`
  }
  return `${(count / 1000).toFixed(1)}k tokens`
}

/**
 * Format credits with readable suffix and icon.
 * Examples: "10 credits", "100 credits"
 */
export function formatCredits(count: number): string {
  return `${count} credits`
}

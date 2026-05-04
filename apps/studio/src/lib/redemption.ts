/**
 * 赎回码验证和工具函数
 */

/**
 * 验证赎回码格式和内容
 */
export function validateRedemptionCode(code: string): {
  valid: boolean
  error?: string
} {
  const normalized = code.trim().toUpperCase()

  if (!normalized) {
    return { valid: false, error: '赎回码不能为空' }
  }

  // 预期格式: GIFT-XXXXXXXX (13 字符)
  // 支持大小写和中间可能没有连字符
  if (!/^[A-Z0-9-]+$/.test(normalized)) {
    return {
      valid: false,
      error: '赎回码格式不正确，只支持数字、字母和连字符',
    }
  }

  if (normalized.length > 20) {
    return {
      valid: false,
      error: '赎回码过长',
    }
  }

  return { valid: true }
}

/**
 * 规范化赎回码（大写、去空格）
 */
export function normalizeRedemptionCode(code: string): string {
  return code.trim().toUpperCase()
}

/**
 * CSV 字段转义 - 处理逗号、引号、换行符
 */
export function escapeCsvField(field: string | number): string {
  const str = String(field)

  // 如果包含特殊字符，用引号包裹，并转义内部的引号
  if (str.includes(',') || str.includes('"') || str.includes('\n')) {
    return `"${str.replace(/"/g, '""')}"`
  }

  return str
}

/**
 * 生成 CSV 内容（带 BOM 用于 Excel）
 */
export function generateCsvContent(
  codes: string[],
  amount: number,
): string {
  const headers = ['赎回码', '积分额度']
  const rows = codes.map((code) => [escapeCsvField(code), escapeCsvField(amount)])

  // UTF-8 BOM (用于 Excel 正确识别中文)
  const bom = '\uFEFF'
  const csvHeader = headers.map(escapeCsvField).join(',')
  const csvRows = rows.map((row) => row.join(','))

  return bom + csvHeader + '\n' + csvRows.join('\n') + '\n'
}

/**
 * 下载 CSV 文件
 */
export function downloadCsv(content: string, filename: string): void {
  const blob = new Blob([content], { type: 'text/csv;charset=utf-8;' })
  const link = document.createElement('a')
  const url = URL.createObjectURL(blob)

  link.setAttribute('href', url)
  link.setAttribute('download', filename)
  link.style.visibility = 'hidden'

  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)

  URL.revokeObjectURL(url)
}

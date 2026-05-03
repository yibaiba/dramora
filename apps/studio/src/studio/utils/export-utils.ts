import type { Asset } from '../../api/types'

// 文件大小限制：100MB
const MAX_FILE_SIZE = 100 * 1024 * 1024
// 单个文件下载超时：30秒
const FETCH_TIMEOUT = 30000

/**
 * 获取认证 token
 */
function getAuthToken(): string | null {
  try {
    // 从 localStorage 尝试获取 token
    const sessionStr = localStorage.getItem('dramora-auth-session')
    if (sessionStr) {
      const session = JSON.parse(sessionStr)
      return session.token
    }
  } catch (error) {
    console.warn('Failed to retrieve auth token:', error)
  }
  return null
}

/**
 * 带超时的 fetch
 */
function fetchWithTimeout(uri: string, timeout: number): Promise<Response> {
  const controller = new AbortController()
  const timeoutId = setTimeout(() => controller.abort(), timeout)

  return fetch(uri, {
    signal: controller.signal,
    headers: {
      'Authorization': `Bearer ${getAuthToken() || ''}`,
    },
  }).finally(() => clearTimeout(timeoutId))
}

/**
 * 导出资产为 ZIP 文件
 * @param assets 要导出的资产列表
 * @param episodeId 用于 ZIP 文件名
 * @param onProgress 进度回调函数（0-100）
 */
export async function exportAssetsAsZip(
  assets: Asset[],
  episodeId: string,
  onProgress?: (percent: number) => void
): Promise<void> {
  // 动态导入 JSZip 库
  const JSZip = (await import('jszip')).default
  const zip = new JSZip()

  if (!assets || assets.length === 0) {
    throw new Error('没有选择要导出的资产')
  }

  let successCount = 0
  const totalCount = assets.length

  for (let i = 0; i < assets.length; i++) {
    const asset = assets[i]
    try {
      // 从 URI 下载资产文件（带认证和超时）
      const response = await fetchWithTimeout(asset.uri, FETCH_TIMEOUT)
      if (!response.ok) {
        console.warn(`Failed to download asset ${asset.id}`)
        continue
      }

      // 检查文件大小
      const contentLength = response.headers.get('content-length')
      if (contentLength && parseInt(contentLength) > MAX_FILE_SIZE) {
        console.warn(`File too large for asset ${asset.id}: ${contentLength} bytes`)
        continue
      }

      const blob = await response.blob()

      // 再次检查 blob 大小
      if (blob.size > MAX_FILE_SIZE) {
        console.warn(`Blob too large for asset ${asset.id}: ${blob.size} bytes`)
        continue
      }

      // 获取文件扩展名
      const ext = getFileExtension(asset.uri, blob.type)

      // 在 ZIP 中创建目录结构：/assets/{kind}/{id}.{ext}
      const filePath = `assets/${asset.kind}/${asset.id}.${ext}`
      zip.file(filePath, blob)
      successCount++

      // 更新进度：基于成功下载的资产数量
      if (onProgress) {
        onProgress(Math.round((successCount / totalCount) * 100))
      }
    } catch (error) {
      if (error instanceof Error && error.name === 'AbortError') {
        console.error(`Download timeout for asset ${asset.id}`)
      } else {
        console.error(`Error processing asset ${asset.id}:`, error)
      }
    }
  }

  if (successCount === 0) {
    throw new Error('没有资产可导出')
  }

  // 生成 ZIP 文件
  const timestamp = new Date().toISOString().split('T')[0]
  const filename = `assets_${episodeId}_${timestamp}.zip`
  const zipBlob = await zip.generateAsync({ type: 'blob' })

  // 浏览器下载
  downloadBlob(zipBlob, filename)
}

/**
 * 从 MIME 类型或 URI 获取文件扩展名
 */
function getFileExtension(uri: string, mimeType: string): string {
  // 尝试从 URI 获取扩展名
  const uriExt = uri.split('.').pop()?.split('?')[0]?.toLowerCase()
  if (uriExt && uriExt.length < 10) {
    return uriExt
  }

  // 从 MIME 类型推断扩展名
  const mimeMap: Record<string, string> = {
    'video/mp4': 'mp4',
    'video/webm': 'webm',
    'video/quicktime': 'mov',
    'image/jpeg': 'jpg',
    'image/png': 'png',
    'image/webp': 'webp',
    'audio/mpeg': 'mp3',
    'audio/wav': 'wav',
  }

  const ext = mimeMap[mimeType]
  return ext || 'bin'
}

/**
 * 触发浏览器下载
 */
function downloadBlob(blob: Blob, filename: string): void {
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
}

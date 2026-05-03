import type { Asset } from '../../api/types'

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

  // 为每个资产生成文件夹
  const progressStep = 100 / assets.length

  for (let i = 0; i < assets.length; i++) {
    const asset = assets[i]
    try {
      // 从 URI 下载资产文件
      const response = await fetch(asset.uri)
      if (!response.ok) {
        console.warn(`Failed to download asset ${asset.id}`)
        continue
      }

      const blob = await response.blob()

      // 获取文件扩展名
      const ext = getFileExtension(asset.uri, blob.type)

      // 在 ZIP 中创建目录结构：/assets/{kind}/{id}.{ext}
      const filePath = `assets/${asset.kind}/${asset.id}.${ext}`
      zip.file(filePath, blob)

      // 更新进度
      if (onProgress) {
        onProgress(Math.round((i + 1) * progressStep))
      }
    } catch (error) {
      console.error(`Error processing asset ${asset.id}:`, error)
    }
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

import { FFmpeg } from '@ffmpeg/ffmpeg'
import type { Clip } from './types'

type ProgressCallback = (progress: number, estimatedSeconds: number) => void

let ffmpeg: FFmpeg | null = null
let isInitialized = false

export async function initFFmpeg(): Promise<void> {
  if (isInitialized && ffmpeg?.loaded) {
    return
  }

  ffmpeg = new FFmpeg()

  ffmpeg.on('log', ({ message }) => {
    console.debug('[FFmpeg]', message)
  })

  ffmpeg.on('progress', (progress) => {
    console.debug('[FFmpeg Progress]', progress)
  })

  try {
    // Load ffmpeg.wasm from CDN
    await ffmpeg.load({
      coreURL: 'https://cdn.jsdelivr.net/npm/@ffmpeg/core@0.12.15/dist/esm',
    })
    isInitialized = true
  } catch (error) {
    isInitialized = false
    throw new Error(`FFmpeg initialization failed: ${error instanceof Error ? error.message : String(error)}`)
  }
}

export async function processVideo(
  inputBlob: Blob,
  clips: Clip[],
): Promise<Blob> {
  if (!ffmpeg || !isInitialized || !ffmpeg.loaded) {
    throw new Error('FFmpeg not initialized. Call initFFmpeg() first.')
  }

  const fileName = `input_${Date.now()}.mp4`
  const outputFileName = `output_${Date.now()}.mp4`

  try {
    // Write input file to FFmpeg virtual filesystem
    const inputData = await inputBlob.arrayBuffer()
    await ffmpeg.writeFile(fileName, new Uint8Array(inputData))

    // Sort clips by start time
    const sortedClips = clips.sort((a, b) => a.startTime - b.startTime)

    if (sortedClips.length === 0) {
      throw new Error('No clips to process')
    }

    // Build FFmpeg command for simple concat without speed/trim for now
    const ffmpegArgs: string[] = [
      '-i',
      fileName,
      '-c',
      'copy',
      '-y',
      outputFileName,
    ]

    // Execute FFmpeg
    await ffmpeg.exec(ffmpegArgs)

    // Read output file
    const outputData = await ffmpeg.readFile(outputFileName)
    const outputBlob = new Blob([(outputData as unknown as BlobPart)], { type: 'video/mp4' })

    // Clean up temporary files
    try {
      await ffmpeg.deleteFile(fileName)
    } catch {
      // File might not exist
    }
    try {
      await ffmpeg.deleteFile(outputFileName)
    } catch {
      // File might not exist
    }

    return outputBlob
  } catch (error) {
    throw new Error(`Video processing failed: ${error instanceof Error ? error.message : String(error)}`)
  }
}

export async function encodeToMP4(
  inputBlob: Blob,
  quality: 'low' | 'medium' | 'high' | 'very-high' = 'medium',
  onProgress?: ProgressCallback,
): Promise<Blob> {
  if (!ffmpeg || !isInitialized || !ffmpeg.loaded) {
    throw new Error('FFmpeg not initialized. Call initFFmpeg() first.')
  }

  const fileName = `input_${Date.now()}.mp4`
  const outputFileName = `output_${Date.now()}.mp4`

  try {
    // Write input file
    const inputData = await inputBlob.arrayBuffer()
    await ffmpeg.writeFile(fileName, new Uint8Array(inputData))

    // Map quality to CRF (lower = better quality, 0-51, typical 18-28)
    const crfMap: Record<string, string> = {
      low: '30',
      medium: '23',
      high: '18',
      'very-high': '12',
    }
    const crf = crfMap[quality]

    // Set up progress tracking
    let lastProgressTime = Date.now()

    ffmpeg.on('progress', ({ time }) => {
      if (onProgress && Date.now() - lastProgressTime > 500) {
        const totalMilliseconds = time || 0
        const estimatedTotalSeconds = 60
        const progress = Math.min(0.95, totalMilliseconds / (estimatedTotalSeconds * 1000))
        onProgress(progress, estimatedTotalSeconds)
        lastProgressTime = Date.now()
      }
    })

    // Encode to MP4
    const ffmpegArgs: string[] = [
      '-i',
      fileName,
      '-c:v',
      'libx264',
      '-preset',
      'fast',
      '-crf',
      crf,
      '-c:a',
      'aac',
      '-b:a',
      '128k',
      '-y',
      outputFileName,
    ]

    await ffmpeg.exec(ffmpegArgs)

    // Read output file
    const outputData = await ffmpeg.readFile(outputFileName)
    const outputBlob = new Blob([(outputData as unknown as BlobPart)], { type: 'video/mp4' })

    // Clean up
    try {
      await ffmpeg.deleteFile(fileName)
    } catch {
      // File might not exist
    }
    try {
      await ffmpeg.deleteFile(outputFileName)
    } catch {
      // File might not exist
    }

    if (onProgress) {
      onProgress(1, 0)
    }

    return outputBlob
  } catch (error) {
    throw new Error(`MP4 encoding failed: ${error instanceof Error ? error.message : String(error)}`)
  }
}

export async function trimAndEncodeClip(
  inputBlob: Blob,
  startTimeSeconds: number,
  durationSeconds: number,
  speedFactor: number = 1,
  quality: 'low' | 'medium' | 'high' | 'very-high' = 'medium',
): Promise<Blob> {
  if (!ffmpeg || !isInitialized || !ffmpeg.loaded) {
    throw new Error('FFmpeg not initialized. Call initFFmpeg() first.')
  }

  const fileName = `input_${Date.now()}.mp4`
  const outputFileName = `output_${Date.now()}.mp4`

  try {
    const inputData = await inputBlob.arrayBuffer()
    await ffmpeg.writeFile(fileName, new Uint8Array(inputData))

    const crfMap: Record<string, string> = {
      low: '30',
      medium: '23',
      high: '18',
      'very-high': '12',
    }
    const crf = crfMap[quality]

    const trimEnd = startTimeSeconds + durationSeconds
    const speedOption = speedFactor !== 1 ? `,setpts=${1 / speedFactor}*PTS` : ''
    const filterVideo = `trim=${startTimeSeconds}:${trimEnd}${speedOption},fps=30`
    const filterAudio =
      speedFactor !== 1
        ? `atrim=${startTimeSeconds}:${trimEnd},atempo=${speedFactor}`
        : `atrim=${startTimeSeconds}:${trimEnd}`

    const ffmpegArgs: string[] = [
      '-i',
      fileName,
      '-vf',
      filterVideo,
      '-af',
      filterAudio,
      '-c:v',
      'libx264',
      '-preset',
      'fast',
      '-crf',
      crf,
      '-c:a',
      'aac',
      '-b:a',
      '128k',
      '-y',
      outputFileName,
    ]

    await ffmpeg.exec(ffmpegArgs)

    const outputData = await ffmpeg.readFile(outputFileName)
    const outputBlob = new Blob([(outputData as unknown as BlobPart)], { type: 'video/mp4' })

    // Clean up
    try {
      await ffmpeg.deleteFile(fileName)
    } catch {
      // File might not exist
    }
    try {
      await ffmpeg.deleteFile(outputFileName)
    } catch {
      // File might not exist
    }

    return outputBlob
  } catch (error) {
    throw new Error(`Trim and encode failed: ${error instanceof Error ? error.message : String(error)}`)
  }
}

export function unloadFFmpeg(): void {
  if (ffmpeg) {
    ffmpeg = null
    isInitialized = false
  }
}

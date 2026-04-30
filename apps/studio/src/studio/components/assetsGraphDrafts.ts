import type { CharacterBible, StoryMapItem } from '../../api/types'

export type GraphNodeKind = 'character' | 'scene' | 'prop'

export type CharacterBibleDraft = CharacterBible

export const characterBibleExpressions = ['中性', '开心', '愤怒', '悲伤', '惊讶', '沉思']

export const characterBibleReferenceAngles = [
  '正面',
  '背面',
  '3/4 左',
  '3/4 右',
  '侧面左',
  '侧面右',
  'T-pose',
]

export function createCharacterBibleDraft(node: StoryMapItem): CharacterBibleDraft {
  if (node.character_bible) {
    return cloneCharacterBibleDraft(node.character_bible)
  }

  return {
    anchor: `${node.name}，保持角色外观稳定；${node.description || '补充年龄、体态、发型、标志性特征和服装主色。'}`,
    expressions: ['中性', '开心', '惊讶'],
    notes: '保存后会写回角色节点的 Character Bible 字段。',
    palette: {
      accent: '#3B82F6',
      costume: '#1F2937',
      eyes: '#22C55E',
      hair: '#1A1A2E',
      skin: '#E8C9A0',
    },
    reference_angles: [...characterBibleReferenceAngles],
    reference_assets: [],
    wardrobe: `${node.code}: 主服装待补充`,
  }
}

export function cloneCharacterBibleDraft(draft: CharacterBibleDraft): CharacterBibleDraft {
  return {
    ...draft,
    expressions: [...draft.expressions],
    palette: { ...draft.palette },
    reference_angles:
      draft.reference_angles.length > 0 ? [...draft.reference_angles] : [...characterBibleReferenceAngles],
    reference_assets: (draft.reference_assets ?? []).map((item) => ({ ...item })),
  }
}

export function isCharacterBibleDraftDirty(
  draft: CharacterBibleDraft,
  baseline: CharacterBibleDraft,
): boolean {
  return JSON.stringify(normalizeCharacterBibleDraft(draft)) !== JSON.stringify(normalizeCharacterBibleDraft(baseline))
}

export function summarizeCharacterBibleDraft(draft: CharacterBibleDraft) {
  return {
    anchorReady: draft.anchor.trim().length > 0,
    expressionCount: draft.expressions.length,
    paletteFilled: Object.values(draft.palette).filter((value) => value.trim().length > 0).length,
    referenceAngleCount: draft.reference_angles.length,
  }
}

function normalizeCharacterBibleDraft(draft: CharacterBibleDraft) {
  return {
    anchor: draft.anchor.trim(),
    expressions: [...draft.expressions].map((value) => value.trim()).filter(Boolean).sort(),
    notes: draft.notes.trim(),
    palette: {
      accent: draft.palette.accent.trim(),
      costume: draft.palette.costume.trim(),
      eyes: draft.palette.eyes.trim(),
      hair: draft.palette.hair.trim(),
      skin: draft.palette.skin.trim(),
    },
    reference_angles: [...draft.reference_angles].map((value) => value.trim()).filter(Boolean).sort(),
    reference_assets: [...(draft.reference_assets ?? [])]
      .map((item) => ({
        angle: item.angle.trim(),
        asset_id: item.asset_id.trim(),
      }))
      .filter((item) => item.angle || item.asset_id)
      .sort((left, right) => left.angle.localeCompare(right.angle)),
    wardrobe: draft.wardrobe.trim(),
  }
}

-- Rollback: Remove IP-Adapter and LoRA parameter fields from shot_prompt_packs table
ALTER TABLE shot_prompt_packs DROP COLUMN IF EXISTS ip_adapter_strength;
ALTER TABLE shot_prompt_packs DROP COLUMN IF EXISTS lora_weight;
ALTER TABLE shot_prompt_packs DROP COLUMN IF EXISTS lora_combination_weight;

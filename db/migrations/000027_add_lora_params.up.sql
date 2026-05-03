-- Add IP-Adapter and LoRA parameter fields to shot_prompt_packs table
ALTER TABLE shot_prompt_packs ADD COLUMN IF NOT EXISTS ip_adapter_strength FLOAT DEFAULT 0.5;
ALTER TABLE shot_prompt_packs ADD COLUMN IF NOT EXISTS lora_weight FLOAT DEFAULT 0.5;
ALTER TABLE shot_prompt_packs ADD COLUMN IF NOT EXISTS lora_combination_weight FLOAT DEFAULT 1.0;

-- Add indexes for efficient queries if needed in future
-- CREATE INDEX idx_shot_prompt_packs_provider ON shot_prompt_packs(provider);

package domain

// OperationType 标识系统中可计费的操作类型。
type OperationType string

const (
	// OperationTypeChat 对话操作。
	OperationTypeChat OperationType = "chat"
	// OperationTypeStoryAnalysis 故事分析操作。
	OperationTypeStoryAnalysis OperationType = "story_analysis"
	// OperationTypeImageGeneration 图像生成操作。
	OperationTypeImageGeneration OperationType = "image_generation"
	// OperationTypeVideoGeneration 视频生成操作。
	OperationTypeVideoGeneration OperationType = "video_generation"
	// OperationTypeStoryboardEdit 故事板编辑操作。
	OperationTypeStoryboardEdit OperationType = "storyboard_edit"
	// OperationTypeCharacterEdit 角色编辑操作。
	OperationTypeCharacterEdit OperationType = "character_edit"
	// OperationTypeSceneEdit 场景编辑操作。
	OperationTypeSceneEdit OperationType = "scene_edit"
)

// OperationCosts 定义所有操作类型的成本（积分）。
// MVP 使用常量；后续可迁移到配置或数据库。
var OperationCosts = map[OperationType]int64{
	OperationTypeChat:            1,
	OperationTypeStoryAnalysis:   50,
	OperationTypeImageGeneration: 100,
	OperationTypeVideoGeneration: 200,
	OperationTypeStoryboardEdit:  5,
	OperationTypeCharacterEdit:   5,
	OperationTypeSceneEdit:       5,
}

// GetOperationCost 返回指定操作类型的成本。
// 若操作类型未知，返回 0 和 error。
func GetOperationCost(opType OperationType) (int64, error) {
	if cost, ok := OperationCosts[opType]; ok {
		return cost, nil
	}
	return 0, ErrInvalidInput
}

// PendingBillingStatus 表示待结算记录的状态。
type PendingBillingStatus string

const (
	// PendingBillingStatusPending 待重试。
	PendingBillingStatusPending PendingBillingStatus = "pending"
	// PendingBillingStatusRetrying 重试中。
	PendingBillingStatusRetrying PendingBillingStatus = "retrying"
	// PendingBillingStatusResolved 已解决（成功扣费）。
	PendingBillingStatusResolved PendingBillingStatus = "resolved"
	// PendingBillingStatusFailed 最终失败（需运营介入）。
	PendingBillingStatusFailed PendingBillingStatus = "failed"
)

// PendingBilling 表示一次待结算的操作。
// 当扣费失败时，创建此记录以支持后续重试。
type PendingBilling struct {
	ID             string
	OrganizationID string
	OperationType  OperationType
	RefType        string // 关联的业务类型（如 "story_analysis_id"）
	RefID          string
	Amount         int64
	Status         PendingBillingStatus
	RetryCount     int
	MaxRetries     int
	LastErrorMsg   string
	CreatedAt      int64 // Unix timestamp
	UpdatedAt      int64 // Unix timestamp
}

package handler

// CreateSettingRequest is the body for POST /settings. A setting is keyed by
// (parent_id, id) within the org; value is an opaque string (often JSON).
type CreateSettingRequest struct {
	ParentId string `json:"parent_id" validate:"omitempty,max=255"`
	Id       string `json:"id" validate:"required,min=1,max=255"`
	Type     string `json:"type" validate:"omitempty,max=64"`
	Value    string `json:"value"`
}

// UpdateSettingRequest is the body for PUT /settings/{parentId}/{id} (upsert).
type UpdateSettingRequest struct {
	Type  string `json:"type" validate:"omitempty,max=64"`
	Value string `json:"value"`
}

package dto


// CreateMetadataInput represents the input for creating a metadata entry
type CreateMetadataInput struct {
	OrgId      string `json:"org_id" binding:"required"`
	ParentId   string `json:"parent_id" binding:"required"`
	ParentType string `json:"parent_type" binding:"required"`
	Key        string `json:"key" binding:"required"`
	Value      string `json:"value" binding:"required"`
	Namespace  string `json:"namespace"`
}

// UpdateMetadataInput represents the input for updating a metadata entry
type UpdateMetadataInput struct {
	OrgId     string `json:"org_id" binding:"required"`
	ParentId  string `json:"parent_id" binding:"required"`
	Key       string `json:"key" binding:"required"`
	Value     string `json:"value" binding:"required"`
	Namespace string `json:"namespace"`
}

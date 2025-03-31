package request

type CreatePspRequest struct {
	Name     string            `json:"name" validate:"required"`
	PspId    string            `json:"psp" validate:"required"`
	Settings map[string]string `json:"settings" validate:"required"`
}

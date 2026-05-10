package dto

type ErrorDetails struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ErrorResponse struct {
	Success bool         `json:"success"`
	Error   ErrorDetails `json:"error"`
}

type SuccessResponse struct {
	Success bool `json:"success"`
	Data    any  `json:"data"`
}

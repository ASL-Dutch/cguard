package model

// ResponseForLwt response for Lwt request
type ResponseForLwt struct {
	CustomsId   string `json:"customs_id"`
	Status      string `json:"status"`
	LwtFilename string `json:"lwt_filename"`
	Error       string `json:"errors"`
	Brief       bool   `json:"brief"`
}

// RequestForLwt Request for Lwt
type RequestForLwt struct {
	CustomsId string `json:"customs_id"`
	Brief     bool   `json:"brief"`
}

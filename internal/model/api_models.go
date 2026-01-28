package model

// ResponseForLwt response for Lwt request
type ResponseForLwt struct {
	CustomsId   string `json:"customs_id"`
	Status      string `json:"status"`
	LwtFilename string `json:"lwt_filename"`
	Error       string `json:"errors"`
	Brief       bool   `json:"brief"`
	// IncProfit 本次生成是否使用利润率版模板/计算逻辑
	IncProfit bool `json:"inc_profit"`
}

// RequestForLwt Request for Lwt
type RequestForLwt struct {
	CustomsId string `json:"customs_id"`
	Brief     bool   `json:"brief"`
	// IncProfit 可选字段：true 表示生成包含利润率的 LWT；不传则默认按 false 处理
	IncProfit *bool `json:"inc_profit,omitempty"`
}

// LwtArticleAdjustment 品类微调参数
type LwtArticleAdjustment struct {
	// ArticleNo 品类序号
	ArticleNo int `json:"article_no"`
	// FinalDeclaredValueOld 当前申报价值
	FinalDeclaredValueOld float64 `json:"final_declared_value_old"`
	// FinalDeclaredValueTarget 原报关单申报价值（微调的目标值）
	FinalDeclaredValueTarget float64 `json:"final_declared_value_target"`
}

// RequestForLwtAdjustment LWT计算参数微调，并根据微调后的参数计算LWT
type RequestForLwtAdjustment struct {
	// CustomsId 海关编号，非必填
	CustomsId string `json:"customs_id"`
	// LwtFilename 已经生成的LWT文件名
	LwtFilename string                 `json:"lwt_filename"`
	Adjustments []LwtArticleAdjustment `json:"adjustments"`
}

// RequestForLwtAdjustment response for Lwt adjustment request
type ResponseForLwtAdjustment struct {
	// CustomsId 海关编号，非必填
	CustomsId string `json:"customs_id"`
	Status    string `json:"status"`
	// LwtFilename 用于下载新的LWT文件
	LwtFilename string `json:"lwt_filename"`
	Error       string `json:"errors"`
	// 是否依然有差异，true表示有差异，false表示没有差异
	HasDiff bool `json:"has_diff"`
}

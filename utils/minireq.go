package utils

import (
	"github.com/aobeom/minireq"
)

// Minireq 初始化
var Minireq *minireq.MiniRequest

// MiniHeaders Headers
type MiniHeaders = minireq.Headers

// MiniParams Params
type MiniParams = minireq.Params

// MiniJSONData JSONData
type MiniJSONData = minireq.JSONData

// MiniFormData FormData
type MiniFormData = minireq.FormData

func init() {
	Minireq = minireq.Requests()
}

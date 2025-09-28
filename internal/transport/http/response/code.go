package response

// 常见业务 系统级错误码（直接基于 HTTP 语义）
const (
	CodeOK           = 0
	CodeBadRequest   = 400
	CodeUnauthorized = 401
	CodeForbidden    = 403
	CodeNotFound     = 404
	CodeServerError  = 500
)

// CodeMsgMap 用于集中管理 code - msg
var CodeMsgMap = map[int]string{
	CodeOK:           "OK",
	CodeBadRequest:   "Bad Request",
	CodeUnauthorized: "Unauthorized",
	CodeForbidden:    "Forbidden",
	CodeNotFound:     "Not Found",
	CodeServerError:  "Internal Server Error",
}

package response

type Resp struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

// New 构造函数（保证 data 不为 null）
func New(code int, msg string, data interface{}) Resp {
	if data == nil {
		data = struct{}{}
	}
	return Resp{Code: code, Msg: msg, Data: data}
}

// OK 成功响应
func OK(data interface{}) Resp {
	return New(CodeOK, CodeMsgMap[CodeOK], data)
}

// Error 失败响应（可以传自定义 msg 覆盖默认）
func Error(code int, customMsg string) Resp {
	msg := CodeMsgMap[code]
	if customMsg != "" {
		msg = customMsg
	}
	return New(code, msg, struct{}{})
}

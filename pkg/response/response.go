package response

import "net/http"

type Response struct {
	Data    any    `json:"data"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}

func Ok(data any) Response {
	return Response{Code: http.StatusOK, Message: "success", Data: data}
}

func Created(data any) Response {
	return Response{Code: http.StatusCreated, Message: "success", Data: data}
}

func BadRequest(msg string) Response {
	return Response{Code: http.StatusBadRequest, Message: msg}
}

func Unauthorized(msg string) Response {
	return Response{Code: http.StatusUnauthorized, Message: msg}
}

func Forbidden(msg string) Response {
	return Response{Code: http.StatusForbidden, Message: msg}
}

func NotFound(msg string) Response {
	return Response{Code: http.StatusNotFound, Message: msg}
}

func Internal() Response {
	return Response{Code: http.StatusInternalServerError, Message: "internal error"}
}

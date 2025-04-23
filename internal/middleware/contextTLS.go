package middleware

import (
	"context"
	"net/http"
)

type HasTLSFlagType string

const HasTLSFlag HasTLSFlagType = "tls"


func SetTLSFlag(r *http.Request, flag bool) *http.Request {
	ctx := context.WithValue(r.Context(), HasTLSFlag, flag)
	return r.WithContext(ctx)
}

func GetTLSFlag(r *http.Request) bool {
	return r.Context().Value(HasTLSFlag).(bool)
}

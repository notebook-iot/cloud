package dashboard

import (
	"net/http"

	"github.com/notebook-iot/cloud/internal/context"
)

func Dashboard(w http.ResponseWriter, r *http.Request, ctx *context.Context) error {
	w.WriteHeader(200)
	_, err := w.Write([]byte("Hello world"))

	return err
}

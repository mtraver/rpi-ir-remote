package main

import (
	"html/template"
	"net/http"

	"github.com/mtraver/gaelog"
)

type statusHandler struct {
	Template *template.Template
}

func (h statusHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	actionLogMux.Lock()
	defer actionLogMux.Unlock()

	if err := h.Template.ExecuteTemplate(w, "status", actionLog); err != nil {
		gaelog.Errorf(ctx, "Could not execute template: %v", err)
	}
}

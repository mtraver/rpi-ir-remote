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
	lg, err := gaelog.New(r)
	if err != nil {
		lg.Errorf("%v", err)
	}
	defer lg.Close()

	actionLogMux.Lock()
	defer actionLogMux.Unlock()

	if err := h.Template.ExecuteTemplate(w, "status", actionLog); err != nil {
		lg.Errorf("Could not execute template: %v", err)
	}
}

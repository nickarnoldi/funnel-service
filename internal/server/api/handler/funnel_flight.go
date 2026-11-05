package handler

import (
	"embed"
	"fmt"
	"log"
	"net/http"
	"sync"
	"text/template"

	"github.com/nickarnoldi/funnel-service/internal/env"
	"github.com/tdewolff/minify"
	"github.com/tdewolff/minify/js"
)

type CachedTemplate struct {
	*sync.Mutex
	template *template.Template
}

var cachedTemplate *CachedTemplate = &CachedTemplate{
	Mutex: &sync.Mutex{},
}

//go:embed static/funnel-flight.js
var embeddedJS embed.FS

func (handler *Handler) HandleGetFunnelFlightJS(w http.ResponseWriter, r *http.Request) {
	pageTemplate := cachedTemplate.template

	if pageTemplate == nil {
		fileContents, err := createMinifiedContentsFromTemplate()
		if err != nil {
			sendInternalError(w, "Unable to read cached JavaScript file")
			return
		}

		pageTemplate, err = template.New("funnel-flight.min.js").Parse(string(fileContents))
		if err != nil {
			sendInternalError(w, "Unable to parse JavaScript template")
			return
		}

		if cachedTemplate.TryLock() {
			log.Println("Caching funnel-flight.min.js template")
			cachedTemplate.template = pageTemplate
			cachedTemplate.Unlock()
		}
	}

	endpoint := env.Get("ABANDONED_LEAD_URL", "")
	if endpoint == "" {
		if env.Get("ENVIRONMENT", "local") != "production" {
			endpoint = fmt.Sprintf("http://%s/abandoned", r.Host)
		} else {
			endpoint = "https://tracking.example.com/abandoned"
		}
	}

	templateData := struct {
		EndpointURL string
	}{
		EndpointURL: endpoint,
	}

	w.Header().Set("Content-Type", "application/javascript")

	if err := pageTemplate.Execute(w, templateData); err != nil {
		log.Printf("Error executing JavaScript template: %v", err)
		sendInternalError(w, "Unable to execute JavaScript template")
	}
}

func createMinifiedContentsFromTemplate() ([]byte, error) {
	jsContent, err := embeddedJS.ReadFile("static/funnel-flight.js")
	if err != nil {
		return nil, fmt.Errorf("unable to read JavaScript file: %w", err)
	}

	m := minify.New()
	m.AddFunc("application/javascript", js.Minify)

	minifiedFileContents, err := m.Bytes("application/javascript", jsContent)
	if err != nil {
		return nil, fmt.Errorf("unable to minify JavaScript file: %w", err)
	}

	return minifiedFileContents, nil
}

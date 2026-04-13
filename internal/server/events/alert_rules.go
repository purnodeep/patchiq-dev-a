package events

import (
	"bytes"
	"strings"
	"text/template"
)

// isAlertEventType returns true if the event type starts with "alert." or "alert_rule."
// These events are produced by the alert system itself and must be skipped
// by the AlertSubscriber to prevent infinite loops.
func isAlertEventType(eventType string) bool {
	return strings.HasPrefix(eventType, "alert.") || strings.HasPrefix(eventType, "alert_rule.")
}

// renderTemplate renders a Go text/template with the given data map.
// Missing keys render as empty string. If the template is invalid,
// returns the fallback string.
func renderTemplate(tmpl string, data map[string]any, fallback string) string {
	t, err := template.New("").Option("missingkey=error").Parse(tmpl)
	if err != nil {
		if fallback != "" {
			return fallback
		}
		return tmpl
	}

	if data == nil {
		data = map[string]any{}
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		// Template had missing keys. Re-render with missingkey=zero for graceful output,
		// but replace "<no value>" artifacts with empty string.
		buf.Reset()
		t2, _ := template.New("").Parse(tmpl)
		_ = t2.Execute(&buf, data)
		result := strings.ReplaceAll(buf.String(), "<no value>", "")
		if result != "" {
			return result
		}
		if fallback != "" {
			return fallback
		}
		return tmpl
	}
	return buf.String()
}

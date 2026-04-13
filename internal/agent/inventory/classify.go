package inventory

import "strings"

// sectionToCategory maps dpkg sections to user-friendly categories.
var sectionToCategory = map[string]string{
	// Application
	"gnome":    "Application",
	"kde":      "Application",
	"x11":      "Application",
	"sound":    "Application",
	"video":    "Application",
	"games":    "Application",
	"graphics": "Application",
	"web":      "Application",
	"mail":     "Application",
	"editors":  "Application",
	"shells":   "Application",
	"news":     "Application",
	"zope":     "Application",
	"comm":     "Application",

	// System
	"admin":        "System",
	"utils":        "System",
	"base":         "System",
	"misc":         "System",
	"embedded":     "System",
	"metapackages": "System",
	"tasks":        "System",

	// Library
	"libs":    "Library",
	"oldlibs": "Library",

	// Development
	"devel":    "Development",
	"libdevel": "Development",
	"debug":    "Development",
	"vcs":      "Development",
	"doc":      "Development",

	// Language Runtime
	"python":       "Language Runtime",
	"perl":         "Language Runtime",
	"ruby":         "Language Runtime",
	"php":          "Language Runtime",
	"java":         "Language Runtime",
	"interpreters": "Language Runtime",
	"lisp":         "Language Runtime",
	"ocaml":        "Language Runtime",
	"haskell":      "Language Runtime",
	"rust":         "Language Runtime",
	"golang":       "Language Runtime",

	// Network
	"net":   "Network",
	"httpd": "Network",

	// Kernel
	"kernel": "Kernel",

	// Font
	"fonts": "Font",

	// Security
	"restricted": "Security",
}

// knownApplications overrides section-based classification for well-known
// end-user applications that dpkg miscategorizes (e.g. VS Code → devel).
var knownApplications = map[string]bool{
	"code":                   true, // VS Code
	"google-chrome-stable":   true,
	"google-chrome-beta":     true,
	"google-chrome-unstable": true,
	"chromium-browser":       true,
	"microsoft-edge-stable":  true,
	"brave-browser":          true,
	"slack-desktop":          true,
	"discord":                true,
	"spotify-client":         true,
	"zoom":                   true,
	"teams":                  true,
	"signal-desktop":         true,
	"telegram-desktop":       true,
	"1password":              true,
	"docker-ce":              true,
	"docker-ce-cli":          true,
	"containerd.io":          true,
	"git":                    true,
	"gh":                     true,
	"nodejs":                 true,
	"postgresql-16":          true,
	"mysql-server":           true,
	"nginx":                  true,
	"apache2":                true,
	"redis-server":           true,
	"obs-studio":             true,
	"gimp":                   true,
	"inkscape":               true,
	"blender":                true,
	"vlc":                    true,
	"audacity":               true,
	"steam":                  true,
	"lutris":                 true,
	"virtualbox":             true,
	"vagrant":                true,
	"ansible":                true,
	"terraform":              true,
	"kubectl":                true,
	"helm":                   true,
}

// ClassifyPackage maps a dpkg section to a user-friendly category.
// It uses both the section string and the package name for accuracy.
func ClassifyPackage(section, name string) string {
	section = strings.TrimSpace(section)
	name = strings.TrimSpace(name)

	if section == "" && name == "" {
		return "Other"
	}

	// Override for well-known applications regardless of dpkg section.
	if knownApplications[name] {
		return "Application"
	}

	// Handle snap packages (no section, just name).
	if section == "" || section == "snap" {
		return classifySnap(name)
	}

	// Check for "security" anywhere in the section.
	if strings.Contains(strings.ToLower(section), "security") {
		return "Security"
	}

	// Handle multiverse/*, universe/* — strip prefix, then classify suffix.
	lower := strings.ToLower(section)
	if strings.HasPrefix(lower, "multiverse/") || strings.HasPrefix(lower, "universe/") {
		_, suffix, _ := strings.Cut(lower, "/")
		if suffix != "" {
			if cat, ok := sectionToCategory[suffix]; ok {
				return cat
			}
			return "Multimedia"
		}
		return "Multimedia"
	}

	// Direct section lookup.
	if cat, ok := sectionToCategory[lower]; ok {
		return cat
	}

	return "Other"
}

// classifySnap classifies a snap package by name heuristics.
func classifySnap(name string) string {
	lower := strings.ToLower(name)

	switch {
	case lower == "firefox" || lower == "chromium" || lower == "thunderbird":
		return "Application"
	case lower == "snapd" || lower == "bare" || strings.HasPrefix(lower, "core"):
		return "System"
	case strings.HasPrefix(lower, "gnome-") || strings.HasPrefix(lower, "gtk-"):
		return "Library"
	default:
		return "Application"
	}
}

package inventory

import (
	"testing"
)

func TestClassifyPackage(t *testing.T) {
	tests := []struct {
		name    string
		section string
		pkg     string
		want    string
	}{
		// Empty inputs
		{name: "both empty", section: "", pkg: "", want: "Other"},

		// Known application overrides
		{name: "known app: code (VS Code)", section: "devel", pkg: "code", want: "Application"},
		{name: "known app: docker-ce", section: "admin", pkg: "docker-ce", want: "Application"},
		{name: "known app: docker-ce-cli", section: "admin", pkg: "docker-ce-cli", want: "Application"},
		{name: "known app: google-chrome-stable", section: "web", pkg: "google-chrome-stable", want: "Application"},
		{name: "known app: git (overrides devel)", section: "devel", pkg: "git", want: "Application"},
		{name: "known app: nginx", section: "httpd", pkg: "nginx", want: "Application"},
		{name: "known app: kubectl", section: "", pkg: "kubectl", want: "Application"},
		{name: "known app: terraform", section: "net", pkg: "terraform", want: "Application"},
		{name: "known app: vlc", section: "video", pkg: "vlc", want: "Application"},
		{name: "known app: slack-desktop", section: "net", pkg: "slack-desktop", want: "Application"},

		// Section-based: Application
		{name: "section gnome", section: "gnome", pkg: "gnome-shell", want: "Application"},
		{name: "section kde", section: "kde", pkg: "plasma-desktop", want: "Application"},
		{name: "section x11", section: "x11", pkg: "xorg", want: "Application"},
		{name: "section sound", section: "sound", pkg: "alsa-utils", want: "Application"},
		{name: "section video", section: "video", pkg: "ffmpeg", want: "Application"},
		{name: "section games", section: "games", pkg: "0ad", want: "Application"},
		{name: "section graphics", section: "graphics", pkg: "imagemagick", want: "Application"},
		{name: "section web", section: "web", pkg: "curl", want: "Application"},
		{name: "section mail", section: "mail", pkg: "postfix", want: "Application"},
		{name: "section editors", section: "editors", pkg: "vim", want: "Application"},
		{name: "section shells", section: "shells", pkg: "bash", want: "Application"},

		// Section-based: System
		{name: "section admin", section: "admin", pkg: "sudo", want: "System"},
		{name: "section utils", section: "utils", pkg: "coreutils", want: "System"},
		{name: "section base", section: "base", pkg: "base-files", want: "System"},
		{name: "section misc", section: "misc", pkg: "some-pkg", want: "System"},
		{name: "section metapackages", section: "metapackages", pkg: "ubuntu-minimal", want: "System"},
		{name: "section tasks", section: "tasks", pkg: "tasksel", want: "System"},

		// Section-based: Library
		{name: "section libs", section: "libs", pkg: "libc6", want: "Library"},
		{name: "section oldlibs", section: "oldlibs", pkg: "libfoo-old", want: "Library"},

		// Section-based: Development
		{name: "section devel", section: "devel", pkg: "gcc", want: "Development"},
		{name: "section libdevel", section: "libdevel", pkg: "libc6-dev", want: "Development"},
		{name: "section debug", section: "debug", pkg: "libc6-dbg", want: "Development"},
		{name: "section vcs", section: "vcs", pkg: "subversion", want: "Development"},
		{name: "section doc", section: "doc", pkg: "manpages", want: "Development"},

		// Section-based: Language Runtime
		{name: "section python", section: "python", pkg: "python3", want: "Language Runtime"},
		{name: "section perl", section: "perl", pkg: "perl", want: "Language Runtime"},
		{name: "section ruby", section: "ruby", pkg: "ruby", want: "Language Runtime"},
		{name: "section golang", section: "golang", pkg: "golang-go", want: "Language Runtime"},
		{name: "section rust", section: "rust", pkg: "rustc", want: "Language Runtime"},
		{name: "section java", section: "java", pkg: "default-jdk", want: "Language Runtime"},
		{name: "section interpreters", section: "interpreters", pkg: "lua5.3", want: "Language Runtime"},

		// Section-based: Network
		{name: "section net", section: "net", pkg: "openssh-client", want: "Network"},
		{name: "section httpd", section: "httpd", pkg: "lighttpd", want: "Network"},

		// Section-based: Kernel
		{name: "section kernel", section: "kernel", pkg: "linux-image-generic", want: "Kernel"},

		// Section-based: Font
		{name: "section fonts", section: "fonts", pkg: "fonts-dejavu", want: "Font"},

		// Section-based: Security
		{name: "section restricted", section: "restricted", pkg: "pkg", want: "Security"},

		// Security keyword in section string
		{name: "security in section string", section: "net/security", pkg: "pkg", want: "Security"},
		{name: "security substring in section", section: "security-updates", pkg: "pkg", want: "Security"},
		{name: "section admin/security substring", section: "admin-security", pkg: "pkg", want: "Security"},

		// multiverse/* prefix stripping
		{name: "multiverse/libs", section: "multiverse/libs", pkg: "lib-nonfree", want: "Library"},
		{name: "multiverse/devel", section: "multiverse/devel", pkg: "build-tool", want: "Development"},
		{name: "multiverse/net", section: "multiverse/net", pkg: "net-tool", want: "Network"},
		{name: "multiverse/unknown-suffix", section: "multiverse/widgets", pkg: "widget-pkg", want: "Multimedia"},
		{name: "multiverse only (no suffix)", section: "multiverse/", pkg: "pkg", want: "Multimedia"},

		// universe/* prefix stripping
		{name: "universe/admin", section: "universe/admin", pkg: "tool", want: "System"},
		{name: "universe/python", section: "universe/python", pkg: "python3-extra", want: "Language Runtime"},
		{name: "universe/unknown-suffix", section: "universe/foo", pkg: "pkg", want: "Multimedia"},

		// Unknown section falls back to Other
		{name: "unknown section", section: "unknownsection", pkg: "random-pkg", want: "Other"},
		{name: "blank section unknown pkg", section: "   ", pkg: "some-unknown-snap", want: "Application"},

		// Whitespace trimming
		{name: "section with leading space", section: "  libs  ", pkg: "libc6", want: "Library"},
		{name: "pkg with trailing space (known)", section: "devel", pkg: "  code  ", want: "Application"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ClassifyPackage(tc.section, tc.pkg)
			if got != tc.want {
				t.Errorf("ClassifyPackage(%q, %q) = %q, want %q", tc.section, tc.pkg, got, tc.want)
			}
		})
	}
}

func TestClassifySnap(t *testing.T) {
	tests := []struct {
		name string
		snap string
		want string
	}{
		// System snaps
		{name: "snapd", snap: "snapd", want: "System"},
		{name: "bare", snap: "bare", want: "System"},
		{name: "core prefix: core", snap: "core", want: "System"},
		{name: "core prefix: core18", snap: "core18", want: "System"},
		{name: "core prefix: core20", snap: "core20", want: "System"},
		{name: "core prefix: core22", snap: "core22", want: "System"},
		{name: "core prefix: core24", snap: "core24", want: "System"},

		// Library snaps
		{name: "gnome- prefix", snap: "gnome-42-2204", want: "Library"},
		{name: "gtk- prefix", snap: "gtk-common-themes", want: "Library"},

		// Known application snaps
		{name: "firefox", snap: "firefox", want: "Application"},
		{name: "chromium", snap: "chromium", want: "Application"},
		{name: "thunderbird", snap: "thunderbird", want: "Application"},

		// Default: application
		{name: "unknown snap defaults to Application", snap: "some-app-snap", want: "Application"},
		{name: "lxd snap", snap: "lxd", want: "Application"},
		{name: "vlc snap", snap: "vlc", want: "Application"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := classifySnap(tc.snap)
			if got != tc.want {
				t.Errorf("classifySnap(%q) = %q, want %q", tc.snap, got, tc.want)
			}
		})
	}
}

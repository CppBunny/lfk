package images

import "testing"

func TestResolve_OverrideWinsOverDefault(t *testing.T) {
	if got := Resolve("", "my-corp/toolbox:3", "busybox"); got != "my-corp/toolbox:3" {
		t.Errorf("override ignored: got %q", got)
	}
}

func TestResolve_BlankOverrideFallsBackToDefault(t *testing.T) {
	for _, override := range []string{"", "   ", "\t"} {
		if got := Resolve("", override, "busybox"); got != "busybox" {
			t.Errorf("Resolve(%q) = %q, want the default", override, got)
		}
	}
}

func TestResolve_RegistryPrefix(t *testing.T) {
	tests := []struct {
		name     string
		registry string
		override string
		def      string
		want     string
	}{
		{"bare name gets prefixed", "registry.internal:5000", "", "busybox", "registry.internal:5000/busybox"},
		{"org/name gets prefixed", "registry.internal:5000", "", "nicolaka/netshoot:v0.13", "registry.internal:5000/nicolaka/netshoot:v0.13"},
		{"override gets prefixed too", "mirror.example.com", "toolbox:3", "busybox", "mirror.example.com/toolbox:3"},
		{"digest pin keeps digest", "mirror.example.com", "", "nicolaka/netshoot@sha256:abc", "mirror.example.com/nicolaka/netshoot@sha256:abc"},
		{"trailing slash on registry is not doubled", "mirror.example.com/", "", "busybox", "mirror.example.com/busybox"},
		{"registry with path prefix", "mirror.example.com/proxy/dockerhub", "", "busybox", "mirror.example.com/proxy/dockerhub/busybox"},
		{"blank registry leaves image alone", "", "", "busybox", "busybox"},
		{"whitespace registry leaves image alone", "   ", "", "busybox", "busybox"},

		// Already-qualified images must NOT be double-prefixed: the whole
		// point of a per-image override is to name an image the mirror
		// prefix would not reach.
		{"dotted host is left alone", "mirror.example.com", "my-corp.jfrog.io/toolbox:3", "busybox", "my-corp.jfrog.io/toolbox:3"},
		{"host:port is left alone", "mirror.example.com", "other.reg:5000/toolbox:3", "busybox", "other.reg:5000/toolbox:3"},
		{"localhost is left alone", "mirror.example.com", "localhost:5000/toolbox", "busybox", "localhost:5000/toolbox"},
		{"bare localhost is left alone", "mirror.example.com", "localhost/toolbox", "busybox", "localhost/toolbox"},
		{"default already qualified is left alone", "mirror.example.com", "", "ghcr.io/x/y:1", "ghcr.io/x/y:1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Resolve(tt.registry, tt.override, tt.def); got != tt.want {
				t.Errorf("Resolve(%q, %q, %q) = %q, want %q", tt.registry, tt.override, tt.def, got, tt.want)
			}
		})
	}
}

// A tag's colon must never be mistaken for a host:port. "busybox:1.36" has a
// dot AND a colon but no slash, so it has no registry host at all.
func TestResolve_TagIsNotMistakenForRegistryHost(t *testing.T) {
	if got := Resolve("mirror.example.com", "", "busybox:1.36"); got != "mirror.example.com/busybox:1.36" {
		t.Errorf("tag read as a registry host: got %q", got)
	}
	if got := Resolve("mirror.example.com", "", "busybox@sha256:abc"); got != "mirror.example.com/busybox@sha256:abc" {
		t.Errorf("digest read as a registry host: got %q", got)
	}
}

func TestResolve_TrimsWhitespaceAroundOverride(t *testing.T) {
	if got := Resolve("", "  toolbox:3  ", "busybox"); got != "toolbox:3" {
		t.Errorf("override not trimmed: got %q", got)
	}
}

func TestHasRegistryHost(t *testing.T) {
	qualified := []string{
		"registry.internal:5000/busybox",
		"ghcr.io/x/y",
		"localhost/x",
		"localhost:5000/x",
		"my-corp.jfrog.io/a/b/c:1",
	}
	for _, img := range qualified {
		if !hasRegistryHost(img) {
			t.Errorf("hasRegistryHost(%q) = false, want true", img)
		}
	}
	bare := []string{
		"busybox",
		"busybox:1.36",
		"busybox@sha256:abc",
		"nicolaka/netshoot:v0.13",
		"library/busybox",
		"a/b/c",
	}
	for _, img := range bare {
		if hasRegistryHost(img) {
			t.Errorf("hasRegistryHost(%q) = true, want false", img)
		}
	}
}

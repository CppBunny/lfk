package images

import (
	"strings"
	"testing"
)

// snapshot restores every Config* global so a test's overrides don't leak
// into the next one (the globals are process-wide by design).
func snapshot(t *testing.T) {
	t.Helper()
	reg, dbg, pod, mnt, node, tcap := ConfigRegistry, ConfigDebug, ConfigDebugPod, ConfigDebugMount, ConfigNodeShell, ConfigTrafficCapture
	t.Cleanup(func() {
		ConfigRegistry, ConfigDebug, ConfigDebugPod, ConfigDebugMount, ConfigNodeShell, ConfigTrafficCapture = reg, dbg, pod, mnt, node, tcap
	})
}

func TestAccessors_DefaultsWhenUnset(t *testing.T) {
	snapshot(t)
	ConfigRegistry, ConfigDebug, ConfigDebugPod = "", "", ""
	ConfigDebugMount, ConfigNodeShell, ConfigTrafficCapture = "", "", ""

	for _, tt := range []struct {
		name string
		got  string
		want string
	}{
		{"Debug", Debug(), DefaultDebug},
		{"DebugPod", DebugPod(), DefaultDebugPod},
		{"DebugMount", DebugMount(), DefaultDebugMount},
		{"NodeShell", NodeShell(), DefaultNodeShell},
		{"TrafficCapture", TrafficCapture(), DefaultTrafficCapture},
	} {
		if tt.got != tt.want {
			t.Errorf("%s() = %q, want %q", tt.name, tt.got, tt.want)
		}
	}
}

// Each accessor must read its OWN override — a copy-paste slip that wires
// two accessors to the same global is exactly what this catches.
func TestAccessors_EachReadsItsOwnOverride(t *testing.T) {
	snapshot(t)
	ConfigRegistry = ""
	ConfigDebug, ConfigDebugPod, ConfigDebugMount = "img-debug", "img-pod", "img-mount"
	ConfigNodeShell, ConfigTrafficCapture = "img-node", "img-cap"

	for _, tt := range []struct {
		name string
		got  string
		want string
	}{
		{"Debug", Debug(), "img-debug"},
		{"DebugPod", DebugPod(), "img-pod"},
		{"DebugMount", DebugMount(), "img-mount"},
		{"NodeShell", NodeShell(), "img-node"},
		{"TrafficCapture", TrafficCapture(), "img-cap"},
	} {
		if tt.got != tt.want {
			t.Errorf("%s() = %q, want %q", tt.name, tt.got, tt.want)
		}
	}
}

func TestAccessors_RegistryAppliesToEveryDefault(t *testing.T) {
	snapshot(t)
	ConfigRegistry = "mirror.example.com"
	ConfigDebug, ConfigDebugPod, ConfigDebugMount = "", "", ""
	ConfigNodeShell, ConfigTrafficCapture = "", ""

	for _, tt := range []struct {
		name string
		got  string
		want string
	}{
		{"Debug", Debug(), "mirror.example.com/" + DefaultDebug},
		{"DebugPod", DebugPod(), "mirror.example.com/" + DefaultDebugPod},
		{"DebugMount", DebugMount(), "mirror.example.com/" + DefaultDebugMount},
		{"NodeShell", NodeShell(), "mirror.example.com/" + DefaultNodeShell},
		{"TrafficCapture", TrafficCapture(), "mirror.example.com/" + DefaultTrafficCapture},
	} {
		if tt.got != tt.want {
			t.Errorf("%s() = %q, want %q", tt.name, tt.got, tt.want)
		}
	}
}

// The shipped defaults must stay tag- or digest-pinned rather than
// resolving to an implicit ":latest". Guards the same property the traffic
// capture backend test used to guard alone.
func TestDefaults_ArePinned(t *testing.T) {
	// A bare repo name resolves to an implicit ":latest", which silently
	// drifts under the capture container's NET_ADMIN/NET_RAW privileges.
	img := DefaultTrafficCapture
	repo := img
	if i := strings.LastIndex(img, "/"); i >= 0 {
		repo = img[i+1:]
	}
	if !strings.ContainsAny(repo, ":@") {
		t.Errorf("DefaultTrafficCapture (%q) must carry an explicit tag or digest", img)
	}
}

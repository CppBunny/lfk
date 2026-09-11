// Package images centralizes the container images lfk spawns for its own
// helper workloads (debug containers, debug pods, node shells, traffic
// capture) and the optional registry prefix applied to them.
//
// It exists as a standalone package because both internal/app (which builds
// the kubectl argv for the debug actions) and internal/k8s (which builds the
// traffic-capture argv) need the resolved values, and internal/k8s must not
// import internal/ui. internal/ui owns the config parsing and writes the
// Config* globals here at startup, mirroring how it drives
// scheduler.Config* and model.Config*.
package images

// Default* are the images lfk uses when the config sets no override. Each
// carries a hard requirement from the feature that runs it:
//
//   - DefaultNodeShell must ship nsenter; the node shell enters the host
//     namespaces with `nsenter --target 1`.
//   - DefaultTrafficCapture must ship tcpdump, and runs with
//     NET_ADMIN/NET_RAW in the target pod's network namespace. Changing it
//     needs a security review.
//   - The remaining three only need a POSIX shell at /bin/sh.
//
// DefaultTrafficCapture is pinned by tag rather than digest to keep the
// build self-contained. For hardened deployments set
// images.traffic_capture to a digest pin
// ("nicolaka/netshoot@sha256:...") so registry tampering is detected.
const (
	DefaultDebug          = "busybox"
	DefaultDebugPod       = "alpine"
	DefaultDebugMount     = "alpine"
	DefaultNodeShell      = "busybox"
	DefaultTrafficCapture = "nicolaka/netshoot:v0.13"
)

// Config* hold the raw values from the config file's images section. Empty
// means "unset" and falls back to the matching Default. They are written
// once at startup by ui.applyImagesConfig, before any tea.Cmd goroutine
// starts, and read-only thereafter — same lifecycle as the other Config*
// globals this package's callers already read.
var (
	// ConfigRegistry is prefixed onto any helper image that does not
	// already name a registry host. See Resolve for the exact rule.
	ConfigRegistry       string
	ConfigDebug          string
	ConfigDebugPod       string
	ConfigDebugMount     string
	ConfigNodeShell      string
	ConfigTrafficCapture string
)

// Debug is the image for the Debug action's ephemeral container.
func Debug() string { return Resolve(ConfigRegistry, ConfigDebug, DefaultDebug) }

// DebugPod is the image for the standalone Debug Pod action.
func DebugPod() string { return Resolve(ConfigRegistry, ConfigDebugPod, DefaultDebugPod) }

// DebugMount is the image for the Debug Mount action's PVC-mounting pod.
func DebugMount() string { return Resolve(ConfigRegistry, ConfigDebugMount, DefaultDebugMount) }

// NodeShell is the image for the Node Shell action's privileged pod. Must
// ship nsenter.
func NodeShell() string { return Resolve(ConfigRegistry, ConfigNodeShell, DefaultNodeShell) }

// TrafficCapture is the image for the Traffic Capture overlay's
// kubectl-debug backend. Must ship tcpdump.
func TrafficCapture() string {
	return Resolve(ConfigRegistry, ConfigTrafficCapture, DefaultTrafficCapture)
}

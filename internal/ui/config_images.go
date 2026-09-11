package ui

import (
	"strings"

	"github.com/janosmiko/lfk/internal/images"
)

// ImagesConfig is the on-disk schema for the images section. Every field
// is optional; an empty value keeps the matching images.Default*.
//
// Registry is prefixed onto any image below that does not already name a
// registry host, so a mirror only has to be configured once. A
// fully-qualified per-image override (one whose first path component
// carries a "." or ":", or is "localhost") is used verbatim instead.
type ImagesConfig struct {
	// Registry is the registry (optionally with a path prefix) that
	// hosts the helper images, e.g. "registry.internal:5000" or
	// "mirror.example.com/proxy/dockerhub". Empty pulls from the image's
	// own default registry.
	Registry string `json:"registry" yaml:"registry"`
	// Debug is the image for the Debug action's ephemeral container.
	Debug string `json:"debug" yaml:"debug"`
	// DebugPod is the image for the standalone Debug Pod action.
	DebugPod string `json:"debug_pod" yaml:"debug_pod"`
	// DebugMount is the image for the Debug Mount action's pod, which
	// mounts the selected PVC at /data.
	DebugMount string `json:"debug_mount" yaml:"debug_mount"`
	// NodeShell is the image for the Node Shell action's privileged pod.
	// Must ship nsenter — the pod enters the host namespaces with
	// `nsenter --target 1`.
	NodeShell string `json:"node_shell" yaml:"node_shell"`
	// TrafficCapture is the image for the Traffic Capture overlay's
	// kubectl-debug backend. Must ship tcpdump. Runs with
	// NET_ADMIN/NET_RAW in the target pod's network namespace, so
	// changing it warrants a security review.
	TrafficCapture string `json:"traffic_capture" yaml:"traffic_capture"`
}

// applyImagesConfig wires the images section into the internal/images
// globals that internal/app and internal/k8s read when they build a
// kubectl argv.
//
// Every destination is reset before the section is read, so a reload that
// drops a key returns to the compiled default instead of keeping a stale
// override — the same rule applyExplorerLayout and applyGotoTargets
// follow. A nil section therefore clears all six.
//
// Values are trimmed and empties dropped so a key present but blank in
// YAML keeps the compiled default rather than producing an empty
// `--image=` flag. The reference itself is not validated: kubectl reports
// a malformed or unreachable image far better than a startup guess could.
func applyImagesConfig(cfg configFile) {
	images.ConfigRegistry = ""
	images.ConfigDebug = ""
	images.ConfigDebugPod = ""
	images.ConfigDebugMount = ""
	images.ConfigNodeShell = ""
	images.ConfigTrafficCapture = ""
	if cfg.Images == nil {
		return
	}
	for _, f := range []struct {
		value string
		dest  *string
	}{
		{cfg.Images.Registry, &images.ConfigRegistry},
		{cfg.Images.Debug, &images.ConfigDebug},
		{cfg.Images.DebugPod, &images.ConfigDebugPod},
		{cfg.Images.DebugMount, &images.ConfigDebugMount},
		{cfg.Images.NodeShell, &images.ConfigNodeShell},
		{cfg.Images.TrafficCapture, &images.ConfigTrafficCapture},
	} {
		if v := strings.TrimSpace(f.value); v != "" {
			*f.dest = v
		}
	}
}

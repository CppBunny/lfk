package ui

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"

	"github.com/janosmiko/lfk/internal/images"
)

// resetImagesGlobals clears the images globals and restores them after the
// test, so each case starts from "nothing configured".
func resetImagesGlobals(t *testing.T) {
	t.Helper()
	reg, dbg := images.ConfigRegistry, images.ConfigDebug
	pod, mnt := images.ConfigDebugPod, images.ConfigDebugMount
	node, tcap := images.ConfigNodeShell, images.ConfigTrafficCapture
	t.Cleanup(func() {
		images.ConfigRegistry, images.ConfigDebug = reg, dbg
		images.ConfigDebugPod, images.ConfigDebugMount = pod, mnt
		images.ConfigNodeShell, images.ConfigTrafficCapture = node, tcap
	})
	images.ConfigRegistry, images.ConfigDebug = "", ""
	images.ConfigDebugPod, images.ConfigDebugMount = "", ""
	images.ConfigNodeShell, images.ConfigTrafficCapture = "", ""
}

func TestApplyImagesConfig_NilSectionLeavesDefaults(t *testing.T) {
	resetImagesGlobals(t)
	applyImagesConfig(configFile{})

	assert.Empty(t, images.ConfigRegistry, "registry")
	assert.Equal(t, images.DefaultDebug, images.Debug(), "debug falls back")
	assert.Equal(t, images.DefaultTrafficCapture, images.TrafficCapture(), "traffic_capture falls back")
}

// A key present but blank must keep the default. Writing the empty value
// through would produce a bare `--image=` flag that kubectl rejects.
func TestApplyImagesConfig_BlankValuesKeepDefaults(t *testing.T) {
	resetImagesGlobals(t)
	applyImagesConfig(configFile{Images: &ImagesConfig{
		Registry: "  ", Debug: "", DebugPod: "\t", DebugMount: " ",
		NodeShell: "", TrafficCapture: "   ",
	}})

	assert.Empty(t, images.ConfigRegistry, "registry")
	assert.Equal(t, images.DefaultDebug, images.Debug(), "debug")
	assert.Equal(t, images.DefaultDebugPod, images.DebugPod(), "debug_pod")
	assert.Equal(t, images.DefaultDebugMount, images.DebugMount(), "debug_mount")
	assert.Equal(t, images.DefaultNodeShell, images.NodeShell(), "node_shell")
	assert.Equal(t, images.DefaultTrafficCapture, images.TrafficCapture(), "traffic_capture")
}

func TestApplyImagesConfig_TrimsValues(t *testing.T) {
	resetImagesGlobals(t)
	applyImagesConfig(configFile{Images: &ImagesConfig{
		Registry: "  registry.internal:5000  ",
		Debug:    "  toolbox:3  ",
	}})

	assert.Equal(t, "registry.internal:5000", images.ConfigRegistry, "registry")
	assert.Equal(t, "toolbox:3", images.ConfigDebug, "debug")
	assert.Equal(t, "registry.internal:5000/toolbox:3", images.Debug(), "resolved debug")
}

// Registry-only config is the common air-gapped case: every default gets
// retargeted without naming each image.
func TestApplyImagesConfig_RegistryOnlyRetargetsEveryDefault(t *testing.T) {
	resetImagesGlobals(t)
	applyImagesConfig(configFile{Images: &ImagesConfig{Registry: "mirror.example.com"}})

	assert.Equal(t, "mirror.example.com/"+images.DefaultDebug, images.Debug(), "debug")
	assert.Equal(t, "mirror.example.com/"+images.DefaultDebugPod, images.DebugPod(), "debug_pod")
	assert.Equal(t, "mirror.example.com/"+images.DefaultDebugMount, images.DebugMount(), "debug_mount")
	assert.Equal(t, "mirror.example.com/"+images.DefaultNodeShell, images.NodeShell(), "node_shell")
	assert.Equal(t, "mirror.example.com/"+images.DefaultTrafficCapture, images.TrafficCapture(), "traffic_capture")
}

// A per-image override naming its own host must not be re-prefixed, or the
// escape hatch for "my mirror doesn't mirror this one" would not work.
func TestApplyImagesConfig_QualifiedOverrideBeatsRegistry(t *testing.T) {
	resetImagesGlobals(t)
	applyImagesConfig(configFile{Images: &ImagesConfig{
		Registry:  "mirror.example.com",
		NodeShell: "other.registry.io/ops/toolbox:2",
		Debug:     "toolbox",
	}})

	assert.Equal(t, "other.registry.io/ops/toolbox:2", images.NodeShell(), "qualified override used verbatim")
	assert.Equal(t, "mirror.example.com/toolbox", images.Debug(), "bare override still prefixed")
}

func TestApplyImagesConfig_LoadedFromYAML(t *testing.T) {
	resetImagesGlobals(t)
	var cfg configFile
	require.NoError(t, yaml.Unmarshal([]byte(`
images:
  registry: registry.internal:5000
  traffic_capture: nicolaka/netshoot@sha256:abc
`), &cfg))
	applyImagesConfig(cfg)

	assert.Equal(t, "registry.internal:5000", images.ConfigRegistry, "registry")
	assert.Equal(t, "registry.internal:5000/nicolaka/netshoot@sha256:abc",
		images.TrafficCapture(), "digest pin prefixed and preserved")
}

// A reload that drops a key must fall back to the default rather than keep
// the previous override — the convention applyExplorerLayout established
// and commit 73688240 fixed for the explorer layout.
func TestApplyImagesConfig_ReloadDroppingKeysResetsToDefaults(t *testing.T) {
	resetImagesGlobals(t)
	applyImagesConfig(configFile{Images: &ImagesConfig{
		Registry: "mirror.example.com", Debug: "toolbox:3", NodeShell: "node:1",
	}})
	require.Equal(t, "mirror.example.com/toolbox:3", images.Debug(), "precondition")

	// Second load: the images section is gone entirely.
	applyImagesConfig(configFile{})

	assert.Empty(t, images.ConfigRegistry, "registry must not survive")
	assert.Equal(t, images.DefaultDebug, images.Debug(), "debug must return to the default")
	assert.Equal(t, images.DefaultNodeShell, images.NodeShell(), "node_shell must return to the default")
}

// A reload that drops only one key must reset that key alone.
func TestApplyImagesConfig_ReloadDroppingOneKeyKeepsOthers(t *testing.T) {
	resetImagesGlobals(t)
	applyImagesConfig(configFile{Images: &ImagesConfig{Debug: "toolbox:3", NodeShell: "node:1"}})
	require.Equal(t, "toolbox:3", images.Debug(), "precondition")

	applyImagesConfig(configFile{Images: &ImagesConfig{NodeShell: "node:1"}})

	assert.Equal(t, images.DefaultDebug, images.Debug(), "dropped key resets")
	assert.Equal(t, "node:1", images.NodeShell(), "retained key stays")
}

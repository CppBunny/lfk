package images

import "strings"

// Resolve picks the image to run and applies the registry prefix.
//
// The override wins over def when set (whitespace-only counts as unset), so
// a user who names an image outright gets exactly that repository. The
// registry prefix is then applied only when the resulting reference does
// not already name a registry host — see hasRegistryHost. That keeps the
// two knobs composable: `registry` retargets the bundled defaults at a
// mirror, while a fully-qualified per-image override reaches a repository
// the mirror prefix never would.
func Resolve(registry, override, def string) string {
	image := strings.TrimSpace(override)
	if image == "" {
		image = strings.TrimSpace(def)
	}
	reg := strings.Trim(strings.TrimSpace(registry), "/")
	if reg == "" || image == "" || hasRegistryHost(image) {
		return image
	}
	return reg + "/" + image
}

// hasRegistryHost reports whether image already names a registry host,
// applying the same rule the OCI/Docker reference grammar uses: the first
// slash-separated component is a host only if it contains a "." or a ":",
// or is exactly "localhost". Everything else is a repository path on the
// default registry.
//
// The leading-component restriction is what keeps a tag or digest from
// being read as a host:port — "busybox:1.36" and "busybox@sha256:..." have
// no slash, so there is no candidate host component at all, and
// "nicolaka/netshoot:v0.13" is tested on "nicolaka" rather than on the
// tagged tail.
func hasRegistryHost(image string) bool {
	head, _, found := strings.Cut(image, "/")
	if !found {
		return false
	}
	return head == "localhost" || strings.ContainsAny(head, ".:")
}

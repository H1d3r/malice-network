package rpc

import (
	"testing"

	"github.com/chainreactors/malice-network/server/internal/configs"
)

func TestRemoteRootOperationDeniedByDefault(t *testing.T) {
	method := "/clientrpc.RootRPC/ListListeners"

	if remoteRootOperationAllowed(method, "192.168.239.110", nil) {
		t.Fatal("remote root operation allowed with nil config, want denied")
	}

	if remoteRootOperationAllowed(method, "192.168.239.110", &configs.RootRPCConfig{}) {
		t.Fatal("remote root operation allowed with default config, want denied")
	}
}

func TestRemoteRootOperationAllowsRemoteWhenExplicitlyEnabled(t *testing.T) {
	cfg := &configs.RootRPCConfig{
		AllowRemote: true,
	}

	if !remoteRootOperationAllowed("/clientrpc.RootRPC/AddListener", "192.168.239.110", cfg) {
		t.Fatal("remote root operation denied with allow_remote enabled and no restrictions")
	}
}

func TestRemoteRootOperationHonorsAllowedCIDRs(t *testing.T) {
	cfg := &configs.RootRPCConfig{
		AllowRemote:  true,
		AllowedCIDRs: []string{"192.168.239.0/24", "10.10.10.10"},
	}

	if !remoteRootOperationAllowed("/clientrpc.RootRPC/ListListeners", "192.168.239.110", cfg) {
		t.Fatal("remote root operation denied for allowed CIDR")
	}

	if !remoteRootOperationAllowed("/clientrpc.RootRPC/ListListeners", "10.10.10.10", cfg) {
		t.Fatal("remote root operation denied for allowed exact IP")
	}

	if remoteRootOperationAllowed("/clientrpc.RootRPC/ListListeners", "172.16.1.20", cfg) {
		t.Fatal("remote root operation allowed outside configured CIDRs")
	}
}

func TestRemoteRootOperationHonorsAllowedMethods(t *testing.T) {
	cfg := &configs.RootRPCConfig{
		AllowRemote:    true,
		AllowedMethods: []string{"/clientrpc.RootRPC/ListListeners", "/clientrpc.RootRPC/AddListener"},
	}

	if !remoteRootOperationAllowed("/clientrpc.RootRPC/ListListeners", "192.168.239.110", cfg) {
		t.Fatal("configured exact root method was denied")
	}

	if !remoteRootOperationAllowed("/clientrpc.RootRPC/AddListener", "192.168.239.110", cfg) {
		t.Fatal("configured wildcard root method was denied")
	}

	if remoteRootOperationAllowed("/clientrpc.RootRPC/RemoveListener", "192.168.239.110", cfg) {
		t.Fatal("unconfigured root method was allowed")
	}
}

// --- Wildcard allowed_methods ---

func TestRemoteRootMethodWildcardAllowsAllInService(t *testing.T) {
	cfg := &configs.RootRPCConfig{
		AllowRemote:    true,
		AllowedMethods: []string{"/clientrpc.RootRPC/*"},
	}

	for _, method := range []string{
		"/clientrpc.RootRPC/ListListeners",
		"/clientrpc.RootRPC/AddListener",
		"/clientrpc.RootRPC/RemoveListener",
		"/clientrpc.RootRPC/AddClient",
	} {
		if !remoteRootOperationAllowed(method, "10.0.0.1", cfg) {
			t.Fatalf("wildcard should allow %s", method)
		}
	}

	if remoteRootOperationAllowed("/clientrpc.MaliceRPC/GetListeners", "10.0.0.1", cfg) {
		t.Fatal("RootRPC wildcard must not match MaliceRPC methods")
	}
}

// --- Empty / whitespace entries ---

func TestRemoteRootMethodIgnoresEmptyAndWhitespace(t *testing.T) {
	cfg := &configs.RootRPCConfig{
		AllowRemote:    true,
		AllowedMethods: []string{"", "  ", "/clientrpc.RootRPC/ListListeners"},
	}
	if !remoteRootOperationAllowed("/clientrpc.RootRPC/ListListeners", "10.0.0.1", cfg) {
		t.Fatal("valid method denied when list contains empty entries")
	}
	if remoteRootOperationAllowed("/clientrpc.RootRPC/AddClient", "10.0.0.1", cfg) {
		t.Fatal("unlisted method allowed despite non-empty allow list")
	}
}

func TestRemoteRootCIDRIgnoresEmptyAndWhitespace(t *testing.T) {
	cfg := &configs.RootRPCConfig{
		AllowRemote:  true,
		AllowedCIDRs: []string{"", "  ", "10.0.0.0/24"},
	}
	if !remoteRootOperationAllowed("/clientrpc.RootRPC/ListListeners", "10.0.0.5", cfg) {
		t.Fatal("valid CIDR denied when list contains empty entries")
	}
	if remoteRootOperationAllowed("/clientrpc.RootRPC/ListListeners", "172.16.0.1", cfg) {
		t.Fatal("out-of-range IP allowed despite non-empty CIDR list")
	}
}

// --- Invalid CIDR entries ---

func TestRemoteRootCIDRIgnoresInvalidEntries(t *testing.T) {
	cfg := &configs.RootRPCConfig{
		AllowRemote:  true,
		AllowedCIDRs: []string{"not-a-cidr", "also/invalid", "10.0.0.0/24"},
	}
	if !remoteRootOperationAllowed("/clientrpc.RootRPC/ListListeners", "10.0.0.5", cfg) {
		t.Fatal("valid CIDR denied when invalid entries present")
	}
	if remoteRootOperationAllowed("/clientrpc.RootRPC/ListListeners", "192.168.1.1", cfg) {
		t.Fatal("invalid CIDR entry must not cause false allow")
	}
}

// --- Empty / invalid remote IP ---

func TestRemoteRootEmptyIPDenied(t *testing.T) {
	cfg := &configs.RootRPCConfig{
		AllowRemote:  true,
		AllowedCIDRs: []string{"0.0.0.0/0"},
	}
	if remoteRootOperationAllowed("/clientrpc.RootRPC/ListListeners", "", cfg) {
		t.Fatal("empty remote IP must be denied")
	}
}

func TestRemoteRootInvalidIPDenied(t *testing.T) {
	cfg := &configs.RootRPCConfig{
		AllowRemote:  true,
		AllowedCIDRs: []string{"0.0.0.0/0"},
	}
	if remoteRootOperationAllowed("/clientrpc.RootRPC/ListListeners", "not-an-ip", cfg) {
		t.Fatal("invalid remote IP must be denied")
	}
}

// --- IPv6 ---

func TestRemoteRootIPv6ExactMatch(t *testing.T) {
	cfg := &configs.RootRPCConfig{
		AllowRemote:  true,
		AllowedCIDRs: []string{"fd00::1"},
	}
	if !remoteRootOperationAllowed("/clientrpc.RootRPC/ListListeners", "fd00::1", cfg) {
		t.Fatal("IPv6 exact match denied")
	}
	if remoteRootOperationAllowed("/clientrpc.RootRPC/ListListeners", "fd00::2", cfg) {
		t.Fatal("different IPv6 address allowed by exact match")
	}
}

func TestRemoteRootIPv6CIDR(t *testing.T) {
	cfg := &configs.RootRPCConfig{
		AllowRemote:  true,
		AllowedCIDRs: []string{"fd00::/64"},
	}
	if !remoteRootOperationAllowed("/clientrpc.RootRPC/ListListeners", "fd00::abcd", cfg) {
		t.Fatal("IPv6 within /64 denied")
	}
	if remoteRootOperationAllowed("/clientrpc.RootRPC/ListListeners", "fd01::1", cfg) {
		t.Fatal("IPv6 outside /64 allowed")
	}
}

// --- enforceRootRemoteGate unit tests ---

func TestEnforceRootRemoteGatePassesNonRootMethods(t *testing.T) {
	identity := &PeerIdentity{RemoteIP: "192.168.1.1", IsLoopback: false}
	if err := enforceRootRemoteGate("/clientrpc.MaliceRPC/GetSessions", identity); err != nil {
		t.Fatalf("non-root method should pass, got: %v", err)
	}
}

func TestEnforceRootRemoteGatePassesLoopback(t *testing.T) {
	identity := &PeerIdentity{RemoteIP: "127.0.0.1", IsLoopback: true}
	if err := enforceRootRemoteGate("/clientrpc.RootRPC/ListListeners", identity); err != nil {
		t.Fatalf("loopback root method should pass, got: %v", err)
	}
}

func TestEnforceRootRemoteGateDeniesRemoteRootByDefault(t *testing.T) {
	identity := &PeerIdentity{RemoteIP: "10.0.0.5", IsLoopback: false}
	err := enforceRootRemoteGate("/clientrpc.RootRPC/ListListeners", identity)
	if err == nil {
		t.Fatal("remote root operation should be denied without config")
	}
}

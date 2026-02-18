package handlers

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/crucial707/hci-asset/internal/models"
	"github.com/crucial707/hci-asset/internal/repo"
)

const (
	graphMaxAssets = 2000
	untaggedGroup  = "Untagged"
)

// NetworkHandler serves network topology / graph data.
type NetworkHandler struct {
	Repo *repo.AssetRepo
}

// NetworkGraphResponse is the JSON shape for GET /v1/network/graph.
type NetworkGraphResponse struct {
	Nodes  []NetworkNode  `json:"nodes"`
	Groups []NetworkGroup `json:"groups"`
}

// NetworkNode is a single node in the graph (one asset).
type NetworkNode struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Group string `json:"group"`
	Title string `json:"title,omitempty"`
	// AssetID for linking to asset detail
	AssetID int `json:"asset_id,omitempty"`
}

// NetworkGroup represents a segment (tag or subnet) for styling.
type NetworkGroup struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// subnetForIP returns a /24 subnet string for IPv4 (e.g. "192.168.1.0/24") or "/64" style for IPv6.
// Returns empty string if ip is not parseable.
func subnetForIP(ipStr string) string {
	ip := net.ParseIP(strings.TrimSpace(ipStr))
	if ip == nil {
		return ""
	}
	if ip4 := ip.To4(); ip4 != nil {
		// IPv4: mask to /24
		ip4[3] = 0
		return ip4.String() + "/24"
	}
	// IPv6: zero last 64 bits for /64 prefix
	if ip6 := ip.To16(); ip6 != nil {
		for i := 8; i < 16; i++ {
			ip6[i] = 0
		}
		return ip.String() + "/64"
	}
	return ""
}

// NetworkGraph returns all assets as graph nodes grouped by subnet (when network_name is set) or first tag (or "Untagged").
// Used by the network visualization UI to show segmentation.
func (h *NetworkHandler) NetworkGraph(w http.ResponseWriter, r *http.Request) {
	assets, err := h.Repo.List(r.Context(), graphMaxAssets, 0)
	if err != nil {
		log.Printf("NetworkGraph list assets: %v", err)
		JSONError(w, ErrMessageInternal, http.StatusInternalServerError)
		return
	}

	groupSet := make(map[string]struct{})
	groupSet[untaggedGroup] = struct{}{}
	var nodes []NetworkNode
	for _, a := range assets {
		group := untaggedGroup
		if subnet := subnetForIP(a.NetworkName); subnet != "" {
			group = subnet
			groupSet[group] = struct{}{}
		} else if len(a.Tags) > 0 {
			group = a.Tags[0]
			groupSet[group] = struct{}{}
		}
		label := a.Name
		if label == "" {
			label = "Asset " + strconv.Itoa(a.ID)
		}
		title := a.Description
		if a.NetworkName != "" {
			if title != "" {
				title += " · "
			}
			title += a.NetworkName
		}
		nodes = append(nodes, NetworkNode{
			ID:     nodeID(a),
			Label:  label,
			Group:  group,
			Title:  title,
			AssetID: a.ID,
		})
	}

	var groups []NetworkGroup
	for g := range groupSet {
		groups = append(groups, NetworkGroup{ID: g, Label: g})
	}
	// Ensure stable order: Untagged first, then alphabetical
	sortNetworkGroups(groups)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(NetworkGraphResponse{Nodes: nodes, Groups: groups})
}

func nodeID(a models.Asset) string {
	return "asset-" + strconv.Itoa(a.ID)
}

func sortNetworkGroups(groups []NetworkGroup) {
	sort.Slice(groups, func(i, j int) bool {
		a, b := groups[i].Label, groups[j].Label
		if a == untaggedGroup {
			return true
		}
		if b == untaggedGroup {
			return false
		}
		return a < b
	})
}

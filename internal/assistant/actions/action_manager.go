package actions

import (
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ActionNode represents a node in the Action tree.
type ActionNode struct {
	Action   Action
	ParentID uuid.UUID
	Children []*ActionNode
}

// ActionChainTree is a tree of actions sharing one chain ID.
type ActionChainTree struct {
	ChainID   uuid.UUID
	CreatedAt time.Time
	UpdatedAt time.Time

	// We store nodes in a map for quick lookups, but we
	// can also keep a pointer to the root node(s) if needed.
	Nodes   map[uuid.UUID]*ActionNode
	Results []string
}

type ActionManager struct {
	activeChains sync.Map // map[uuid.UUID]*ActionChain
	chains       sync.Map
	logger       *slog.Logger
}

func NewActionManager(logger *slog.Logger) *ActionManager {
	am := &ActionManager{
		logger: logger,
	}

	return am
}

func (m *ActionManager) RegisterAction(action Action) {
	chainID := action.Context.ChainID

	raw, loaded := m.activeChains.LoadOrStore(chainID, &ActionChainTree{
		ChainID:   chainID,
		CreatedAt: time.Now(),
		Nodes:     make(map[uuid.UUID]*ActionNode),
	})

	chain := raw.(*ActionChainTree)
	if loaded {
		chain.UpdatedAt = time.Now()
	}

	// Update existing node if present
	if existing, exists := chain.Nodes[action.ID]; exists {
		// m.logger.Debug("RegisterAction, update", "context", action)
		existing.Action = action
		return
	}

	// m.logger.Debug(
	// 	"RegisterAction, create",
	// 	"context", action,
	// )

	node := &ActionNode{
		Action:   action,
		ParentID: action.Context.ParentID,
		Children: make([]*ActionNode, 0),
	}
	chain.Nodes[action.ID] = node

	if action.Context.ParentID != uuid.Nil {
		if parentNode, ok := chain.Nodes[action.Context.ParentID]; ok {
			// Check if child already exists before appending
			exists := false
			for _, child := range parentNode.Children {
				if child.Action.ID == action.ID {
					exists = true
					break
				}
			}
			if !exists {
				parentNode.Children = append(parentNode.Children, node)
			}
		}
	}

	m.activeChains.Store(chainID, chain)
}

// Return the chain tree for a given chainID
func (m *ActionManager) GetChainTree(chainID uuid.UUID) (*ActionChainTree, bool) {
	raw, ok := m.activeChains.Load(chainID)
	if !ok {
		return nil, false
	}
	return raw.(*ActionChainTree), true
}

// Return the root nodes (those with no parent).
func (t *ActionChainTree) RootNodes() []*ActionNode {
	roots := make([]*ActionNode, 0)
	for _, node := range t.Nodes {
		if node.ParentID == uuid.Nil {
			roots = append(roots, node)
		}
	}
	return roots
}

func (m *ActionManager) DumpActionChainTree(chainID uuid.UUID) string {
	var sb strings.Builder

	t, ok := m.GetChainTree(chainID)
	if !ok {
		m.logger.Warn("Chain not found", "chain_id", chainID)
		return ""
	}

	// For each root node, do a DFS
	for _, root := range t.RootNodes() {
		m.dumpNodeRec(root, 0, &sb)
	}

	return sb.String()
}

func (m *ActionManager) dumpNodeRec(node *ActionNode, depth int, sb *strings.Builder) {
	indent := strings.Repeat("  ", depth)
	status := " "
	if node.Action.Result != nil && node.Action.Result.Success {
		status = "✓ "
	}

	sb.WriteString(fmt.Sprintf(
		"%s- %s%s \n",
		indent,
		status,
		node.Action.String(),
	))

	// Sort children by creation time for consistent output
	children := make([]*ActionNode, len(node.Children))
	copy(children, node.Children)
	sort.Slice(children, func(i, j int) bool {
		return children[i].Action.Context.CreatedAt.Before(
			children[j].Action.Context.CreatedAt)
	})

	for _, child := range children {
		m.dumpNodeRec(child, depth+1, sb)
	}
}

func (m *ActionManager) AddResult(chainID uuid.UUID, result string) {
	if record, ok := m.activeChains.Load(chainID); ok {
		chain := record.(*ActionChainTree)
		chain.Results = append(chain.Results, result)
		chain.UpdatedAt = time.Now()
		m.activeChains.Store(chainID, chain)
	}
}

func (m *ActionManager) AddError(chainID uuid.UUID, action Action, err error) {
	errMsg := fmt.Sprintf("ERROR: action_id=%s - %v", action.ID, err)
	m.AddResult(chainID, errMsg)
}

func FindRootAction(m *ActionManager, action Action) Action {
	current := action
	for {
		if current.Context.ParentID == uuid.Nil {
			return current
		}

		if chain, ok := m.GetChainTree(current.Context.ChainID); ok {
			if parentNode, exists := chain.Nodes[current.Context.ParentID]; exists {
				current = parentNode.Action
			} else {
				return current // Parent not found, exit
			}
		} else {
			return current // Chain not found, exit
		}
	}
}

func (m *ActionManager) GetAllChains() []*ActionChainTree {
	var chains []*ActionChainTree
	m.activeChains.Range(func(_, v interface{}) bool {
		chains = append(chains, v.(*ActionChainTree))
		return true
	})
	return chains
}

func (t *ActionChainTree) GetActionsSorted() []Action {
	actions := make([]Action, 0, len(t.Nodes))
	for _, node := range t.Nodes {
		actions = append(actions, node.Action)
	}
	// Sort by CreatedAt
	sort.Slice(actions, func(i, j int) bool {
		return actions[i].Context.CreatedAt.After(actions[j].Context.CreatedAt)
	})
	return actions
}

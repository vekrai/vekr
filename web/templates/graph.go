package templates

import (
	"math"

	"vekr/internal/models"
)

const (
	graphWidth   = 600.0
	graphRowH    = 150.0
	graphMaxCols = 4
	graphMaxR    = 38.0
)

type GraphNode struct {
	ID    uint
	Title string
	X, Y  float64
}

type GraphEdge struct {
	X1, Y1, X2, Y2 float64
	Prerequisite   bool
}

type GraphGroup struct {
	TopicID uint
	Label   string
	X, Y    float64
}

type GraphView struct {
	Width, Height float64
	Nodes         []GraphNode
	Edges         []GraphEdge
	Groups        []GraphGroup
}

func BuildGraph(nodes []models.Node, links []models.NodeLink, topics []models.Topic, unsortedLabel string) GraphView {
	var view GraphView
	if len(nodes) == 0 {
		return view
	}

	byTopic := map[uint][]models.Node{}
	var unsorted []models.Node
	for _, n := range nodes {
		if n.TopicID == nil {
			unsorted = append(unsorted, n)
			continue
		}
		byTopic[*n.TopicID] = append(byTopic[*n.TopicID], n)
	}

	type group struct {
		topicID uint
		label   string
		notes   []models.Node
	}
	var groups []group
	for _, t := range topics {
		if notes := byTopic[t.ID]; len(notes) > 0 {
			groups = append(groups, group{topicID: t.ID, label: t.Title, notes: notes})
			delete(byTopic, t.ID)
		}
	}
	for _, notes := range byTopic {
		unsorted = append(unsorted, notes...)
	}
	if len(unsorted) > 0 {
		groups = append(groups, group{label: unsortedLabel, notes: unsorted})
	}

	cols := min(len(groups), graphMaxCols)
	colW := graphWidth / float64(cols)
	rows := (len(groups) + cols - 1) / cols
	view.Width = graphWidth
	view.Height = float64(rows) * graphRowH

	positions := map[uint][2]float64{}
	for i, g := range groups {
		cx := colW * (float64(i%cols) + 0.5)
		cy := float64(i/cols)*graphRowH + graphRowH/2 - 12
		radius := math.Min(graphMaxR, colW/2-24)
		for k, n := range g.notes {
			x, y := cx, cy
			if len(g.notes) > 1 {
				angle := 2*math.Pi*float64(k)/float64(len(g.notes)) - math.Pi/2
				x = cx + radius*math.Cos(angle)
				y = cy + radius*math.Sin(angle)
			}
			positions[n.ID] = [2]float64{x, y}
			view.Nodes = append(view.Nodes, GraphNode{ID: n.ID, Title: n.Title, X: x, Y: y})
		}
		view.Groups = append(view.Groups, GraphGroup{TopicID: g.topicID, Label: g.label, X: cx, Y: cy + radius + 26})
	}

	for _, l := range links {
		from, okFrom := positions[l.FromNodeID]
		to, okTo := positions[l.ToNodeID]
		if !okFrom || !okTo {
			continue
		}
		view.Edges = append(view.Edges, GraphEdge{X1: from[0], Y1: from[1], X2: to[0], Y2: to[1], Prerequisite: l.Relation == models.RelationPrerequisiteOf})
	}
	return view
}

func graphLabel(label string, width float64) string {
	limit := int(width/7.5) - 1
	runes := []rune(label)
	if limit < 4 || len(runes) <= limit {
		return label
	}
	return string(runes[:limit-1]) + "…"
}

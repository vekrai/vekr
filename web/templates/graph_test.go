package templates

import (
	"testing"

	"vekr/internal/models"
)

func TestBuildGraphGroupsNotesByTopic(t *testing.T) {
	one, two := uint(1), uint(2)
	nodes := []models.Node{
		{ID: 10, Title: "a", TopicID: &one},
		{ID: 11, Title: "b", TopicID: &one},
		{ID: 12, Title: "c", TopicID: &two},
		{ID: 13, Title: "loose"},
	}
	links := []models.NodeLink{
		{FromNodeID: 10, ToNodeID: 11, Relation: models.RelationPrerequisiteOf},
		{FromNodeID: 10, ToNodeID: 99, Relation: models.RelationRelated},
	}
	topics := []models.Topic{{ID: 1, Title: "One"}, {ID: 2, Title: "Two"}}

	view := BuildGraph(nodes, links, topics, "Unsorted")

	if len(view.Nodes) != 4 {
		t.Fatalf("got %d nodes, want 4", len(view.Nodes))
	}
	labels := []string{}
	for _, g := range view.Groups {
		labels = append(labels, g.Label)
	}
	if len(labels) != 3 || labels[0] != "One" || labels[1] != "Two" || labels[2] != "Unsorted" {
		t.Errorf("got groups %v, want [One Two Unsorted]", labels)
	}
	if len(view.Edges) != 1 || !view.Edges[0].Prerequisite {
		t.Errorf("got edges %+v, want one prerequisite edge (the link to a missing node is dropped)", view.Edges)
	}
	for _, n := range view.Nodes {
		if n.X < 0 || n.X > view.Width || n.Y < 0 || n.Y > view.Height {
			t.Errorf("node %d at (%.0f, %.0f) is outside the %.0fx%.0f canvas", n.ID, n.X, n.Y, view.Width, view.Height)
		}
	}
}

func TestBuildGraphEmpty(t *testing.T) {
	if view := BuildGraph(nil, nil, nil, "Unsorted"); len(view.Nodes) != 0 || view.Height != 0 {
		t.Errorf("got %+v, want an empty view", view)
	}
}

func TestGraphLabelTruncates(t *testing.T) {
	if got := graphLabel("Present Perfect vs. Simple Past", 120); got != "Present Perfec…" {
		t.Errorf("got %q", got)
	}
	if got := graphLabel("Short", 120); got != "Short" {
		t.Errorf("got %q", got)
	}
}

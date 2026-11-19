package models

import "testing"

func TestPracticeItemChoicesSplitsStemAndOptions(t *testing.T) {
	item := PracticeItem{Type: PracticeItemTypeMCQ, Prompt: "Choose the correct sentence.\n\nA) I have seen that film last night.\nB) I saw that film last night.\nc. I have saw that film."}

	stem, choices := item.Choices()

	if stem != "Choose the correct sentence." {
		t.Errorf("got stem %q", stem)
	}
	want := []Choice{
		{Letter: "A", Text: "I have seen that film last night."},
		{Letter: "B", Text: "I saw that film last night."},
		{Letter: "C", Text: "I have saw that film."},
	}
	if len(choices) != len(want) {
		t.Fatalf("got %d choices, want %d", len(choices), len(want))
	}
	for i, c := range want {
		if choices[i] != c {
			t.Errorf("choice %d: got %+v, want %+v", i, choices[i], c)
		}
	}
}

func TestPracticeItemChoicesFallsBackWithoutOptions(t *testing.T) {
	item := PracticeItem{Type: PracticeItemTypeMCQ, Prompt: "Which is right: since or for?"}

	stem, choices := item.Choices()

	if stem != item.Prompt || choices != nil {
		t.Errorf("got stem %q choices %v, want the raw prompt and no choices", stem, choices)
	}
}

func TestPracticeItemExpectedAnswerIncludesChoiceText(t *testing.T) {
	key := "b"
	item := PracticeItem{Type: PracticeItemTypeMCQ, AnswerKey: &key, Prompt: "Pick.\nA) one\nB) two"}

	if got := item.ExpectedAnswer(); got != "B) two" {
		t.Errorf("got %q, want %q", got, "B) two")
	}
}

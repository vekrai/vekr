package main

import (
	"log"
	"time"

	"gorm.io/gorm"

	"vekr/internal/models"
)

type seedAttempt struct {
	submission string
	passed     bool
	feedback   string
	mistake    string
	ago        time.Duration
}

type seedItem struct {
	kind     string
	prompt   string
	answer   string
	concepts []string
	attempts []seedAttempt
}

type seedNote struct {
	title   string
	content string
}

type seedTopic struct {
	title    string
	content  string
	status   string
	ago      time.Duration
	notes    []seedNote
	concepts []string
	items    []seedItem
}

type seedChat struct {
	note     string
	ago      time.Duration
	messages []string
}

const hour = time.Hour
const day = 24 * time.Hour

var topics = []seedTopic{
	{
		title:    "Present Perfect vs. Simple Past",
		content:  "When to say \"I have seen\" and when to say \"I saw\": finished time vs. time up to now, signal words, and the most common mistakes learners make.",
		status:   models.StatusPracticing,
		ago:      2 * hour,
		concepts: []string{"finished-time-markers", "experience-vs-event", "since-vs-for"},
		notes: []seedNote{
			{"Simple Past: Finished Time", "Use the simple past for actions at a **specific, finished time**: *I visited Paris **in 2019***. Signal words: *yesterday, last week, ago, in 2019, when I was a child*."},
			{"Present Perfect: Time Up to Now", "Use *have/has + past participle* when the time period is **not finished** or **not stated**: *I **have visited** Paris twice.* The focus is on experience or the present result, not on when."},
			{"Since vs. For", "*For* + a length of time (*for three years*). *Since* + a starting point (*since 2021, since Monday*). Both usually go with the present perfect: *I have lived here **since** 2021.*"},
		},
		items: []seedItem{
			{models.PracticeItemTypeMCQ, "Choose the correct sentence.\n\nA) I have seen that film last night.\nB) I saw that film last night.\nC) I have saw that film last night.", "B", []string{"finished-time-markers"}, []seedAttempt{
				{"A", false, "*Last night* is a finished time, so the present perfect doesn't fit. Use the simple past: *I saw*.", models.ErrorTypeMisconception, 3 * day},
				{"B", true, "Correct: a finished time marker takes the simple past.", "", 2 * hour},
			}},
			{models.PracticeItemTypeOpen, "Fill in the blank and explain your choice: \"She ___ (live) in London for five years, and she still loves it.\"", "has lived; the action started in the past and continues now.", []string{"since-vs-for", "experience-vs-event"}, []seedAttempt{
				{"lived", false, "*She still loves it* tells us she still lives there, so the period isn't finished: *has lived*.", models.ErrorTypeKnowledgeGap, 2 * day},
			}},
			{models.PracticeItemTypeFlashcard, "Since or for? \"I have studied English ___ 2020.\"", "since", []string{"since-vs-for"}, []seedAttempt{
				{"since", true, "Right: 2020 is a starting point.", "", 5 * hour},
			}},
			{models.PracticeItemTypeOpen, "Correct the sentence: \"Did you ever eat sushi?\"", "Have you ever eaten sushi?", []string{"experience-vs-event"}, []seedAttempt{
				{"Have you ever ate sushi?", true, "The tense is right, but the participle of *eat* is *eaten*: *Have you ever eaten sushi?*", models.ErrorTypeCarelessSlip, 4 * hour},
			}},
		},
	},
	{
		title:    "Articles: a, an, the, or nothing",
		content:  "Choosing between indefinite, definite, and zero articles, with countable vs. uncountable nouns and first vs. second mention.",
		status:   models.StatusLearning,
		ago:      30 * time.Minute,
		concepts: []string{"countable-nouns", "first-vs-second-mention"},
		notes: []seedNote{
			{"A vs. An: It's About Sound", "Choose by the **sound**, not the letter: *an hour, an MBA, a university, a one-way ticket*."},
			{"First Mention vs. Second Mention", "Introduce something new with *a/an*, then refer back with *the*: *I bought **a** book. **The** book is about grammar.*"},
			{"Zero Article with Uncountable and Plural Nouns", "Talking in general? No article: ***Water** is essential. **Dogs** are loyal.* Add *the* only for specific ones: ***The** water in this bottle is cold.*"},
		},
		items: []seedItem{
			{models.PracticeItemTypeMCQ, "Choose the correct article: \"She is ___ honest person.\"\n\nA) a\nB) an\nC) the", "B", []string{"countable-nouns"}, nil},
			{models.PracticeItemTypeOpen, "Why is it \"I saw a dog. The dog was barking.\" and not \"I saw the dog. A dog was barking.\"?", "First mention introduces with a; second mention refers back with the.", []string{"first-vs-second-mention"}, nil},
			{models.PracticeItemTypeFlashcard, "Article or none? \"___ information is power.\"", "No article: information is uncountable and general.", []string{"countable-nouns"}, nil},
		},
	},
	{
		title:    "Conditionals: Zero to Third",
		content:  "Real and unreal conditions: facts, likely futures, imagined presents, and regrets about the past.",
		status:   models.StatusPracticing,
		ago:      day,
		concepts: []string{"unreal-past", "if-clause-tense"},
		notes: []seedNote{
			{"First vs. Second Conditional", "First: a real possibility, *If it **rains**, I **will stay** home.* Second: imagined or unlikely, *If I **won** the lottery, I **would travel**.* Never put *will* or *would* in the if-clause."},
			{"Third Conditional for Regrets", "*If + had + past participle, would have + past participle*: *If I **had studied**, I **would have passed**.* It talks about a past that didn't happen."},
		},
		items: []seedItem{
			{models.PracticeItemTypeMCQ, "Choose the correct sentence.\n\nA) If I will see her, I will tell her.\nB) If I see her, I will tell her.\nC) If I would see her, I tell her.", "B", []string{"if-clause-tense"}, []seedAttempt{
				{"A", false, "Don't use *will* in the if-clause; the present simple already refers to the future: *If I see her*.", models.ErrorTypeMisconception, 20 * hour},
			}},
			{models.PracticeItemTypeOpen, "Complete: \"If she ___ (leave) earlier, she wouldn't have missed the train.\"", "had left", []string{"unreal-past"}, []seedAttempt{
				{"had left", true, "Correct third conditional.", "", 22 * hour},
			}},
			{models.PracticeItemTypeFlashcard, "Which conditional describes general truths, like \"If you heat ice, it melts\"?", "Zero conditional", nil, nil},
		},
	},
	{
		title:   "Countable and Uncountable Nouns",
		content: "Much vs. many, a few vs. a little, and nouns that are uncountable in English but not in other languages.",
		status:  models.StatusMastered,
		ago:     4 * day,
		notes: []seedNote{
			{"Much, Many, a Few, a Little", "*Many / a few* + plural countable nouns (*many books, a few friends*). *Much / a little* + uncountable nouns (*much time, a little advice*)."},
		},
		items: []seedItem{
			{models.PracticeItemTypeFlashcard, "Much or many? \"How ___ advice did she give you?\"", "much", nil, []seedAttempt{
				{"much", true, "Correct: *advice* is uncountable.", "", 4 * day},
			}},
			{models.PracticeItemTypeMCQ, "Which sentence is correct?\n\nA) I need some informations.\nB) I need some information.\nC) I need an information.", "B", nil, []seedAttempt{
				{"B", true, "Correct: *information* has no plural and takes no *a/an*.", "", 4 * day},
			}},
		},
	},
}

var chats = []seedChat{
	{"", 20 * time.Minute, []string{
		"I always mix up a, an and the. Can you help?",
		"Sure. I've drafted a topic **Articles: a, an, the, or nothing** with notes on *a* vs. *an*, first vs. second mention, and the zero article. Want me to add practice items?",
	}},
	{"Present Perfect: Time Up to Now", 21 * hour, []string{
		"Why is \"I have seen him yesterday\" wrong?",
		"*Yesterday* is a finished time, and the present perfect can't be used with a finished time. Say *I saw him yesterday*, or drop the time: *I have seen him.*",
	}},
	{"", 2 * day, []string{
		"Quiz me on present perfect vs. simple past",
		"I added 4 practice items to **Present Perfect vs. Simple Past**. Answer them on the topic's practice page and I'll grade each one.",
	}},
}

func main() {
	db, err := models.Connect()
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}

	var count int64
	if err := db.Model(&models.Topic{}).Count(&count).Error; err != nil {
		log.Fatalf("count topics: %v", err)
	}
	if count > 0 {
		log.Printf("db:seed: %d topics already exist, skipping", count)
		return
	}

	if err := db.Transaction(seed); err != nil {
		log.Fatalf("db:seed: %v", err)
	}
	log.Printf("db:seed: inserted %d topics and %d chats", len(topics), len(chats))
}

func seed(tx *gorm.DB) error {
	now := time.Now()
	notesByTitle := map[string]uint{}

	for _, t := range topics {
		at := now.Add(-t.ago)
		topic := models.Topic{Title: t.title, Content: t.content, Status: t.status, CreatedAt: at, UpdatedAt: at}
		if err := tx.Create(&topic).Error; err != nil {
			return err
		}

		var previous uint
		for i, n := range t.notes {
			noteAt := at.Add(-time.Duration(len(t.notes)-i) * time.Minute)
			note := models.Node{Title: n.title, Content: n.content, TopicID: &topic.ID, CreatedAt: noteAt, UpdatedAt: noteAt}
			if err := tx.Create(&note).Error; err != nil {
				return err
			}
			notesByTitle[n.title] = note.ID
			if previous != 0 {
				link := models.NodeLink{FromNodeID: previous, ToNodeID: note.ID, Relation: models.RelationPrerequisiteOf}
				if err := tx.Create(&link).Error; err != nil {
					return err
				}
			}
			previous = note.ID
		}

		conceptIDs := map[string]uint{}
		for _, name := range t.concepts {
			concept := models.Concept{TopicID: &topic.ID, Name: name}
			if err := tx.Create(&concept).Error; err != nil {
				return err
			}
			conceptIDs[name] = concept.ID
		}

		for _, it := range t.items {
			answer := it.answer
			item := models.PracticeItem{TopicID: topic.ID, Type: it.kind, Prompt: it.prompt, AnswerKey: &answer, CreatedAt: at}
			if err := tx.Create(&item).Error; err != nil {
				return err
			}
			for _, name := range it.concepts {
				if err := tx.Create(&models.PracticeItemConcept{PracticeItemID: item.ID, ConceptID: conceptIDs[name]}).Error; err != nil {
					return err
				}
			}
			for _, a := range it.attempts {
				passed := a.passed
				attempt := models.Attempt{PracticeItemID: item.ID, Submission: a.submission, Passed: &passed, AgentFeedback: a.feedback, CreatedAt: now.Add(-a.ago)}
				if err := tx.Create(&attempt).Error; err != nil {
					return err
				}
				if a.mistake == "" {
					continue
				}
				for _, name := range it.concepts {
					mistake := models.MistakeLog{AttemptID: attempt.ID, ConceptID: conceptIDs[name], ErrorType: a.mistake, CreatedAt: attempt.CreatedAt}
					if err := tx.Create(&mistake).Error; err != nil {
						return err
					}
				}
			}
		}
	}

	for _, c := range chats {
		at := now.Add(-c.ago)
		chat := models.Chat{CreatedAt: at}
		if c.note != "" {
			id := notesByTitle[c.note]
			chat.NodeID = &id
		}
		if err := tx.Create(&chat).Error; err != nil {
			return err
		}
		for i, content := range c.messages {
			role := "user"
			if i%2 == 1 {
				role = "assistant"
			}
			message := models.ChatMessage{ChatID: chat.ID, Role: role, Content: content, CreatedAt: at.Add(time.Duration(i) * time.Second)}
			if err := tx.Create(&message).Error; err != nil {
				return err
			}
		}
	}

	return tx.Exec(`
		INSERT INTO concept_weakness (concept_id, miss_count, attempt_count, last_missed_at)
		SELECT c.id,
		       (SELECT COUNT(*) FROM mistake_log m WHERE m.concept_id = c.id),
		       (SELECT COUNT(DISTINCT a.id) FROM attempts a JOIN practice_item_concepts pic ON pic.practice_item_id = a.practice_item_id WHERE pic.concept_id = c.id),
		       (SELECT MAX(created_at) FROM mistake_log m WHERE m.concept_id = c.id)
		FROM concepts c
	`).Error
}

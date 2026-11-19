package models

import (
	"strings"

	"gorm.io/gorm"
)

type NoteHit struct {
	NodeID  uint   `gorm:"column:node_id"`
	TopicID *uint  `gorm:"column:topic_id"`
	Title   string `gorm:"column:title"`
	Snippet string `gorm:"column:snippet"`
}

func escapeLikePattern(s string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return "%" + replacer.Replace(s) + "%"
}

func SearchNotes(db *gorm.DB, query string, limit int) ([]NoteHit, error) {
	var hits []NoteHit
	err := db.Raw(`
		WITH q AS (
			SELECT replace(plainto_tsquery('english', ?)::text, ' & ', ' | ')::tsquery AS tsq
		),
		candidates AS (
			SELECT n.id, n.topic_id, n.title, n.content,
			       to_tsvector('english', n.title || ' ' || n.content) AS tsv,
			       q.tsq
			FROM nodes n, q
		)
		SELECT id AS node_id, topic_id, title,
		       CASE
		           WHEN tsv @@ tsq THEN ts_headline('english', content, tsq, 'MaxFragments=1,MaxWords=25')
		           ELSE left(content, 160)
		       END AS snippet
		FROM candidates
		WHERE tsv @@ tsq OR (title || ' ' || content) ILIKE ?
		ORDER BY
		    CASE WHEN tsv @@ tsq THEN ts_rank(tsv, tsq) ELSE 0 END DESC,
		    similarity(title || ' ' || content, ?) DESC
		LIMIT ?
	`, query, escapeLikePattern(query), query, limit).Scan(&hits).Error
	return hits, err
}

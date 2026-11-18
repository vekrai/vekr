ALTER TABLE paths RENAME TO topics;
ALTER SEQUENCE paths_id_seq RENAME TO topics_id_seq;
ALTER INDEX paths_pkey RENAME TO topics_pkey;
ALTER TABLE topics ADD COLUMN content TEXT NOT NULL DEFAULT '';
ALTER TABLE topics ADD COLUMN status TEXT NOT NULL DEFAULT 'not_started' CHECK (status IN ('not_started', 'learning', 'practicing', 'mastered'));
ALTER TABLE topics ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT now();

ALTER TABLE topics ADD COLUMN legacy_node_id BIGINT;
INSERT INTO topics (title, content, legacy_node_id)
SELECT title, content, id FROM nodes WHERE type = 'topic' AND path_id IS NULL;
UPDATE nodes SET path_id = t.id FROM topics t WHERE t.legacy_node_id = nodes.id;
ALTER TABLE topics DROP COLUMN legacy_node_id;

UPDATE topics t SET status = CASE
    WHEN s.total > 0 AND s.mastered = s.total THEN 'mastered'
    WHEN s.practicing > 0 OR s.mastered > 0 THEN 'practicing'
    WHEN s.learning > 0 THEN 'learning'
    ELSE 'not_started'
END
FROM (
    SELECT path_id,
           COUNT(*) AS total,
           COUNT(*) FILTER (WHERE status = 'mastered') AS mastered,
           COUNT(*) FILTER (WHERE status = 'practicing') AS practicing,
           COUNT(*) FILTER (WHERE status = 'learning') AS learning
    FROM nodes
    WHERE type = 'topic'
    GROUP BY path_id
) s
WHERE s.path_id = t.id;

ALTER TABLE practice_items ADD COLUMN topic_id BIGINT;
UPDATE practice_items pi SET topic_id = n.path_id FROM nodes n WHERE n.id = pi.node_id;
DELETE FROM mistake_log WHERE attempt_id IN (SELECT a.id FROM attempts a JOIN practice_items pi ON pi.id = a.practice_item_id WHERE pi.topic_id IS NULL);
DELETE FROM attempts WHERE practice_item_id IN (SELECT id FROM practice_items WHERE topic_id IS NULL);
DELETE FROM practice_item_concepts WHERE practice_item_id IN (SELECT id FROM practice_items WHERE topic_id IS NULL);
DELETE FROM practice_items WHERE topic_id IS NULL;
ALTER TABLE practice_items ALTER COLUMN topic_id SET NOT NULL;
ALTER TABLE practice_items DROP COLUMN node_id;
CREATE INDEX practice_items_topic_id_idx ON practice_items (topic_id);

UPDATE topics t SET status = CASE WHEN s.total > 0 AND s.passed = s.total THEN 'mastered' ELSE 'practicing' END
FROM (
    SELECT pi.topic_id,
           COUNT(*) AS total,
           COUNT(*) FILTER (WHERE latest.passed AND NOT EXISTS (SELECT 1 FROM mistake_log m WHERE m.attempt_id = latest.id)) AS passed,
           COUNT(latest.id) AS attempted
    FROM practice_items pi
    LEFT JOIN LATERAL (
        SELECT a.id, a.passed FROM attempts a WHERE a.practice_item_id = pi.id ORDER BY a.created_at DESC, a.id DESC LIMIT 1
    ) latest ON true
    GROUP BY pi.topic_id
) s
WHERE s.topic_id = t.id AND s.attempted > 0 AND t.status <> 'mastered';

ALTER TABLE nodes RENAME COLUMN path_id TO topic_id;
ALTER TABLE nodes DROP COLUMN status;
ALTER TABLE nodes DROP COLUMN type;
CREATE INDEX nodes_topic_id_idx ON nodes (topic_id);

ALTER TABLE concepts RENAME COLUMN path_id TO topic_id;

DROP TABLE node_concepts;

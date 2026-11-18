CREATE TABLE node_concepts (
    node_id    BIGINT NOT NULL,
    concept_id BIGINT NOT NULL,
    PRIMARY KEY (node_id, concept_id)
);

ALTER TABLE concepts RENAME COLUMN topic_id TO path_id;

DROP INDEX nodes_topic_id_idx;
ALTER TABLE nodes ADD COLUMN type TEXT NOT NULL DEFAULT 'note' CHECK (type IN ('note', 'topic'));
ALTER TABLE nodes ADD COLUMN status TEXT CHECK (status IN ('not_started', 'learning', 'practicing', 'mastered'));
ALTER TABLE nodes RENAME COLUMN topic_id TO path_id;
UPDATE nodes n SET type = 'topic', status = t.status FROM topics t WHERE t.id = n.path_id;

DROP INDEX practice_items_topic_id_idx;
ALTER TABLE practice_items ADD COLUMN node_id BIGINT;
UPDATE practice_items pi SET node_id = (SELECT MIN(n.id) FROM nodes n WHERE n.path_id = pi.topic_id);
INSERT INTO nodes (title, content, type, path_id, status)
SELECT t.title, t.content, 'topic', t.id, t.status
FROM topics t
WHERE EXISTS (SELECT 1 FROM practice_items pi WHERE pi.topic_id = t.id AND pi.node_id IS NULL);
UPDATE practice_items pi SET node_id = (SELECT MIN(n.id) FROM nodes n WHERE n.path_id = pi.topic_id) WHERE node_id IS NULL;
ALTER TABLE practice_items ALTER COLUMN node_id SET NOT NULL;
ALTER TABLE practice_items DROP COLUMN topic_id;

ALTER TABLE topics DROP COLUMN updated_at;
ALTER TABLE topics DROP COLUMN status;
ALTER TABLE topics DROP COLUMN content;
ALTER TABLE topics RENAME TO paths;
ALTER INDEX topics_pkey RENAME TO paths_pkey;
ALTER SEQUENCE topics_id_seq RENAME TO paths_id_seq;

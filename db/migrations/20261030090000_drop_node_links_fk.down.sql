ALTER TABLE node_links ADD CONSTRAINT node_links_from_node_id_fkey FOREIGN KEY (from_node_id) REFERENCES nodes(id);
ALTER TABLE node_links ADD CONSTRAINT node_links_to_node_id_fkey FOREIGN KEY (to_node_id) REFERENCES nodes(id);

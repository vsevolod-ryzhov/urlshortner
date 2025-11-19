CREATE TABLE links (
    id BIGINT PRIMARY KEY,
    short VARCHAR(32) NOT NULL,
    original TEXT NOT NULL
);

CREATE INDEX idx_links_short ON links(short);
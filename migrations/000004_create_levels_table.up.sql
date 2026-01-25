-- Create levels table
CREATE TABLE levels (
    level INTEGER PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    min_xp INTEGER NOT NULL
);

-- Insert default levels
INSERT INTO levels (level, name, min_xp) VALUES
    (1, 'Rookie', 0),
    (2, 'Performer', 100),
    (3, 'Pro', 500),
    (4, 'Ace', 1500),
    (5, 'Legend', 5000),
    (6, 'Champion', 15000);

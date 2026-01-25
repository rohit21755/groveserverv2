-- Create chat_members table
CREATE TABLE chat_members (
    room_id UUID NOT NULL REFERENCES chat_rooms(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    joined_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (room_id, user_id)
);

-- Create indexes
CREATE INDEX idx_chat_members_room_id ON chat_members(room_id);
CREATE INDEX idx_chat_members_user_id ON chat_members(user_id);

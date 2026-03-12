-- name: CreateTaskConversation :one
INSERT INTO task_conversations (status) VALUES ('active')
RETURNING *;

-- name: GetTaskConversation :one
SELECT * FROM task_conversations WHERE id = $1;

-- name: UpdateTaskConversationStatus :one
UPDATE task_conversations SET status = $2 WHERE id = $1
RETURNING *;

-- name: LinkTaskConversationParent :one
UPDATE task_conversations SET parent_task_id = $2 WHERE id = $1
RETURNING *;

-- name: CreateTaskMessage :one
INSERT INTO task_messages (conversation_id, role, content) VALUES ($1, $2, $3)
RETURNING *;

-- name: GetTaskMessages :many
SELECT * FROM task_messages WHERE conversation_id = $1 ORDER BY created_at ASC;

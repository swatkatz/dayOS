-- name: CreateTaskConversation :one
INSERT INTO task_conversations (user_id, status) VALUES (@user_id, 'active')
RETURNING *;

-- name: GetTaskConversation :one
SELECT * FROM task_conversations WHERE id = $1 AND user_id = @user_id;

-- name: UpdateTaskConversationStatus :one
UPDATE task_conversations SET status = $2 WHERE id = $1 AND user_id = @user_id
RETURNING *;

-- name: LinkTaskConversationParent :one
UPDATE task_conversations SET parent_task_id = $2 WHERE id = $1 AND user_id = @user_id
RETURNING *;

-- name: CreateTaskMessage :one
INSERT INTO task_messages (conversation_id, role, content) VALUES ($1, $2, $3)
RETURNING *;

-- name: GetTaskMessages :many
SELECT * FROM task_messages WHERE conversation_id = $1 ORDER BY created_at ASC;

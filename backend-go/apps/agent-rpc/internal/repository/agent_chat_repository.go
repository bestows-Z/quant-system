package repository

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"quant-system/backend-go/apps/agent-rpc/agent"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ChatRepository struct{ db *sql.DB }

func NewChatRepository(db *sql.DB) *ChatRepository { return &ChatRepository{db: db} }

func (r *ChatRepository) CreateSession(ctx context.Context, userID uint64, title string) (*agent.AgentChatSession, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		title = "新研究会话"
	}
	if len([]rune(title)) > 80 {
		title = string([]rune(title)[:80])
	}
	res, err := r.db.ExecContext(ctx, `INSERT INTO agent_chat_sessions(user_id,title,status) VALUES(?,?,'ACTIVE')`, userID, title)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return r.GetSession(ctx, userID, uint64(id))
}

func (r *ChatRepository) GetSession(ctx context.Context, userID, sessionID uint64) (*agent.AgentChatSession, error) {
	var item agent.AgentChatSession
	var last sql.NullTime
	var created time.Time
	err := r.db.QueryRowContext(ctx, `SELECT id,title,status,last_message_at,created_at FROM agent_chat_sessions WHERE id=? AND user_id=? AND deleted_at IS NULL`, sessionID, userID).Scan(&item.Id, &item.Title, &item.Status, &last, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, status.Error(codes.NotFound, "研究会话不存在")
	}
	if err != nil {
		return nil, err
	}
	item.CreatedAt = created.Format("2006-01-02 15:04:05")
	if last.Valid {
		item.LastMessageAt = last.Time.Format("2006-01-02 15:04:05")
	}
	return &item, nil
}

func (r *ChatRepository) ListSessions(ctx context.Context, userID uint64) ([]*agent.AgentChatSession, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,title,status,last_message_at,created_at FROM agent_chat_sessions WHERE user_id=? AND deleted_at IS NULL ORDER BY COALESCE(last_message_at,created_at) DESC,id DESC LIMIT 100`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]*agent.AgentChatSession, 0)
	for rows.Next() {
		var item agent.AgentChatSession
		var last sql.NullTime
		var created time.Time
		if err := rows.Scan(&item.Id, &item.Title, &item.Status, &last, &created); err != nil {
			return nil, err
		}
		item.CreatedAt = created.Format("2006-01-02 15:04:05")
		if last.Valid {
			item.LastMessageAt = last.Time.Format("2006-01-02 15:04:05")
		}
		items = append(items, &item)
	}
	return items, rows.Err()
}

func (r *ChatRepository) RenameSession(ctx context.Context, userID, sessionID uint64, title string) (*agent.AgentChatSession, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, status.Error(codes.InvalidArgument, "会话名称不能为空")
	}
	if len([]rune(title)) > 80 {
		title = string([]rune(title)[:80])
	}
	res, err := r.db.ExecContext(ctx, `UPDATE agent_chat_sessions SET title=? WHERE id=? AND user_id=? AND deleted_at IS NULL`, title, sessionID, userID)
	if err != nil {
		return nil, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil, status.Error(codes.NotFound, "研究会话不存在")
	}
	return r.GetSession(ctx, userID, sessionID)
}

func (r *ChatRepository) DeleteSession(ctx context.Context, userID, sessionID uint64) (uint64, error) {
	res, err := r.db.ExecContext(ctx, `UPDATE agent_chat_sessions SET status='DELETED',deleted_at=CURRENT_TIMESTAMP WHERE id=? AND user_id=? AND deleted_at IS NULL`, sessionID, userID)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return uint64(n), nil
}

func (r *ChatRepository) ClearMessages(ctx context.Context, userID, sessionID uint64) (uint64, error) {
	if _, err := r.GetSession(ctx, userID, sessionID); err != nil {
		return 0, err
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()
	res, err := tx.ExecContext(ctx, `DELETE FROM agent_chat_messages WHERE session_id=? AND user_id=?`, sessionID, userID)
	if err != nil {
		return 0, err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE agent_chat_sessions SET last_message_at=NULL WHERE id=? AND user_id=?`, sessionID, userID); err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	if err = tx.Commit(); err != nil {
		return 0, err
	}
	return uint64(n), nil
}

func (r *ChatRepository) ListMessages(ctx context.Context, userID, sessionID uint64) ([]*agent.AgentChatMessage, error) {
	if _, err := r.GetSession(ctx, userID, sessionID); err != nil {
		return nil, err
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id,session_id,request_id,role,content,status,provider,model,input_tokens,output_tokens,latency_ms,COALESCE(CAST(citations_json AS CHAR),'[]'),COALESCE(CAST(tool_calls_json AS CHAR),'[]'),error_message,created_at FROM agent_chat_messages WHERE session_id=? AND user_id=? ORDER BY id ASC LIMIT 500`, sessionID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]*agent.AgentChatMessage, 0)
	for rows.Next() {
		item, err := scanChatMessage(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *ChatRepository) FindAssistant(ctx context.Context, userID, sessionID uint64, requestID string) (*agent.AgentChatMessage, error) {
	item, err := scanChatMessage(r.db.QueryRowContext(ctx, `SELECT id,session_id,request_id,role,content,status,provider,model,input_tokens,output_tokens,latency_ms,COALESCE(CAST(citations_json AS CHAR),'[]'),COALESCE(CAST(tool_calls_json AS CHAR),'[]'),error_message,created_at FROM agent_chat_messages WHERE session_id=? AND user_id=? AND request_id=? AND role='assistant'`, sessionID, userID, requestID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return item, err
}

func (r *ChatRepository) InsertMessage(ctx context.Context, userID, sessionID uint64, item *agent.AgentChatMessage) (*agent.AgentChatMessage, error) {
	if _, err := r.GetSession(ctx, userID, sessionID); err != nil {
		return nil, err
	}
	res, err := r.db.ExecContext(ctx, `INSERT INTO agent_chat_messages(session_id,user_id,request_id,role,content,status,provider,model,input_tokens,output_tokens,latency_ms,citations_json,tool_calls_json,error_message) VALUES(?,?,?,?,?,?,?,?,?,?,?,CAST(? AS JSON),CAST(? AS JSON),?)`, sessionID, userID, item.RequestId, item.Role, item.Content, item.Status, item.Provider, item.Model, item.InputTokens, item.OutputTokens, item.LatencyMs, jsonOrEmpty(item.CitationsJson, "[]"), jsonOrEmpty(item.ToolCallsJson, "[]"), item.ErrorMessage)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	_, _ = r.db.ExecContext(ctx, `UPDATE agent_chat_sessions SET last_message_at=CURRENT_TIMESTAMP WHERE id=? AND user_id=?`, sessionID, userID)
	item.Id = uint64(id)
	item.SessionId = sessionID
	item.CreatedAt = time.Now().Format("2006-01-02 15:04:05")
	return item, nil
}

func jsonOrEmpty(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func scanChatMessage(s scanner) (*agent.AgentChatMessage, error) {
	var item agent.AgentChatMessage
	var created time.Time
	if err := s.Scan(&item.Id, &item.SessionId, &item.RequestId, &item.Role, &item.Content, &item.Status, &item.Provider, &item.Model, &item.InputTokens, &item.OutputTokens, &item.LatencyMs, &item.CitationsJson, &item.ToolCallsJson, &item.ErrorMessage, &created); err != nil {
		return nil, err
	}
	item.CreatedAt = created.Format("2006-01-02 15:04:05")
	return &item, nil
}

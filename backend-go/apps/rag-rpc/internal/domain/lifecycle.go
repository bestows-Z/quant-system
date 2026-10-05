package domain

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"quant-system/backend-go/apps/rag-rpc/rag"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Service) EnsureLifecycleSchema(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS rag_vector_cleanup_tasks (
id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY, operation VARCHAR(32) NOT NULL, collection_name VARCHAR(255) NOT NULL,
point_ids_json JSON DEFAULT NULL, status VARCHAR(24) NOT NULL DEFAULT 'PENDING', retry_count INT NOT NULL DEFAULT 0,
last_error VARCHAR(1000) DEFAULT NULL, next_retry_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
KEY idx_cleanup_status_retry(status,next_retry_at,id)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`)
	return err
}

func (s *Service) DeleteDocument(ctx context.Context, userID, documentID uint64) (*rag.DeleteDocumentResp, error) {
	if userID == 0 || documentID == 0 {
		return nil, status.Error(codes.InvalidArgument, "删除参数错误")
	}
	var collection string
	var deletedAt sql.NullTime
	err := s.db.QueryRowContext(ctx, `SELECT kb.collection_name,d.deleted_at FROM rag_documents d JOIN knowledge_bases kb ON kb.id=d.knowledge_base_id WHERE d.id=? AND kb.user_id=? AND kb.deleted_at IS NULL`, documentID, userID).Scan(&collection, &deletedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, status.Error(codes.NotFound, "文档不存在或无权删除")
	}
	if err != nil {
		return nil, err
	}
	if deletedAt.Valid {
		return &rag.DeleteDocumentResp{Ok: true, AlreadyDeleted: true}, nil
	}
	pointIDs, err := s.documentPointIDs(ctx, documentID)
	if err != nil {
		return nil, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `UPDATE rag_documents SET status='DELETED',deleted_at=CURRENT_TIMESTAMP,updated_at=CURRENT_TIMESTAMP WHERE id=? AND deleted_at IS NULL`, documentID); err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(ctx, `DELETE e FROM rag_embeddings e JOIN rag_chunks c ON c.id=e.chunk_id WHERE c.document_id=?`, documentID); err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM rag_chunks WHERE document_id=?`, documentID); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	if err = s.qdrant.DeletePoints(ctx, collection, pointIDs); err != nil {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if queueErr := s.queueCleanup(cleanupCtx, "DELETE_POINTS", collection, pointIDs, err); queueErr != nil {
			return nil, fmt.Errorf("文档已软删除，Qdrant 清理和补偿任务写入均失败: %v / %w", err, queueErr)
		}
		return &rag.DeleteDocumentResp{Ok: true, VectorCleanupPending: true}, nil
	}
	return &rag.DeleteDocumentResp{Ok: true}, nil
}

func (s *Service) DeleteKnowledgeBase(ctx context.Context, userID, kbID uint64) (*rag.DeleteKnowledgeBaseResp, error) {
	if userID == 0 || kbID == 0 {
		return nil, status.Error(codes.InvalidArgument, "删除参数错误")
	}
	var collection string
	var deletedAt sql.NullTime
	err := s.db.QueryRowContext(ctx, `SELECT collection_name,deleted_at FROM knowledge_bases WHERE id=? AND user_id=?`, kbID, userID).Scan(&collection, &deletedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, status.Error(codes.NotFound, "知识库不存在或无权删除")
	}
	if err != nil {
		return nil, err
	}
	if collection == systemCollection {
		return nil, status.Error(codes.PermissionDenied, "系统量化研究库不可删除")
	}
	if deletedAt.Valid {
		return &rag.DeleteKnowledgeBaseResp{Ok: true, AlreadyDeleted: true}, nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `UPDATE rag_documents SET status='DELETED',deleted_at=CURRENT_TIMESTAMP,updated_at=CURRENT_TIMESTAMP WHERE knowledge_base_id=? AND deleted_at IS NULL`, kbID); err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM rag_embeddings WHERE knowledge_base_id=?`, kbID); err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM rag_chunks WHERE knowledge_base_id=?`, kbID); err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE knowledge_bases SET deleted_at=CURRENT_TIMESTAMP,updated_at=CURRENT_TIMESTAMP WHERE id=? AND deleted_at IS NULL`, kbID); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	if err = s.qdrant.DeleteCollection(ctx, collection); err != nil {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if queueErr := s.queueCleanup(cleanupCtx, "DELETE_COLLECTION", collection, nil, err); queueErr != nil {
			return nil, fmt.Errorf("知识库已软删除，Qdrant 清理和补偿任务写入均失败: %v / %w", err, queueErr)
		}
		return &rag.DeleteKnowledgeBaseResp{Ok: true, VectorCleanupPending: true}, nil
	}
	return &rag.DeleteKnowledgeBaseResp{Ok: true}, nil
}

func (s *Service) documentPointIDs(ctx context.Context, documentID uint64) ([]uint64, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id FROM rag_chunks WHERE document_id=? ORDER BY id`, documentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := make([]uint64, 0)
	for rows.Next() {
		var id uint64
		if err = rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (s *Service) queueCleanup(ctx context.Context, operation, collection string, pointIDs []uint64, cause error) error {
	raw, _ := json.Marshal(pointIDs)
	_, err := s.db.ExecContext(ctx, `INSERT INTO rag_vector_cleanup_tasks(operation,collection_name,point_ids_json,status,last_error,next_retry_at) VALUES(?,?,CAST(? AS JSON),'PENDING',?,DATE_ADD(CURRENT_TIMESTAMP,INTERVAL 1 MINUTE))`, operation, collection, string(raw), cause.Error())
	return err
}

func (s *Service) StartVectorCleanupScheduler(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := s.retryVectorCleanup(ctx); err != nil {
					logx.Errorf("RAG vector cleanup retry failed: %v", err)
				}
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (s *Service) retryVectorCleanup(ctx context.Context) error {
	rows, err := s.db.QueryContext(ctx, `SELECT id,operation,collection_name,COALESCE(CAST(point_ids_json AS CHAR),'[]'),retry_count FROM rag_vector_cleanup_tasks WHERE status='PENDING' AND next_retry_at<=CURRENT_TIMESTAMP ORDER BY id LIMIT 50`)
	if err != nil {
		return err
	}
	type task struct {
		id                            uint64
		operation, collection, points string
		retry                         int
	}
	tasks := make([]task, 0)
	for rows.Next() {
		var item task
		if err = rows.Scan(&item.id, &item.operation, &item.collection, &item.points, &item.retry); err != nil {
			rows.Close()
			return err
		}
		tasks = append(tasks, item)
	}
	rows.Close()
	for _, item := range tasks {
		var cleanupErr error
		if item.operation == "DELETE_COLLECTION" {
			cleanupErr = s.qdrant.DeleteCollection(ctx, item.collection)
		} else {
			var ids []uint64
			_ = json.Unmarshal([]byte(item.points), &ids)
			cleanupErr = s.qdrant.DeletePoints(ctx, item.collection, ids)
		}
		if cleanupErr == nil {
			_, err = s.db.ExecContext(ctx, `UPDATE rag_vector_cleanup_tasks SET status='SUCCESS',last_error=NULL,updated_at=CURRENT_TIMESTAMP WHERE id=?`, item.id)
		} else {
			delay := 1 << min(item.retry, 6)
			_, err = s.db.ExecContext(ctx, `UPDATE rag_vector_cleanup_tasks SET retry_count=retry_count+1,last_error=?,next_retry_at=DATE_ADD(CURRENT_TIMESTAMP,INTERVAL ? MINUTE),updated_at=CURRENT_TIMESTAMP WHERE id=?`, cleanupErr.Error(), delay, item.id)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

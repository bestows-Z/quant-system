package domain

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"quant-system/backend-go/apps/rag-rpc/internal/config"
	"quant-system/backend-go/apps/rag-rpc/rag"
	"quant-system/backend-go/common/constants"
	"quant-system/backend-go/common/idgen"
	"quant-system/backend-go/common/trace"
	"quant-system/backend-go/common/xmq"
	"quant-system/backend-go/common/xqdrant"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

const disclaimer = "RAG 结果来自知识库检索与可追溯片段摘要，仅供研究，不构成投资建议"

const systemCollection = "quant_system_research"

var systemDocuments = []struct {
	Title   string
	Content string
}{
	{"A股技术指标研究方法", "移动平均线用于观察不同周期的价格趋势，MA5、MA10、MA20需要结合成交量与市场环境解释，单次交叉不能单独视为交易依据。MACD由快慢指数移动平均线差值DIF、信号线DEA和柱体构成，背离与交叉都应进行样本外检验。RSI用于描述一段窗口内上涨与下跌动能的相对强弱，超买超卖阈值不是确定性反转信号。布林带由中轨均线和波动率带宽组成，突破需要结合带宽、成交量和回归风险评估。所有指标必须避免未来函数，并明确复权方式、计算窗口和数据截止时间。"},
	{"因子研究与回测验证规范", "候选因子必须经过数据清洗、去极值、标准化以及必要的行业和市值中性化。评价指标包括IC、Rank IC、ICIR、覆盖率、换手率、衰减和因子间相关性。策略验证应划分训练、验证和样本外区间，使用滚动前推或带隔离区间的交叉验证，并记录手续费、滑点、涨跌停、停牌和成交容量。不得仅根据单次历史收益挑选参数；最大回撤、收益波动、胜率、盈亏比和稳定性需要联合展示。"},
	{"A股模拟执行与T加一约束", "A股普通股票模拟交易应区分总持仓、可卖持仓、当日买入和冻结数量。当日买入的普通股票在下一交易日才转为可卖数量。委托检查应覆盖可用资金、可卖数量、最小申报单位、手续费、滑点、涨跌停和停牌状态。市价模拟不能任意指定成交价：有盘口时按可成交档位逐档消耗数量，没有盘口时必须明确使用最新价加滑点的简化模型，并标记数据局限。"},
	{"量化研究报告证据规范", "研究结论应分别说明数据面、技术面、策略面、风险面和事件信息的证据来源、截止时间与完整度。行情或资金流不可用时必须明确写明缺失，不得补造数值。语言模型只负责解释和归纳，结构化信号、风险约束和模拟撮合由确定性服务执行。任何候选排序和策略信号仅供研究，不构成投资建议；历史表现不代表未来结果。"},
	{"A股市场与板块交易制度方法库v2", "沪市主板、深市主板、创业板、科创板和北交所证券应按市场、板块、上市状态和特殊处理状态匹配交易规则。价格涨跌幅限制、申报数量、最小价格变动单位、集合竞价和连续竞价时段必须配置化。停牌、退市整理、ST、上市初期和无有效盘口证券不得被普通自动策略视为正常可交易标的。"},
	{"量价与流动性因子方法库v2", "量价研究包括收益率、对数收益率、成交量变化率、换手率、量比、振幅、VWAP、OBV、MFI、Amihud非流动性和成交额冲击。成交量突破必须与价格突破、历史分位数和市场整体成交状态联合解释。免费行情缺少成交额时应保留缺失状态，不能用收盘价乘成交量伪装成真实成交额。"},
	{"盘口微观结构因子方法库v2", "五档盘口可计算买卖量不平衡、加权微价格、最优价差、盘口斜率和逐档深度。盘口快照必须携带行情时间和来源，过期或时间倒序的快照不能用于自动撮合。盘口数量只代表可见流动性，模拟市价单应逐档消耗并允许部分成交，不能假设任意数量都按最新价成交。"},
	{"趋势动量与反转策略方法库v2", "趋势策略可组合均线排列、MACD、ADX、突破和多周期动量；反转策略可使用RSI、布林带偏离、短期收益反转和成交量衰竭。策略必须明确参数、信号方向、持有期、退出条件和适用市场状态。同一指标在趋势和震荡环境下含义可能相反，因此应先进行市场状态识别并做样本外验证。"},
	{"多因子选股与中性化方法库v2", "多因子选股可覆盖价值、质量、成长、盈利、动量、低波动、流动性和情绪因子。截面处理通常包括缺失值管理、去极值、标准化、行业中性化和市值中性化。组合构建需控制行业暴露、单票权重、换手率和交易容量，并保存每次评分使用的因子版本与证据快照。"},
	{"因子IC与生命周期管理方法库v2", "因子评价包括Pearson IC、Spearman Rank IC、IC均值、IC标准差、ICIR、分组收益、单调性、覆盖率、换手率和衰减。因子状态应按候选、影子、晋级、退役管理。自动流程可以生成候选和影子结果，但未经足够样本外观测、稳定性检验和风险门槛不得直接替换已运行策略。"},
	{"回测偏差与稳健性检查方法库v2", "回测必须防止未来函数、幸存者偏差、前视偏差、复权错误和参数过拟合。应使用滚动前推、训练验证测试分离、隔离窗口和多市场状态测试。绩效除收益率外还应展示最大回撤、波动率、Sharpe、Sortino、Calmar、胜率、盈亏比、交易次数、换手率和容量约束。"},
	{"组合风险预算与压力测试方法库v2", "组合风险应覆盖单票集中度、行业集中度、相关性、波动率、Beta、VaR、CVaR、最大回撤和连续亏损。压力测试需考虑指数下跌、行业冲击、流动性下降、涨跌停和行情源中断。风险规则必须先于订单执行，达到最大回撤或数据完整度阈值时自动暂停策略。"},
	{"龙虎榜与市场情绪研究方法库v2", "龙虎榜应同时观察上榜原因、股票级买入卖出金额、净额、总成交占比以及买卖前五席位。机构专用、沪深股通和普通营业部应分类展示。单日席位净买入不等于后续上涨结论，需要结合重复上榜、席位集中度、价格位置、成交量、公告事件和后续样本统计分析。"},
	{"公告新闻与事件因子方法库v2", "公告和资讯应保存标题、正文摘要、发布时间、来源链接、涉及证券和事件类型。事件可分类为业绩、回购、增减持、并购、合同中标、监管、诉讼、停复牌和风险警示，并计算方向、置信度和时间衰减。消息面只能作为可追溯因子输入，不能把媒体标题直接转换成确定性买卖指令。"},
	{"自动模拟交易治理方法库v2", "AI只能生成受限JSON策略DSL和参数候选，不能执行任意代码。候选策略先做语法与指标白名单校验，再完成回测、样本外评估和至少二十个交易日影子运行。已晋级策略由确定性引擎执行，AI不可用时继续运行既有策略；风险越界、数据缺失或盘口过期时无需等待AI即可暂停。"},
	{"模型分析与RAG引用规范v2", "模型提示应包含结构化行情、指标、策略、风控、公告与知识库证据，并限制上下文预算。回答需要区分事实、计算结果和推断，列出引用来源与数据截止时间。模型调用保存供应商、模型、输入输出Token、延迟、状态和报告关联；模型失败时应回退到本地确定性量化报告。"},
}

type Service struct {
	db       *sql.DB
	embedder Embedder
	qdrant   *xqdrant.Client
	producer xmq.Producer
}

type kbRow struct {
	ID                uint64
	UserID            uint64
	Name              string
	Description       string
	Visibility        string
	EmbeddingProvider string
	VectorStore       string
	CollectionName    string
	CreatedAt         time.Time
}

type docRow struct {
	ID              uint64
	KnowledgeBaseID uint64
	UserID          uint64
	Title           string
	DocType         string
	SourceURL       string
	Status          string
	MetadataJSON    string
	ChunkCount      int64
	CreatedAt       time.Time
}

func NewService(db *sql.DB, qdrant *xqdrant.Client, producer xmq.Producer, _ config.Config) *Service {
	return &Service{db: db, embedder: LocalEmbedder{}, qdrant: qdrant, producer: producer}
}

func (s *Service) EnsureSystemKnowledge(ctx context.Context) error {
	if err := s.qdrant.EnsureCollection(ctx, systemCollection, s.embedder.Dimension()); err != nil {
		return fmt.Errorf("初始化系统知识库 Qdrant collection: %w", err)
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO knowledge_bases (user_id,name,description,visibility,embedding_provider,vector_store,collection_name)
VALUES (NULL,'系统量化研究库','内置量化方法、回测规范、A股执行约束与报告证据规范','PUBLIC',?,'qdrant',?)
ON DUPLICATE KEY UPDATE name=VALUES(name),description=VALUES(description),visibility='PUBLIC',embedding_provider=VALUES(embedding_provider),updated_at=CURRENT_TIMESTAMP`, s.embedder.Provider(), systemCollection)
	if err != nil {
		return err
	}
	var kbID uint64
	if err := s.db.QueryRowContext(ctx, `SELECT id FROM knowledge_bases WHERE collection_name=? AND deleted_at IS NULL`, systemCollection).Scan(&kbID); err != nil {
		return err
	}
	for _, item := range systemDocuments {
		var count int
		if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM rag_documents WHERE knowledge_base_id=? AND title=? AND deleted_at IS NULL`, kbID, item.Title).Scan(&count); err != nil {
			return err
		}
		if count > 0 {
			continue
		}
		if _, err := s.CreateDocument(ctx, &rag.CreateDocumentReq{UserId: 0, KnowledgeBaseId: kbID, Title: item.Title, DocType: "system_methodology", Content: item.Content, MetadataJson: `{"scope":"system","verified":true}`}); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) CreateKnowledgeBase(ctx context.Context, in *rag.CreateKnowledgeBaseReq) (*rag.KnowledgeBase, error) {
	name := strings.TrimSpace(in.GetName())
	if in.GetUserId() == 0 || name == "" {
		return nil, status.Error(codes.InvalidArgument, "知识库名称不能为空")
	}
	visibility := strings.ToUpper(strings.TrimSpace(in.GetVisibility()))
	if visibility == "" {
		visibility = "PRIVATE"
	}
	if visibility != "PRIVATE" && visibility != "TEAM" && visibility != "PUBLIC" {
		return nil, status.Error(codes.InvalidArgument, "visibility 只能是 PRIVATE、TEAM 或 PUBLIC")
	}
	collection := fmt.Sprintf("kb_%d_%s", in.GetUserId(), idgen.NewEventID()[:12])
	if err := s.qdrant.EnsureCollection(ctx, collection, s.embedder.Dimension()); err != nil {
		return nil, status.Errorf(codes.Unavailable, "Qdrant collection 创建失败: %v", err)
	}
	res, err := s.db.ExecContext(ctx, `
INSERT INTO knowledge_bases (user_id, name, description, visibility, embedding_provider, vector_store, collection_name)
VALUES (?, ?, ?, ?, ?, 'qdrant', ?)`, in.GetUserId(), name, strings.TrimSpace(in.GetDescription()), visibility, s.embedder.Provider(), collection)
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	s.publish(ctx, constants.EventRagDocumentUploaded, in.GetUserId(), map[string]any{"knowledge_base_id": id, "event": "knowledge_base.created"})
	return s.GetKnowledgeBase(ctx, in.GetUserId(), uint64(id))
}

func (s *Service) ListKnowledgeBases(ctx context.Context, userID uint64) ([]*rag.KnowledgeBase, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, COALESCE(user_id,0), name, COALESCE(description,''), visibility, embedding_provider, vector_store, collection_name, created_at
FROM knowledge_bases
WHERE deleted_at IS NULL AND (user_id = ? OR visibility = 'PUBLIC')
ORDER BY id DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*rag.KnowledgeBase
	for rows.Next() {
		item, err := scanKB(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, kbPB(item))
	}
	return out, rows.Err()
}

func (s *Service) GetKnowledgeBase(ctx context.Context, userID, id uint64) (*rag.KnowledgeBase, error) {
	row, err := s.selectKB(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	return kbPB(row), nil
}

func (s *Service) CreateDocument(ctx context.Context, in *rag.CreateDocumentReq) (*rag.Document, error) {
	if in.GetKnowledgeBaseId() == 0 || strings.TrimSpace(in.GetTitle()) == "" {
		return nil, status.Error(codes.InvalidArgument, "文档参数错误")
	}
	content := strings.TrimSpace(in.GetContent())
	if content == "" {
		return nil, status.Error(codes.InvalidArgument, "文档内容不能为空")
	}
	kb, err := s.selectKB(ctx, in.GetUserId(), in.GetKnowledgeBaseId())
	if err != nil {
		return nil, err
	}
	if kb.UserID != in.GetUserId() {
		return nil, status.Error(codes.PermissionDenied, "公共知识库只读，不能上传用户文档")
	}
	if err := s.qdrant.EnsureCollection(ctx, kb.CollectionName, s.embedder.Dimension()); err != nil {
		return nil, status.Errorf(codes.Unavailable, "Qdrant collection 不可用: %v", err)
	}
	metadata := normalizeJSON(in.GetMetadataJson())
	chunks := splitChunks(content, 800, 100)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	res, err := tx.ExecContext(ctx, `
INSERT INTO rag_documents (knowledge_base_id, user_id, title, doc_type, source_url, status, metadata_json)
VALUES (?, ?, ?, ?, ?, 'INDEXING', ?)`, in.GetKnowledgeBaseId(), in.GetUserId(), strings.TrimSpace(in.GetTitle()), defaultString(in.GetDocType(), "text"), strings.TrimSpace(in.GetSourceUrl()), metadata)
	if err != nil {
		return nil, err
	}
	docID, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	for i, chunk := range chunks {
		chunkMeta := fmt.Sprintf(`{"source":"document-ingest","chunk_index":%d}`, i)
		chunkRes, err := tx.ExecContext(ctx, `
INSERT INTO rag_chunks (document_id, knowledge_base_id, chunk_index, content, token_count, metadata_json)
VALUES (?, ?, ?, ?, ?, ?)`, docID, in.GetKnowledgeBaseId(), i, chunk, approxTokens(chunk), chunkMeta)
		if err != nil {
			return nil, err
		}
		chunkID, err := chunkRes.LastInsertId()
		if err != nil {
			return nil, err
		}
		vector := s.embedder.Embed(chunk)
		vectorJSON, _ := json.Marshal(vector)
		vectorID := fmt.Sprintf("chunk_%d", chunkID)
		if _, err := tx.ExecContext(ctx, `
INSERT INTO rag_embeddings (chunk_id, knowledge_base_id, provider, model, dimension, vector_id, vector_json, status)
VALUES (?, ?, ?, ?, ?, ?, ?, 'PENDING')`, chunkID, in.GetKnowledgeBaseId(), s.embedder.Provider(), s.embedder.Model(), s.embedder.Dimension(), vectorID, string(vectorJSON)); err != nil {
			return nil, err
		}
		if err := s.qdrant.Upsert(ctx, kb.CollectionName, xqdrant.Point{ID: uint64(chunkID), Vector: vector, Payload: map[string]any{
			"knowledge_base_id": in.GetKnowledgeBaseId(), "chunk_id": chunkID,
			"document_id": docID, "document_title": strings.TrimSpace(in.GetTitle()),
			"content": chunk, "metadata_json": chunkMeta,
		}}); err != nil {
			return nil, status.Errorf(codes.Unavailable, "Qdrant 向量写入失败: %v", err)
		}
		if _, err := tx.ExecContext(ctx, `UPDATE rag_embeddings SET status='READY', updated_at=CURRENT_TIMESTAMP WHERE chunk_id=?`, chunkID); err != nil {
			return nil, err
		}
	}
	if _, err := tx.ExecContext(ctx, `UPDATE rag_documents SET status='READY', updated_at=CURRENT_TIMESTAMP WHERE id=?`, docID); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	s.publish(ctx, constants.EventRagDocumentUploaded, in.GetUserId(), map[string]any{"document_id": docID, "knowledge_base_id": in.GetKnowledgeBaseId()})
	s.publish(ctx, constants.EventRagDocumentChunked, in.GetUserId(), map[string]any{"document_id": docID, "chunks": len(chunks)})
	s.publish(ctx, constants.EventRagEmbeddingCompleted, in.GetUserId(), map[string]any{"document_id": docID, "provider": s.embedder.Provider(), "model": s.embedder.Model()})
	return s.GetDocument(ctx, in.GetUserId(), uint64(docID))
}

func (s *Service) ListDocuments(ctx context.Context, userID, kbID uint64) ([]*rag.Document, error) {
	if _, err := s.selectKB(ctx, userID, kbID); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT d.id, d.knowledge_base_id, COALESCE(d.user_id,0), d.title, d.doc_type, COALESCE(d.source_url,''), d.status, COALESCE(CAST(d.metadata_json AS CHAR),'{}'), COUNT(c.id), d.created_at
FROM rag_documents d
LEFT JOIN rag_chunks c ON c.document_id = d.id
WHERE d.deleted_at IS NULL AND d.knowledge_base_id = ?
GROUP BY d.id
ORDER BY d.id DESC`, kbID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*rag.Document
	for rows.Next() {
		item, err := scanDoc(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, docPB(item))
	}
	return out, rows.Err()
}

func (s *Service) GetDocument(ctx context.Context, userID, id uint64) (*rag.Document, error) {
	row, err := s.selectDoc(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	return docPB(row), nil
}

func (s *Service) ListChunks(ctx context.Context, userID, docID uint64) ([]*rag.Chunk, error) {
	if _, err := s.selectDoc(ctx, userID, docID); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT id, document_id, knowledge_base_id, chunk_index, content, token_count, COALESCE(CAST(metadata_json AS CHAR),'{}')
FROM rag_chunks WHERE document_id = ? ORDER BY chunk_index`, docID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*rag.Chunk
	for rows.Next() {
		var item rag.Chunk
		if err := rows.Scan(&item.Id, &item.DocumentId, &item.KnowledgeBaseId, &item.ChunkIndex, &item.Content, &item.TokenCount, &item.MetadataJson); err != nil {
			return nil, err
		}
		out = append(out, &item)
	}
	return out, rows.Err()
}

func (s *Service) Search(ctx context.Context, in *rag.SearchReq) (*rag.SearchResp, error) {
	start := time.Now()
	if in.GetUserId() == 0 || strings.TrimSpace(in.GetQuery()) == "" {
		return nil, status.Error(codes.InvalidArgument, "检索参数错误")
	}
	kbID, err := s.resolveKnowledgeBaseID(ctx, in.GetKnowledgeBaseId())
	if err != nil {
		return nil, err
	}
	kb, err := s.selectKB(ctx, in.GetUserId(), kbID)
	if err != nil {
		return nil, err
	}
	topK := int(in.GetTopK())
	if topK <= 0 || topK > 20 {
		topK = 5
	}
	queryVec := s.embedder.Embed(in.GetQuery())
	if hits, qdrantErr := s.qdrant.Search(ctx, kb.CollectionName, queryVec, topK); qdrantErr == nil {
		results := make([]*rag.SearchResult, 0, len(hits)+topK)
		candidateIDs := make([]uint64, 0, len(hits))
		for _, hit := range hits {
			if id := payloadUint(hit.Payload, "chunk_id"); id > 0 {
				candidateIDs = append(candidateIDs, id)
			}
		}
		activeIDs, activeErr := s.activeChunkIDs(ctx, kbID, candidateIDs)
		if activeErr != nil {
			return nil, activeErr
		}
		for _, hit := range hits {
			chunkID := payloadUint(hit.Payload, "chunk_id")
			if !activeIDs[chunkID] {
				continue
			}
			results = append(results, &rag.SearchResult{
				ChunkId:       chunkID,
				DocumentId:    payloadUint(hit.Payload, "document_id"),
				DocumentTitle: payloadString(hit.Payload, "document_title"),
				Content:       payloadString(hit.Payload, "content"),
				MetadataJson:  defaultString(payloadString(hit.Payload, "metadata_json"), "{}"),
				Score:         hit.Score,
			})
		}
		if keywordHits, keywordErr := s.keywordSearch(ctx, kbID, in.GetQuery(), topK); keywordErr == nil {
			results = mergeSearchResults(results, keywordHits, topK)
		}
		provider := s.embedder.Provider() + "+qdrant"
		s.logSearch(ctx, in.GetUserId(), kbID, in.GetQuery(), topK, provider, results, time.Since(start))
		return &rag.SearchResp{Items: results, Provider: provider, Model: s.embedder.Model(), Disclaimer: disclaimer}, nil
	}

	// MySQL keeps the original vectors as an explicit disaster-recovery path.
	// Normal retrieval always uses Qdrant above.
	rows, err := s.db.QueryContext(ctx, `
SELECT c.id, c.document_id, d.title, c.content, COALESCE(CAST(c.metadata_json AS CHAR),'{}'), COALESCE(CAST(e.vector_json AS CHAR),'[]')
FROM rag_chunks c
JOIN rag_documents d ON d.id = c.document_id
JOIN rag_embeddings e ON e.chunk_id = c.id AND e.status = 'READY'
WHERE c.knowledge_base_id = ? AND d.deleted_at IS NULL`, kbID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var results []*rag.SearchResult
	queryLower := strings.ToLower(in.GetQuery())
	for rows.Next() {
		var item rag.SearchResult
		var vectorJSON string
		if err := rows.Scan(&item.ChunkId, &item.DocumentId, &item.DocumentTitle, &item.Content, &item.MetadataJson, &vectorJSON); err != nil {
			return nil, err
		}
		var vec []float64
		_ = json.Unmarshal([]byte(vectorJSON), &vec)
		score := cosine(queryVec, vec)
		if strings.Contains(strings.ToLower(item.Content), queryLower) {
			score += 0.2
		}
		item.Score = score
		results = append(results, &item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	sort.Slice(results, func(i, j int) bool { return results[i].Score > results[j].Score })
	if len(results) > topK {
		results = results[:topK]
	}
	provider := s.embedder.Provider() + "+mysql-fallback"
	s.logSearch(ctx, in.GetUserId(), kbID, in.GetQuery(), topK, provider, results, time.Since(start))
	return &rag.SearchResp{Items: results, Provider: provider, Model: s.embedder.Model(), Disclaimer: disclaimer}, nil
}

func (s *Service) activeChunkIDs(ctx context.Context, kbID uint64, ids []uint64) (map[uint64]bool, error) {
	active := make(map[uint64]bool, len(ids))
	if len(ids) == 0 {
		return active, nil
	}
	placeholders := make([]string, len(ids))
	args := make([]any, 0, len(ids)+1)
	args = append(args, kbID)
	for index, id := range ids {
		placeholders[index] = "?"
		args = append(args, id)
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT c.id FROM rag_chunks c
JOIN rag_documents d ON d.id=c.document_id AND d.deleted_at IS NULL
JOIN rag_embeddings e ON e.chunk_id=c.id AND e.status='READY'
WHERE c.knowledge_base_id=? AND c.id IN (`+strings.Join(placeholders, ",")+`)`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id uint64
		if err = rows.Scan(&id); err != nil {
			return nil, err
		}
		active[id] = true
	}
	return active, rows.Err()
}

func (s *Service) keywordSearch(ctx context.Context, kbID uint64, query string, limit int) ([]*rag.SearchResult, error) {
	tokens := searchTokens(query, 8)
	if len(tokens) == 0 {
		return nil, nil
	}
	conditions := make([]string, 0, len(tokens))
	args := []any{kbID}
	for _, token := range tokens {
		conditions = append(conditions, `(c.content LIKE ? OR d.title LIKE ?)`)
		pattern := "%" + token + "%"
		args = append(args, pattern, pattern)
	}
	args = append(args, max(limit*4, 20))
	rows, err := s.db.QueryContext(ctx, `SELECT c.id,c.document_id,d.title,c.content,COALESCE(CAST(c.metadata_json AS CHAR),'{}') FROM rag_chunks c JOIN rag_documents d ON d.id=c.document_id WHERE c.knowledge_base_id=? AND d.deleted_at IS NULL AND (`+strings.Join(conditions, " OR ")+`) LIMIT ?`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]*rag.SearchResult, 0)
	for rows.Next() {
		var item rag.SearchResult
		if err := rows.Scan(&item.ChunkId, &item.DocumentId, &item.DocumentTitle, &item.Content, &item.MetadataJson); err != nil {
			return nil, err
		}
		text := strings.ToLower(item.DocumentTitle + " " + item.Content)
		matches := 0
		for _, token := range tokens {
			if strings.Contains(text, strings.ToLower(token)) {
				matches++
			}
		}
		item.Score = 0.45 + 0.5*float64(matches)/float64(len(tokens))
		items = append(items, &item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Score > items[j].Score })
	return items, rows.Err()
}

func mergeSearchResults(vector, keyword []*rag.SearchResult, limit int) []*rag.SearchResult {
	merged := make(map[uint64]*rag.SearchResult, len(vector)+len(keyword))
	for _, item := range vector {
		if item == nil {
			continue
		}
		cloned := proto.Clone(item).(*rag.SearchResult)
		cloned.Score *= 0.7
		merged[item.ChunkId] = cloned
	}
	for _, item := range keyword {
		if item == nil {
			continue
		}
		if existing := merged[item.ChunkId]; existing != nil {
			existing.Score += item.Score * 0.3
		} else {
			cloned := proto.Clone(item).(*rag.SearchResult)
			cloned.Score *= 0.3
			merged[item.ChunkId] = cloned
		}
	}
	items := make([]*rag.SearchResult, 0, len(merged))
	for _, item := range merged {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Score > items[j].Score })
	if len(items) > limit {
		items = items[:limit]
	}
	return items
}

func searchTokens(query string, limit int) []string {
	parts := strings.FieldsFunc(strings.TrimSpace(query), func(r rune) bool {
		return r == ' ' || r == ',' || r == '，' || r == '。' || r == '?' || r == '？' || r == '/' || r == ':' || r == '：'
	})
	seen := map[string]bool{}
	items := make([]string, 0, limit)
	for _, part := range parts {
		part = strings.TrimSpace(part)
		runes := []rune(part)
		if len(runes) < 2 {
			continue
		}
		candidates := []string{part}
		if len(runes) > 6 {
			candidates = candidates[:0]
			for i := 0; i+1 < len(runes); i += 2 {
				end := min(i+4, len(runes))
				candidates = append(candidates, string(runes[i:end]))
			}
		}
		for _, candidate := range candidates {
			if seen[candidate] {
				continue
			}
			seen[candidate] = true
			items = append(items, candidate)
			if len(items) == limit {
				return items
			}
		}
	}
	return items
}

func (s *Service) Ask(ctx context.Context, in *rag.AskReq) (*rag.AskResp, error) {
	start := time.Now()
	kbID, err := s.resolveKnowledgeBaseID(ctx, in.GetKnowledgeBaseId())
	if err != nil {
		return nil, err
	}
	search, err := s.Search(ctx, &rag.SearchReq{UserId: in.GetUserId(), KnowledgeBaseId: kbID, Query: in.GetQuestion(), TopK: in.GetTopK()})
	if err != nil {
		return nil, err
	}
	answer := buildGroundedAnswer(in.GetQuestion(), search.Items)
	refsJSON, _ := json.Marshal(search.Items)
	_, _ = s.db.ExecContext(ctx, `
INSERT INTO rag_qa_logs (knowledge_base_id, user_id, question, answer, references_json, llm_provider, model, latency_ms)
VALUES (?, ?, ?, ?, ?, 'local', 'extractive-rag-v1', ?)`, kbID, in.GetUserId(), in.GetQuestion(), answer, string(refsJSON), time.Since(start).Milliseconds())
	return &rag.AskResp{Answer: answer, References: search.Items, Provider: "local", Model: "extractive-rag-v1", Disclaimer: disclaimer}, nil
}

func (s *Service) resolveKnowledgeBaseID(ctx context.Context, requested uint64) (uint64, error) {
	if requested != 0 {
		return requested, nil
	}
	var id uint64
	err := s.db.QueryRowContext(ctx, `SELECT id FROM knowledge_bases WHERE collection_name=? AND visibility='PUBLIC' AND deleted_at IS NULL LIMIT 1`, systemCollection).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, status.Error(codes.Unavailable, "系统量化知识库尚未初始化")
	}
	return id, err
}

func (s *Service) selectKB(ctx context.Context, userID, id uint64) (*kbRow, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, COALESCE(user_id,0), name, COALESCE(description,''), visibility, embedding_provider, vector_store, collection_name, created_at
FROM knowledge_bases
WHERE id = ? AND deleted_at IS NULL AND (user_id = ? OR visibility = 'PUBLIC')`, id, userID)
	item, err := scanKB(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, status.Error(codes.NotFound, "知识库不存在")
	}
	return item, err
}

func (s *Service) selectDoc(ctx context.Context, userID, id uint64) (*docRow, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT d.id, d.knowledge_base_id, COALESCE(d.user_id,0), d.title, d.doc_type, COALESCE(d.source_url,''), d.status, COALESCE(CAST(d.metadata_json AS CHAR),'{}'), COUNT(c.id), d.created_at
FROM rag_documents d
JOIN knowledge_bases kb ON kb.id = d.knowledge_base_id
LEFT JOIN rag_chunks c ON c.document_id = d.id
WHERE d.id = ? AND d.deleted_at IS NULL AND kb.deleted_at IS NULL AND (kb.user_id = ? OR kb.visibility = 'PUBLIC')
GROUP BY d.id`, id, userID)
	item, err := scanDoc(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, status.Error(codes.NotFound, "文档不存在")
	}
	return item, err
}

func (s *Service) logSearch(ctx context.Context, userID, kbID uint64, query string, topK int, provider string, results []*rag.SearchResult, latency time.Duration) {
	payload, _ := json.Marshal(results)
	_, _ = s.db.ExecContext(ctx, `
INSERT INTO rag_retrieval_logs (knowledge_base_id, user_id, query_text, top_k, provider, results_json, latency_ms)
VALUES (?, ?, ?, ?, ?, ?, ?)`, kbID, userID, query, topK, provider, string(payload), latency.Milliseconds())
}

func (s *Service) publish(ctx context.Context, eventType string, userID uint64, payload map[string]any) {
	if s.producer == nil {
		return
	}
	_ = s.producer.Publish(ctx, xmq.Event{EventID: idgen.NewEventID(), EventType: eventType, TraceID: trace.FromContext(ctx), UserID: userID, Payload: payload, CreatedAt: time.Now()})
}

type scanner interface{ Scan(dest ...any) error }

func scanKB(row scanner) (*kbRow, error) {
	var item kbRow
	if err := row.Scan(&item.ID, &item.UserID, &item.Name, &item.Description, &item.Visibility, &item.EmbeddingProvider, &item.VectorStore, &item.CollectionName, &item.CreatedAt); err != nil {
		return nil, err
	}
	return &item, nil
}

func scanDoc(row scanner) (*docRow, error) {
	var item docRow
	if err := row.Scan(&item.ID, &item.KnowledgeBaseID, &item.UserID, &item.Title, &item.DocType, &item.SourceURL, &item.Status, &item.MetadataJSON, &item.ChunkCount, &item.CreatedAt); err != nil {
		return nil, err
	}
	return &item, nil
}

func kbPB(item *kbRow) *rag.KnowledgeBase {
	return &rag.KnowledgeBase{Id: item.ID, UserId: item.UserID, Name: item.Name, Description: item.Description, Visibility: item.Visibility, EmbeddingProvider: item.EmbeddingProvider, VectorStore: item.VectorStore, CollectionName: item.CollectionName, CreatedAt: item.CreatedAt.Format("2006-01-02 15:04:05")}
}

func docPB(item *docRow) *rag.Document {
	return &rag.Document{Id: item.ID, KnowledgeBaseId: item.KnowledgeBaseID, UserId: item.UserID, Title: item.Title, DocType: item.DocType, SourceUrl: item.SourceURL, Status: item.Status, MetadataJson: item.MetadataJSON, ChunkCount: item.ChunkCount, CreatedAt: item.CreatedAt.Format("2006-01-02 15:04:05")}
}

func splitChunks(content string, size, overlap int) []string {
	runes := []rune(content)
	if len(runes) <= size {
		return []string{content}
	}
	var chunks []string
	for start := 0; start < len(runes); {
		end := start + size
		if end > len(runes) {
			end = len(runes)
		}
		chunks = append(chunks, strings.TrimSpace(string(runes[start:end])))
		if end == len(runes) {
			break
		}
		start = end - overlap
		if start < 0 {
			start = 0
		}
	}
	return chunks
}

func approxTokens(text string) int {
	runes := utf8.RuneCountInString(text)
	if runes == 0 {
		return 0
	}
	return (runes + 1) / 2
}

func normalizeJSON(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "{}"
	}
	var raw any
	if json.Unmarshal([]byte(value), &raw) != nil {
		return "{}"
	}
	return value
}

func defaultString(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func buildGroundedAnswer(question string, refs []*rag.SearchResult) string {
	if len(refs) == 0 {
		return "未在当前知识库中检索到足够相关的上下文。请补充文档后重试。仅供研究，不构成投资建议。"
	}
	var b strings.Builder
	b.WriteString("基于当前知识库的检索摘要：")
	b.WriteString("问题「")
	b.WriteString(question)
	b.WriteString("」可参考以下上下文：")
	for i, ref := range refs {
		if i >= 3 {
			break
		}
		snippet := []rune(strings.TrimSpace(ref.Content))
		if len(snippet) > 120 {
			snippet = snippet[:120]
		}
		b.WriteString(fmt.Sprintf("\n%d. %s", i+1, string(snippet)))
	}
	b.WriteString("\n以上内容由可追溯知识片段提取汇总，仅供研究，不构成投资建议。")
	return b.String()
}

func payloadString(payload map[string]any, key string) string {
	value, ok := payload[key]
	if !ok || value == nil {
		return ""
	}
	if text, ok := value.(string); ok {
		return text
	}
	return fmt.Sprint(value)
}

func payloadUint(payload map[string]any, key string) uint64 {
	switch value := payload[key].(type) {
	case float64:
		return uint64(value)
	case float32:
		return uint64(value)
	case int:
		return uint64(value)
	case int64:
		return uint64(value)
	case uint64:
		return value
	case json.Number:
		parsed, _ := value.Int64()
		return uint64(parsed)
	default:
		return 0
	}
}

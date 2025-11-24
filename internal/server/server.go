package server

import (
	"context"
	"fmt"
	"net"

	"google.golang.org/grpc"

	"github.com/hainn191297/myDb/internal/config"
	"github.com/hainn191297/myDb/internal/logging"
	"github.com/hainn191297/myDb/internal/schema"
	"github.com/hainn191297/myDb/internal/server/session"
	"github.com/hainn191297/myDb/internal/sql/executor"
	"github.com/hainn191297/myDb/internal/sql/parser"
	"github.com/hainn191297/myDb/internal/sql/planner"
	"github.com/hainn191297/myDb/internal/storage/provider"
	"github.com/hainn191297/myDb/internal/txn"

	pb "github.com/hainn191297/myDb/api/proto"
)

// Server implements MyDBService gRPC server
type Server struct {
	pb.UnimplementedMyDBServiceServer

	cfg      config.Config
	sessMgr  *session.Manager
	txnMgr   *txn.Manager
	catalog  *schema.Catalog
	provider *provider.Provider
	grpcSrv  *grpc.Server
}

// New constructs a server with default middleware wired.
func New(cfg config.Config) (*Server, error) {
	prov, err := provider.New(cfg.DataDir, cfg.BufferPoolPages)
	if err != nil {
		return nil, fmt.Errorf("server: init storage provider: %w", err)
	}
	cat, err := prov.LoadCatalog(context.Background())
	if err != nil {
		return nil, fmt.Errorf("server: load catalog: %w", err)
	}
	// Propagate catalog back into provider so engines can leverage schema info
	prov.SetCatalog(cat)

	logging.Infof("server initialized with data dir %s", cfg.DataDir)
	return &Server{
		cfg:      cfg,
		sessMgr:  session.NewManager(cfg.MaxSessions, cfg.IdleSessionExpiry),
		txnMgr:   txn.NewManager(prov, prov),
		catalog:  cat,
		provider: prov,
	}, nil
}

// ExecuteSQL implements the gRPC ExecuteSQL method
func (s *Server) ExecuteSQL(ctx context.Context, req *pb.ExecuteSQLRequest) (*pb.ExecuteSQLResponse, error) {
	// 1. Parse SQL
	ast, err := parser.Parse(req.Sql)
	if err != nil {
		return &pb.ExecuteSQLResponse{
			Result: &pb.ExecuteSQLResponse_Error{
				Error: &pb.ErrorResult{
					Message: err.Error(),
					Code:    "PARSE_ERROR",
				},
			},
		}, nil
	}

	// 2. Build plan
	plan, err := planner.Build(ast, s.catalog)
	if err != nil {
		return &pb.ExecuteSQLResponse{
			Result: &pb.ExecuteSQLResponse_Error{
				Error: &pb.ErrorResult{
					Message: err.Error(),
					Code:    "PLAN_ERROR",
				},
			},
		}, nil
	}

	// 3. Session Management
	var sess session.Session
	var errSess error

	if req.SessionId == "" {
		sess, errSess = s.sessMgr.Create(ctx)
	} else {
		sess, errSess = s.sessMgr.Get(req.SessionId)
		if errSess != nil {
			// If session not found (expired/invalid), create new one
			logging.Warnf("session %s not found, creating new one", req.SessionId)
			sess, errSess = s.sessMgr.Create(ctx)
		}
	}
	if errSess != nil {
		return &pb.ExecuteSQLResponse{
			Result: &pb.ExecuteSQLResponse_Error{
				Error: &pb.ErrorResult{
					Message: "failed to manage session: " + errSess.Error(),
					Code:    "SESSION_ERROR",
				},
			},
		}, nil
	}
	s.sessMgr.Touch(sess.ID)

	// 4. Create executor
	sessionTxn := &executor.SessionTxn{Current: sess.TxnState}
	exec := executor.New(plan, executor.Options{
		Catalog:    s.catalog,
		Provider:   s.provider,
		TxnManager: s.txnMgr,
		SessionTxn: sessionTxn,
	})
	defer exec.Close()

	// 5. Format response based on statement type
	resp, err := s.formatResponse(ctx, exec, ast.Type)

	// Update session with new txn state
	sess.TxnState = sessionTxn.Current
	s.sessMgr.Update(sess)

	if resp != nil {
		resp.SessionId = sess.ID
	}

	return resp, err
}

// formatResponse converts executor results to protobuf response
func (s *Server) formatResponse(ctx context.Context, exec *executor.Executor, stmtType parser.StatementType) (*pb.ExecuteSQLResponse, error) {
	switch stmtType {
	case parser.SelectStmt:
		return s.formatQueryResult(ctx, exec)

	case parser.InsertStmt, parser.UpdateStmt, parser.DeleteStmt,
		parser.CreateTableStmt, parser.DropTableStmt,
		parser.CreateIndexStmt, parser.DropIndexStmt,
		parser.BeginStmt, parser.CommitStmt, parser.RollbackStmt:
		return s.formatCommandResult(ctx, exec, stmtType)

	default:
		return &pb.ExecuteSQLResponse{
			Result: &pb.ExecuteSQLResponse_Error{
				Error: &pb.ErrorResult{
					Message: fmt.Sprintf("unsupported statement type: %s", stmtType),
					Code:    "UNSUPPORTED_STATEMENT",
				},
			},
		}, nil
	}
}

// formatQueryResult formats SELECT query results
func (s *Server) formatQueryResult(ctx context.Context, exec *executor.Executor) (*pb.ExecuteSQLResponse, error) {
	var rows []*pb.Row
	var columns []string

	for {
		hasNext, err := exec.Next(ctx)
		if err != nil {
			return &pb.ExecuteSQLResponse{
				Result: &pb.ExecuteSQLResponse_Error{
					Error: &pb.ErrorResult{
						Message: err.Error(),
						Code:    "EXECUTION_ERROR",
					},
				},
			}, nil
		}
		if !hasNext {
			break
		}

		row := exec.Row()
		if len(columns) == 0 {
			columns = row.Columns
		}

		// Build row values
		values := make([][]byte, len(row.Values))
		copy(values, row.Values)
		rows = append(rows, &pb.Row{Values: values})
	}

	return &pb.ExecuteSQLResponse{
		Result: &pb.ExecuteSQLResponse_QueryResult{
			QueryResult: &pb.QueryResult{
				Columns: columns,
				Rows:    rows,
			},
		},
	}, nil
}

// formatCommandResult formats DDL/DML command results
func (s *Server) formatCommandResult(ctx context.Context, exec *executor.Executor, stmtType parser.StatementType) (*pb.ExecuteSQLResponse, error) {
	// Execute the command (DDL/DML operations complete on first Next())
	_, err := exec.Next(ctx)
	if err != nil {
		return &pb.ExecuteSQLResponse{
			Result: &pb.ExecuteSQLResponse_Error{
				Error: &pb.ErrorResult{
					Message: err.Error(),
					Code:    "EXECUTION_ERROR",
				},
			},
		}, nil
	}

	// Generate appropriate success message
	var message string
	switch stmtType {
	case parser.InsertStmt:
		message = "INSERT 1"
	case parser.UpdateStmt:
		message = "UPDATE 1"
	case parser.DeleteStmt:
		message = "DELETE 1"
	case parser.CreateTableStmt:
		message = "CREATE TABLE"
	case parser.DropTableStmt:
		message = "DROP TABLE"
	case parser.CreateIndexStmt:
		message = "CREATE INDEX"
	case parser.DropIndexStmt:
		message = "DROP INDEX"
	case parser.BeginStmt:
		message = "BEGIN"
	case parser.CommitStmt:
		message = "COMMIT"
	case parser.RollbackStmt:
		message = "ROLLBACK"
	default:
		message = "OK"
	}

	return &pb.ExecuteSQLResponse{
		Result: &pb.ExecuteSQLResponse_CommandResult{
			CommandResult: &pb.CommandResult{
				Message:      message,
				RowsAffected: 1, // TODO: Track actual row count
			},
		},
	}, nil
}

// GetMetadata implements the gRPC GetMetadata method
func (s *Server) GetMetadata(ctx context.Context, req *pb.GetMetadataRequest) (*pb.GetMetadataResponse, error) {
	schemaName := req.Schema
	if schemaName == "" {
		schemaName = "public" // Default schema
	}

	allTables := s.catalog.ListTables()
	var tableInfos []*pb.TableInfo

	for _, tableDef := range allTables {
		// Filter by schema
		if tableDef.Schema != schemaName {
			continue
		}

		tableInfo := &pb.TableInfo{
			Schema:  tableDef.Schema,
			Name:    tableDef.Table,
			Columns: make([]*pb.ColumnInfo, len(tableDef.Columns)),
		}

		for i, col := range tableDef.Columns {
			tableInfo.Columns[i] = &pb.ColumnInfo{
				Name:       col.Name,
				Type:       col.Type.String(),
				Nullable:   col.Nullable,
				PrimaryKey: col.PrimaryKey,
			}
		}

		// Get indexes
		indexes, err := s.catalog.GetIndexes(schemaName, tableDef.Table)
		if err != nil {
			logging.Warnf("failed to get indexes for table %s.%s: %v", schemaName, tableDef.Table, err)
		}
		tableInfo.Indexes = make([]*pb.IndexInfo, 0, len(indexes))
		for _, idx := range indexes {
			tableInfo.Indexes = append(tableInfo.Indexes, &pb.IndexInfo{
				Name:    idx.IndexName,
				Columns: idx.Columns,
				Unique:  idx.Unique,
			})
		}

		tableInfos = append(tableInfos, tableInfo)
	}

	return &pb.GetMetadataResponse{
		Tables: tableInfos,
	}, nil
}

// Start starts the gRPC server
func (s *Server) Start(ctx context.Context) error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.cfg.GRPCPort))
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	s.grpcSrv = grpc.NewServer(
		grpc.MaxRecvMsgSize(10 * 1024 * 1024), // 10MB max message size
	)
	pb.RegisterMyDBServiceServer(s.grpcSrv, s)

	// Graceful shutdown
	go func() {
		<-ctx.Done()
		logging.Warnf("shutting down gRPC server")
		s.grpcSrv.GracefulStop()
		if err := s.provider.Close(); err != nil {
			logging.Errorf("error closing provider: %v", err)
		}
	}()

	logging.Infof("gRPC server listening on :%d", s.cfg.GRPCPort)
	return s.grpcSrv.Serve(lis)
}

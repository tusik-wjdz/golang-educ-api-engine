package psqlservice
import (
	"context"
)

type IEntity interface {
    GetTableName() 		string
    GetID() 			int
    GetColumns() 		[]string
    GetValues() 		[]any
    GetKeyValuePair() 	map[string]any
    SetID(id int)
}

type IDatabaseDriver[PT IEntity] interface {
    DoFindByRawQuery(ctx context.Context, rawSql string, limit int, offset int, vals... any) ([]PT, error)
    DoFindById(ctx context.Context, id int) (PT, bool)
    DoFindByCriteria(ctx context.Context, c map[string]map[string]string, limit int, offset int) ([]PT, error)
    DoBatchUpdateTx(ctx context.Context, data []PT) error
    DoBatchInsertTx(ctx context.Context, data []PT) ([]int, error)
    DoBatchDelete(ctx context.Context, data []PT, ignoreUnsaved bool) (int64, error) 
    DoExecRawQuery(ctx context.Context, q string, args... any) QueryResult[PT]
    DoRawQuery(ctx context.Context, q string, useStruct bool, args... any) QueryResult[PT]
    RunInTransaction(ctx context.Context, fn func(ctx context.Context) error) error    
    GetRelatedTableName() string
    GetColumnsNames() []string
    GetDriver() DbEnv // conn. driver only (.driver)
    GetConnectionPool() DbEnv // connection pool (.pool)
	GetDbEnv() DbEnv // whole ENV
    Close()
}
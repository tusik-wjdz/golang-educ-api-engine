package psqlservice

import (
    "context"
    "fmt"
    "strconv"
    "strings"
    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgxpool"
)

/**
 * PsqlService is prototype of an quasi-adapter between PGX and various implementation
 * of SQL data manipulations patterns.
 * Usable in many scenarios (thanks to interface-based architecture)
 * [IDatabaseDriver/IEntity]
 *
 * todo:
 *  -> connection close issues
 *  -> security issues 
 *
 * Author: W.Dzieciol
 **/

type DbEnv struct {
    conn    *pgx.Conn
    pool    *pgxpool.Pool
}

// psql service
type PsqlService[T any, PT interface{
    *T
    IEntity
}] struct {
    // connection pool
    pool 			*pgxpool.Pool
    // related table name
    tableName		string
    columnsNames    []string
}

// transaction key
type txKey struct {}

// Query Result DTO
type QueryResult[PT IEntity] struct {
    AsMap           []map[string]any // slice of maps      (pointer)
    AsStructPtr     []PT // slice of pointers to sructures (pointer)
    RowsAffected    int64
    Ok              bool
    ErrMsg          string
}

func (db *PsqlService[T, PT]) Close() {
    db.pool.Close()
}

func GetPsqlService[T any, PT interface{
    *T
    IEntity
}](p *pgxpool.Pool) *PsqlService[T, PT] {
    // we need "dummy struct"
    var genericEntity T
    // PT is a pointer to T type so...
    entity := PT(&genericEntity)
    return &PsqlService[T, PT]{
        pool:           p,
        tableName:      entity.GetTableName(),
        columnsNames:   entity.GetColumns(),
    }
}

// region `for interface ...`
func (db *PsqlService[T, PT]) GetRelatedTableName() string {
    return db.tableName
}

func (db *PsqlService[T, PT]) GetColumnsNames() []string {
    return db.columnsNames
}

func (db *PsqlService[T, PT]) GetDriver() DbEnv {
    return DbEnv{conn: nil, pool: db.pool}
}	

func (db *PsqlService[T, PT]) GetConnectionPool() DbEnv {
    return DbEnv{conn: nil, pool: db.pool}
}

func (db *PsqlService[T, PT]) GetDbEnv() DbEnv {
    return DbEnv{conn: nil, pool: db.pool}
}
// endregion

// region TX
func RunInTx(ctx context.Context, pool *pgxpool.Pool, fn func(ctx context.Context) error) error {
    if tryGetTxFromContext(ctx) != nil {
        // ongoing tx - nothing to do
        return fn(ctx)
    }
    tx, err := pool.Begin(ctx)
    if err != nil {
        return err
    }
    newCtx := context.WithValue(ctx, txKey{}, tx)
    defer tx.Rollback(ctx)
    err = fn(newCtx)
    if err != nil {
        // defer will do the job with rollback
        return err
    }
    // final commit
    return tx.Commit(ctx)
}
// try find ongoing transaction in context
func tryGetTxFromContext(ctx context.Context) pgx.Tx {
    if tx, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
        return tx
    }
    return nil
}
// run in TX (method)
func (db *PsqlService[T, PT]) RunInTransaction(
    ctx context.Context,
    fn func(ctx context.Context) error,
) error {
    if tryGetTxFromContext(ctx) != nil {
        // ongoing tx - nothing to do
        return fn(ctx)
    }
    tx, err := db.pool.Begin(ctx)
    if err != nil {
        return err
    }
    newCtx := context.WithValue(ctx, txKey{}, tx)
    defer tx.Rollback(ctx)
    err = fn(newCtx)
    if err != nil {
        // defer will do the job with rollback
        return err
    }
    // final commit
    return tx.Commit(ctx)
}
// endregion
// todo: case sensitive / insensitive
// todo: create syntax pre-check
func (db *PsqlService[T, PT]) buildCriteria(
    c map[string]map[string]string,
) (string, []any, error) {
    var stmt strings.Builder
    input 	:= make([]string, 0)
    n 		:= 1
    fn 		:= func(sub *map[string]string, operator string, n *int) string {
        pattern := "%s%s%s "
        stmt := ""
        // small preparation
        op, cl := "", ""
        if operator == "or" {
            if *n > 1 {
                op, cl = "OR (", ")" 
            } else {
                op, cl = "(", ")"
            }
        }
        for fieldWithCond, inputVal := range *sub {
            // todo: case for *n < 1
            if *n > 1 {
                if operator == "or" {
                    if stmt != "" {
                        stmt += "OR "
                    }
                } else {
                    stmt += "AND "
                }
            }
            // in case of check against NULL
            if strings.ToUpper(inputVal) == "NULL" {
                stmt += fieldWithCond + " NULL"
                continue
            }
            // in any other case
            input = append(input, inputVal)
            stmt += fieldWithCond + " $" + strconv.Itoa(*n) + " "
            *n++
        }
        return fmt.Sprintf(pattern, op, stmt, cl)
    }

    operator 	:= ""
    directives  := []string{"where", "orWhere", "whereIn", "whereNotIn", "like", "orLike","order"}
    for _, d := range directives {
        cInfo, ok := c[d]
        if !ok {
            continue
        }
        if strings.Contains(d, "or") {
            // change operator if required
            operator = "or"
        } else {
            // defualt
            operator = "and"
        }
        // switch over directives
        switch(d) {
        case "where", "orWhere":
            if n == 1 {
                stmt.WriteString("WHERE ")
            }
            stmt.WriteString(fn(&cInfo, operator, &n))
        case "like", "orLike":
            if n == 1 {
                stmt.WriteString("LIKE ")
            }
            stmt.WriteString(fn(&cInfo, operator, &n))
        case "whereIn", "whereNotIn": // orWherIn orWhereNotIn
            tmpStmt := ""
            pattern := ""
            if d == "whereNotIn" {
                pattern = "%s NOT IN (%s) "
            } else {
                pattern = "%s IN (%s) "
            }
            for find, set := range cInfo {
                if len(set) < 1 {
                    return "", nil, fmt.Errorf(
                        "Invalid syntax for whereIn/wherNotIn. At least one expression is required.",
                    )
                }
                if n > 1 {
                    if strings.Contains(d, "or") {
                        tmpStmt = "OR " + pattern
                    } else {
                        tmpStmt = "AND " + pattern			
                    }
                } else {
                    tmpStmt = "WHERE " + pattern
                }
                // append set
                whereInSet := strings.Split(set, ",")
                input       = append(input, whereInSet...)
                fmt.Fprintf(
                    &stmt,
                    tmpStmt,
                    find,
                    generatePlaceHolders(len(whereInSet), n),
                )
                n = n + (len(whereInSet))
            }
        case "order":
            pattern := " ORDER BY %s %s"
            for by, dir := range cInfo {
                switch(dir) {
                case "desc": dir = "DESC"
                case "asc": dir = "ASC"
                default:
                    // we must break it
                    return "", nil, fmt.Errorf("Unknown order direction")
                }
                stmt.WriteString(fmt.Sprintf(pattern, by, dir))
            }
        }
    }
    queryStr 	:= stmt.String()
    output 		:= make([]any, len(input))
    // prepare output
    for i, e 	:= range input {
        output[i] = e
    }
    return queryStr, output, nil
}
// find by ID
func (db *PsqlService[T, PT]) DoFindById(ctx context.Context, id int) (PT, bool) {	
    idStr := strconv.Itoa(id)
    rawQuery := fmt.Sprintf("SELECT * FROM %s WHERE id = %s", db.tableName, idStr)
    results, err := db.DoFindByRawQuery(ctx, rawQuery, 1, 0)
    if err != nil  {
        return nil, false
    }
    if len(results) != 1 {
        return nil, false
    }
    return results[0], true
}
// advanced search via specified filters (c param -> map[string]map[string]string)
func (db *PsqlService[T, PT]) DoFindByCriteria(
    ctx context.Context,
    c map[string]map[string]string,
    limit int,
    offset int,
) ([]PT, error) {
    stmt, args, err := db.buildCriteria(c)
    if err != nil {
        return []PT{}, fmt.Errorf("Unable to build criteria based on passed map. Reason: %s", err)
    }
    rawQuery := fmt.Sprintf("SELECT * FROM %s", db.tableName)
    if stmt != "" {
        rawQuery += " " + stmt
    }
    return db.DoFindByRawQuery(ctx, rawQuery, limit, offset, args...)
}
// simple find by `raw` query
func (db *PsqlService[T, PT]) DoFindByRawQuery(
    ctx context.Context,
    rawSql string,
    limit int,
    offset int,
    vals... any,
) ([]PT, error) {
    // check limit
    if limit != 0 { // todo: check contains limit / offset already
        rawSql += " LIMIT " + strconv.Itoa(limit)
    }
    // check offset
    if offset != 0 {
        rawSql += " OFFSET " + strconv.Itoa(offset)
    }
    // do query
    rows, err := db.pool.Query(ctx, rawSql, vals...)
    if err != nil {
        return nil, fmt.Errorf("Unable to fetch row: %w", err)
    }
    defer rows.Close()
    rawResults, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[T])
    if err != nil {
        return nil, fmt.Errorf("Error while mapping rows to struct: %w", err)
    }
    // prepare results
    results := make([]PT, len(rawResults))
    for n, result := range rawResults {
        results[n] = PT(result)
    }	
    return results, nil
}
// method for batch insert in TX (with ongoing TX detection)
func (db *PsqlService[T, PT]) DoBatchInsertTx(ctx context.Context, data []PT) ([]int, error) {
    // get transaction status
    tx 			:= tryGetTxFromContext(ctx)
    ongoingTx 	:= (tx != nil)

    var err error
    // check tx
    if !ongoingTx {
        tx, err = db.pool.Begin(ctx)
        if err != nil {
            return nil, err
        }
        defer tx.Rollback(ctx)
    }

    ids 	:= make([]int, 0)
    batch 	:= &pgx.Batch{}
    for _, entity := range data {
        // get columns for each entity (could be any struct implement IEntity interface)
        columns := entity.GetColumns()
        // get values
        values 	:= entity.GetValues()
        // check
        if len(columns) != len(values) {
            return nil, fmt.Errorf("Defines columns does not equal passed values.")
        }
        columnsStr := strings.Join(columns, ",")
        // todo: count columns / values (must be same value)
        queryStr := fmt.Sprintf(
            "INSERT INTO %s (%s) VALUES (%s) RETURNING id", entity.GetTableName(),
            columnsStr,
            generatePlaceHolders(len(values), 1),
        )
        batch.Queue(queryStr, values...)
    }
    // send batch
    results := tx.SendBatch(ctx, batch)
    defer results.Close()
    // fetch IDs
    for i := 0; i < len(data); i++ {
        var realID int
        err = results.QueryRow().Scan(&realID)
        if err != nil {
            return nil, fmt.Errorf("Error occurred while trying to fetch row %d -> %w", i, err)
        }
        // assign fetched ID
        data[i].SetID(realID)
        ids = append(ids, realID)
    }
    // close batch
    results.Close()
    // close our "nested" tx (second cond. just for linter)
    if !ongoingTx && tx != nil {
        return ids, tx.Commit(ctx)
    }
    return ids, nil
}
// method for batch update in TX (with ongoing TX detection)
func (db *PsqlService[T, PT]) DoBatchUpdateTx(ctx context.Context, data []PT) error {
    // get tx status
    tx 			:= tryGetTxFromContext(ctx)    
    ongoingTx 	:= (tx != nil)

    var err error
    // check tx
    if !ongoingTx {
        tx, err = db.pool.Begin(ctx)
        if err != nil {
            return err
        }
        defer tx.Rollback(ctx)
    }
    
    batch := &pgx.Batch{}
    for _, entity := range data {
        // get columns
        columns := entity.GetColumns()
        // must be recalculated of each entity
        numberOfColumns := len(columns)
        if numberOfColumns != len(entity.GetValues()) {
            return fmt.Errorf("Number of columns doesn't match to passed params -> %w", err)
        }
        // check number of columns and values
        setStmt := make([]string, 0)
        for i := 0; i < numberOfColumns; i++ {
            // todo: escaping strings
            // todo: check against original set
            setStmt = append(setStmt, fmt.Sprintf("%s = $%d", columns[i], i + 1))
        }
        setStmtStr := strings.Join(setStmt, ",")
        // todo: count columns / values (must be same value)
        queryStr := fmt.Sprintf(
            "UPDATE %s SET %s WHERE id = %d;",
            entity.GetTableName(),
            setStmtStr,
            entity.GetID(), // todo: consider extra where 
        )
        // todo: debug option in PsqlService        
        batch.Queue(queryStr, entity.GetValues()...)
    }
    // send batch
    results := tx.SendBatch(ctx, batch)
    defer results.Close()
    // check results 
    for i := 0; i < len(data); i++  {
        _, err := results.Exec()
        if err != nil {
            return fmt.Errorf("Error occurred while trying to update row: %d -> %w", i, err)
        }
    }
    results.Close()
    // close our "nested" tx (second cond. just for linter)
    if ! ongoingTx && tx != nil {
        return tx.Commit(ctx)
    }
    return nil
}
// batch delete (also usable for single query without lost of performance)
func (db *PsqlService[T, PT]) DoBatchDelete(
    ctx context.Context,
    data []PT,
    ignoreUnsaved bool,
) (int64, error) {
    var err error
    // entity IDs counter
    IDs         := make([]string, len(data))
    // related SQL tablename from property (if set)
    tableName   := db.tableName    
    // collect entities IDs
    for i, e := range data {
        id := e.GetID()
        if id < 1 && ignoreUnsaved {
            // just do nothing
            continue
        } else if (id < 1) {
            // yell !
            return 0, fmt.Errorf("Unsaved entity detected... [index in passed set: %d]", i)
        }
        // just in case
        if tableName == "" {
            tableName = e.GetTableName()
        }
        IDs[i] = strconv.Itoa(id)
    }
    // prepare ids set
    idsStr      := strings.Join(IDs, ",")
    // prepare DELETE statement
    query       := fmt.Sprintf("DELETE FROM %s WHERE id IN (%s)", tableName, idsStr)
    // try exec
    cmdTag, err := db.pool.Exec(ctx, query)
    if(err != nil) {
        return 0, fmt.Errorf("Unable to DELETE passed set: %v", err)
    }
    // otherwise get affected rows counter
    ra          := cmdTag.RowsAffected()
    // clean up - we have to "detach" ID from each entity stored in slice,
    // but do not touch anything else
    for _, e := range data {
        e.SetID(0)
    }
    return ra, nil
}
// delete inc. criteria created by builder in method above
func (db *PsqlService[T, PT]) DoDeleteByCriteria(
    ctx context.Context,
    c map[string]map[string]string,
) (int64, error) {
    stmt, args, err := db.buildCriteria(c)
    if err != nil {
        return 0, fmt.Errorf("Unable to build criteria based on passed map. Reason: %s", err)
    }
    delQuery := fmt.Sprintf("DELETE FROM %s", db.tableName)
    // check...
    if stmt != "" && len(args) > 0 {
        delQuery += " " + stmt
    } else {
        return 0, fmt.Errorf("Empty criteria... ")
    }
    // if ok try exec
    cmdTag, err := db.pool.Exec(ctx, delQuery, args...)
    if err != nil {
        // db err, pass to "upper" layer
        return 0, err
    }
    ra := cmdTag.RowsAffected()
    return ra, nil
}

// region handle_raw_queries
// simple query, returns QueryResult DTO without fetching any data (e.g. SELECT, DELETE)
func (db *PsqlService[T, PT]) DoExecRawQuery(
    ctx context.Context,
    q string,
    args... any,
) QueryResult[PT] {
    cmdTag, err := db.pool.Exec(ctx, q, args...)
    ra := cmdTag.RowsAffected()
    if err != nil {
        // debug mode for detailed info
        return QueryResult[PT]{
            Ok:     false,
            ErrMsg: fmt.Sprintf("Unable to exec command: %v", err),
        }
    }
    // return in QueryResult DTO
    return QueryResult[PT]{
        Ok:             true,
        RowsAffected:   ra,
    }
}
// simple query, returns QueryResult DTO (returs data as map or struct)
func (db *PsqlService[T, PT]) DoRawQuery(
    ctx context.Context,
    q string,
    useStruct bool,
    args... any,
) QueryResult[PT] {
    // trying to exec query
    rows, err := db.pool.Query(ctx, q, args...)
    if err != nil {
        return QueryResult[PT]{
            Ok:             true,
            ErrMsg:         fmt.Sprintf("Unable to exec query: %v", err),
        }
    }
    // don't forget close the door...
    defer rows.Close()
    // use structs (service lvl. defined) ...
    if useStruct {
        res, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[T])
        // handle error
        if err != nil {
            return QueryResult[PT]{
                Ok:         false,
                ErrMsg:     fmt.Sprintf("Unable to exec query (while mapping to struct): %v", err),
            }
        }
        // prepare target slice
        results := make([]PT, len(res))
        // fill it with enity pointers (`generic` cast)
        for n, e := range res {
            results[n] = PT(e) // use pointer instead of copy of entity
        }
        return QueryResult[PT]{
            Ok:             true,
            AsStructPtr:    results,
        }
    // or just a raw map
    } else {
        results, err := pgx.CollectRows(rows, pgx.RowToMap)
        // handle error
        if err != nil {
            return QueryResult[PT]{
                Ok:         false,
                ErrMsg:     fmt.Sprintf("Unable to exec query (while mapping to map): %v", err),
            }
        }
        return QueryResult[PT]{
            Ok:     true,
            AsMap:  results,
        }
    }
}
// endregion

// region funcs_exec_queries
// simple query (result returns as struct defined by T)
func DoRawQueryGetStruct[T any, PT interface {
    *T
    IEntity
}] (
    ctx context.Context,
    pool *pgxpool.Pool,
    q string,    
    args... any,
) ([]PT, error) {
    // trying to exec query
    rows, err := pool.Query(ctx, q, args...)
    if err != nil {        
        return nil, fmt.Errorf("Unable to exec query: %w", err)
    }
    // don't forget close the door...
    defer rows.Close()
    // use structs (service lvl. defined) ...
    res, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[T])
    // handle error (todo: debugMode)
    if err != nil {
        return nil, fmt.Errorf("Unable to exec query (while mapping to struct): %w", err)
    }
    // prepare target slice
    results := make([]PT, len(res))
    // fill it with entity pointers (`generic` cast)
    for n, e := range res {
        results[n] = PT(e) // use pointer instead of copy of entity
    }
    return results, nil    
}
// simple query (result returns as map[string]any w. type defined by PT)
func DoRawQueryGetMap[PT IEntity](
    ctx context.Context,
    pool *pgxpool.Pool,
    q string,
    args... any,
) ([]map[string]any, error) {
    rows, err := pool.Query(ctx, q, args...)
    if err != nil {
        return nil, fmt.Errorf("Unable to exec query: %w", err)
    }
    // don't forget close the door...
    defer rows.Close()
    results, err := pgx.CollectRows(rows, pgx.RowToMap)
    // handle error
    if err != nil {
        return nil, fmt.Errorf("Unable to exec query (while mapping to map): %w", err)
    }
    return results, nil
}
// func for simple delete based on PT type, returns affected rows
func DoBatchDelete[PT IEntity](
    ctx context.Context,
    pool *pgxpool.Pool,
    data []PT,
    ignoreUnsaved bool,
) (int64, error) {
    var err error
    // entity IDs counter
    if len(data) < 1 {
        return 0, fmt.Errorf("Empty set has been passed, Nothing to do ...")
    }
    IDs         := make([]string, len(data))
    // get related SQL tablename (based on last entity in set)
    tableName   := data[len(data)-1].GetTableName()
    // collect entities IDs
    for i, e := range data {
        id := e.GetID()
        if id < 1 && ignoreUnsaved {
            // just do nothing
            continue
        } else if id < 1 {
            // yell !
            return 0, fmt.Errorf("Unsaved entity detected... [index in passed set: %d]", i)
        }
        if tableName == "" {
            tableName = e.GetTableName()
        }
        IDs[i] = strconv.Itoa(id)
    }
    // prepare ids set
    idsStr      := strings.Join(IDs, ",")
    // prepare DELETE statement
    query       := fmt.Sprintf("DELETE FROM %s WHERE id IN (%s)", tableName, idsStr)
    // try exec
    cmdTag, err := pool.Exec(ctx, query)
    if err != nil {
        return 0, fmt.Errorf("Unable to DELETE passed set: %v", err)
    }
    // get affected rows
    ra          := cmdTag.RowsAffected()
    // clean up (like in method)
    for _, e := range data {
        e.SetID(0)
    }
    return ra, nil
}
// simple query (based on type defined behind PT), returs affected rows
func DoExecRawQuery[PT IEntity](
    ctx context.Context,
    pool *pgxpool.Pool,
    q string,
    args... any,
) (int64, error) {
    cmdTag, err := pool.Exec(ctx, q, args...)
    ra          := cmdTag.RowsAffected()
    if err != nil {
        return 0, fmt.Errorf("Unable to exec command: %w", err)
    }
    return ra, nil
}
// endregion

// region helpers
// helper for generating placeholders
func generatePlaceHolders(num int, startFrom int) string {
    if num <= 0 {
        return ""
    }
    if startFrom < 1 {
        startFrom = 1
    }
    result := ""
    for i := startFrom; i <= num + startFrom - 1 ; i++ {
        result += ("$" + strconv.Itoa(i) + ",")
    }
    return strings.TrimSuffix(result, ",")
}
// endregion

// region deprecated
// deprecated section
// deprecated
func BeginTx(ctx context.Context, pool *pgxpool.Pool) (context.Context, error) {
    ongoingTx := tryGetTxFromContext(ctx)
    if nil != ongoingTx {
        // Postgres TX detected so do nothing and silently return NIL
        return ctx, nil
    }
    tx, err := pool.Begin(ctx)
    if err != nil {
        return ctx, err
    }
    newCtx := context.WithValue(ctx, txKey{}, tx)
    return newCtx, nil
}

// deprecated
func RollbackTx(ctx context.Context, pool *pgxpool.Pool) (error) {
    ongoingTx := tryGetTxFromContext(ctx)
    if nil == ongoingTx {
        return nil
    }
    result := ongoingTx.Rollback(ctx)    
    return result
}
// endregion
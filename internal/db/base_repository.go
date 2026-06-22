package db

import (
	"context"
	"fmt"
	"log"
	"slices"
	dbc "test-api/internal/pkg/psqlservice"
)

const (
    RESULT_AS_MAP       uint16          = 0xFFFA
    RESULT_AS_STRUCT    uint16          = 0xFFFB
    PFOR_CREATE         uint16          = 1216
    PFOR_MODIFY         uint16          = 1218
    ORDER_ASC           string          = "asc"
    ORDER_DESC          string          = "desc"
)

/**
 * Modified version of Repository Pattern implementation
 * Tested on PGX.
 *
 * todo:
 *  -> test cover
 *  -> for remove-like methods - wrapper "BY ENTITY/ID/CRITERIA"
 *
 * Author: W.Dzieciol
 **/
type Repository[PT dbc.IEntity] struct {
    // PT - pointer for real IEntity
    // auto persist on find like methods
    AutoPersist			bool
    // realated table
    TableName           string
    // columns
    Columns             []string
    // columns allowed to p
    allowedForOrderBy   map[string]string // alias - realcolumn pair
    // db driver
    driver 				dbc.IDatabaseDriver[PT]
    // sets of entities stored in memory
    forCreate			[]PT  		// insert
    forModify			map[int]PT 	// update, delete (consider string in the future)
    lastInsertedIds		[]int
}

func (r *Repository[PT]) Find(ctx context.Context, id int) (PT, bool) {
    e, ok := r.driver.DoFindById(ctx, id)
    if !ok {
        return e, false
    }
    if r.AutoPersist {
        r.forModify[id] = e
    }	
    return e, true
}

func (r *Repository[PT]) FindBy(
    ctx context.Context,
    where map[string]string,
    order map[string]string,
    limit int,
    offset int,
) ([]PT, error) {
    if limit < 0 { limit = 0}
    if offset < 0 { offset = 0}
    criteria := map[string]map[string]string{}
    if where == nil {
        return make([]PT, 0), fmt.Errorf("Empty WHERE set. Check your syntax.")
    }
    criteria["where"] = where
    if order != nil {
        criteria["order"] = order
    }
    result, err := r.driver.DoFindByCriteria(ctx, criteria, limit, offset)
    if err != nil {
        return make([]PT, 0), err
    }
    r.collectDataInAutoPersistMode(result, PFOR_MODIFY)
    return result, nil
}

func (r *Repository[PT]) FindOneBy(
    ctx context.Context,
    where map[string]string,
) (PT, error) {
    var zPtr PT
    results, err := r.FindBy(ctx, where, nil, 1, 0)
    if err != nil {
        return zPtr, err
    }
    if len(results) == 0 {
        return zPtr, fmt.Errorf("Entity not found.")
    }
    r.collectDataInAutoPersistMode(results, PFOR_MODIFY)
    return results[0], nil
}

// todo: order BY
func (r *Repository[PT]) FindAll(
    ctx context.Context,
    limit int,
    offset int,
    orderBy string,
    direction string,
) ([]PT, error) {
    // just for protection
    if limit < 0 { limit = 0 }
    if offset < 0 { offset = 0 }
    
    result, err := r.driver.DoFindByRawQuery(
        ctx,
        "SELECT * FROM " + r.driver.GetRelatedTableName() + r.prepareOrderBy(orderBy, direction),
        limit,
        offset,        
    )
    if err != nil {
        return make([]PT, 0), err
    }
    r.collectDataInAutoPersistMode(result, PFOR_MODIFY)
    return result, nil
}

func (r *Repository[PT]) FindAllBy(
    ctx context.Context,
    criteria map[string]map[string]string,
    limit int,
    offset int,
) ([]PT, error) {
    if limit < 0 { limit = 0 }
    if offset < 0 { offset = 0 }
    result, err := r.driver.DoFindByCriteria(ctx, criteria, limit, offset)
    if err != nil {
        return make([]PT, 0), err
    }
    r.collectDataInAutoPersistMode(result, PFOR_MODIFY)
    return result, nil
}

func (r *Repository[PT]) ExecQuery(
    ctx context.Context,
    q string,
    args... any,
) (int64, error) {
    result := r.driver.DoExecRawQuery(ctx, q, args...)
    if !result.Ok {        
        return 0, fmt.Errorf("Unable to exec query: %s", result.ErrMsg)
    }
    return result.RowsAffected, nil
}

func (r *Repository[PT]) Query(
    ctx context.Context,
    q string,
    resType uint16,
    args... any,
) (any, error) {
    var res dbc.QueryResult[PT]
    var toReturn any

    switch resType {
    case RESULT_AS_STRUCT:
        res         = r.driver.DoRawQuery(ctx, q, true, args...)
        toReturn    = &res.AsStructPtr
    case RESULT_AS_MAP:
        res         = r.driver.DoRawQuery(ctx, q, false, args...)
        toReturn    = &res.AsMap
    default:
        res = dbc.QueryResult[PT]{Ok: false, ErrMsg: "Invalid return data type."}
    }
    if !res.Ok {        
        return nil, fmt.Errorf("Something went wrong: %s", res.ErrMsg)
    }

    return toReturn, nil
}

func (r *Repository[PT]) collectDataInAutoPersistMode(dataSet []PT, target uint16) {
    if (!r.AutoPersist || len(dataSet) < 1) {
        // nothing to do
        return
    }
    // otherwise
    switch (target) {
    case PFOR_CREATE:
        r.forCreate = append(r.forCreate, dataSet...)
    case PFOR_MODIFY:
        for _, e := range dataSet {
            r.forModify[e.GetID()] = e
        }
    }
    // in any other case just silently return
}

// just a wrapper
func (r *Repository[PT]) Persist(e PT) *Repository[PT] {
    data := make([]PT, 0)    
    return r.PersistSet(append(data, e))
}

// all
func (r *Repository[PT]) PersistSet(data []PT) *Repository[PT] {
    for _ , e := range data {
        id := e.GetID()
        if id == 0 {
            // looks like new entity
            r.forCreate = append(r.forCreate, e)
        } else {
            // exists in db
            r.forModify[id] = e
        }
    }
    return r
}

func (r *Repository[PT]) Flush(ctx context.Context) error {
    if len(r.forCreate) < 1 && len(r.forModify) < 1 {
        // nothing to do
        return nil
    }
    // flush in TX
    result := r.driver.RunInTransaction(ctx, func(contextWithTx context.Context) error {
        defer r.PurgeMem()
        if len(r.forCreate) > 0 {
            ids, err := r.driver.DoBatchInsertTx(contextWithTx, r.forCreate)
            if err != nil {
                return fmt.Errorf("ERR: %s", err)
            }
            r.lastInsertedIds = ids
        }
        // now check for modify (update in this case)
        if len(r.forModify) > 0 {
            // we have to copy pointers to temporary slice (order doesn't matter in this case)
            ptrs := r.convertForModifyToSlice()
            err := r.driver.DoBatchUpdateTx(contextWithTx, ptrs)
            if err != nil {
                return fmt.Errorf("ERR: %s", err)
            }
        }
        return nil
    })
    return result
}

func (r *Repository[PT]) Delete(ctx context.Context) (int64, error) {
    if len(r.forModify) < 1 {
        // nothing to do
        return 0, nil
    }
    ptrs := r.convertForModifyToSlice()
    affected, err := r.driver.DoBatchDelete(ctx, ptrs, true)
    if err != nil {
        return 0, err
    }
    return affected, nil
}

func (r *Repository[PT]) DeleteById(ctx context.Context, id int) (int64, error) {
    result := r.driver.DoExecRawQuery(ctx, "DELETE FROM $1 WHERE id = $2", r.TableName, id)
    if !result.Ok {
        return 0, fmt.Errorf("ERR: %s", result.ErrMsg)
    }
    return result.RowsAffected, nil
}

// save one (must be a pointer to entity)
func (r *Repository[PT]) Save(ctx context.Context, e PT) error {
    return r.Persist(e).Flush(ctx)
}

// save many (must be a slice of pointers to entities)
func (r *Repository[PT]) SaveAll(ctx context.Context, data []PT) error {
    return r.PersistSet(data).Flush(ctx)
}

func (r *Repository[PT]) PurgeMem() {
    r.forCreate = make([]PT, 0)
    r.forModify = make(map[int]PT)
}

func (r *Repository[PT]) convertForModifyToSlice() []PT {
    ptrs := make([]PT, len(r.forModify))
    idx := 0
    for _ , v := range r.forModify {
        ptrs[idx] = v
        idx++
    }
    return ptrs
}
// sets allowed columns for ORDER BY directive, then returns current Repository instance
func (r *Repository[PT]) SetAllowedCols(allowed map[string]string) *Repository[PT] {
    result := make(map[string]string)
    if len(allowed) == 0 {
        result = make(map[string]string)
        for _, col := range r.Columns {
            // alias     =   real_column
            result[col] = col
        }
    } else {
        // now we have to iterate over alias map created during/after repository build
        for alias, col := range allowed {
            if !slices.Contains(r.Columns, col) {
                log.Printf("Column `%s` doesn't exists. Alias: `%s` and will be omitted.", col, alias)
                // we have to omit this pair
                continue
            }
            result[alias] = col
        }
    }
    // set field
    r.allowedForOrderBy = result
    // for chain...
    return r
}
// VERY important because of security reasons
func (r *Repository[PT]) prepareOrderBy(colAlias string, direction string) (string) {
    if len(r.allowedForOrderBy) == 0 {
        r.SetAllowedCols(nil)
    }
    col, exists := r.allowedForOrderBy[colAlias]
    if !exists {
        // set to defaults
        col         = "id"
        direction   = ORDER_ASC
    }
    var directionStr string
    // set and validate direction string (asc, desc)
    switch direction {
    case ORDER_ASC: directionStr = "ASC"
    case ORDER_DESC: directionStr = "DESC"
    default: directionStr = "ASC" // just for sure
    }
    // prepare `order by` syntax
    oStx := fmt.Sprintf(" ORDER BY %s %s", col, directionStr)
    return oStx
}

// args: driver interface, model entity 
func NewRepository[PT dbc.IEntity] (db dbc.IDatabaseDriver[PT]) *Repository[PT] {
    return &Repository[PT] {
        driver:         db,
        AutoPersist:    true, // by default
        TableName:      db.GetRelatedTableName(),
        Columns:        db.GetColumnsNames(),
        forCreate:      make([]PT, 0),
        forModify:      make(map[int]PT),
    }
}


func (r *Repository[PT]) GetDriver() dbc.IDatabaseDriver[PT] {
    return r.driver
}

// helpers
func (r *Repository[PT]) GetDbEnv() dbc.DbEnv {
    return r.driver.GetDbEnv()
}

func (r *Repository[PT]) GetPool() dbc.DbEnv {
    return r.driver.GetDbEnv()
}

// counters (for profiler / tests)
func (r *Repository[PT]) GetNumForUpdate() int {
    return len(r.forModify)
}

func (r *Repository[PT]) GetNumForCreate() int {
    return len(r.forCreate)
}

package internal

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"	
	"strconv"
	"strings"
	"test-api/internal/db"
	dbc "test-api/internal/pkg/psqlservice"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type ProductService struct {
    Products 		    *db.Repository[*Product]
    BulkMode		    bool
    ExternalImportMode  bool
}

func GetProductService() *ProductService {
    return &ProductService{
        // build related repository
        Products: NewProductRepository(
            dbc.GetPsqlService[Product](GetEnv().GetConnPool()),
        ),
        // for massive data import
        BulkMode:               false,
        ExternalImportMode:     false,
    }
}
// toggle `bulk` mode
func (ps *ProductService) SetBulkMode(m bool) {
    ps.BulkMode = m
}
// toggle `external` import mode (allows system user to operate on specified user's / all products
// "out of" the current session
func (ps *ProductService) SetExternalImportMode(m bool) {
    ps.ExternalImportMode = m
}
// find by id
func (ps *ProductService) Find(ctx context.Context, id int) (*Product, bool) {
    return ps.Products.Find(ctx, id)
}
// find by id but owned by specified user
func (ps *ProductService) FindProductByOwner(
    ctx context.Context,
    u *User,
    id int,
) (*Product, bool) {
    sqlStr  := "SELECT * FROM " + ps.Products.TableName + " WHERE created_by = $1"
    pool    := GetEnv().GetConnPool()
    // exec custom query
    prods, err := dbc.DoRawQueryGetStruct[Product](ctx, pool, sqlStr, u.GetID())
    if err != nil || len(prods) != 1 {
        log.Printf("SQL Error while trying to fetch users's product: [%v]\n", err)
        return &Product{}, false
    }
    // get last
    p := prods[len(prods)-1]
    return p, true
}
// list products
func (ps *ProductService) List(
    ctx context.Context,
    orderBy string,
    direction string,
    limit int,
    offset int,
) ([]*Product, error) {
    dir := db.ORDER_ASC
    switch direction {
    case "asc":     dir = db.ORDER_ASC
    case "desc":    dir = db.ORDER_DESC
    }
    prods, err := ps.Products.FindAll(ctx, limit, offset, orderBy, dir)
    if err != nil {
        log.Println(err)
        return nil, ErrorFactory(ERR_GET_LIST, "Product")
    }
    return prods, nil
}
// list products of specified user
func (ps *ProductService) ListByOwner(
    ctx context.Context,
    u *User,
    limit int,
    offset int,
) ([]*Product, error) {
    criteria := map[string]map[string]string {
        "where": {"created_by =": strconv.Itoa(u.GetID())},
        "order": {"id" : "asc"},
    }
    prods, err := ps.Products.FindAllBy(ctx, criteria, limit, offset)
    if err != nil {
        log.Println(err)
        return nil, ErrorFactory(ERR_GET_LIST, "Product")
    }
    return prods, nil
}
// create
func (ps *ProductService) Create(
    ctx context.Context,
    name string,
    priceStr string,
    qty int,
    desc string,
    col string,
) (*Product, error) {
    // todo add validation
    as := GetAuthService()
    if qty < 0 { qty = 0 }
    // fetch logged user or system user in case of bulk import    
    updater, isLogged := as.GetLoggedUserPtr(ctx)    
    if ! isLogged {
        // bulk import? We need system user (bot)
        var ok bool
        if updater, ok = as.GetSystemUser(ctx); ! ok {
            return &Product{}, ErrorFactory(ERR_GET_SYSTEM_USER)            
        }
    }
    uID := updater.GetID()
    // creation time    
    cTime := time.Now().Unix()
    // prepare price
    priceInt, err := ps.pConvertToCents(priceStr)
    if err != nil {        
        return &Product{}, ErrorFactory(ERR_CONV_TOCENTS, err)
    }    
    // create new product in memory
    p := Product{
        Name:           name,
        Price:          priceInt,
        Qty:            qty,
        Description:    desc,
        Color:          col,
        CreatedAt:      cTime,
        UpdatedAt:      &cTime,
        CreatedBy:      uID,
        UpdatedBy:      &uID,
    }    
    if !ps.BulkMode {
        // generate fingerprint (product's checksum) - only in `standard-save` mode
        if _, err := ps.GenerateProductFingerprint(&p); err != nil {
            // log and return
            log.Println(err)
            return &p, ErrorFactory(ERR_GENERATE_FINGERPRINT)
        }
    }
    ps.Products.Persist(&p)
    if ps.BulkMode { // for bulk mode
        return &p, nil
    }
    // save
    if err := ps.Products.Flush(ctx); err != nil { // for traditional way
        return &Product{}, ErrorFactory(ERR_ENTITY_SAVE, "Product")
    }
    return &p, nil
}
// update
func (ps *ProductService) Update(
    ctx context.Context,
    p *Product,
    incomingData map[string]IncomingValue,
) (*Product, error) {
    if p.ID < 1 {
        // looks like unsaved product
        return p, ErrorFactory(ERR_ENTITY_UNSAVED, "Product")
    }
    var updator *User
    // fetch auth service
    as                  := GetAuthService()
    // set flags
    isModified          := false    
    // get logged user (editor/updator) or if not present - system user bot instead
    updator, err        := as.GetLoggedOrSystemUser(ctx, ps.ExternalImportMode)
    if err != nil {
        return &Product{}, err
    }
    // fetch updator ID
    uID := updator.GetID()
    // we have to check product owner (always true for privileged users) 
    if !ps.canOperateOnProduct(ctx, updator, p) {
        return &Product{}, ErrorFactory(ERR_INV_PROD_OWNER)
    }
    // modification time
    uTime := time.Now().Unix()
    // check values
    for k, iV  := range incomingData {
        switch k {
        case "name":
            if ApplyStrV(&p.Name, iV) { isModified = true }
        case "price":
            currPriceCents := ps.pConvertFromCents(p.Price)
            if ApplyStrV(&currPriceCents, iV) {
                isModified = true
                price, err := ps.pConvertToCents(iV.GetRawValue())
                if err != nil {                    
                    return &Product{}, ErrorFactory(ERR_CONV_TOCENTS, err)
                }
                // assert
                p.Price = price
            }
        case "qty":
            if ApplyNumV(&p.Qty, iV) { isModified = true }
        case "description":
            if ApplyStrV(&p.Description, iV) { isModified = true }
        case "color":
            if ApplyStrV(&p.Color, iV) { isModified = true }
        }
    }
    // check
    if isModified {
        p.UpdatedAt = &uTime
        p.UpdatedBy = &uID        
        if !ps.BulkMode {
            // re-generate fingerprint (product's checksum) in `standard` save mode           
            if _, err := ps.GenerateProductFingerprint(p); err != nil {
                // just log and continue
                log.Println(err)
            }
        }
        ps.Products.Persist(p)
        // for bulk mode... just return
        if ps.BulkMode {            
            return p, nil
        }
        // otherwise try save
        if err := ps.Products.Flush(ctx); err != nil {
            return &Product{}, ErrorFactory(ERR_ENTITY_SAVE, "Product")
        }
    }
    return p, nil
}
// removes product by passed entity
func (ps *ProductService) Remove(ctx context.Context, p *Product) error {
    if p.ID < 1 {
        return ErrorFactory(ERR_ENTITY_UNSAVED, "Product")
    }
    // fetch service
    as := GetAuthService()
    u, err := as.GetLoggedOrSystemUser(ctx, ps.ExternalImportMode)
    if err != nil {
        return err
    }
    // check if we can operate on this product
    if ! ps.canOperateOnProduct(ctx, u, p) {
        return ErrorFactory(ERR_INV_PROD_OWNER)
    }
    // do delete
    affRows, err := ps.Products.Persist(p).Delete(ctx)
    if err != nil {
        log.Println(err) // log
        return ErrorFactory(ERR_ENTITY_REMOVE, "Product")
    }
    log.Printf("Removed products: %v", affRows) // must be 1
    // looks ok
    return nil
}
// removes product by ID
func (ps *ProductService) RemoveById(ctx context.Context, id int) error {
    if p, ok := ps.Find(ctx, id); ok {
        return ps.Remove(ctx, p)
    }
    return ErrorFactory(ERR_ENTITY_NOT_FOUND, "Product")
}
// for massive dump (sync. mode)
// warning: this structure doesn't have something like "internal_id"
// thats why you can add same product many times
// If you read it in the future (btw. heavy storm outside... )
// keep in mind that you have to add to unique internal_id or something like that to the structure
func (ps *ProductService) DumpIntoDb(ctx context.Context) error {
    // todo: chunk
    if ! ps.BulkMode {
        // not in bulk mode, so nothing to do
        return nil
    }
    defer ps.SetBulkMode(false)
    if err := ps.Products.Flush(ctx); err != nil {
        log.Println(err)        
        return ErrorFactory(ERR_BULK_MODE_SAVE_FAILED)
    }
    // looks fine
    return nil
}
// Updates products fingerprints (spec. checksums) async.
// packetSize means records preparing for "one shot" to DB
// offset is just a start point...
// maxRecords to... process
//
// if maxRecords is smaller than 1 then then ALL records will be processed
// it can take a while...
// Resuming is possible. Just check (for example in logs) last offset
func (ps *ProductService) UpdateProductsFingerprints(
    ctx context.Context,
    packetSize int,
    offset int,
    maxRecords int,
    updateAll bool,
) error {
    // todo: custom errors
    counter         := 0
    // batch size
    numOfRecords    := packetSize
    // done chan
    done            := make(chan int)
    // search criteria (pointed to not updated products where checksum IS NULL)
    criteria        := map[string]map[string]string {
        "where": {"checksum IS": "null"}, "order": {"id": "asc"},
    }    
    defer close(done) // do not forget...

    // main loop
    var err error
    for {
        var processedProducts []*Product
        var products []*Product
        // find packet
        if updateAll {
            // get all products
            products, err = ps.Products.FindAll(ctx, numOfRecords, offset, "id", "asc")
        } else {
            // get products with NULL checksums
            products, err = ps.Products.FindAllBy(ctx, criteria, numOfRecords, offset)
	    }
        if err != nil {
            log.Println(err)
            // stop on db error            
            return fmt.Errorf("Can't fetch next set of product entity.")
        }
        // no more products for processing
        if len(products) == 0 {
            break
        }
        // generate in-stream chan. (of Products)
	    inStream    := productStream(done, products)
        // time to fine out over workers (done chan, inStrem<products>, worker delegate)
        workers     := fanOut(done, inStream, ps.generateProductFingerprintAsync)
        // new product stream (after fanIn)
        prodStream  := fanIn(done, workers...)
        // add to results
        for p := range prodStream {
            processedProducts = append(processedProducts, p)
        }
        log.Printf("Rows to update fingerprint: %d\n", ps.Products.GetNumForUpdate())
        // try save next part
        // Flush is not concurrent-safe, so DO NOT change it to: go func() { err := ps.Products.Flush(ctx) ... }()
        if err = ps.Products.Flush(ctx); err != nil {
            // looks like something went wrong
            log.Println(err)
            return fmt.Errorf("Flush failed while UPDATE product's fingerprint. Check logs.")
        }
        log.Println("Done.")
        counter += len(processedProducts)
        processedProducts = nil
        // next page
        offset += numOfRecords
        log.Printf("Current offset: %d\n", offset)
        if maxRecords > 0 && counter >= maxRecords {
            // defined limit reached ?
            log.Printf("Limit was set after reach [%d] records. \n", maxRecords)
            break
        }
    }
    return nil 
}
// for single product
func (ps *ProductService) GenerateProductFingerprint(p *Product) (*Product, error) {
	rawData := fmt.Sprintf(
        "%d|%s|%d|%d|SECSALT2026",
        p.GetID(),
        p.Name,
        p.Price,
        p.CreatedAt,
    )
	// Pre-hashing for blowfish algo
	shaHash := sha256.Sum256([]byte(rawData))
	hexStr  := hex.EncodeToString(shaHash[:])
	// then blowfish algo in `extreme` mode
	hash, err := bcrypt.GenerateFromPassword([]byte(hexStr), 14)
	if err != nil {
        return p, err
	}
    // otherwise
    hashStr := string(hash) // convert to string
    p.Checksum = &hashStr   // use ptr
    return p, nil
}
// async mode for fanIn / fanOut pattern (or just select/done)
func (ps *ProductService) generateProductFingerprintAsync(
    done <-chan int,
    prodStream <-chan *Product,
) <-chan *Product {
    // create new channel
    processed := make(chan *Product)
    go func() {
        defer close(processed)
        for {
            select {
            case <-done:
                return
            case p, ok := <-prodStream:
                if !ok {
                    // !ok -> it means prodstream is already close, end of data packet -> worker can die
                    return
                }
                // use "standard func"
                p, err := ps.GenerateProductFingerprint(p)
                if err != nil {
                    // todo: send it to err chan
                    continue
                }
                select {
                case <-done:
                    return
                case processed <-p:
                }
            }
        }
    }()
    return processed
}
// fan-in-out helper - creates product stream chan as a source data
func productStream(done <-chan int, productSet []*Product) <-chan *Product {
	s := make(chan *Product) // product stream chan
	go func() {
		defer close(s)
        for _, p := range productSet {
			select {
			case <-done:
				return
			case s <- p:
			}
        }
	}()
	return s
}
// checks if specified user (usable for checks against logged users) can operate on this product
func (ps *ProductService) canOperateOnProduct(
    ctx context.Context,
    u *User,
    p *Product,
) bool {
    // privileged users (e.g. admins, system-bot)
    if GetUserService().IsPrivileged(ctx, u) {
        return true
    }
    // otherwise
    return u.GetID() == p.CreatedBy     
}
// convert from common `currency` format to cents (integer val.)
func (ps *ProductService) pConvertToCents(rawVal string) (int64, error) {
    // trim whitespaces
    str := strings.TrimSpace(rawVal)
    if str == "" { return 0, nil }
    // replace potential ","" to "."
    str = strings.ReplaceAll(str, ",", ".")
    // split
    sArr := strings.Split(str, ".")
    if len(sArr) > 2 { return 0, fmt.Errorf("Invalid number format.")}
    // integer part of value
    intPartStr := sArr[0]
    if intPartStr == "" {
        intPartStr = "0"
    }
    intPart, err := strconv.ParseInt(intPartStr, 10, 64)
    if err != nil {
        return 0, fmt.Errorf("Num. parse error [int. part.]")
    }
    // decimal part of value
    decPartStr := "00" // set our "base"
    if len(sArr) == 2 {
        decPartStr = sArr[1]
        // cut to 2
        if len(decPartStr) > 2 {
            decPartStr = decPartStr[:2]
        // otherwise add one more zero (e.g 8.1 to 8.10)
        } else if len(decPartStr) == 1 {
            decPartStr += "0"
        }
    }
    // time to convert
    decPart, err := strconv.ParseInt(decPartStr, 10, 64)
    if err != nil {
        return 0, fmt.Errorf("Num. parse error [dec. part.]")
    }
    // finally...
    return (intPart * 100) + decPart, nil
}
// conv. from cents to `typical` currency format
func (ps *ProductService) pConvertFromCents(num int64) string {
    sign := ""
    // just in case
    if num < 0 {
        sign = "-"
        num = -num // conv. to positive val.
    }
    intPart     := num / 100
    remainder   := num % 100
    // format and return
    return fmt.Sprintf("%s%d.%02d", sign, intPart, remainder)
}
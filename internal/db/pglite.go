package db

import (
	"database/sql"
	"fmt"
	"net"
	"strings"

	"github.com/jackc/pgx/v5/pgproto3"
	_ "github.com/lib/pq"

	"github.com/qiangli/ai/internal/log"
)

type PgLiteBackend struct {
	backend   *pgproto3.Backend
	conn      net.Conn
	responder func(*pgproto3.Backend, *sql.DB, string)
	dbname    string

	db                 *sql.DB
	preparedStatements map[string]string
	activePortals      map[string]*activePortal
}

type activePortal struct {
	statementName string
	params        []interface{}
}

func NewPgLiteBackend(conn net.Conn, dbname string, responder func(*pgproto3.Backend, *sql.DB, string)) *PgLiteBackend {
	backend := pgproto3.NewBackend(conn, conn)
	connHandler := &PgLiteBackend{
		backend:            backend,
		conn:               conn,
		responder:          responder,
		dbname:             dbname,
		preparedStatements: make(map[string]string),
		activePortals:      make(map[string]*activePortal),
	}

	return connHandler
}

func (p *PgLiteBackend) Run() error {
	defer p.Close()

	err := p.handleStartup()
	if err != nil {
		return err
	}

	for {
		msg, err := p.backend.Receive()
		if err != nil {
			return fmt.Errorf("error receiving message: %w", err)
		}

		log.Debugf("Received message %v\n", msg)

		switch qmsg := msg.(type) {
		case *pgproto3.Query:
			query := qmsg.String
			log.Debugf("Received query: %s", query)
			p.responder(p.backend, p.db, query)
			continue
		case *pgproto3.Terminate:
			if p.db != nil {
				p.db.Close()
				p.db = nil
			}
			log.Debugf("Client terminated connection")
			return nil
		case *pgproto3.Parse:
			name := qmsg.Name
			query := qmsg.Query
			fmt.Printf("Received Parse: %v, Name: %s\n", query, name)
			p.preparedStatements[name] = query
			p.backend.Send(&pgproto3.ParseComplete{})
			p.backend.Flush()
			continue
		case *pgproto3.Bind:
			portalName := qmsg.DestinationPortal
			statementName := qmsg.PreparedStatement

			// Check if the prepared statement exists
			if _, exists := p.preparedStatements[statementName]; !exists {
				sendError(p.backend, fmt.Errorf("prepared statement not found: %s", statementName))
				continue
			}

			// Convert parameter values
			paramValues := make([]interface{}, len(qmsg.Parameters))
			for i, paramVal := range qmsg.Parameters {
				paramValues[i] = string(paramVal)
			}

			// Store the portal information
			p.activePortals[portalName] = &activePortal{statementName: statementName, params: paramValues}

			p.backend.Send(&pgproto3.BindComplete{})
			p.backend.Flush()

		case *pgproto3.Execute:
			portalName := qmsg.Portal
			portal, exists := p.activePortals[portalName]
			if !exists {
				sendError(p.backend, fmt.Errorf("portal not found: %s", portalName))
				continue
			}

			query := p.preparedStatements[portal.statementName]

			executeQuery(p.backend, p.db, query, portal.params)

			// Optionally, you can remove the portal once executed if it's single-use
			// delete(p.activePortals, portalName)
		case *pgproto3.Describe:
			name := qmsg.Name
			objectType := qmsg.ObjectType
			fmt.Printf("Received Describe: object type %v, name %v\n", objectType, name)

			switch objectType {
			case 'S': // Statement
				if query, exists := p.preparedStatements[name]; exists {
					// Determine fields based on the actual query analysis
					fields, err := getQueryMetadata(p.db, query)
					if err != nil {
						sendError(p.backend, err)
					} else {
						p.backend.Send(&pgproto3.RowDescription{Fields: fields})
						continue
					}
				} else {
					sendError(p.backend, fmt.Errorf("prepared statement not found: %s", name))
				}
			case 'P': // Portal
				// For simplicity, return no data as we're not managing portals here
				p.backend.Send(&pgproto3.NoData{})
			}

			p.backend.Send(&pgproto3.CommandComplete{CommandTag: []byte("DESCRIBE")})
			p.backend.Flush()
		case *pgproto3.Sync:
			log.Debugf("Received Sync")

			// Send a ReadyForQuery response, indicating processing is complete
			p.backend.Send(&pgproto3.ReadyForQuery{TxStatus: 'I'})
			p.backend.Flush()
		case *pgproto3.Close:
			objectType := qmsg.ObjectType
			name := qmsg.Name
			log.Debugf("Received Close: object type %c, name %v\n", objectType, name)

			// Perform the close operation based on the type
			switch objectType {
			case 'S': // Statement
				if _, exists := p.preparedStatements[name]; exists {
					delete(p.preparedStatements, name)
					log.Debugf("Closed statement: %s", name)
				}
			case 'P': // Portal
				if _, exists := p.activePortals[name]; exists {
					delete(p.activePortals, name)
					log.Debugf("Closed portal: %s", name)
				}
			}

			// Send CloseComplete message
			p.backend.Send(&pgproto3.CloseComplete{})
			p.backend.Flush()
		case *pgproto3.Flush:
			// Handle Flush message
			log.Debugf("Received Flush")
			p.backend.Flush()
		default:
			return fmt.Errorf("received message other than Query from client: %#v", msg)
		}
	}
}

// Helper function to retrieve query metadata
func getQueryMetadata(db *sql.DB, query string) ([]pgproto3.FieldDescription, error) {
	// This function should prepare a statement, execute an empty query to get metadata, and return FieldDescriptions

	stmt, err := db.Prepare(query)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	// Assuming we execute to just get the schema
	rows, err := stmt.Query()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols, err := rows.ColumnTypes()
	if err != nil {
		return nil, err
	}

	fields := make([]pgproto3.FieldDescription, len(cols))
	for i, col := range cols {
		fields[i] = pgproto3.FieldDescription{
			Name:        []byte(col.Name()),
			DataTypeOID: 25, // Simplified example; derive OID from actual type
			// ... more attributes if needed
		}
	}
	return fields, nil
}

func (p *PgLiteBackend) handleStartup() error {
	// &{196608 map[client_encoding:UTF8 database:postgres user:postgres]}
	// &{196608 map[application_name:psql client_encoding:SQL_ASCII database:postgres user:postgres]}
	// &{196608 map[client_encoding:UTF8 database:postgres user:postgres]}
	msg, err := p.backend.ReceiveStartupMessage()
	if err != nil {
		return fmt.Errorf("error receiving startup message: %w", err)
	}

	log.Debugf("Received startup message %v\n", msg)

	auth := func() (*pgproto3.PasswordMessage, error) {
		// Request password from client
		authReq := &pgproto3.AuthenticationCleartextPassword{}
		buf := mustEncode(authReq.Encode(nil))
		_, err = p.conn.Write(buf)
		if err != nil {
			return nil, fmt.Errorf("error sending password request: %w", err)
		}
		p.backend.Flush()
		// Receive password
		pwdMsg, err := p.backend.Receive()
		if err != nil {
			return nil, fmt.Errorf("error receiving password message: %w", err)
		}
		pwd, ok := pwdMsg.(*pgproto3.PasswordMessage)
		if !ok {
			return nil, fmt.Errorf("expected PasswordMessage, got: %#v", pwdMsg)
		}
		return pwd, nil
	}

	// TODO
	// validate password
	validate := func(pwd string) bool {
		return pwd == "password"
	}

	switch smsg := msg.(type) {
	case *pgproto3.StartupMessage:
		user := smsg.Parameters["user"]
		dbname := smsg.Parameters["database"]
		if dbname == "" {
			dbname = p.dbname
		}
		// "pgAdmin 4 - DB:postgres"
		// "psql"
		appname := smsg.Parameters["application_name"]
		host := smsg.Parameters["host"]
		if host == "" {
			host = "localhost"
		}
		password := smsg.Parameters["password"]
		if password == "" {
			pwd, err := auth()
			if err != nil {
				sendError(p.backend, err)
				return err
			}

			if !validate(pwd.Password) {
				sendFatal(p.backend, fmt.Errorf("password authentication failed for user %q", user))
				return fmt.Errorf("authentication failed: invalid password")
			}

			password = pwd.Password
		}

		//
		dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s sslmode=disable", host, user, password, dbname)
		log.Debugf("startup message %s from %s\n", dsn, appname)
		db, err := sql.Open("postgres", dsn)
		if err != nil {
			return err
		}
		p.db = db

		buf := mustEncode((&pgproto3.AuthenticationOk{}).Encode(nil))
		buf = mustEncode((&pgproto3.ReadyForQuery{TxStatus: 'I'}).Encode(buf))
		_, err = p.conn.Write(buf)
		if err != nil {
			return fmt.Errorf("error sending ready for query: %w", err)
		}
	case *pgproto3.SSLRequest:
		_, err = p.conn.Write([]byte("N"))
		if err != nil {
			return fmt.Errorf("error sending deny SSL request: %w", err)
		}
		return p.handleStartup()
	case *pgproto3.GSSEncRequest:
		log.Debugf("Received GSSEncRequest message")
		// Respond with 'N' to indicate that GSS encryption is not supported
		_, err = p.conn.Write([]byte("N"))
		if err != nil {
			return fmt.Errorf("error sending deny GSSEncRequest: %w", err)
		}
		return p.handleStartup()
	case *pgproto3.CancelRequest:
		log.Debugf("Received CancelRequest message")
		return p.Close()
	default:
		return fmt.Errorf("unknown startup message: %v", smsg)
	}

	return nil
}

func (p *PgLiteBackend) Close() error {
	if p.conn != nil {
		err := p.conn.Close()
		p.conn = nil
		return err
	}
	return nil
}

func mustEncode(buf []byte, err error) []byte {
	if err != nil {
		panic(err)
	}
	return buf
}

func handleQuery(be *pgproto3.Backend, db *sql.DB, query string) {
	query = strings.TrimSpace(query)

	Q := strings.ToUpper(query)
	switch {
	case strings.HasPrefix(Q, "SET"):
		result, err := db.Exec(query)
		if err != nil {
			sendError(be, err)
			return
		}
		affeted, _ := result.RowsAffected()
		resp := fmt.Sprintf("%v rows affected", affeted)
		be.Send(&pgproto3.CommandComplete{CommandTag: []byte(resp)})
	case strings.HasPrefix(Q, "SELECT"):
		executeQuery(be, db, query, nil)
	default:
		// For non-select queries, just execute
		result, err := db.Exec(query)
		if err != nil {
			sendError(be, err)
			return
		}
		affeted, _ := result.RowsAffected()
		resp := fmt.Sprintf("%v rows affected", affeted)
		be.Send(&pgproto3.CommandComplete{CommandTag: []byte(resp)})
	}

	be.Send(&pgproto3.ReadyForQuery{TxStatus: 'I'})
	be.Flush()
}

func executeQuery(be *pgproto3.Backend, db *sql.DB, query string, params []any) {
	stmt, err := db.Prepare(query)
	if err != nil {
		sendError(be, err)
		return
	}
	defer stmt.Close()
	//
	log.Debugf("query: %v\n", query)
	var rows *sql.Rows

	if len(params) > 0 {
		rows, err = stmt.Query(params)
	} else {
		rows, err = stmt.Query()
	}
	if err != nil {
		sendError(be, err)
		return
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		sendError(be, err)
		return
	}
	// Send RowDescription
	fields := make([]pgproto3.FieldDescription, len(cols))
	for i, col := range cols {
		fields[i] = pgproto3.FieldDescription{
			Name:                 []byte(col),
			TableOID:             0,
			TableAttributeNumber: 0,
			DataTypeOID:          25, // TEXT oid
			DataTypeSize:         -1,
			TypeModifier:         -1,
			Format:               0, // text format
		}
	}
	be.Send(&pgproto3.RowDescription{Fields: fields})

	rowCount := 0
	for rows.Next() {
		values, err := rowsValues(rows, len(cols))
		if err != nil {
			sendError(be, err)
			return
		}
		be.Send(&pgproto3.DataRow{Values: values})
		rowCount++
	}
	if err := rows.Err(); err != nil {
		sendError(be, err)
		return
	}

	be.Send(&pgproto3.CommandComplete{CommandTag: []byte(fmt.Sprintf("SELECT %d", rowCount))})
}

func rowsValues(rows *sql.Rows, numCols int) ([][]byte, error) {
	dest := make([]interface{}, numCols)
	for i := range dest {
		dest[i] = new(interface{})
	}

	if err := rows.Scan(dest...); err != nil {
		return nil, err
	}

	result := make([][]byte, numCols)
	for i, val := range dest {
		if b, ok := (*(val.(*interface{}))).([]byte); ok {
			result[i] = b
		} else {
			result[i] = []byte(fmt.Sprintf("%v", *(val.(*interface{}))))
		}

		log.Debugf("row[%v]: %s\n", i, string(result[i]))
	}

	return result, nil
}

func sendError(be *pgproto3.Backend, err error) {
	be.Send(&pgproto3.ErrorResponse{
		Severity: "ERROR",
		Message:  err.Error(),
	})
	be.Send(&pgproto3.ReadyForQuery{TxStatus: 'I'})
	be.Flush()
}

func sendFatal(be *pgproto3.Backend, err error) {
	be.Send(&pgproto3.ErrorResponse{
		Severity: "FATAL",
		Message:  err.Error(),
	})
	be.Send(&pgproto3.ReadyForQuery{TxStatus: 'I'})
	be.Flush()
}

func commandTag(query string) string {
	q := strings.ToUpper(query)
	if strings.HasPrefix(q, "SELECT") {
		return fmt.Sprintf("SELECT %d", 0) // Row count not tracked here
	} else if strings.HasPrefix(q, "INSERT") {
		return "INSERT 0 1"
	} else if strings.HasPrefix(q, "UPDATE") {
		return "UPDATE 0"
	} else if strings.HasPrefix(q, "DELETE") {
		return "DELETE 0"
	} else {
		return "COMMAND"
	}
}

func information_schema(query string) bool {
	if strings.Contains(query, "information_schema") {
		log.Debugf("Query is related to information_schema: %s", query)
		return true
	}
	return false
}

var schemaMap = map[string][]string{
	// PostgreSQL default schemas and tables
	"pg_catalog":         {},
	"information_schema": {},
	// Add more default system tables if necessary
	"versioning_info": {"version_id", "version_name", "applied_on"},
	"user_account":    {"user_id", "username", "email", "created_at"},
	"table_metadata":  {"table_id", "table_name", "schema_name", "created_at"},
}

func initializeSchemaMap(db *sql.DB) error {
	// schemaMap will be populated with user-defined tables and columns from SQLite
	// via the `initializeSchemaMap` function.

	// Initialize dynamic schemaMap using the SQLite database
	rows, err := db.Query(`
		SELECT m.name AS table_name, p.name AS column_name
		FROM sqlite_master m
		JOIN pragma_table_info((m.name)) p
		WHERE m.type='table'
		ORDER BY m.name, p.cid;
	`)
	if err != nil {
		return fmt.Errorf("error retrieving schema: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var tableName, columnName string
		if err := rows.Scan(&tableName, &columnName); err != nil {
			return fmt.Errorf("error scanning schema row: %w", err)
		}
		schemaMap[tableName] = append(schemaMap[tableName], columnName)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating schema rows: %w", err)
	}

	return nil
}

func StartPG(address string, dbname string) {
	// conn, err := sql.Open("file::memory:?mode=memory&cache=shared")
	// db, err := sql.Open("sqlite", "test.db")
	// db, err := sql.Open("postgres", "host=localhost user=postgres password=password dbname=postgres sslmode=disable")
	// if err != nil {
	// 	log.Fatalf("failed to open database: %v", err)
	// }

	// defer db.Close()

	// err = initializeSchemaMap(db)
	// if err != nil {
	// 	log.Fatalf("failed to initialize schema map: %v", err)
	// }

	ln, err := net.Listen("tcp", address)
	if err != nil {
		log.Errorf("failed to listen on %s: %v\n", address, err)
		return
	}
	log.Infof("Postgres listening on %s...\n", address)

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Debugf("failed to accept connection: %v\n", err)
			continue
		}

		log.Debugf("Accepted connection from: %v\n", conn.RemoteAddr())

		pg := NewPgLiteBackend(conn, dbname, func(be *pgproto3.Backend, db *sql.DB, query string) {
			handleQuery(be, db, query)
		})

		go func() {
			err := pg.Run()
			if err != nil {
				log.Debugln(err)
			}
			log.Debugf("Closed connection from %v\n", conn.RemoteAddr())
		}()
	}
}
